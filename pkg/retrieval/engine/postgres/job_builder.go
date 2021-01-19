/*
2020 © Postgres.ai
*/

// Package postgres contains data retrieval jobs for a Postgres engine.
package postgres

import (
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/snapshot"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/pool"
)

const (
	// EngineType defines a Postgres engine type.
	EngineType = "postgres"
)

// JobBuilder defines a struct for job building.
type JobBuilder struct {
	cloneManager pool.FSManager
	globalCfg    *dblabCfg.Global
}

// NewJobBuilder create a new job builder.
func NewJobBuilder(global *dblabCfg.Global, cm pool.FSManager) *JobBuilder {
	return &JobBuilder{
		globalCfg:    global,
		cloneManager: cm,
	}
}

// BuildJob builds a new job by configuration.
func (s *JobBuilder) BuildJob(jobCfg config.JobConfig) (components.JobRunner, error) {
	switch jobCfg.Spec.Name {
	case logical.DumpJobType:
		return logical.NewDumpJob(jobCfg, s.globalCfg)

	case logical.RestoreJobType:
		return logical.NewJob(jobCfg, s.globalCfg)

	case physical.RestoreJobType:
		return physical.NewJob(jobCfg, s.globalCfg)

	case snapshot.LogicalInitialType:
		return snapshot.NewLogicalInitialJob(jobCfg, s.globalCfg, s.cloneManager)

	case snapshot.PhysicalInitialType:
		return snapshot.NewPhysicalInitialJob(jobCfg, s.globalCfg, s.cloneManager)
	}

	return nil, errors.Errorf("unknown job type: %q", jobCfg.Spec.Name)
}
