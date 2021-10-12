/*
2019 © Postgres.ai
*/

// Package config provides access to the Database Lab configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"gopkg.in/yaml.v2"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/estimator"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
	retConfig "gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	srvCfg "gitlab.com/postgres-ai/database-lab/v2/pkg/srv/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

const (
	configName     = "server.yml"
	instanceIDFile = "instance_id"
)

// Config contains a common database-lab configuration.
type Config struct {
	Server      srvCfg.Config    `yaml:"server"`
	Provision   provision.Config `yaml:"provision"`
	Cloning     cloning.Config   `yaml:"cloning"`
	Platform    platform.Config  `yaml:"platform"`
	Global      global.Config    `yaml:"global"`
	Retrieval   retConfig.Config `yaml:"retrieval"`
	Observer    observer.Config  `yaml:"observer"`
	Estimator   estimator.Config `yaml:"estimator"`
	PoolManager pool.Config      `yaml:"poolManager"`
}

// LoadConfiguration instances a new application configuration.
func LoadConfiguration() (*Config, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	log.SetDebug(cfg.Global.Debug)
	log.Dbg("Config loaded", cfg)

	return cfg, cfg.loadInstanceID()
}

// loadInstanceID tries to make instance ID persistent across runs and load its value after restart
func (cfg *Config) loadInstanceID() error {
	idFilepath := filepath.Join(cfg.PoolManager.MountDir, instanceIDFile)

	data, err := os.ReadFile(idFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.Global.InstanceID = xid.New().String()
			log.Dbg("no instance_id file was found, generate new instance ID", cfg.Global.InstanceID)

			return os.WriteFile(idFilepath, []byte(cfg.Global.InstanceID), 0544)
		}

		return fmt.Errorf("failed to load instanceid, %w", err)
	}

	cfg.Global.InstanceID = string(data)

	return nil
}

// readConfig reads application configuration.
func readConfig() (*Config, error) {
	configPath, err := util.GetConfigPath(configName)
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

	return cfg, cfg.loadInstanceID()
}
