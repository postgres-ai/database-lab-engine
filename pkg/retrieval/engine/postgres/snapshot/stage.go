/*
2020 © Postgres.ai
*/

// Package snapshot provides components of a snapshot stage.
package snapshot

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
)

const (
	// StageType declares a snapshot stage type.
	StageType = "snapshot"
)

// Stage defines a snapshot stage.
type Stage struct {
	name string
	jobs []components.JobRunner
}

// NewStage creates a new snapshot stage.
func NewStage(name string) *Stage {
	return &Stage{
		name: name,
	}
}

// BuildJob builds stage jobs.
func (s *Stage) BuildJob(jobCfg config.JobConfig) (components.JobRunner, error) {
	switch jobCfg.Name {
	case snapshottingType:
		return NewSnapshottingJob(jobCfg)

	case maskingType:
		return NewMaskingJob(jobCfg)
	}

	return nil, errors.New("unknown job type")
}

// AddJob applies jobs to the current stage.
func (s *Stage) AddJob(job components.JobRunner) {
	s.jobs = append(s.jobs, job)
}

// Run starts the snapshot stage.
func (s *Stage) Run(ctx context.Context) error {
	log.Msg(fmt.Sprintf("Running the stage: %s", s.name))

	for _, j := range s.jobs {
		if err := j.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}
