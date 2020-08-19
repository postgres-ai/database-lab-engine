/*
2020 © Postgres.ai
*/

// Package postgres contains data retrieval stages and jobs for a Postgres engine.
package postgres

import (
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/snapshot"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

const (
	// EngineType defines a Postgres engine type.
	EngineType = "postgres"
)

// JobBuilder defines a struct for job building.
type JobBuilder struct {
	dockerClient *client.Client
	cloneManager thinclones.Manager
	dbMarker     *dbmarker.Marker
	globalCfg    *dblabCfg.Global
}

// NewJobBuilder create a new job builder.
func NewJobBuilder(global *dblabCfg.Global, dockerClient *client.Client, cloneManager thinclones.Manager) *JobBuilder {
	return &JobBuilder{
		dockerClient: dockerClient,
		globalCfg:    global,
		cloneManager: cloneManager,
		dbMarker:     dbmarker.NewMarker(global.DataDir),
	}
}

// BuildJob builds a new job by configuration.
func (s *JobBuilder) BuildJob(jobCfg config.JobConfig) (components.JobRunner, error) {
	switch jobCfg.Name {
	case logical.DumpJobType:
		return logical.NewDumpJob(jobCfg, s.dockerClient, s.globalCfg, s.dbMarker)

	case logical.RestoreJobType:
		return logical.NewJob(jobCfg, s.dockerClient, s.globalCfg, s.dbMarker)

	case physical.RestoreJobType:
		return physical.NewJob(jobCfg, s.dockerClient, s.globalCfg, s.dbMarker)

	case snapshot.LogicalInitialType:
		return snapshot.NewLogicalInitialJob(jobCfg, s.cloneManager, s.globalCfg, s.dbMarker)

	case snapshot.PhysicalInitialType:
		return snapshot.NewPhysicalInitialJob(jobCfg, s.dockerClient, s.cloneManager, s.globalCfg, s.dbMarker)
	}

	return nil, errors.Errorf("unknown job type: %q", jobCfg.Name)
}
