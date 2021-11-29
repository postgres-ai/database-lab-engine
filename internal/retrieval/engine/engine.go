/*
2020 © Postgres.ai
*/

// Package engine provides different engines.
package engine

import (
	"errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
)

// JobBuilder provides a new job builder.
func JobBuilder(globalCfg *global.Config, engineProps global.EngineProps, cloneManager pool.FSManager,
	tm *telemetry.Agent) (components.JobBuilder, error) {
	switch globalCfg.Engine {
	case postgres.EngineType:
		return postgres.NewJobBuilder(globalCfg, engineProps, cloneManager, tm), nil

	default:
		return nil, errors.New("failed to get engine")
	}
}
