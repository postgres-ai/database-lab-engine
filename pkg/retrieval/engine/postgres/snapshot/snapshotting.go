/*
2020 © Postgres.ai
*/

package snapshot

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
)

const (
	// snapshottingType defines a snapshotting type.
	snapshottingType = "snapshotting"
)

// SnapshottingJob describe a snapshotting job.
type SnapshottingJob struct {
	name string
	Options
}

// Options defines snapshot options.
type Options struct {
	Schedule  string           `yaml:"schedule"`
	Retention RetentionOptions `yaml:"retention"`
}

// RetentionOptions defines retention options.
type RetentionOptions struct {
	MaxSnapshotCount int `yaml:"maxSnapshotCount"`
}

// NewSnapshottingJob create a new snapshot job.
func NewSnapshottingJob(cfg config.JobConfig) (*SnapshottingJob, error) {
	snapshottingJob := &SnapshottingJob{
		name: cfg.Name,
	}

	if err := options.Unmarshal(cfg.Options, &snapshottingJob.Options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	return snapshottingJob, nil
}

// Name returns a name of the job.
func (s *SnapshottingJob) Name() string {
	return s.name
}

// Run starts the job.
func (s *SnapshottingJob) Run(ctx context.Context) error {
	fmt.Println("TBD: ", s.name, "Options: ", s.Options)

	return nil
}
