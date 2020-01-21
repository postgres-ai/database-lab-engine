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

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// createEnvironment creates a new CLI environment.
func createEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		configFilename, err := GetFilename()
		if err != nil {
			return err
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return err
		}

		environmentID := cliCtx.Args().First()
		if err := AddEnvironmentToConfig(cliCtx, cfg, environmentID); err != nil {
			return err
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The %q environment is successfully created.\n",
			environmentID)

		return err
	}
}

// updateEnvironment updates an existing CLI environment.
func updateEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		configFilename, err := GetFilename()
		if err != nil {
			return err
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return err
		}

		environmentID := cliCtx.Args().First()
		if err := updateEnvironmentInConfig(cliCtx, cfg, environmentID); err != nil {
			return err
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The %q environment is successfully updated.\n",
			environmentID)

		return err
	}
}

// view displays status of a CLI environment.
func view() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		environmentID := cfg.CurrentEnvironment

		if cliCtx.NArg() > 0 {
			environmentID = cliCtx.Args().First()
		}

		environment, ok := cfg.Environments[environmentID]
		if !ok {
			return errors.Errorf("Configuration of environment %q not found.", environmentID)
		}

		environment.EnvironmentID = environmentID

		commandResponse, err := json.MarshalIndent(environment, "", "    ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return err
	}
}

// list displays all available CLI environments.
func list() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := getConfig()
		if err != nil {
			return err
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

		return err
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
	return func(cliCtx *cli.Context) error {
		configFilename, err := GetFilename()
		if err != nil {
			return err
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return err
		}

		environmentID := cliCtx.Args().First()
		if err := switchToEnvironment(cfg, environmentID); err != nil {
			return err
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The CLI environment is successfully switched to %q.\n",
			environmentID)

		return err
	}
}

// removeEnvironment removes an existing CLI environment.
func removeEnvironment() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		configFilename, err := GetFilename()
		if err != nil {
			return err
		}

		cfg, err := Load(configFilename)
		if err != nil {
			return err
		}

		environmentID := cliCtx.Args().First()
		if err := removeByID(cfg, environmentID); err != nil {
			return err
		}

		if err := SaveConfig(configFilename, cfg); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "Environment %q is successfully removed.\nThe current environment is %q.\n",
			environmentID, cfg.CurrentEnvironment)

		return err
	}
}
