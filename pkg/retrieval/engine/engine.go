/*
2020 © Postgres.ai
*/

// Package engine provides different engines.
package engine

import (
	"errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/telemetry"
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
