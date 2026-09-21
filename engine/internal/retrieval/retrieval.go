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
	"slices"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
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
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/goroutine"

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
	State        State
	imageState   *db.ImageContent
	cfgMu        sync.RWMutex
	cfg          *config.Config
	global       global.Config
	engineProps  *global.EngineProps
	docker       *client.Client
	poolManager  *pool.Manager
	tm           *telemetry.Agent
	runner       runners.Runner
	runMu        sync.RWMutex
	scheduler    Scheduler
	ctxCancel    context.CancelFunc
	statefulJobs []components.JobRunner
}

// Scheduler defines a refresh scheduler. Reload rebuilds it while HTTP handlers report the next
// refresh time, so it is kept private and guarded by runMu.
type Scheduler struct {
	Cron *cron.Cron
	Spec cron.Schedule
}

var (
	ErrRefreshInProgress = errors.New("The data refresh/snapshot is currently in progress. Skip a new data refresh iteration")
	ErrRefreshPending    = errors.New("Data retrieving suspended because Retrieval state is pending")
	ErrNoAvailablePool   = errors.New("Pool to perform full refresh not found. Skip refreshing")
)

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, engineProps *global.EngineProps, docker *client.Client, pm *pool.Manager, tm *telemetry.Agent,
	runner runners.Runner) (*Retrieval, error) {
	r := &Retrieval{
		global:      cfg.Global,
		engineProps: engineProps,
		docker:      docker,
		poolManager: pm,
		tm:          tm,
		runner:      runner,
		State: State{
			status: models.Inactive,
			alerts: make(map[models.AlertType]models.Alert),
		},
		imageState: db.NewImageContent(*engineProps),
	}

	retrievalCfg, err := ValidateConfig(&cfg.Retrieval)
	if err != nil {
		return nil, err
	}

	r.setup(retrievalCfg, cfg.Global)

	if err := checkPendingMarker(r); err != nil {
		return nil, fmt.Errorf("failed to check pending marker: %w", err)
	}

	return r, nil
}

// ImageContent provides the content of foundation Docker image.
func (r *Retrieval) ImageContent() *db.ImageContent {
	return r.imageState
}

// GetRetrievalMode returns the current retrieval mode.
func (r *Retrieval) GetRetrievalMode() models.RetrievalMode {
	return r.State.Mode()
}

// GetRetrievalStatus returns the current retrieval status.
func (r *Retrieval) GetRetrievalStatus() models.RetrievalStatus {
	return r.State.Status()
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

	r.State.SetStatus(models.Pending)

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

	r.State.SetStatus(models.Inactive)

	return nil
}

// Reload reloads retrieval configuration.
func (r *Retrieval) Reload(ctx context.Context, retrievalCfg *config.Config, globalCfg global.Config) {
	r.setup(retrievalCfg, globalCfg)
	r.reloadStatefulJobs()
	r.stopScheduler()
	r.setupScheduler(ctx)
}

func (r *Retrieval) setup(retrievalCfg *config.Config, globalCfg global.Config) {
	r.cfgMu.Lock()
	r.cfg = retrievalCfg
	r.global = globalCfg
	r.cfgMu.Unlock()

	r.defineRetrievalMode()
}

// Config returns the retrieval configuration. Reload swaps the whole struct rather than writing
// into it, so the returned pointer stays consistent after the lock is released.
func (r *Retrieval) Config() *config.Config {
	r.cfgMu.RLock()
	defer r.cfgMu.RUnlock()

	return r.cfg
}

// GlobalConfig returns a snapshot of the global configuration.
func (r *Retrieval) GlobalConfig() global.Config {
	r.cfgMu.RLock()
	defer r.cfgMu.RUnlock()

	return r.global
}

func (r *Retrieval) reloadStatefulJobs() {
	jobsSpec := r.Config().JobsSpec

	for _, job := range r.statefulJobList() {
		cfg, ok := jobsSpec[job.Name()]
		if !ok {
			log.Msg("Skip reloading of the stateful retrieval job. Spec not found", job.Name())
			continue
		}
		// todo should we remove if jobs are not there ?
		// todo should we check for completion before ?
		if err := job.Reload(cfg.Options); err != nil {
			log.Err("failed to reload configuration of retrieval job", job.Name(), err)
		}
	}
}

// statefulJobList returns a copy of the jobs that survive a configuration reload. The snapshot
// pipeline replaces the slice while a reload walks it, so the caller must not iterate the
// original.
func (r *Retrieval) statefulJobList() []components.JobRunner {
	r.runMu.RLock()
	defer r.runMu.RUnlock()

	return slices.Clone(r.statefulJobs)
}

func (r *Retrieval) setStatefulJobs(jobs []components.JobRunner) {
	r.runMu.Lock()
	r.statefulJobs = jobs
	r.runMu.Unlock()
}

// restartRunContext cancels the context of the previous pipeline run and installs a fresh one.
// The old cancel is invoked after the lock is released, so it never runs under runMu.
func (r *Retrieval) restartRunContext(ctx context.Context) context.Context {
	runCtx, cancel := context.WithCancel(ctx)

	r.runMu.Lock()
	previousCancel := r.ctxCancel
	r.ctxCancel = cancel
	r.runMu.Unlock()

	if previousCancel != nil {
		previousCancel()
	}

	return runCtx
}

// Run start retrieving process.
func (r *Retrieval) Run(ctx context.Context) error {
	runCtx := r.restartRunContext(ctx)

	log.Msg("Retrieval mode:", r.State.Mode())

	if err := r.collectFoundationImageContent(); err != nil {
		return fmt.Errorf("failed to collect content lists from the foundation Docker image of the logicalDump job: %w", err)
	}

	if refresh := r.Config().Refresh; refresh != nil && refresh.SkipStartRefresh {
		log.Msg("Continue without performing initial data refresh because the `skipStartRefresh` option is enabled")
		r.setupScheduler(ctx)

		return nil
	}

	fsManager, err := r.getNextPoolToDataRetrieving()
	if err != nil {
		var skipError *SkipRefreshingError
		if errors.As(err, &skipError) {
			r.State.SetStatus(models.Finished)

			log.Msg("Continue without performing a full refresh:", skipError.Error())
			r.setupScheduler(ctx)

			return nil
		}

		alert := telemetry.Alert{
			Level:   models.RefreshFailed,
			Message: "Pool to perform data refresh not found",
		}

		r.State.SetStatus(models.Failed)
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)

		return fmt.Errorf("failed to choose pool to refresh: %w", err)
	}

	log.Msg("Pool to perform data retrieving: ", fsManager.Pool().Name)

	if r.State.Status() == models.Pending {
		log.Msg("Data retrieving suspended because Retrieval state is pending")

		return nil
	}

	if err := r.run(runCtx, fsManager); err != nil {
		r.State.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: err.Error()})
		// Build a generic message to avoid sending sensitive data.
		r.tm.SendEvent(ctx, telemetry.AlertEvent, telemetry.Alert{Level: models.RefreshFailed,
			Message: fmt.Sprintf("Failed to perform initial data retrieving: %s", r.State.Mode())})

		return err
	}

	r.setupScheduler(ctx)

	return nil
}

func (r *Retrieval) collectFoundationImageContent() error {
	if _, ok := r.Config().JobsSpec[logical.DumpJobType]; !ok {
		if r.State.Mode() == models.Logical {
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
	dbs := make([]string, 0, len(definitions))

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
	if r.State.Mode() != models.Logical {
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
	if r.engineProps.GetEdition() == global.StandardEdition {
		if err := r.engineProps.CheckBilling(); err != nil {
			return fmt.Errorf("skip snapshotting: %w", err)
		}
	}

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

	if r.State.Status() == models.Renewed {
		r.State.cleanAlerts()
	}

	if err := r.SnapshotData(ctx, poolName); err != nil && !isSnapshotExempt(err) {
		return err
	}

	if r.State.Status() == models.Finished {
		r.poolManager.MakeActive(poolElement)
		r.State.cleanAlerts()
	}

	if err := fsm.InitBranching(); err != nil {
		return fmt.Errorf("failed to init branching: %w", err)
	}

	if err := fsm.VerifyBranchMetadata(); err != nil {
		log.Warn(fmt.Sprintf("failed to verify branch metadata: %v", err))
	}

	return nil
}

// isSnapshotExempt reports whether a SnapshotData error must not abort the run: having no
// snapshot jobs or an already existing snapshot still leaves the pool ready to be activated.
func isSnapshotExempt(err error) bool {
	var existsErr *thinclones.SnapshotExistsError

	return errors.Is(err, errNoJobs) || errors.As(err, &existsErr)
}

// RefreshData runs a group of data refresh jobs.
func (r *Retrieval) RefreshData(ctx context.Context, poolName string) error {
	fsm, err := r.poolManager.GetFSManager(poolName)
	if err != nil {
		return fmt.Errorf("failed to get %q FSManager: %w", poolName, err)
	}

	if status := r.State.Status(); status == models.Refreshing || status == models.Snapshotting {
		return fmt.Errorf("skip refreshing the data because the pool is still busy: %s", status)
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

	r.State.SetStatus(models.Refreshing)
	r.State.SetLastRefresh(models.NewLocalTime(time.Now().Truncate(time.Second)))

	defer func() {
		r.State.SetStatus(models.Renewed)

		if err != nil {
			r.State.SetStatus(models.Failed)
			r.State.addAlert(telemetry.Alert{
				Level:   models.RefreshFailed,
				Message: err.Error(),
			})

			fsm.Pool().SetStatus(resources.EmptyPool)
		}

		r.State.SetCurrentJob(nil)
	}()

	for _, j := range jobs {
		r.State.SetCurrentJob(j)

		if err = j.Run(ctx); err != nil {
			return err
		}
	}

	r.State.SetCurrentJob(nil)

	return nil
}

// SnapshotData runs a group of data snapshot jobs.
func (r *Retrieval) SnapshotData(ctx context.Context, poolName string) error {
	fsm, err := r.poolManager.GetFSManager(poolName)
	if err != nil {
		return fmt.Errorf("failed to get %q FSManager: %w", poolName, err)
	}

	if status := r.State.Status(); status != models.Inactive && status != models.Renewed && status != models.Finished {
		return fmt.Errorf("pool is not ready to take a snapshot: %s", status)
	}

	jobs, err := r.buildJobs(fsm, snapshotJobs)
	if err != nil {
		return fmt.Errorf("failed to build snapshot jobs for %s: %w", poolName, err)
	}

	if r.State.Mode() == models.Physical {
		r.setStatefulJobs(jobs)
	}

	if len(jobs) == 0 {
		log.Dbg(errNoJobs, fsm.Pool())
		return errNoJobs
	}

	log.Dbg("Taking a snapshot on the pool: ", fsm.Pool())

	r.State.SetStatus(models.Snapshotting)

	defer func() {
		r.State.SetStatus(models.Finished)

		var existsErr *thinclones.SnapshotExistsError

		if err != nil && !errors.As(err, &existsErr) {
			r.State.SetStatus(models.Failed)
			r.State.addAlert(telemetry.Alert{
				Level:   models.RefreshFailed,
				Message: err.Error(),
			})

			fsm.Pool().SetStatus(resources.EmptyPool)
		}

		r.State.SetCurrentJob(nil)
	}()

	for _, j := range jobs {
		r.State.SetCurrentJob(j)

		if err = j.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}

// buildJobs processes the configuration spec to build data retrieval jobs.
func (r *Retrieval) buildJobs(fsm pool.FSManager, groupName jobGroup) ([]components.JobRunner, error) {
	globalCfg := r.GlobalConfig()

	retrievalRunner, err := engine.JobBuilder(&globalCfg, r.engineProps, fsm, r.tm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a job builder")
	}

	dbMarker := dbmarker.NewMarker(fsm.Pool().DataDir())
	jobs := make([]components.JobRunner, 0)
	cfg := r.Config()

	for _, jobName := range cfg.Jobs {
		jobSpec, ok := cfg.JobsSpec[jobName]
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
	jobsSpec := r.Config().JobsSpec

	if hasPhysicalJob(jobsSpec) {
		r.State.SetMode(models.Physical)
		return
	}

	if hasLogicalJob(jobsSpec) {
		r.State.SetMode(models.Logical)
		return
	}

	r.State.SetMode(models.Unknown)
}

// ScheduleSpec returns the schedule of the full-refresh timetable, or nil when no timetable is
// configured. Callers must use the returned value rather than reading the schedule twice, because
// a concurrent reload may drop it in between.
func (r *Retrieval) ScheduleSpec() cron.Schedule {
	r.runMu.RLock()
	defer r.runMu.RUnlock()

	return r.scheduler.Spec
}

func (r *Retrieval) setupScheduler(ctx context.Context) {
	r.stopScheduler()

	refresh := r.Config().Refresh
	if refresh == nil || refresh.Timetable == "" {
		return
	}

	specParser := cron.NewParser(parseOption)

	spec, err := specParser.Parse(refresh.Timetable)
	if err != nil {
		log.Err(errors.Wrapf(err, "failed to parse schedule timetable %q", refresh.Timetable))
		return
	}

	scheduler := cron.New()
	scheduler.Schedule(spec, cron.FuncJob(func() {
		goroutine.Run("scheduled full refresh", r.refreshFunc(ctx))
	}))
	scheduler.Start()

	r.runMu.Lock()
	r.scheduler.Cron = scheduler
	r.scheduler.Spec = spec
	r.runMu.Unlock()
}

func (r *Retrieval) refreshFunc(ctx context.Context) func() {
	return func() {
		err := r.FullRefresh(ctx)
		if err == nil || IsRefreshSkipped(err) {
			return
		}

		alert := telemetry.Alert{Level: models.RefreshFailed, Message: err.Error()}
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, telemetry.Alert{Level: models.RefreshFailed, Message: "Failed to run full-refresh"})
		log.Err(alert.Message)
	}
}

// IsRefreshSkipped reports whether the error means the refresh never started. FullRefresh has
// already alerted and logged in that case, so the caller must not report it as a failure.
func IsRefreshSkipped(err error) bool {
	return errors.Is(err, ErrRefreshInProgress) || errors.Is(err, ErrRefreshPending)
}

// FullRefresh performs full refresh for an unused storage pool and makes it active. It claims the
// single refresh slot and releases it on every return path. When the slot cannot be claimed it
// returns ErrRefreshInProgress or ErrRefreshPending, having already alerted and logged; callers
// must filter those with IsRefreshSkipped instead of reporting them as failures.
func (r *Retrieval) FullRefresh(ctx context.Context) error {
	if err := r.State.TryStartRefresh(); err != nil {
		switch {
		case errors.Is(err, ErrRefreshInProgress):
			alert := telemetry.Alert{
				Level:   models.RefreshSkipped,
				Message: err.Error(),
			}
			r.State.addAlert(alert)
			r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)
			log.Msg(alert.Message)

		case errors.Is(err, ErrRefreshPending):
			log.Msg(err.Error())
		}

		return err
	}

	// Release the slot on every return path, including the early ones below.
	defer r.State.FinishRefresh()

	// Stop previous runs and snapshot schedulers.
	runCtx := r.restartRunContext(ctx)

	if err := r.HasAvailablePool(); err != nil {
		alert := telemetry.Alert{
			Level:   models.RefreshSkipped,
			Message: err.Error(),
		}
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)
		log.Msg(err.Error() + ". Hint: Check that there is at least one pool that does not have clones running. " +
			"Refresh can be performed only to a pool without clones.")

		return nil
	}

	elementToUpdate := r.poolManager.GetPoolToUpdate()

	poolToUpdate, err := r.poolManager.GetFSManager(elementToUpdate.Value.(string))
	if err != nil {
		return errors.Wrap(err, "failed to get FSManager")
	}

	log.Msg("Pool selected to perform full refresh: ", poolToUpdate.Pool())

	// Stop service containers: sync-instance, etc.
	if cleanUpErr := cont.CleanUpControlContainers(runCtx, r.docker, r.engineProps.InstanceID); cleanUpErr != nil {
		log.Err("failed to clean up service containers:", cleanUpErr)

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
	r.runMu.Lock()
	scheduler := r.scheduler.Cron
	r.scheduler.Cron = nil
	r.scheduler.Spec = nil
	r.runMu.Unlock()

	if scheduler != nil {
		scheduler.Stop()
	}
}

// ReportState collects the current restore state.
func (r *Retrieval) ReportState() telemetry.Restore {
	var refreshingTimetable string

	cfg := r.Config()

	if cfg.Refresh != nil {
		refreshingTimetable = cfg.Refresh.Timetable
	}

	return telemetry.Restore{
		Mode:       r.State.Mode(),
		Refreshing: refreshingTimetable,
		Jobs:       cfg.Jobs,
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
	stageSpec, ok := r.Config().JobsSpec[stage]
	if !ok {
		return config.JobSpec{}, ErrStageNotFound
	}

	return stageSpec, nil
}

// ReportSyncStatus return status of sync containers.
func (r *Retrieval) ReportSyncStatus(ctx context.Context) (*models.Sync, error) {
	if r.State.Mode() != models.Physical {
		return &models.Sync{
			Status: models.Status{Code: models.SyncStatusNotAvailable},
		}, nil
	}

	filterArgs := make(client.Filters).Add("label", fmt.Sprintf("%s=%s", cont.DBLabControlLabel, cont.DBLabSyncLabel))

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
	resp, err := r.docker.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %w", err)
	}

	state := resp.Container.State

	if state == nil {
		return nil, fmt.Errorf("failed to read container state")
	}

	if state.Health != nil && state.Health.Status == container.Unhealthy {
		// in case of Unhealthy state, add health check output to status
		var healthCheckOutput = ""

		if healthCheckLength := len(state.Health.Log); healthCheckLength > 0 {
			if lastHealthCheck := state.Health.Log[healthCheckLength-1]; lastHealthCheck.ExitCode > 1 {
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

	firstPool := r.poolManager.First()
	if firstPool == nil {
		return &models.Sync{
			Status: models.Status{
				Code:    models.SyncStatusError,
				Message: "Data pool is not available",
			},
		}, nil
	}

	socketPath := filepath.Join(firstPool.Pool().SocketDir(), resp.Container.Name)
	globalCfg := r.GlobalConfig()

	value, err := status.FetchSyncMetrics(ctx, &globalCfg, socketPath)

	if err != nil {
		log.Warn("Failed to fetch synchronization metrics", err)

		return &models.Sync{
			Status: models.Status{
				Code:    models.SyncStatusError,
				Message: err.Error(),
			},
		}, nil
	}

	value.StartedAt = state.StartedAt

	return value, nil
}

// CanStartRefresh reports whether a full refresh may start. It claims nothing: FullRefresh takes
// the slot itself, so a handler precheck cannot consume it and leave the refresh a no-op.
func (r *Retrieval) CanStartRefresh() error {
	return r.State.CanStartRefresh()
}

func (r *Retrieval) HasAvailablePool() error {
	element := r.poolManager.GetPoolToUpdate()
	if element == nil || element.Value == nil {
		return ErrNoAvailablePool
	}

	return nil
}
