/*
2020 © Postgres.ai
*/

// Package postgres contains data retrieval jobs for a Postgres engine.
package postgres

import (
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/snapshot"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/telemetry"
)

const (
	// EngineType defines a Postgres engine type.
	EngineType = "postgres"
)

// JobBuilder defines a struct for job building.
type JobBuilder struct {
	cloneManager pool.FSManager
	globalCfg    *global.Config
	tm           *telemetry.Agent
}

// NewJobBuilder create a new job builder.
func NewJobBuilder(global *global.Config, cm pool.FSManager, tm *telemetry.Agent) *JobBuilder {
	return &JobBuilder{
		globalCfg:    global,
		cloneManager: cm,
		tm:           tm,
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

	case snapshot.LogicalSnapshotType:
		return snapshot.NewLogicalInitialJob(jobCfg, s.globalCfg, s.cloneManager, s.tm)

	case snapshot.PhysicalSnapshotType:
		return snapshot.NewPhysicalInitialJob(jobCfg, s.globalCfg, s.cloneManager, s.tm)
	}

	return nil, errors.Errorf("unknown job type: %q", jobCfg.Spec.Name)
}
