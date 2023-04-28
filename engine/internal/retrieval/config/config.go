/*
2020 © Postgres.ai
*/

// Package config contains configuration options of the data retrieval.
package config

import (
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
)

// Config describes of data retrieval jobs.
type Config struct {
	Refresh  Refresh            `yaml:"refresh"`
	Jobs     []string           `yaml:"jobs,flow"`
	JobsSpec map[string]JobSpec `yaml:"spec"`
}

// Refresh describes full-refresh options.
type Refresh struct {
	Timetable string `yaml:"timetable"`
}

// JobSpec contains details about a job.
type JobSpec struct {
	Name    string                 `yaml:"name"`
	Options map[string]interface{} `yaml:"options"`
}

// JobConfig describes a job configuration.
type JobConfig struct {
	Spec   JobSpec
	Docker *client.Client
	Marker *dbmarker.Marker
	FSPool *resources.Pool
}
