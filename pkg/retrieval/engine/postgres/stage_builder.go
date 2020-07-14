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
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/snapshot"
)

const (
	// EngineType defines a Postgres engine type.
	EngineType = "postgres"
)

// StageBuilder provides a Postgres stage builder.
type StageBuilder struct {
	dockerClient *client.Client
	globalCfg    *dblabCfg.Global
}

// NewStageBuilder create a new Postgres stage builder.
func NewStageBuilder(globalCfg *dblabCfg.Global, dockerClient *client.Client) *StageBuilder {
	return &StageBuilder{
		dockerClient: dockerClient,
		globalCfg:    globalCfg,
	}
}

// BuildStageRunner builds a stage runner.
func (s *StageBuilder) BuildStageRunner(name string) (components.StageRunner, error) {
	switch name {
	case initialize.StageType:
		return initialize.NewStage(name, s.dockerClient, s.globalCfg), nil

	case snapshot.StageType:
		return snapshot.NewStage(name), nil
	}

	return nil, errors.Errorf("unknown stage given: %q", name)
}
