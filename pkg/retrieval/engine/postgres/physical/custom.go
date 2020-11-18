/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"fmt"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/defaults"
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
func (c *custom) GetRecoveryConfig(pgVersion float64) []byte {
	buffer := bytes.Buffer{}

	if c.options.RestoreCommand != "" {
		buffer.WriteString("\n")
		buffer.WriteString(fmt.Sprintf("restore_command = '%s'\n", c.options.RestoreCommand))

		if pgVersion < defaults.PGVersion12 {
			buffer.WriteString("standby_mode = 'on'\n")
		}
	}

	return buffer.Bytes()
}
