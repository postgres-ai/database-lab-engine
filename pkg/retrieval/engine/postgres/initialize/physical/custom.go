/*
2020 © Postgres.ai
*/

package physical

import (
	"strings"

	"github.com/docker/docker/api/types/mount"
)

const (
	customTool = "customTool"
)

type custom struct {
	options customOptions
}

type customOptions struct {
	Command string `yaml:"command"`
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
func (c *custom) GetRestoreCommand() []string {
	return strings.Split(c.options.Command, " ")
}
