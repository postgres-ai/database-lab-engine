/*
2020 © Postgres.ai
*/

package config

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
)

// CLIConfig defines a format of CLI configuration.
type CLIConfig struct {
	Version            string                 `yaml:"version" json:"version"`
	CurrentEnvironment string                 `yaml:"current_environment" json:"current_environment"`
	Environments       map[string]Environment `yaml:"environments" json:"environments"`
}

// Environment defines a format of environment configuration.
type Environment struct {
	EnvironmentID string `yaml:"-" json:"environment_id"`
	URL           string `yaml:"url" json:"url"`
	Token         string `yaml:"token" json:"token"`
}

// AddEnvironmentToConfig adds a new environment to CLIConfig.
func AddEnvironmentToConfig(c *cli.Context, cfg *CLIConfig, environmentID string) error {
	if environmentID == "" {
		return errors.New("environment_id must not be empty")
	}

	if _, ok := cfg.Environments[environmentID]; ok {
		return errors.Errorf("Environment %q is already initialized.", environmentID)
	}

	env := Environment{
		URL:   c.String(commands.URLKey),
		Token: c.String(commands.TokenKey),
	}

	if cfg.Environments == nil {
		cfg.Environments = make(map[string]Environment, 1)
	}

	cfg.Environments[environmentID] = env
	cfg.CurrentEnvironment = environmentID

	return nil
}

// updateEnvironmentInConfig updates an existing environment config.
func updateEnvironmentInConfig(c *cli.Context, cfg *CLIConfig, environmentID string) error {
	if environmentID == "" {
		return errors.New("environment_id must not be empty")
	}

	environment, ok := cfg.Environments[environmentID]
	if !ok {
		return errors.Errorf("Environment %q not found.", environmentID)
	}

	if c.String(commands.URLKey) != "" {
		environment.URL = c.String(commands.URLKey)
	}

	if c.String(commands.TokenKey) != "" {
		environment.Token = c.String(commands.TokenKey)
	}

	cfg.Environments[environmentID] = environment
	cfg.CurrentEnvironment = environmentID

	return nil
}

// switchToEnvironment switches to another CLI environment.
func switchToEnvironment(cfg *CLIConfig, environmentID string) error {
	if environmentID == "" {
		return errors.New("environment_id must not be empty")
	}

	_, ok := cfg.Environments[environmentID]
	if !ok {
		return errors.Errorf("Environment %q not found.", environmentID)
	}

	cfg.CurrentEnvironment = environmentID

	return nil
}

// removeByID removes an existing CLI environment from config.
func removeByID(cfg *CLIConfig, environmentID string) error {
	if environmentID == "" {
		return errors.New("environment_id must not be empty")
	}

	_, ok := cfg.Environments[environmentID]
	if !ok {
		return errors.Errorf("Environment %q not found.", environmentID)
	}

	delete(cfg.Environments, environmentID)

	if cfg.CurrentEnvironment == environmentID {
		// Switch to random environment.
		cfg.CurrentEnvironment = ""

		for envName := range cfg.Environments {
			cfg.CurrentEnvironment = envName
			break
		}
	}

	return nil
}
