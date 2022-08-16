/*
2020 © Postgres.ai
*/

// Package retrieval provides data retrieval pipeline.
package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"

	dblabCfg "gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	refreshJobs  jobGroup = "refresh"
	snapshotJobs jobGroup = "snapshot"
)

type jobGroup string

// Retrieval describes a data retrieval.
type Retrieval struct {
	Scheduler    Scheduler
	State        State
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
	}

	retrievalCfg, err := ValidateConfig(&cfg.Retrieval)
	if err != nil {
		return nil, err
	}

	r.setup(retrievalCfg)

	return r, nil
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

	if err := r.run(runCtx, fsManager); err != nil {
		alert := telemetry.Alert{Level: models.RefreshFailed,
			Message: fmt.Sprintf("Failed to perform initial data retrieving: %s", r.State.Mode)}
		r.State.addAlert(alert)
		r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)

		return err
	}

	r.setupScheduler(ctx)

	return nil
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

	if err := r.SnapshotData(ctx, poolName); err != nil {
		return err
	}

	if r.State.Status == models.Finished {
		r.poolManager.MakeActive(poolElement)
		r.State.cleanAlerts()
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
		log.Dbg("no jobs to snapshot pool data:", fsm.Pool())
		return nil
	}

	log.Dbg("Taking a snapshot on the pool: ", fsm.Pool())

	r.State.Status = models.Snapshotting

	defer func() {
		r.State.Status = models.Finished

		if err != nil {
			r.State.Status = models.Failed

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

	if r.cfg.Refresh.Timetable == "" {
		return
	}

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

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
		if err := r.fullRefresh(ctx); err != nil {
			alert := telemetry.Alert{Level: models.RefreshFailed, Message: "Failed to run full-refresh"}
			r.State.addAlert(alert)
			r.tm.SendEvent(ctx, telemetry.AlertEvent, alert)
			log.Err(alert.Message, err)
		}
	}
}

// fullRefresh performs full refresh for an unused storage pool and makes it active.
func (r *Retrieval) fullRefresh(ctx context.Context) error {
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
		log.Msg(alert.Message)

		return nil
	}

	poolToUpdate, err := r.poolManager.GetFSManager(elementToUpdate.Value.(string))
	if err != nil {
		return errors.Wrap(err, "failed to get FSManager")
	}

	log.Msg("Pool to a full refresh: ", poolToUpdate.Pool())

	if err := preparePoolToRefresh(poolToUpdate); err != nil {
		return errors.Wrap(err, "failed to prepare the pool to a full refresh")
	}

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

func preparePoolToRefresh(poolToUpdate pool.FSManager) error {
	cloneList, err := poolToUpdate.ListClonesNames()
	if err != nil {
		return errors.Wrap(err, "failed to check running clones")
	}

	if len(cloneList) > 0 {
		return errors.Errorf("there are active clones in the requested pool: %s\nDestroy them to perform a full refresh",
			strings.Join(cloneList, " "))
	}

	poolToUpdate.RefreshSnapshotList()

	snapshots := poolToUpdate.SnapshotList()
	if len(snapshots) == 0 {
		log.Msg(fmt.Sprintf("no snapshots for pool %s", poolToUpdate.Pool().Name))
		return nil
	}

	for _, snapshotEntry := range snapshots {
		if err := poolToUpdate.DestroySnapshot(snapshotEntry.ID); err != nil {
			return errors.Wrap(err, "failed to destroy the existing snapshot")
		}
	}

	return nil
}

// ReportState collects the current restore state.
func (r *Retrieval) ReportState() telemetry.Restore {
	return telemetry.Restore{
		Mode:       r.State.Mode,
		Refreshing: r.cfg.Refresh.Timetable,
		Jobs:       r.cfg.Jobs,
	}
}
