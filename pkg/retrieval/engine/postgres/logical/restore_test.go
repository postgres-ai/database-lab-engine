/*
2020 © Postgres.ai
*/

package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestoreCommandBuilding(t *testing.T) {
	logicalJob := &RestoreJob{}

	testCases := []struct {
		CopyOptions RestoreOptions
		Command     []string
	}{
		{
			CopyOptions: RestoreOptions{
				DBName:       "testDB",
				ParallelJobs: 1,
				ForceInit:    false,
				DumpFile:     "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "postgres", "--dbname", "postgres", "--create", "--no-privileges", "--jobs", "1", "/tmp/db.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 4,
				ForceInit:    true,
			},
			Command: []string{"pg_restore", "--username", "postgres", "--dbname", "postgres", "--create", "--no-privileges", "--clean", "--if-exists", "--jobs", "4", ""},
		},
		{
			CopyOptions: RestoreOptions{
				DBName:       "testDB",
				ParallelJobs: 1,
				Partial:      Partial{Tables: []string{"test", "users"}},
				DumpFile:     "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "postgres", "--dbname", "postgres", "--create", "--no-privileges", "--jobs", "1", "--table", "test", "--table", "users", "/tmp/db.dump"},
		},
	}

	for _, tc := range testCases {
		logicalJob.RestoreOptions = tc.CopyOptions
		restoreCommand := logicalJob.buildLogicalRestoreCommand()

		assert.Equal(t, restoreCommand, tc.Command)
	}
}
