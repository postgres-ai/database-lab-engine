/*
2020 © Postgres.ai
*/

// Package retrieval provides data retrieval pipeline.
package retrieval

import (
	"context"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

// Retrieval describes a data retrieval.
type Retrieval struct {
	cfg             *config.Config
	globalCfg       *dblabCfg.Global
	retrievalRunner components.JobBuilder
	cloneManager    thinclones.Manager
	jobs            []components.JobRunner
}

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, dockerCLI *client.Client, cloneManager thinclones.Manager) (*Retrieval, error) {
	retrievalRunner, err := engine.JobBuilder(&cfg.Global, dockerCLI, cloneManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a job builder")
	}

	return &Retrieval{
		cfg:             &cfg.Retrieval,
		globalCfg:       &cfg.Global,
		retrievalRunner: retrievalRunner,
		cloneManager:    cloneManager,
	}, nil
}

// Run start retrieving process.
func (r *Retrieval) Run(ctx context.Context) error {
	if len(r.cfg.Jobs) == 0 {
		return nil
	}

	if err := r.parseJobs(); err != nil {
		return errors.Wrap(err, "failed to parse retrieval jobs")
	}

	if err := r.validate(); err != nil {
		return errors.Wrap(err, "invalid data retrieval configuration")
	}

	if err := r.prepareEnvironment(); err != nil {
		return errors.Wrap(err, "failed to prepare retrieval environment")
	}

	for _, j := range r.jobs {
		if err := j.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}

// parseJobs processes configuration to define data retrieval jobs.
func (r *Retrieval) parseJobs() error {
	for _, jobName := range r.cfg.Jobs {
		jobConfig, ok := r.cfg.JobsSpec[jobName]
		if !ok {
			return errors.Errorf("Job %q not found", jobName)
		}

		jobConfig.Name = jobName

		job, err := r.retrievalRunner.BuildJob(jobConfig)
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
	jobsList := make(map[string]struct{}, len(r.jobs))

	for _, job := range r.jobs {
		jobsList[job.Name()] = struct{}{}
	}

	_, hasLogical := jobsList[logical.RestoreJobType]
	_, hasPhysical := jobsList[physical.RestoreJobType]

	if hasLogical && hasPhysical {
		return errors.New("must not contain physical and logical restore jobs simultaneously")
	}

	return nil
}

func (r *Retrieval) prepareEnvironment() error {
	if err := os.MkdirAll(r.globalCfg.DataDir, 0700); err != nil {
		return err
	}

	return filepath.Walk(r.globalCfg.DataDir, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			// PGDATA dir permissions must be 0700 to avoid errors.
			err = os.Chmod(name, 0700)
		}

		return err
	})
}
