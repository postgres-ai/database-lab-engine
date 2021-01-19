/*
2020 © Postgres.ai
*/

// Package engine provides different engines.
package engine

import (
	"errors"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/pool"
)

// JobBuilder provides a new job builder.
func JobBuilder(globalCfg *config.Global, cloneManager pool.FSManager) (components.JobBuilder, error) {
	switch globalCfg.Engine {
	case postgres.EngineType:
		return postgres.NewJobBuilder(globalCfg, cloneManager), nil

	default:
		return nil, errors.New("failed to get engine")
	}
}
