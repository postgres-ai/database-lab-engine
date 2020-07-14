/*
2020 © Postgres.ai
*/

// Package components provides the key components of data retrieval.
package components

import (
	"context"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
)

// StageBuilder builds a StageRunner.
type StageBuilder interface {
	BuildStageRunner(string) (StageRunner, error)
}

// JobRunner performs a job.
type JobRunner interface {
	// Name returns a job name.
	Name() string

	// Run starts a job.
	Run(ctx context.Context) error
}

// StageRunner declares stage content and performs stage jobs.
type StageRunner interface {
	// AddJob applies jobs to the current stage.
	AddJob(JobRunner)

	// BuildJob builds stage jobs.
	BuildJob(config.JobConfig) (JobRunner, error)

	// Run starts stage jobs.
	Run(ctx context.Context) error
}
