/*
2019 © Postgres.ai
*/

// Package config provides access to the Database Lab configuration.
package config

import (
	"fmt"
	"io/ioutil"
	"os/user"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/srv"
	"gitlab.com/postgres-ai/database-lab/pkg/util"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config contains a common database-lab configuration.
type Config struct {
	Server    srv.Config       `yaml:"server"`
	Provision provision.Config `yaml:"provision"`
	Cloning   cloning.Config   `yaml:"cloning"`
	Platform  platform.Config  `yaml:"platform"`
	Global    Global           `yaml:"global"`
	Retrieval config.Config    `yaml:"retrieval"`
}

// Global contains global Database Lab configurations.
type Global struct {
	InstanceID string
	Engine     string `yaml:"engine"`
	Debug      bool   `yaml:"debug"`
	DataDir    string `yaml:"dataDir"`
	MountDir   string // TODO (akartasov): Use MountDir for the ModeLocalConfig of a Provision service.
}

// LoadConfig instances a new Config by configuration filename.
func LoadConfig(name string) (*Config, error) {
	path, err := util.GetConfigPath(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config path")
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("error loading %s config file", name)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("error parsing %s config", name)
	}

	osUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current user")
	}

	cfg.Provision.OSUsername = osUser.Username

	return cfg, nil
}
