/*
2020 © Postgres.ai
*/

// Package retrieval provides data retrieval pipeline.
package retrieval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	dblabCfg "gitlab.com/postgres-ai/database-lab/v2/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/thinclones/zfs"
)

// Retrieval describes a data retrieval.
type Retrieval struct {
	cfg           *config.Config
	global        *dblabCfg.Global
	docker        *client.Client
	poolManager   *pool.Manager
	runner        runners.Runner
	jobs          []components.JobRunner
	scheduler     *cron.Cron
	retrieveMutex sync.Mutex
	ctxCancel     context.CancelFunc
	jobSpecs      map[string]config.JobSpec
}

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, docker *client.Client, pm *pool.Manager, runner runners.Runner) *Retrieval {
	return &Retrieval{
		cfg:         &cfg.Retrieval,
		global:      &cfg.Global,
		docker:      docker,
		poolManager: pm,
		runner:      runner,
		jobSpecs:    make(map[string]config.JobSpec, len(cfg.Retrieval.Jobs)),
	}
}

// Reload reloads retrieval configuration.
func (r *Retrieval) Reload(ctx context.Context, cfg *dblabCfg.Config) {
	*r.cfg = cfg.Retrieval

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

	r.setupScheduler(ctx)
}

// Run start retrieving process.
func (r *Retrieval) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	r.ctxCancel = cancel

	if err := r.run(runCtx, r.poolManager.Active()); err != nil {
		return err
	}

	r.setupScheduler(ctx)

	return nil
}

func (r *Retrieval) run(ctx context.Context, fsm pool.FSManager) error {
	r.retrieveMutex.Lock()
	defer r.retrieveMutex.Unlock()

	if err := r.configure(fsm); err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	if err := r.prepareEnvironment(fsm); err != nil {
		return errors.Wrap(err, "failed to prepare retrieval environment")
	}

	// Check the pool aliveness.
	if _, err := fsm.GetDiskState(); err != nil {
		return errors.Wrap(errors.Unwrap(err), "filesystem manager is not ready")
	}

	for _, j := range r.jobs {
		if err := j.Run(ctx); err != nil {
			return err
		}
	}

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
	retrievalRunner, err := engine.JobBuilder(r.global, fsm)
	if err != nil {
		return errors.Wrap(err, "failed to get a job builder")
	}

	dbMarker := dbmarker.NewMarker(fsm.Pool().DataDir())

	r.jobs = make([]components.JobRunner, 0, len(r.cfg.Jobs))

	for _, jobName := range r.cfg.Jobs {
		jobSpec, ok := r.cfg.JobsSpec[jobName]
		if !ok {
			return errors.Errorf("Job %q not found", jobName)
		}

		jobSpec.Name = jobName
		r.jobSpecs[jobName] = jobSpec

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
	_, hasLogicalRestore := r.jobSpecs[logical.RestoreJobType]
	_, hasPhysicalRestore := r.jobSpecs[physical.RestoreJobType]

	if hasLogicalRestore && hasPhysicalRestore {
		return errors.New("must not contain physical and logical restore jobs simultaneously")
	}

	return nil
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

	if r.cfg.Refresh.Timetable != "" {
		if err := r.validateRefreshSchedule(); err != nil {
			log.Err(errors.Wrap(err, "an invalid full-refresh schedule"))
			return
		}

		r.scheduler = cron.New()

		if _, err := r.scheduler.AddFunc(r.cfg.Refresh.Timetable, r.refreshFunc(ctx)); err != nil {
			log.Err(errors.Wrap(err, "failed to add a full-refresh func"))
		}

		r.scheduler.Start()
	}
}

func (r *Retrieval) validateRefreshSchedule() error {
	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	if _, err := specParser.Parse(r.cfg.Refresh.Timetable); err != nil {
		return errors.Wrapf(err, "failed to parse schedule timetable %q", r.cfg.Refresh.Timetable)
	}

	return nil
}

func (r *Retrieval) refreshFunc(ctx context.Context) func() {
	return func() {
		if err := r.fullRefresh(ctx); err != nil {
			log.Err("Failed to run full-refresh: ", err)
		}
	}
}

// fullRefresh makes a full refresh for an old filesystem pool.
func (r *Retrieval) fullRefresh(ctx context.Context) error {
	// Stop previous runs and snapshot schedulers.
	if r.ctxCancel != nil {
		r.ctxCancel()
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.ctxCancel = cancel
	poolToUpdate := r.poolManager.Oldest()

	if poolToUpdate == nil {
		log.Msg("Pool to a full refresh not found. Skip refreshing.")
		return nil
	}

	log.Msg("Pool to a full refresh: ", poolToUpdate.Pool())

	if err := preparePoolToRefresh(poolToUpdate); err != nil {
		return errors.Wrap(err, "failed to prepare the pool to a full refresh")
	}

	// Stop service containers: sync-instance, etc.
	if cleanUpErr := cont.CleanUpControlContainers(runCtx, r.docker, r.global.InstanceID); cleanUpErr != nil {
		log.Err("Failed to clean up service containers:", cleanUpErr)

		return cleanUpErr
	}

	current := r.poolManager.Active()

	if err := r.run(runCtx, poolToUpdate); err != nil {
		return err
	}

	r.poolManager.SetOldest(current)
	r.poolManager.SetActive(poolToUpdate)

	return nil
}

// Stop stops a retrieval service.
func (r *Retrieval) Stop() {
	r.stopScheduler()
}

func (r *Retrieval) stopScheduler() {
	if r.scheduler != nil {
		r.scheduler.Stop()
	}
}

// IsValidConfig checks if the retrieval configuration is valid.
func IsValidConfig(cfg *dblabCfg.Config) error {
	rs := New(cfg, nil, nil, nil)

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
		emptyErr, ok := errors.Cause(err).(*zfs.EmptyPoolError)
		if !ok {
			return errors.Wrap(err, "failed to check existing snapshots")
		}

		log.Msg(emptyErr.Error())
	}

	for _, snapshot := range snapshots {
		if err := poolToUpdate.DestroySnapshot(snapshot.ID); err != nil {
			return errors.Wrap(err, "failed to destroy the existing snapshot")
		}
	}

	return nil
}
