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
			},
			Command: []string{"pg_restore", "-U", "postgres", "-C", "-d", "testDB", "-j", "1", "/tmp/db.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 4,
				ForceInit:    true,
			},
			Command: []string{"pg_restore", "-U", "postgres", "-C", "-d", "postgres", "--clean", "--if-exists", "-j", "4", "/tmp/db.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				DBName:       "testDB",
				ParallelJobs: 1,
				Partial:      Partial{Tables: []string{"test", "users"}},
			},
			Command: []string{"pg_restore", "-U", "postgres", "-C", "-d", "testDB", "-j", "1", "-t", "test", "-t", "users", "/tmp/db.dump"},
		},
	}

	for _, tc := range testCases {
		logicalJob.RestoreOptions = tc.CopyOptions
		restoreCommand := logicalJob.buildLogicalRestoreCommand()

		assert.Equal(t, restoreCommand, tc.Command)
	}
}
