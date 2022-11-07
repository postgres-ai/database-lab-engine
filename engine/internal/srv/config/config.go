/*
2021 © Postgres.ai
*/

// Package config contains configuration options of HTTP server.
package config

// Config provides configuration management via DLE API
type Config struct {
	VerificationToken         string `yaml:"verificationToken" json:"-"`
	Host                      string `yaml:"host"`
	Port                      uint   `yaml:"port"`
	DisableConfigModification bool   `yaml:"disableConfigModification" json:"-"`
}
