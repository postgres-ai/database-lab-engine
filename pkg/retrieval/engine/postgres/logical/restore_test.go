/*
2020 © Postgres.ai
*/

package logical

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				ParallelJobs: 1,
				ForceInit:    false,
				Databases: map[string]DBDefinition{
					"testDB": {
						Format: customFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--create", "--jobs", "1", "/tmp/db.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 4,
				ForceInit:    true,
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--create", "--clean", "--if-exists", "--jobs", "4", ""},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 2,
				ForceInit:    false,
				Databases:    map[string]DBDefinition{"testDB": {}},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--create", "--jobs", "2", "/tmp/db.dump/testDB"},
		},
		{
			CopyOptions: RestoreOptions{
				ParallelJobs: 1,
				Databases: map[string]DBDefinition{
					"testDB": {
						Tables: []string{"test", "users"},
						Format: directoryFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--create", "--jobs", "1", "--table", "test", "--table", "users", "/tmp/db.dump/testDB"},
		},
	}

	for _, tc := range testCases {
		logicalJob.RestoreOptions = tc.CopyOptions
		for dbName, definition := range tc.CopyOptions.Databases {
			restoreCommand := logicalJob.buildLogicalRestoreCommand(dbName, definition)
			assert.Equal(t, restoreCommand, tc.Command)
		}
	}
}

func TestDiscoverDumpDirectories(t *testing.T) {
	r := &RestoreJob{}

	tmpDirRoot, err := ioutil.TempDir("", "dblab_test_restore_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirRoot) }()

	tmpDirDB1, err := ioutil.TempDir(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB1) }()

	tmpDirDB2, err := ioutil.TempDir(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB2) }()

	tmpDirDB3, err := ioutil.TempDir(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB3) }()

	tmpFile, err := ioutil.TempFile(tmpDirRoot, "file_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	expectedMap := map[string]DBDefinition{
		path.Base(tmpDirDB1): {Format: directoryFormat},
		path.Base(tmpDirDB2): {Format: directoryFormat},
		path.Base(tmpDirDB3): {Format: directoryFormat},
	}

	dumpDirectories, err := r.discoverDumpDirectories(tmpDirRoot)
	assert.Nil(t, err)

	assert.Equal(t, expectedMap, dumpDirectories)
}

func TestDumpLocation(t *testing.T) {
	r := &RestoreJob{}
	r.RestoreOptions.DumpLocation = "/tmp/dblab_test"

	testCases := []struct {
		format           string
		dbname           string
		expectedLocation string
	}{
		{format: directoryFormat, dbname: "postgres", expectedLocation: "/tmp/dblab_test/postgres"},
		{format: customFormat, dbname: "postgres", expectedLocation: "/tmp/dblab_test"},
		{format: plainFormat, dbname: "postgres", expectedLocation: "/tmp/dblab_test"},
	}

	for _, tc := range testCases {
		dumpLocation := r.getDumpLocation(tc.format, tc.dbname)
		assert.Equal(t, tc.expectedLocation, dumpLocation)
	}
}
