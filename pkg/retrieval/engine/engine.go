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
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
)

// StageBuilder provides a new stage builder.
func StageBuilder(globalCfg *config.Global, dockerCli *client.Client, prov provision.Provision) (components.StageBuilder, error) {
	switch globalCfg.Engine {
	case postgres.EngineType:
		return postgres.NewStageBuilder(globalCfg, dockerCli, prov), nil

	default:
		return nil, errors.New("failed to get engine")
	}
}
