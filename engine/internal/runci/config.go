/*
2021 © Postgres.ai
*/

// Package runci provides tools to run and check migrations in CI.
package runci

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/runci/source"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/envvar"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	configFilename = "ci_checker.yml"
)

// Config contains a runner configuration.
type Config struct {
	App      App             `yaml:"app"`
	DLE      DLE             `yaml:"dle"`
	Platform platform.Config `yaml:"platform"`
	Source   source.Config   `yaml:"source"`
	Runner   Runner          `yaml:"runner"`
}

// App defines a general configuration of the application.
type App struct {
	Host              string `yaml:"host"`
	Port              uint   `yaml:"port"`
	VerificationToken string `yaml:"verificationToken"`
	Debug             bool   `yaml:"debug"`
}

// DLE describes the configuration of the Database Lab Engine server.
type DLE struct {
	VerificationToken string `yaml:"verificationToken"`
	URL               string `yaml:"url"`
	DBName            string `yaml:"dbName"`
	Container         string `yaml:"container"`
}

// Runner defines runner configuration.
type Runner struct {
	Image string `yaml:"image"`
}

// LoadConfiguration loads configuration of DB Migration Checker.
func LoadConfiguration() (*Config, error) {
	configPath, err := util.GetConfigPath(configFilename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config path")
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Errorf("error loading %s config file", configPath)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, errors.WithMessagef(err, "error parsing %s config", configPath)
	}

	if err := envvar.ExpandFields([]envvar.Field{
		{Name: "app.verificationToken", Ptr: &cfg.App.VerificationToken},
		{Name: "dle.verificationToken", Ptr: &cfg.DLE.VerificationToken},
		{Name: "platform.accessToken", Ptr: &cfg.Platform.AccessToken},
		{Name: "source.token", Ptr: &cfg.Source.Token},
	}); err != nil {
		return nil, errors.Wrap(err, "failed to resolve environment placeholders")
	}

	return cfg, nil
}
