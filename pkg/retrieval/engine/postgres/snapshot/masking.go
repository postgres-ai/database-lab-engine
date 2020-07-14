/*
2020 © Postgres.ai
*/

package snapshot

import (
	"context"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
)

const (
	// maskingType defines a masking type.
	maskingType = "masking"
)

// MaskingJob describe a masking job.
type MaskingJob struct {
	name string
}

// NewMaskingJob creates a new masking job.
func NewMaskingJob(cfg config.JobConfig) (*MaskingJob, error) {
	maskingJob := &MaskingJob{
		name: cfg.Name,
	}

	return maskingJob, nil
}

// Name returns a name of the job.
func (m *MaskingJob) Name() string {
	return m.name
}

// Run starts the job.
func (m *MaskingJob) Run(ctx context.Context) error {
	return nil
}
