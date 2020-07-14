/*
2020 © Postgres.ai
*/

// Package config contains configuration options of the data retrieval.
package config

// Config describes of data retrieval stages.
type Config struct {
	Stages    []string        `yaml:"stages,flow"`
	StageSpec map[string]Spec `yaml:"spec"`
}

// Spec describes a retrieval job.
type Spec struct {
	Jobs []JobConfig `yaml:"jobs,flow"`
}

// JobConfig contains details about a job.
type JobConfig struct {
	Name    string                 `yaml:"name"`
	Options map[string]interface{} `yaml:"options"`
}
