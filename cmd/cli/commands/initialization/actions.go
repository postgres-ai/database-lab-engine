/*
2020 © Postgres.ai
*/

// Package initialization provides commands for a CLI initialization.
package initialization

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/config"
)

func initCLI(c *cli.Context) error {
	dirname, err := config.GetDirname()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dirname, 0755); err != nil {
		return errors.Wrapf(err, "Cannot create config directory %s", dirname)
	}

	filename := config.BuildFileName(dirname)

	cfg, err := config.Load(filename)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &config.CLIConfig{}
		} else {
			return err
		}
	}

	cfg.Version = c.App.Version

	environmentID := c.String(commands.EnvironmentIDKey)
	if err := config.AddEnvironmentToConfig(c, cfg, environmentID); err != nil {
		return err
	}

	if err := config.SaveConfig(filename, cfg); err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.App.Writer, "Database Lab CLI is successfully initialized. Environment %q is created.\n",
		environmentID)

	return err
}
