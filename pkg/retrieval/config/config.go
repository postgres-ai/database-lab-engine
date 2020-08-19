/*
2020 © Postgres.ai
*/

// Package config contains configuration options of the data retrieval.
package config

// Config describes of data retrieval stages.
type Config struct {
	Jobs     []string             `yaml:"jobs,flow"`
	JobsSpec map[string]JobConfig `yaml:"spec"`
}

// JobConfig contains details about a job.
type JobConfig struct {
	Name    string                 `yaml:"name"`
	Options map[string]interface{} `yaml:"options"`
}
