/*
2020 © Postgres.ai
*/

package config

import (
	"os"
	"os/user"
	"path"

	"gopkg.in/yaml.v2"
)

const (
	configPath     = ".dblab"
	configFilename = "cli.yml"
)

// GetDirname returns the CLI config path located in the current user's home directory.
func GetDirname() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	dirname := path.Join(currentUser.HomeDir, configPath)

	return dirname, nil
}

// GetFilename returns the CLI config filename located in the current user's home directory.
func GetFilename() (string, error) {
	dirname, err := GetDirname()
	if err != nil {
		return "", nil
	}

	return BuildFileName(dirname), nil
}

// BuildFileName builds a config filename.
func BuildFileName(dirname string) string {
	return path.Join(dirname, configFilename)
}

// Load loads a CLI config by a provided filename.
func Load(filename string) (*CLIConfig, error) {
	configData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &CLIConfig{}
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getConfig provides a loaded CLI config.
func getConfig() (*CLIConfig, error) {
	configFilename, err := GetFilename()
	if err != nil {
		return nil, err
	}

	return Load(configFilename)
}

// SaveConfig persists a CLI config.
func SaveConfig(filename string, cfg *CLIConfig) error {
	configData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, configData, 0600); err != nil {
		return err
	}

	return nil
}
