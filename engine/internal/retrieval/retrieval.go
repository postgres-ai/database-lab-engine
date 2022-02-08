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
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
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

// Retrieval describes a data retrieval.
type Retrieval struct {
	Scheduler     Scheduler
	State         State
	cfg           *config.Config
	global        *global.Config
	engineProps   global.EngineProps
	docker        *client.Client
	poolManager   *pool.Manager
	tm            *telemetry.Agent
	runner        runners.Runner
	jobs          []components.JobRunner
	retrieveMutex sync.Mutex
	ctxCancel     context.CancelFunc
	jobSpecs      map[string]config.JobSpec
}

// Scheduler defines a refresh scheduler.
type Scheduler struct {
	Cron *cron.Cron
	Spec cron.Schedule
}

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, engineProps global.EngineProps, docker *client.Client, pm *pool.Manager, tm *telemetry.Agent,
	runner runners.Runner) *Retrieval {
	r := &Retrieval{
		cfg:         &cfg.Retrieval,
		global:      &cfg.Global,
		engineProps: engineProps,
		docker:      docker,
		poolManager: pm,
		tm:          tm,
		runner:      runner,
		jobSpecs:    make(map[string]config.JobSpec, len(cfg.Retrieval.Jobs)),
		State: State{
			Status: models.Inactive,
			alerts: make(map[models.AlertType]models.Alert),
		},
	}

	r.formatJobsSpec()
	r.defineRetrievalMode()

	return r
}

// Reload reloads retrieval configuration.
func (r *Retrieval) Reload(ctx context.Context, cfg *dblabCfg.Config) {
	*r.cfg = cfg.Retrieval

	r.formatJobsSpec()

	for _, job := range r.jobs {
		cfg, ok := r.cfg.JobsSpec[job.Name()]
		if !ok {
			log.Msg("Skip reloading of the retrieval job", job.Name())
			continue
		}

		if err := job.Reload(cfg.Options); err != nil {
			log.Err("Failed to reload configuration of the retrieval job", job.Name(), err)
		}
	}

	r.stopScheduler()
	r.setupScheduler(ctx)
}

func (r *Retrieval) formatJobsSpec() {
	for _, jobName := range r.cfg.Jobs {
		jobSpec, ok := r.cfg.JobsSpec[jobName]
		if !ok {
			continue
		}

		jobSpec.Name = jobName
		r.jobSpecs[jobName] = jobSpec
	}
}

// Run start retrieving process.
func (r *Retrieval) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	r.ctxCancel = cancel

	log.Msg("Retrieval mode:", r.State.Mode)

	fsManager, err := r.getPoolToDataRetrieving()
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

func (r *Retrieval) getPoolToDataRetrieving() (pool.FSManager, error) {
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
	if err := r.configure(fsm); err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	if err := r.prepareEnvironment(fsm); err != nil {
		return errors.Wrap(err, "failed to prepare retrieval environment")
	}

	// Check the pool aliveness.
	if _, err := fsm.GetFilesystemState(); err != nil {
		return errors.Wrap(errors.Unwrap(err), "filesystem manager is not ready")
	}

	poolByName := r.poolManager.GetPoolByName(fsm.Pool().Name)
	if poolByName == nil {
		return errors.Errorf("pool %s not found", fsm.Pool().Name)
	}

	if len(r.jobs) > 0 {
		fsm.Pool().SetStatus(resources.RefreshingPool)

		r.retrieveMutex.Lock()
		r.State.Status = models.Refreshing
		r.State.LastRefresh = pointer.ToTimeOrNil(time.Now().Truncate(time.Second))

		defer func() {
			r.State.Status = models.Finished

			if err != nil {
				r.State.Status = models.Failed

				fsm.Pool().SetStatus(resources.EmptyPool)
			}

			r.retrieveMutex.Unlock()
		}()

		for _, j := range r.jobs {
			if err := j.Run(ctx); err != nil {
				return err
			}
		}
	}

	r.poolManager.MakeActive(poolByName)
	r.State.cleanAlerts()

	return nil
}

// configure configures retrieval service.
func (r *Retrieval) configure(fsm pool.FSManager) error {
	if len(r.cfg.Jobs) == 0 {
		return nil
	}

	if err := r.parseJobs(fsm); err != nil {
		return errors.Wrap(err, "failed to parse retrieval jobs")
	}

	if err := r.validate(); err != nil {
		return errors.Wrap(err, "invalid data retrieval configuration")
	}

	return nil
}

// parseJobs processes configuration to define data retrieval jobs.
func (r *Retrieval) parseJobs(fsm pool.FSManager) error {
	retrievalRunner, err := engine.JobBuilder(r.global, r.engineProps, fsm, r.tm)
	if err != nil {
		return errors.Wrap(err, "failed to get a job builder")
	}

	dbMarker := dbmarker.NewMarker(fsm.Pool().DataDir())

	r.jobs = make([]components.JobRunner, 0, len(r.cfg.Jobs))

	for _, jobName := range r.cfg.Jobs {
		jobSpec, ok := r.jobSpecs[jobName]
		if !ok {
			return errors.Errorf("Job %q not found", jobName)
		}

		jobCfg := config.JobConfig{
			Spec:   jobSpec,
			Docker: r.docker,
			Marker: dbMarker,
			FSPool: fsm.Pool(),
		}

		job, err := retrievalRunner.BuildJob(jobCfg)
		if err != nil {
			return errors.Wrap(err, "failed to build job")
		}

		r.addJob(job)
	}

	return nil
}

// addJob applies a job to the current data retrieval.
func (r *Retrieval) addJob(job components.JobRunner) {
	r.jobs = append(r.jobs, job)
}

func (r *Retrieval) validate() error {
	if r.hasLogicalJob() && r.hasPhysicalJob() {
		return errors.New("must not contain physical and logical jobs simultaneously")
	}

	return nil
}

func (r *Retrieval) hasLogicalJob() bool {
	if len(r.jobSpecs) == 0 {
		return false
	}

	if _, hasLogicalDump := r.jobSpecs[logical.DumpJobType]; hasLogicalDump {
		return true
	}

	if _, hasLogicalRestore := r.jobSpecs[logical.RestoreJobType]; hasLogicalRestore {
		return true
	}

	if _, hasLogicalSnapshot := r.jobSpecs[snapshot.LogicalSnapshotType]; hasLogicalSnapshot {
		return true
	}

	return false
}

func (r *Retrieval) hasPhysicalJob() bool {
	if len(r.jobSpecs) == 0 {
		return false
	}

	if _, hasPhysicalRestore := r.jobSpecs[physical.RestoreJobType]; hasPhysicalRestore {
		return true
	}

	if _, hasPhysicalSnapshot := r.jobSpecs[snapshot.PhysicalSnapshotType]; hasPhysicalSnapshot {
		return true
	}

	return false
}

func (r *Retrieval) defineRetrievalMode() {
	if r.hasPhysicalJob() {
		r.State.Mode = models.Physical
		return
	}

	if r.hasLogicalJob() {
		r.State.Mode = models.Logical
		return
	}

	r.State.Mode = models.Unknown
}

func (r *Retrieval) prepareEnvironment(fsm pool.FSManager) error {
	dataDir := fsm.Pool().DataDir()
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	return filepath.Walk(dataDir, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			// PGDATA dir permissions must be 0700 to avoid errors.
			err = os.Chmod(name, 0700)
		}

		return err
	})
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
	if r.State.Status == models.Refreshing {
		alert := telemetry.Alert{
			Level:   models.RefreshSkipped,
			Message: "The data refresh is currently in progress. Skip a new data refresh iteration",
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

// IsValidConfig checks if the retrieval configuration is valid.
func IsValidConfig(cfg *dblabCfg.Config) error {
	rs := New(cfg, global.EngineProps{}, nil, nil, nil, nil)

	cm, err := pool.NewManager(nil, pool.ManagerConfig{
		Pool: &resources.Pool{
			Name: "",
			Mode: "zfs",
		},
	})
	if err != nil {
		return nil
	}

	if err := rs.configure(cm); err != nil {
		return err
	}

	if err := rs.validate(); err != nil {
		return err
	}

	return nil
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

	snapshots, err := poolToUpdate.GetSnapshots()
	if err != nil {
		var emptyErr *zfs.EmptyPoolError
		if !errors.As(err, &emptyErr) {
			return errors.Wrap(err, "failed to check existing snapshots")
		}

		log.Msg(emptyErr.Error())
	}

	for _, snapshotEntry := range snapshots {
		if err := poolToUpdate.DestroySnapshot(snapshotEntry.ID); err != nil {
			return errors.Wrap(err, "failed to destroy the existing snapshot")
		}
	}

	return nil
}

// CollectRestoreTelemetry collect restore data.
func (r *Retrieval) CollectRestoreTelemetry() telemetry.Restore {
	return telemetry.Restore{
		Mode:       r.State.Mode,
		Refreshing: r.cfg.Refresh.Timetable,
		Jobs:       r.cfg.Jobs,
	}
}
