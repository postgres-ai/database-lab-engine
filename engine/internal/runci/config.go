/*
2021 © Postgres.ai
*/

// Package runci provides tools to run and check migrations in CI.
package runci

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"

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

	cfg, err := parseConfig(b)
	if err != nil {
		return nil, errors.WithMessagef(err, "error parsing %s config", configPath)
	}

	return cfg, nil
}

// parseConfig resolves environment placeholders on the parsed document and then
// decodes it, matching how the engine loads its own config: walk the tree with
// yaml.v3, decode with yaml.v2 so decode semantics are unchanged.
func parseConfig(b []byte) (*Config, error) {
	var root yaml.Node

	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("failed to parse config document: %w", err)
	}

	if err := envvar.ExpandNode(&root); err != nil {
		return nil, fmt.Errorf("failed to resolve environment placeholders: %w", err)
	}

	expanded, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild config document: %w", err)
	}

	cfg := &Config{}
	if err := yamlv2.Unmarshal(expanded, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return cfg, nil
}
