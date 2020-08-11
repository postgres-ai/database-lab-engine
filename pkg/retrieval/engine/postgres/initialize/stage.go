/*
2020 © Postgres.ai
*/

// Package initialize provides components of an initialization stage.
package initialize

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/logical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/physical"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/snapshot"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

const (
	// StageType declares an initialization stage type.
	StageType = "initialize"
)

// Stage defines an initialization stage.
type Stage struct {
	name         string
	dockerClient *client.Client
	cloneManager thinclones.Manager
	dbMarker     *dbmarker.Marker
	globalCfg    *dblabCfg.Global
	jobs         []components.JobRunner
}

// NewStage create a new initialization stage.
func NewStage(name string, dockerClient *client.Client, global *dblabCfg.Global, cloneManager thinclones.Manager) *Stage {
	return &Stage{
		name:         name,
		dockerClient: dockerClient,
		globalCfg:    global,
		cloneManager: cloneManager,
		dbMarker:     dbmarker.NewMarker(global.DataDir),
	}
}

// BuildJob builds stage jobs.
func (s *Stage) BuildJob(jobCfg config.JobConfig) (components.JobRunner, error) {
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

	return nil, errors.New("unknown job type")
}

// AddJob applies jobs to the current stage.
func (s *Stage) AddJob(job components.JobRunner) {
	s.jobs = append(s.jobs, job)
}

// Run starts the initialization stage.
func (s *Stage) Run(ctx context.Context) error {
	log.Msg(fmt.Sprintf("Running the stage: %s", s.name))

	if err := s.validate(); err != nil {
		return errors.Wrap(err, "invalid initialize stage configuration")
	}

	for _, j := range s.jobs {
		if err := j.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stage) validate() error {
	jobsList := make(map[string]struct{}, len(s.jobs))

	for _, job := range s.jobs {
		jobsList[job.Name()] = struct{}{}
	}

	_, hasLogical := jobsList[logical.RestoreJobType]
	_, hasPhysical := jobsList[physical.RestoreJobType]

	if hasLogical && hasPhysical {
		return errors.New("must not contain physical and logical restore jobs simultaneously")
	}

	return nil
}
