/*
2020 © Postgres.ai
*/

// Package retrieval provides data retrieval pipeline.
package retrieval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/snapshot"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/status"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"

	dblabCfg "gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	parseOption           = cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow
	refreshJobs  jobGroup = "refresh"
	snapshotJobs jobGroup = "snapshot"

	pendingFilename = "pending.retrieval"
)

var errNoJobs = errors.New("no jobs to snapshot pool data")

type jobGroup string

// Retrieval describes a data retrieval.
type Retrieval struct {
	Scheduler    Scheduler
	State        State
	imageState   *db.ImageContent
	cfg          *config.Config
	global       *global.Config
	engineProps  global.EngineProps
	docker       *client.Client
	poolManager  *pool.Manager
	tm           *telemetry.Agent
	runner       runners.Runner
	ctxCancel    context.CancelFunc
	statefulJobs []components.JobRunner
}

// Scheduler defines a refresh scheduler.
type Scheduler struct {
	Cron *cron.Cron
	Spec cron.Schedule
}

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, engineProps global.EngineProps, docker *client.Client, pm *pool.Manager, tm *telemetry.Agent,
	runner runners.Runner) (*Retrieval, error) {
	r := &Retrieval{
		global:      &cfg.Global,
		engineProps: engineProps,
		docker:      docker,
		poolManager: pm,
		tm:          tm,
		runner:      runner,
		State: State{
			Status: models.Inactive,
			alerts: make(map[models.AlertType]models.Alert),
		},
		imageState: db.NewImageContent(engineProps),
	}

	retrievalCfg, err := ValidateConfig(&cfg.Retrieval)
	if err != nil {
		return nil, err
	}

	r.setup(retrievalCfg)

	if err := checkPendingMarker(r); err != nil {
		return nil, fmt.Errorf("failed to check pending marker: %w", err)
	}

	return r, nil
}

// ImageContent provides the content of foundation Docker image.
func (r *Retrieval) ImageContent() *db.ImageContent {
	return r.imageState
}

func checkPendingMarker(r *Retrieval) error {
	pendingPath, err := util.GetMetaPath(pendingFilename)
	if err != nil {
		return fmt.Errorf("failed to build pending filename: %w", err)
	}

	if _, err := os.Stat(pendingPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("failed to get pending file info: %w", err)
	}

	r.State.Status = models.Pending

	return nil
}

// RemovePendingMarker removes the file from the metadata directory which specifies that retrieval is pending.
func (r *Retrieval) RemovePendingMarker() error {
	pending, err := util.GetMetaPath(pendingFilename)
	if err != nil {
		return fmt.Errorf("failed to build pending filename: %w", err)
	}

	if err := os.Remove(pending); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	r.State.Status = models.Inactive

	return nil
}

// Reload reloads retrieval configuration.
func (r *Retrieval) Reload(ctx context.Context, retrievalCfg *config.Config) {
	r.setup(retrievalCfg)
	r.reloadStatefulJobs()
	r.stopScheduler()
	r.setupScheduler(ctx)
}

func (r *Retrieval) setup(retrievalCfg *config.Config) {
	r.cfg = retrievalCfg

	r.defineRetrievalMode()
}

func (r *Retrieval) reloadStatefulJobs() {
	for _, job := range r.statefulJobs {
		cfg, ok := r.cfg.JobsSpec[job.Name()]
		if !ok {
			log.Msg("Skip reloading of the stateful retrieval job. Spec not found", job.Name())
			continue
		}
		// todo should we remove if jobs are not there ?
		// todo should we check for completion before ?
		if err := job.Reload(cfg.Options); err != nil {
			log.Err("Failed to reload configuration of the retrieval job", job.Name(), err)
		}
	}
}

// Run start retrieving process.
func (r *Retrieval) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	r.ctxCancel = cancel

	log.Msg("Retrieval mode:", r.State.Mode)

	if err := r.collectFoundationImageContent(); err != nil {
		return fmt.Errorf("failed to collect content lists from the foundation Docker image of the logicalDump job: %w", err)
	}

	if r.cfg.Refresh != nil && r.cfg.Refresh.SkipStartRefresh {
		log.Msg("Continue without performing initial data refresh because the `skipStartRefresh` option is enabled")
		r.setupScheduler(ctx)

		return nil
	}

	fsManager, err := r.getNextPoolToDataRetrieving()
	if err != nil {
		var skipError *SkipRefreshingError
		if errors.As(err, &skipError) {
			r.State.Status = models.Finished

			log.Msg("Continue without performing a full refresh:", skipError.Error())
			r.setupScheduler(ctx)

			return nil
		}

		alert := telemetry.Alert{
			Level:   models.RefreshFailed,
			Message: "Pool to perform data refresh not found",
		}
		r.State.Status = models.Failed
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)

		return fmt.Errorf("failed to choose pool to refresh: %w", err)
	}

	log.Msg("Pool to perform data retrieving: ", fsManager.Pool().Name)

	if r.State.Status == models.Pending {
		log.Msg("Data retrieving suspended because Retrieval state is pending")

		return nil
	}

	if err := r.run(runCtx, fsManager); err != nil {
		r.State.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: err.Error()})
		// Build a generic message to avoid sending sensitive data.
		r.tm.SendEvent(ctx, telemetry.AlertEvent, telemetry.Alert{Level: models.RefreshFailed,
			Message: fmt.Sprintf("Failed to perform initial data retrieving: %s", r.State.Mode)})

		return err
	}

	r.setupScheduler(ctx)

	return nil
}

func (r *Retrieval) collectFoundationImageContent() error {
	if _, ok := r.cfg.JobsSpec[logical.DumpJobType]; !ok {
		if r.State.Mode == models.Logical {
			log.Msg("logicalDump job is not enabled. Docker image extensions and locales will not be checked")
		}

		return nil
	}

	dumpOptions := &logical.DumpOptions{}

	if err := r.JobConfig(logical.DumpJobType, &dumpOptions); err != nil {
		return fmt.Errorf("failed to get config of %s job: %w", logical.DumpJobType, err)
	}

	if err := r.imageState.Collect(dumpOptions.DockerImage); err != nil {
		return err
	}

	// Collect a list of databases mentioned in the Retrieval config. An empty list means all databases.
	dbs := make([]string, 0)

	if len(dumpOptions.Databases) != 0 {
		dbs = append(dbs, collectDBList(dumpOptions.Databases)...)

		restoreOptions := &logical.RestoreOptions{}

		if err := r.JobConfig(logical.RestoreJobType, &restoreOptions); err == nil && len(restoreOptions.Databases) != 0 {
			dbs = append(dbs, collectDBList(restoreOptions.Databases)...)
		}
	}

	r.imageState.SetDatabases(dbs)

	return nil
}

func collectDBList(definitions map[string]logical.DumpDefinition) []string {
	dbs := []string{}

	for dbName := range definitions {
		dbs = append(dbs, dbName)
	}

	return dbs
}

func (r *Retrieval) getNextPoolToDataRetrieving() (pool.FSManager, error) {
	firstPool := r.poolManager.First()
	if firstPool == nil {
		return nil, errors.New("no available pools")
	}

	if firstPool.Pool().Status() == resources.EmptyPool {
		return firstPool, nil
	}

	// For physical or unknown modes, changing the pool is possible only by the refresh timetable.
	if r.State.Mode != models.Logical {
		return firstPool, nil
	}

	// For logical mode try to find another pool to avoid rewriting prepared data.
	elementToRefresh := r.poolManager.GetPoolToUpdate()

	if elementToRefresh == nil || elementToRefresh.Value == nil {
		if firstPool.Pool().Status() == resources.ActivePool {
			return nil, NewSkipRefreshingError("pool to refresh not found, but the current pool is active")
		}

		return nil, errors.New("pool to perform data refresh not found")
	}

	poolToRefresh, err := r.poolManager.GetFSManager(elementToRefresh.Value.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to get FSManager: %w", err)
	}

	return poolToRefresh, nil
}

func (r *Retrieval) run(ctx context.Context, fsm pool.FSManager) (err error) {
	// Check the pool aliveness.
	if _, err := fsm.GetFilesystemState(); err != nil {
		return errors.Wrap(errors.Unwrap(err), "filesystem manager is not ready")
	}

	poolName := fsm.Pool().Name
	poolElement := r.poolManager.GetPoolByName(poolName)

	if poolElement == nil {
		return errors.Errorf("pool %s not found", poolName)
	}

	if err := r.RefreshData(ctx, poolName); err != nil {
		return err
	}

	if r.State.Status == models.Renewed {
		r.State.cleanAlerts()
	}

	if err := r.SnapshotData(ctx, poolName); err != nil && err != errNoJobs {
		return err
	}

	if r.State.Status == models.Finished {
		r.poolManager.MakeActive(poolElement)
		r.State.cleanAlerts()
	}

	if err := fsm.InitBranching(); err != nil {
		return fmt.Errorf("failed to init branching: %w", err)
	}

	return nil
}

// RefreshData runs a group of data refresh jobs.
func (r *Retrieval) RefreshData(ctx context.Context, poolName string) error {
	fsm, err := r.poolManager.GetFSManager(poolName)
	if err != nil {
		return fmt.Errorf("failed to get %q FSManager: %w", poolName, err)
	}

	if r.State.Status == models.Refreshing || r.State.Status == models.Snapshotting {
		return fmt.Errorf("skip refreshing the data because the pool is still busy: %s", r.State.Status)
	}

	jobs, err := r.buildJobs(fsm, refreshJobs)
	if err != nil {
		return fmt.Errorf("failed to build refresh jobs for %s: %w", poolName, err)
	}

	if len(jobs) == 0 {
		log.Dbg("no jobs to refresh pool:", fsm.Pool())
		return nil
	}

	log.Dbg("Refreshing data pool: ", fsm.Pool())

	fsm.Pool().SetStatus(resources.RefreshingPool)

	r.State.Status = models.Refreshing
	r.State.LastRefresh = models.NewLocalTime(time.Now().Truncate(time.Second))

	defer func() {
		r.State.Status = models.Renewed

		if err != nil {
			r.State.Status = models.Failed
			r.State.addAlert(telemetry.Alert{
				Level:   models.RefreshFailed,
				Message: err.Error(),
			})

			fsm.Pool().SetStatus(resources.EmptyPool)
		}

		r.State.CurrentJob = nil
	}()

	if r.State.Mode == models.Logical {
		if err := preparePoolToRefresh(fsm, r.runner); err != nil {
			return fmt.Errorf("failed to prepare pool for initial refresh: %w", err)
		}
	}

	for _, j := range jobs {
		r.State.CurrentJob = j

		if err = j.Run(ctx); err != nil {
			return err
		}
	}

	r.State.CurrentJob = nil

	return nil
}

// SnapshotData runs a group of data snapshot jobs.
func (r *Retrieval) SnapshotData(ctx context.Context, poolName string) error {
	fsm, err := r.poolManager.GetFSManager(poolName)
	if err != nil {
		return fmt.Errorf("failed to get %q FSManager: %w", poolName, err)
	}

	if r.State.Status != models.Inactive && r.State.Status != models.Renewed && r.State.Status != models.Finished {
		return fmt.Errorf("pool is not ready to take a snapshot: %s", r.State.Status)
	}

	jobs, err := r.buildJobs(fsm, snapshotJobs)
	if err != nil {
		return fmt.Errorf("failed to build snapshot jobs for %s: %w", poolName, err)
	}

	if r.State.Mode == models.Physical {
		r.statefulJobs = jobs
	}

	if len(jobs) == 0 {
		log.Dbg(errNoJobs, fsm.Pool())
		return errNoJobs
	}

	log.Dbg("Taking a snapshot on the pool: ", fsm.Pool())

	r.State.Status = models.Snapshotting

	defer func() {
		r.State.Status = models.Finished

		if err != nil {
			r.State.Status = models.Failed
			r.State.addAlert(telemetry.Alert{
				Level:   models.RefreshFailed,
				Message: err.Error(),
			})

			fsm.Pool().SetStatus(resources.EmptyPool)
		}

		r.State.CurrentJob = nil
	}()

	for _, j := range jobs {
		r.State.CurrentJob = j

		if err = j.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}

// buildJobs processes the configuration spec to build data retrieval jobs.
func (r *Retrieval) buildJobs(fsm pool.FSManager, groupName jobGroup) ([]components.JobRunner, error) {
	retrievalRunner, err := engine.JobBuilder(r.global, r.engineProps, fsm, r.tm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a job builder")
	}

	dbMarker := dbmarker.NewMarker(fsm.Pool().DataDir())
	jobs := make([]components.JobRunner, 0)

	for _, jobName := range r.cfg.Jobs {
		jobSpec, ok := r.cfg.JobsSpec[jobName]
		if !ok {
			return nil, errors.Errorf("job %q not found", jobName)
		}

		if getJobGroup(jobSpec.Name) != groupName {
			log.Dbg(fmt.Sprintf("Skip the %s job because it does not belong to the %s group", jobName, groupName))
			continue
		}

		jobCfg := config.JobConfig{
			Spec:   jobSpec,
			Docker: r.docker,
			Marker: dbMarker,
			FSPool: fsm.Pool(),
		}

		job, err := retrievalRunner.BuildJob(jobCfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build job")
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func getJobGroup(name string) jobGroup {
	switch name {
	case logical.DumpJobType, logical.RestoreJobType, physical.RestoreJobType:
		return refreshJobs

	case snapshot.LogicalSnapshotType, snapshot.PhysicalSnapshotType:
		return snapshotJobs
	}

	return ""
}

func (r *Retrieval) defineRetrievalMode() {
	if hasPhysicalJob(r.cfg.JobsSpec) {
		r.State.Mode = models.Physical
		return
	}

	if hasLogicalJob(r.cfg.JobsSpec) {
		r.State.Mode = models.Logical
		return
	}

	r.State.Mode = models.Unknown
}

func (r *Retrieval) setupScheduler(ctx context.Context) {
	r.stopScheduler()

	if r.cfg.Refresh == nil || r.cfg.Refresh.Timetable == "" {
		return
	}

	specParser := cron.NewParser(parseOption)

	spec, err := specParser.Parse(r.cfg.Refresh.Timetable)
	if err != nil {
		log.Err(errors.Wrapf(err, "failed to parse schedule timetable %q", r.cfg.Refresh.Timetable))
		return
	}

	r.Scheduler.Cron = cron.New()
	r.Scheduler.Spec = spec
	r.Scheduler.Cron.Schedule(r.Scheduler.Spec, cron.FuncJob(r.refreshFunc(ctx)))
	r.Scheduler.Cron.Start()
}

func (r *Retrieval) refreshFunc(ctx context.Context) func() {
	return func() {
		if err := r.FullRefresh(ctx); err != nil {
			alert := telemetry.Alert{Level: models.RefreshFailed, Message: err.Error()}
			r.State.addAlert(alert)
			r.tm.SendEvent(ctx, telemetry.AlertEvent, telemetry.Alert{Level: models.RefreshFailed, Message: "Failed to run full-refresh"})
			log.Err(alert.Message)
		}
	}
}

// FullRefresh performs full refresh for an unused storage pool and makes it active.
func (r *Retrieval) FullRefresh(ctx context.Context) error {
	if r.State.Status == models.Refreshing || r.State.Status == models.Snapshotting {
		alert := telemetry.Alert{
			Level:   models.RefreshSkipped,
			Message: "The data refresh/snapshot is currently in progress. Skip a new data refresh iteration",
		}
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)
		log.Msg(alert.Message)

		return nil
	}

	if r.State.Status == models.Pending {
		log.Msg("Data retrieving suspended because Retrieval state is pending")

		return nil
	}

	// Stop previous runs and snapshot schedulers.
	if r.ctxCancel != nil {
		r.ctxCancel()
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.ctxCancel = cancel
	elementToUpdate := r.poolManager.GetPoolToUpdate()

	if elementToUpdate == nil || elementToUpdate.Value == nil {
		alert := telemetry.Alert{
			Level:   models.RefreshSkipped,
			Message: "Pool to perform full refresh not found. Skip refreshing",
		}
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)
		log.Msg(alert.Message + ". Hint: Check that there is at least one pool that does not have clones running. " +
			"Refresh can be performed only to a pool without clones.")

		return nil
	}

	poolToUpdate, err := r.poolManager.GetFSManager(elementToUpdate.Value.(string))
	if err != nil {
		return errors.Wrap(err, "failed to get FSManager")
	}

	log.Msg("Pool selected to perform full refresh: ", poolToUpdate.Pool())

	// Stop service containers: sync-instance, etc.
	if cleanUpErr := cont.CleanUpControlContainers(runCtx, r.docker, r.engineProps.InstanceID); cleanUpErr != nil {
		log.Err("Failed to clean up service containers:", cleanUpErr)

		return cleanUpErr
	}

	if err := r.run(runCtx, poolToUpdate); err != nil {
		return err
	}

	r.poolManager.MakeActive(elementToUpdate)
	r.State.cleanAlerts()

	return nil
}

// Stop stops a retrieval service.
func (r *Retrieval) Stop() {
	r.stopScheduler()
}

func (r *Retrieval) stopScheduler() {
	if r.Scheduler.Cron != nil {
		r.Scheduler.Cron.Stop()
		r.Scheduler.Spec = nil
	}
}

// ReportState collects the current restore state.
func (r *Retrieval) ReportState() telemetry.Restore {
	var refreshingTimetable string

	if r.cfg.Refresh != nil {
		refreshingTimetable = r.cfg.Refresh.Timetable
	}

	return telemetry.Restore{
		Mode:       r.State.Mode,
		Refreshing: refreshingTimetable,
		Jobs:       r.cfg.Jobs,
	}
}

// ErrStageNotFound means that the requested stage is not exist in the retrieval jobs config.
var ErrStageNotFound = errors.New("stage not found")

// JobConfig parses job configuration to the provided structure.
func (r *Retrieval) JobConfig(stage string, jobCfg any) error {
	stageSpec, err := r.GetStageSpec(stage)
	if err != nil {
		return err
	}

	if err := options.Unmarshal(stageSpec.Options, jobCfg); err != nil {
		return fmt.Errorf("failed to unmarshal configuration options: %w", err)
	}

	return nil
}

// GetStageSpec returns the stage spec if exists.
func (r *Retrieval) GetStageSpec(stage string) (config.JobSpec, error) {
	stageSpec, ok := r.cfg.JobsSpec[stage]
	if !ok {
		return config.JobSpec{}, ErrStageNotFound
	}

	return stageSpec, nil
}

// ReportSyncStatus return status of sync containers.
func (r *Retrieval) ReportSyncStatus(ctx context.Context) (*models.Sync, error) {
	if r.State.Mode != models.Physical {
		return &models.Sync{
			Status: models.Status{Code: models.SyncStatusNotAvailable},
		}, nil
	}

	filterArgs := filters.NewArgs(
		filters.KeyValuePair{Key: "label",
			Value: fmt.Sprintf("%s=%s", cont.DBLabControlLabel, cont.DBLabSyncLabel)})

	filterArgs.Add("label", fmt.Sprintf("%s=%s", cont.DBLabInstanceIDLabel, r.engineProps.InstanceID))

	ids, err := tools.ListContainersByLabel(ctx, r.docker, filterArgs)
	if err != nil {
		return &models.Sync{
			Status: models.Status{Code: models.SyncStatusError, Message: err.Error()},
		}, fmt.Errorf("failed to list containers by label %w", err)
	}

	if len(ids) != 1 {
		return &models.Sync{
			Status: models.Status{Code: models.SyncStatusError},
		}, fmt.Errorf("failed to match sync container")
	}

	id := ids[0]

	sync, err := r.reportContainerSyncStatus(ctx, id)

	return sync, err
}

func (r *Retrieval) reportContainerSyncStatus(ctx context.Context, containerID string) (*models.Sync, error) {
	resp, err := r.docker.ContainerInspect(ctx, containerID)

	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %w", err)
	}

	if resp.State == nil {
		return nil, fmt.Errorf("failed to read container state")
	}

	if resp.State.Health != nil && resp.State.Health.Status == types.Unhealthy {
		// in case of Unhealthy state, add health check output to status
		var healthCheckOutput = ""

		if healthCheckLength := len(resp.State.Health.Log); healthCheckLength > 0 {
			if lastHealthCheck := resp.State.Health.Log[healthCheckLength-1]; lastHealthCheck.ExitCode > 1 {
				healthCheckOutput = lastHealthCheck.Output
			}
		}

		return &models.Sync{
			Status: models.Status{
				Code:    models.SyncStatusDown,
				Message: healthCheckOutput,
			},
		}, nil
	}

	socketPath := filepath.Join(r.poolManager.First().Pool().SocketDir(), resp.Name)
	value, err := status.FetchSyncMetrics(ctx, r.global, socketPath)

	if err != nil {
		log.Warn("Failed to fetch synchronization metrics", err)

		return &models.Sync{
			Status: models.Status{
				Code:    models.SyncStatusError,
				Message: err.Error(),
			},
		}, nil
	}

	value.StartedAt = resp.State.StartedAt

	return value, nil
}
