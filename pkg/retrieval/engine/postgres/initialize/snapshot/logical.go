/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"context"
	"path"

	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

// LogicalInitial describes a job for preparing a logical initial snapshot.
type LogicalInitial struct {
	name         string
	cloneManager thinclones.Manager
	options      LogicalOptions
	globalCfg    *dblabCfg.Global
	dbMarker     *dbmarker.Marker
}

// LogicalOptions describes options for a logical initialization job.
type LogicalOptions struct {
	PreprocessingScript string            `yaml:"preprocessingScript"`
	Configs             map[string]string `yaml:"configs"`
}

const (
	// LogicalInitialType declares a job type for preparing a logical initial snapshot.
	LogicalInitialType = "logical-snapshot"
)

// NewLogicalInitialJob creates a new logical initial job.
func NewLogicalInitialJob(cfg config.JobConfig, cloneManager thinclones.Manager,
	global *dblabCfg.Global, marker *dbmarker.Marker) (*LogicalInitial, error) {
	li := &LogicalInitial{
		name:         cfg.Name,
		cloneManager: cloneManager,
		globalCfg:    global,
		dbMarker:     marker,
	}

	if err := options.Unmarshal(cfg.Options, &li.options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	return li, nil
}

// Name returns a name of the job.
func (s *LogicalInitial) Name() string {
	return s.name
}

// Run starts the job.
func (s *LogicalInitial) Run(_ context.Context) error {
	if s.options.PreprocessingScript != "" {
		if err := runPreprocessingScript(s.options.PreprocessingScript); err != nil {
			return err
		}
	}

	if err := s.touchConfigFiles(); err != nil {
		return errors.Wrap(err, "failed to create PostgreSQL configuration files")
	}

	// Run basic PostgreSQL configuration.
	if err := configuration.Run(s.globalCfg.DataDir); err != nil {
		return errors.Wrap(err, "failed to adjust PostgreSQL configs")
	}

	// Apply user defined configs.
	if err := applyUsersConfigs(s.options.Configs, path.Join(s.globalCfg.DataDir, "postgresql.conf")); err != nil {
		return errors.Wrap(err, "failed to apply user-defined configs")
	}

	dataStateAt := extractDataStateAt(s.dbMarker)

	if _, err := s.cloneManager.CreateSnapshot("", dataStateAt); err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	return nil
}

func (s *LogicalInitial) touchConfigFiles() error {
	if err := tools.TouchFile(path.Join(s.globalCfg.DataDir, "postgresql.conf")); err != nil {
		return err
	}

	return tools.TouchFile(path.Join(s.globalCfg.DataDir, "pg_hba.conf"))
}
