/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"

	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
)

// DumpJobType declares a job type for logical dumping.
const DumpJobType = "logical-dump"

// DumpJob declares a job for logical dumping.
type DumpJob struct {
	name string
}

// NewDumpJob creates a new DumpJob.
func NewDumpJob(cfg config.JobConfig) (*DumpJob, error) {
	configureJob := &DumpJob{
		name: cfg.Name,
	}

	return configureJob, nil
}

// Name returns a name of the job.
func (d *DumpJob) Name() string {
	return d.name
}

// Run starts the job.
func (d *DumpJob) Run(ctx context.Context) error {
	fmt.Println("TBD: ", d.name)

	return nil
}
