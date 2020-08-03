/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
)

// LogicalInitial describes a job for preparing a logical initial snapshot.
type LogicalInitial struct {
	name         string
	provisionSvc provision.Provision
	options      LogicalOptions
	globalCfg    *dblabCfg.Global
	dbMarker     *dbmarker.Marker
}

// LogicalOptions describes options for a logical initialization job.
type LogicalOptions struct {
	PreprocessingScript string `yaml:"preprocessingScript"`
}

const (
	// LogicalInitialType declares a job type for preparing a logical initial snapshot.
	LogicalInitialType = "logical-snapshot"
)

// NewLogicalInitialJob creates a new logical initial job.
func NewLogicalInitialJob(cfg config.JobConfig, provisionSvc provision.Provision,
	global *dblabCfg.Global, marker *dbmarker.Marker) (*LogicalInitial, error) {
	li := &LogicalInitial{
		name:         cfg.Name,
		provisionSvc: provisionSvc,
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
func (s *LogicalInitial) Run(ctx context.Context) error {
	commandOutput, err := exec.Command("bash", s.options.PreprocessingScript).Output()
	if err != nil {
		return errors.Wrap(err, "failed to run a custom script")
	}

	log.Msg(string(commandOutput))

	// TODO(akartasov): Automated basic Postgres configuration: https://gitlab.com/postgres-ai/database-lab/-/issues/141

	dataStateAt := ""

	dbMark, err := s.dbMarker.GetConfig()
	if err != nil {
		log.Err("Failed to retrieve dataStateAt from a DBMarker config:", err)
	} else {
		dataStateAt = dbMark.DataStateAt
	}

	if err := s.provisionSvc.CreateSnapshot(dataStateAt); err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	return nil
}
