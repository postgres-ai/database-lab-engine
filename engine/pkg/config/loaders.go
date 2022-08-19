package config

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"gopkg.in/yaml.v2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/backup"
)

const numberOfBackups = 10

// LoadConfiguration instances a new application configuration.
func LoadConfiguration() (*Config, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	return cfg, nil
}

// ApplyGlobals applies global configuration to logger.
func ApplyGlobals(cfg *Config) {
	log.SetDebug(cfg.Global.Debug)
	log.Dbg("Config loaded", cfg)
}

// LoadInstanceID tries to make instance ID persistent across runs and load its value after restart
func LoadInstanceID() (string, error) {
	instanceID := ""

	idFilepath, err := util.GetMetaPath(instanceIDFile)
	if err != nil {
		return "", fmt.Errorf("failed to get path of instanceID file: %w", err)
	}

	data, err := os.ReadFile(idFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			instanceID = xid.New().String()
			log.Dbg("no instance_id file was found, generate new instance ID", instanceID)

			if err := os.MkdirAll(path.Dir(idFilepath), 0744); err != nil {
				return "", fmt.Errorf("failed to make directory meta: %w", err)
			}

			return instanceID, os.WriteFile(idFilepath, []byte(instanceID), 0644)
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

// GetConfigBytes returns config bytes.
func GetConfigBytes() ([]byte, error) {
	configPath, err := util.GetConfigPath(configName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config path")
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Errorf("error loading %s config file", configPath)
	}

	return b, nil
}

// RotateConfig store data in config, and backup old config
func RotateConfig(data []byte) error {
	configPath, err := util.GetConfigPath(configName)
	if err != nil {
		return errors.Wrap(err, "failed to get config path")
	}

	backups, err := backup.NewBackupCollection(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to create backup collection")
	}

	err = backups.Rotate(data)
	if err != nil {
		return errors.Wrap(err, "failed to rotate config")
	}

	err = backups.EnsureMaxBackups(numberOfBackups)
	if err != nil {
		return errors.Wrap(err, "failed to ensure max backups")
	}

	return nil
}
