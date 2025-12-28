/*
2020 © Postgres.ai
*/

// Package config provides commands for a CLI config management.
package config

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/format"
)

// createEnvironment creates a new CLI environment.
func createEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		configFilename, err := GetFilename()
		if err != nil {
			return commands.ToActionError(err)
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return commands.ToActionError(err)
		}

		environmentID := cliCtx.Args().First()
		if err := AddEnvironmentToConfig(cliCtx, cfg, environmentID); err != nil {
			return commands.ToActionError(err)
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return commands.ToActionError(err)
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The %q environment is successfully created.\n",
			environmentID)

		return commands.ToActionError(err)
	}
}

// updateEnvironment updates an existing CLI environment.
func updateEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		if err := checkEnvironmentIDBefore(cliCtx); err != nil {
			return commands.ToActionError(err)
		}

		configFilename, err := GetFilename()
		if err != nil {
			return commands.ToActionError(err)
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return commands.ToActionError(err)
		}

		environmentID := cliCtx.Args().First()
		if err := updateEnvironmentInConfig(cliCtx, cfg, environmentID); err != nil {
			return commands.ToActionError(err)
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return commands.ToActionError(err)
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The %q environment is successfully updated.\n",
			environmentID)

		return commands.ToActionError(err)
	}
}

// view displays status of a CLI environment.
func view() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		cfg, err := getConfig()
		if err != nil {
			return commands.ToActionError(err)
		}

		environmentID := cfg.CurrentEnvironment

		if cliCtx.NArg() > 0 {
			environmentID = cliCtx.Args().First()
		}

		environment, ok := cfg.Environments[environmentID]
		if !ok {
			return commands.ActionErrorf("Configuration of environment %q not found.", environmentID)
		}

		environment.EnvironmentID = environmentID

		commandResponse, err := json.MarshalIndent(environment, "", "    ")
		if err != nil {
			return commands.ToActionError(err)
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return commands.ToActionError(err)
	}
}

// list displays all available CLI environments.
func list() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		cfg, err := getConfig()
		if err != nil {
			return commands.ToActionError(err)
		}

		if len(cfg.Environments) == 0 {
			_, err = fmt.Fprintln(cliCtx.App.Writer, "No environments configured.")
			return commands.ToActionError(err)
		}

		environmentNames := make([]string, 0, len(cfg.Environments))

		for environmentName := range cfg.Environments {
			environmentNames = append(environmentNames, environmentName)
		}

		sort.Strings(environmentNames)

		fmtCfg := format.FromContext(cliCtx)

		_, _ = fmt.Fprintln(cliCtx.App.Writer, "Available CLI environments:")

		t := format.NewTable(cliCtx.App.Writer, fmtCfg.NoColor)
		t.SetHeaders("", "ENV", "URL", "FORWARDING SERVER", "LOCAL PORT")

		for _, envName := range environmentNames {
			env := cfg.Environments[envName]
			marker := ""

			if envName == cfg.CurrentEnvironment {
				marker = "*"
			}

			t.Append([]string{
				marker,
				envName,
				env.URL,
				env.Forwarding.ServerURL,
				env.Forwarding.LocalPort,
			})
		}

		t.Render()

		return nil
	}
}

// switchEnvironment switches to another CLI environment.
func switchEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		if err := checkEnvironmentIDBefore(cliCtx); err != nil {
			return commands.ToActionError(err)
		}

		configFilename, err := GetFilename()
		if err != nil {
			return commands.ToActionError(err)
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return commands.ToActionError(err)
		}

		environmentID := cliCtx.Args().First()
		if err := switchToEnvironment(cfg, environmentID); err != nil {
			return commands.ToActionError(err)
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return commands.ToActionError(err)
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The CLI environment is successfully switched to %q.\n",
			environmentID)

		return commands.ToActionError(err)
	}
}

// removeEnvironment removes an existing CLI environment.
func removeEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) (err error) {
		if err := checkEnvironmentIDBefore(cliCtx); err != nil {
			return commands.ToActionError(err)
		}

		configFilename, err := GetFilename()
		if err != nil {
			return commands.ToActionError(err)
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return commands.ToActionError(err)
		}

		environmentID := cliCtx.Args().First()
		if err := removeByID(cfg, environmentID); err != nil {
			return commands.ToActionError(err)
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return commands.ToActionError(err)
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "Environment %q is successfully removed.\nThe current environment is %q.\n",
			environmentID, cfg.CurrentEnvironment)

		return commands.ToActionError(err)
	}
}

// showSettings shows global CLI settings.
func showSettings(cliCtx *cli.Context) error {
	configFilename, err := GetFilename()
	if err != nil {
		return commands.ToActionError(err)
	}

	cfg, err := Load(configFilename)
	if err != nil {
		return commands.ToActionError(err)
	}

	settings, err := json.MarshalIndent(cfg.Settings, "", "    ")
	if err != nil {
		return commands.ToActionError(err)
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(settings))

	return commands.ToActionError(err)
}

// updateSettings updates CLI settings.
func updateSettings(cliCtx *cli.Context) error {
	configFilename, err := GetFilename()
	if err != nil {
		return commands.ToActionError(err)
	}

	cfg, err := Load(configFilename)
	if err != nil {
		return commands.ToActionError(err)
	}

	if err := updateSettingsInConfig(cliCtx, cfg); err != nil {
		return commands.ToActionError(err)
	}

	if err := SaveConfig(configFilename, cfg); err != nil {
		return commands.ToActionError(err)
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "CLI settings has been successfully updated.\n")

	return commands.ToActionError(err)
}
