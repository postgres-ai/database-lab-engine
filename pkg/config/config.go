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

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/embedui"
	"gitlab.com/postgres-ai/database-lab/v3/internal/estimator"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	retConfig "gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	srvCfg "gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
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
	EmbedUI     embedui.Config   `yaml:"embedUI"`
}

// LoadConfiguration instances a new application configuration.
func LoadConfiguration() (*Config, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	log.SetDebug(cfg.Global.Debug)
	log.Dbg("Config loaded", cfg)

	return cfg, nil
}

// LoadInstanceID tries to make instance ID persistent across runs and load its value after restart
func LoadInstanceID(mountDir string) (string, error) {
	instanceID := ""
	idFilepath := filepath.Join(mountDir, instanceIDFile)

	data, err := os.ReadFile(idFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			instanceID = xid.New().String()
			log.Dbg("no instance_id file was found, generate new instance ID", instanceID)

			return instanceID, os.WriteFile(idFilepath, []byte(instanceID), 0544)
		}

		return instanceID, fmt.Errorf("failed to load instanceid, %w", err)
	}

	instanceID = string(data)

	return instanceID, nil
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

	return cfg, nil
}
