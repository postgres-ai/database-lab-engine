/*
2021 © Postgres.ai
*/

// Package config contains configuration options of HTTP server.
package config

// Engine provides configuration for Database Lab Engine.
type Engine struct {
	StateDir          string   `yaml:"stateDir"`
	DockerImage       string   `yaml:"dockerImage"`
	Volumes           []string `yaml:"volumes"`
	Envs              []string `yaml:"envs"`
	VerificationToken string   `yaml:"verificationToken"`
	Host              string   `yaml:"host"`
	Port              int      `yaml:"port"`
}
