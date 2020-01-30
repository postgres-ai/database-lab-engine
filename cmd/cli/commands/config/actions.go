/*
2020 © Postgres.ai
*/

// Package config provides commands for a CLI config management.
package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
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

		environmentNames := make([]string, 0, len(cfg.Environments))
		maxNameLength := 0

		for environmentName := range cfg.Environments {
			environmentNames = append(environmentNames, environmentName)

			nameLength := len(environmentName)
			if maxNameLength < nameLength {
				maxNameLength = nameLength
			}
		}

		sort.Strings(environmentNames)

		listOutput := buildListOutput(cfg, environmentNames, maxNameLength)

		_, err = fmt.Fprintf(cliCtx.App.Writer, "Available CLI environments:\n%s", listOutput)

		return commands.ToActionError(err)
	}
}

func buildListOutput(cfg *CLIConfig, environmentNames []string, maxNameLength int) string {
	const outputAlign = 2

	s := strings.Builder{}

	for _, environmentName := range environmentNames {
		if environmentName == cfg.CurrentEnvironment {
			s.WriteString("[*] ")
		} else {
			s.WriteString("[ ] ")
		}

		s.WriteString(environmentName)
		s.WriteString(strings.Repeat(" ", maxNameLength-len(environmentName)+outputAlign))
		s.WriteString(cfg.Environments[environmentName].URL)
		s.WriteString("\n")
	}

	return s.String()
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
