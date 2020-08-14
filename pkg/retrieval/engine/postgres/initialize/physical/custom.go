/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"fmt"

	"github.com/docker/docker/api/types/mount"
)

const (
	customTool = "customTool"
)

type custom struct {
	options customOptions
}

type customOptions struct {
	RestoreCommand string `yaml:"command"`
	RefreshCommand string `yaml:"refresh_command"`
}

func newCustomTool(options customOptions) *custom {
	return &custom{
		options: options,
	}
}

// GetEnvVariables returns environment variables for a custom restore command.
func (c *custom) GetEnvVariables() []string {
	return []string{}
}

// GetMounts returns volume configurations for a custom restore command.
func (c *custom) GetMounts() []mount.Mount {
	return []mount.Mount{}
}

// GetRestoreCommand returns a custom command to restore data.
func (c *custom) GetRestoreCommand() string {
	return c.options.RestoreCommand
}

// GetRecoveryConfig returns a recovery config to restore data.
func (c *custom) GetRecoveryConfig() []byte {
	buffer := bytes.Buffer{}

	buffer.WriteString("standby_mode = 'on'\n")
	buffer.WriteString(fmt.Sprintf("restore_command = '%s'\n", c.options.RefreshCommand))

	return buffer.Bytes()
}
