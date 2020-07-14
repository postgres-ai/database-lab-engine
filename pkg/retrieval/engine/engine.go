/*
2020 © Postgres.ai
*/

// Package engine provides different engines.
package engine

import (
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres"
)

// StageBuilder provides a new stage builder.
func StageBuilder(globalCfg *config.Global, dockerCli *client.Client) (components.StageBuilder, error) {
	switch globalCfg.Engine {
	case postgres.EngineType:
		return postgres.NewStageBuilder(globalCfg, dockerCli), nil

	default:
		return nil, errors.New("failed to get engine")
	}
}
