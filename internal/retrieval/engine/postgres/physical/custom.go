/*
2020 © Postgres.ai
*/

package physical

import (
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
)

const (
	customTool = "customTool"
)

type custom struct {
	options customOptions
}

type customOptions struct {
	Command        string `yaml:"command"`
	RestoreCommand string `yaml:"restore_command"`
}

func newCustomTool(options customOptions) *custom {
	return &custom{
		options: options,
	}
}

// GetRestoreCommand returns a custom command to restore data.
func (c *custom) GetRestoreCommand() string {
	return c.options.Command
}

// GetRecoveryConfig returns a recovery config to restore data.
func (c *custom) GetRecoveryConfig(pgVersion float64) map[string]string {
	recoveryCfg := make(map[string]string)

	if c.options.RestoreCommand != "" {
		recoveryCfg["restore_command"] = c.options.RestoreCommand

		if pgVersion < defaults.PGVersion12 {
			recoveryCfg["standby_mode"] = "on"
			recoveryCfg["recovery_target_timeline"] = "latest"
		}
	}

	return recoveryCfg
}
