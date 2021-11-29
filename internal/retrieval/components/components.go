/*
2020 © Postgres.ai
*/

// Package components provides the key components of data retrieval.
package components

import (
	"context"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
)

// JobBuilder builds jobs.
type JobBuilder interface {
	// BuildJob builds retrieval jobs.
	BuildJob(jobConfig config.JobConfig) (JobRunner, error)
}

// JobRunner performs a job.
type JobRunner interface {
	// Name returns a job name.
	Name() string

	// Reload reloads job configuration.
	Reload(cfg map[string]interface{}) error

	// Run starts a job.
	Run(ctx context.Context) error
}
