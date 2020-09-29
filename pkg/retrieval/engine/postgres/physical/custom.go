/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"fmt"
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
func (c *custom) GetRecoveryConfig() []byte {
	buffer := bytes.Buffer{}

	if c.options.RestoreCommand != "" {
		buffer.WriteString("\n")
		buffer.WriteString("standby_mode = 'on'\n")
		buffer.WriteString(fmt.Sprintf("restore_command = '%s'\n", c.options.RestoreCommand))
	}

	return buffer.Bytes()
}
