/*
2021 © Postgres.ai
*/

// Package runci provides a tools to run and check migrations in CI.
package runci

import (
	"gitlab.com/postgres-ai/database-lab/v2/pkg/runci/source"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
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
