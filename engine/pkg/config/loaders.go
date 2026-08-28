package config

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/envvar"
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
//
// The config itself is not logged. Placeholders are resolved by the time it is
// decoded, so printing it would put every source-database and object-store
// credential in the process log, where the UI stream's secret filter does not
// reach. Use the admin API, which serves the config masked, to inspect it.
func ApplyGlobals(cfg *Config) {
	log.SetDebug(cfg.Global.Debug)
	log.Dbg("Config loaded")
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

	cfg, err := parseConfig(b)
	if err != nil {
		return nil, errors.WithMessagef(err, "error parsing %s config", configPath)
	}

	return cfg, nil
}

// parseConfig resolves environment placeholders on the parsed document and then
// decodes it. Expanding before the decode is what lets placeholders reach the
// retrieval job specs: those are decoded per job from map[string]interface{},
// so the credentials inside them cannot be named as struct fields.
//
// The document is walked with yaml.v3, which is the only version exposing a
// node tree, but it is decoded with yaml.v2 exactly as before. That split is
// deliberate: the shipped logical-mode examples merge two anchors into one
// mapping ("<<: *db_container" and "<<: *db_configs" as siblings), which v2
// accepts and v3 rejects as a duplicate key. Decoding with v3 would break every
// config derived from those examples. Parsing to a node does not apply that
// check, so expansion can use v3 while decode semantics stay untouched.
func parseConfig(b []byte) (*Config, error) {
	expanded, err := ExpandDocument(b)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yamlv2.Unmarshal(expanded, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return cfg, nil
}

// replacementRulesPath holds observer.replacementRules, a regex -> replacement
// map whose values are Go regexp templates: a template of exactly "${name}" is
// a named backreference there, so the subtree is never expanded.
const replacementRulesPath = "observer.replacementRules"

// References lists every placeholder occurrence in raw YAML, skipping what
// ExpandDocument skips.
func References(b []byte) ([]envvar.Reference, error) {
	var root yaml.Node

	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("failed to parse config document: %w", err)
	}

	return envvar.References(&root, replacementRulesPath), nil
}

// ExpandDocument resolves placeholders in raw YAML and returns the equivalent
// document with them substituted. The output is an in-memory intermediate only:
// the file on disk keeps its placeholders, which is what lets the admin config
// API rewrite the config without persisting resolved secrets. Callers that
// validate a config before writing it use this to check what the engine will
// actually load, while still writing the raw bytes they were given.
func ExpandDocument(b []byte) ([]byte, error) {
	var root yaml.Node

	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("failed to parse config document: %w", err)
	}

	if err := envvar.ExpandNode(&root, replacementRulesPath); err != nil {
		return nil, fmt.Errorf("failed to resolve environment placeholders: %w", err)
	}

	expanded, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild config document: %w", err)
	}

	return expanded, nil
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
