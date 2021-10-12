/*
2021 © Postgres.ai
*/

// Package config contains configuration options of HTTP server.
package config

// Config provides configuration for an HTTP server of the Database Lab.
type Config struct {
	VerificationToken string `yaml:"verificationToken"`
	Host              string `yaml:"host"`
	Port              uint   `yaml:"port"`
}
