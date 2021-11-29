/*
2020 © Postgres.ai
*/

// Package global provides general commands for CLI usage.
package global

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
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

func forward(cliCtx *cli.Context) error {
	remoteURL, err := url.Parse(cliCtx.String(commands.URLKey))
	if err != nil {
		return err
	}

	tunnel, err := commands.BuildTunnel(cliCtx, remoteURL)
	if err != nil {
		return err
	}

	if err := tunnel.Open(); err != nil {
		return err
	}

	defer func() {
		if stopErr := tunnel.Stop(); err == nil {
			err = stopErr
		}
	}()

	log.Msg(fmt.Sprintf("The connection is available by address: %s", tunnel.Endpoints.Local))

	if err := tunnel.Listen(cliCtx.Context); err != nil {
		return err
	}

	return nil
}
