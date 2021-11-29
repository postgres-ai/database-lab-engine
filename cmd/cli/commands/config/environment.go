/*
2020 © Postgres.ai
*/

package config

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
)

// CLIConfig defines a format of CLI configuration.
type CLIConfig struct {
	CurrentEnvironment string                 `yaml:"current_environment" json:"current_environment"`
	Environments       map[string]Environment `yaml:"environments" json:"environments"`
}

// Environment defines a format of environment configuration.
type Environment struct {
	EnvironmentID  string     `yaml:"-" json:"environment_id"`
	URL            string     `yaml:"url" json:"url"`
	Token          string     `yaml:"token" json:"token"`
	Insecure       bool       `yaml:"insecure" json:"insecure"`
	RequestTimeout Duration   `yaml:"request_timeout,omitempty" json:"request_timeout,omitempty"`
	Forwarding     Forwarding `yaml:"forwarding" json:"forwarding"`
}

// Forwarding defines configuration for port forwarding.
type Forwarding struct {
	ServerURL    string `yaml:"server_url" json:"server_url"`
	LocalPort    string `yaml:"local_port" json:"local_port"`
	IdentityFile string `yaml:"identity_file" json:"identity_file"`
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
		URL:            c.String(commands.URLKey),
		Token:          c.String(commands.TokenKey),
		Insecure:       c.Bool(commands.InsecureKey),
		RequestTimeout: Duration(c.Duration(commands.RequestTimeoutKey)),
		Forwarding: Forwarding{
			ServerURL:    c.String(commands.FwServerURLKey),
			LocalPort:    c.String(commands.FwLocalPortKey),
			IdentityFile: c.String(commands.IdentityFileKey),
		},
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

	if c.NumFlags() == 0 {
		return errors.New("config unchanged. Set options to update.") // nolint
	}

	newEnvironment := environment

	if c.IsSet(commands.URLKey) {
		newEnvironment.URL = c.String(commands.URLKey)
	}

	if c.IsSet(commands.TokenKey) {
		newEnvironment.Token = c.String(commands.TokenKey)
	}

	if c.IsSet(commands.InsecureKey) {
		newEnvironment.Insecure = c.Bool(commands.InsecureKey)
	}

	if c.IsSet(commands.RequestTimeoutKey) {
		newEnvironment.RequestTimeout = Duration(c.Duration(commands.RequestTimeoutKey))
	}

	if c.IsSet(commands.FwServerURLKey) {
		newEnvironment.Forwarding.ServerURL = c.String(commands.FwServerURLKey)
	}

	if c.IsSet(commands.FwLocalPortKey) {
		newEnvironment.Forwarding.LocalPort = c.String(commands.FwLocalPortKey)
	}

	if c.IsSet(commands.IdentityFileKey) {
		newEnvironment.Forwarding.IdentityFile = c.String(commands.IdentityFileKey)
	}

	if newEnvironment == environment {
		return errors.New("config unchanged. Set different option values to update.") // nolint
	}

	cfg.Environments[environmentID] = newEnvironment
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
