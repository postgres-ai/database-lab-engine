/*
2021 © Postgres.ai
*/

// Package global provides access to the global Database Lab Engine configuration.
package global

import (
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/defaults"
)

// Config contains global Database Lab configurations.
type Config struct {
	Database  Database  `yaml:"database"`
	Engine    string    `yaml:"engine"`
	Debug     bool      `yaml:"debug"`
	Telemetry Telemetry `yaml:"telemetry"`
}

// Database contains default configurations of the managed database.
type Database struct {
	Username string `yaml:"username"`
	DBName   string `yaml:"dbname"`
}

// User returns default Database username.
func (d *Database) User() string {
	if d.Username != "" {
		return d.Username
	}

	return defaults.Username
}

// Name returns default Database name.
func (d *Database) Name() string {
	if d.DBName != "" {
		return d.DBName
	}

	return defaults.DBName
}

// Telemetry contains configuration of Database Lab Engine telemetry.
type Telemetry struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// EngineProps contains internal Database Lab Engine properties.
type EngineProps struct {
	InstanceID    string
	ContainerName string
	EnginePort    uint
}
