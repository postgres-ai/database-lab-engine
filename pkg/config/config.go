/*
2019 © Postgres.ai
*/

// Package config provides access to the Database Lab configuration.
package config

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/estimator"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
	retConfig "gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/srv"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

// Config contains a common database-lab configuration.
type Config struct {
	Server      srv.Config       `yaml:"server"`
	Provision   provision.Config `yaml:"provision"`
	Cloning     cloning.Config   `yaml:"cloning"`
	Platform    platform.Config  `yaml:"platform"`
	Global      global.Config    `yaml:"global"`
	Retrieval   retConfig.Config `yaml:"retrieval"`
	Observer    observer.Config  `yaml:"observer"`
	Estimator   estimator.Config `yaml:"estimator"`
	PoolManager pool.Config      `yaml:"poolManager"`
}

// LoadConfig instances a new Config by configuration filename.
func LoadConfig(name string) (*Config, error) {
	configPath, err := util.GetConfigPath(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config path")
	}

	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Errorf("error loading %s config file", name)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, errors.WithMessagef(err, "error parsing %s config", name)
	}

	return cfg, nil
}
