/*
2020 © Postgres.ai
*/

// Package components provides the key components of data retrieval.
package components

import (
	"context"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
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

	// ReportActivity reports the current job activity.
	ReportActivity(context.Context) (*activity.Activity, error)
}
