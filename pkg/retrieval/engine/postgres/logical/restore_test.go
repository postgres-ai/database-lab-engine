/*
2020 © Postgres.ai
*/

package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
)

func TestRestoreCommandBuilding(t *testing.T) {
	logicalJob := &RestoreJob{
		globalCfg: &global.Config{
			Database: global.Database{
				Username: "john",
				DBName:   "testdb",
			},
		},
	}

	testCases := []struct {
		CopyOptions RestoreOptions
		Command     []string
	}{
		{
			CopyOptions: RestoreOptions{
				DBName:       "testDB",
				ParallelJobs: 1,
				ForceInit:    false,
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--create", "--no-privileges", "--no-owner", "--jobs", "1", "/tmp/db.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 4,
				ForceInit:    true,
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--create", "--no-privileges", "--no-owner", "--clean", "--if-exists", "--jobs", "4", ""},
		},
		{
			CopyOptions: RestoreOptions{
				DBName:       "testDB",
				ParallelJobs: 1,
				Partial:      Partial{Tables: []string{"test", "users"}},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--create", "--no-privileges", "--no-owner", "--jobs", "1", "--table", "test", "--table", "users", "/tmp/db.dump"},
		},
	}

	for _, tc := range testCases {
		logicalJob.RestoreOptions = tc.CopyOptions
		restoreCommand := logicalJob.buildLogicalRestoreCommand()

		assert.Equal(t, restoreCommand, tc.Command)
	}
}
