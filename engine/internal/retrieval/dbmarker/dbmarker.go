/*
2020 © Postgres.ai
*/

// Package dbmarker provides a tool for marking database data.
package dbmarker

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Marker marks database data depends on a retrieval process.
type Marker struct {
	dataPath string
}

// NewMarker creates a new DBMarker.
func NewMarker(dataPath string) *Marker {
	return &Marker{
		dataPath: dataPath,
	}
}

// Config describes marked data.
type Config struct {
	DataStateAt string `yaml:"dataStateAt"`
	DataType    string `yaml:"dataType"`
}

const (
	configDir      = ".dblab"
	configFilename = "dbmarker"

	// LogicalDataType defines a logical data type.
	LogicalDataType = "logical"

	// PhysicalDataType defines a physical data type.
	PhysicalDataType = "physical"
)

// Init inits DB marker for the data directory.
func (m *Marker) initDBLabDirectory() error {
	dirname := path.Join(m.dataPath, configDir)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return errors.Wrapf(err, "cannot create a DBMarker directory %s", dirname)
	}

	return nil
}

// CreateConfig creates a new DBMarker config file.
func (m *Marker) CreateConfig() error {
	if err := m.initDBLabDirectory(); err != nil {
		return errors.Wrap(err, "failed to init DBMarker")
	}

	dbMarkerFile, err := os.OpenFile(m.buildFileName(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer func() { _ = dbMarkerFile.Close() }()

	return nil
}

// GetConfig provides a loaded DBMarker config.
func (m *Marker) GetConfig() (*Config, error) {
	configData, err := os.ReadFile(m.buildFileName())
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	if len(configData) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig stores a DBMarker config.
func (m *Marker) SaveConfig(cfg *Config) error {
	configData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(m.buildFileName(), configData, 0600); err != nil {
		return err
	}

	return nil
}

// buildFileName builds a DBMarker config filename.
func (m *Marker) buildFileName() string {
	return path.Join(m.dataPath, configDir, configFilename)
}
