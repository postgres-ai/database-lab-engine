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
		{
			CopyOptions: RestoreOptions{
				Databases: map[string]DBDefinition{
					"testDB.dump": {
						Format: plainFormat,
						dbName: "testDB",
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"psql", "--username", "john", "--dbname", "postgres", "--file", "/tmp/db.dump/testDB.dump"},
		},
		{
			CopyOptions: RestoreOptions{
				Databases: map[string]DBDefinition{
					"testDB.dump": {
						Format: plainFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			Command: []string{"psql", "--username", "john", "--dbname", "testDB", "--file", "/tmp/db.dump/testDB.dump"},
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
		path.Base(tmpDirDB1):      {Format: directoryFormat},
		path.Base(tmpDirDB2):      {Format: directoryFormat},
		path.Base(tmpDirDB3):      {Format: directoryFormat},
		path.Base(tmpFile.Name()): {Format: plainFormat},
	}

	dumpDirectories, err := r.discoverDumpLocation(tmpDirRoot)
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
		{format: plainFormat, dbname: "postgres", expectedLocation: "/tmp/dblab_test/postgres"},
	}

	for _, tc := range testCases {
		dumpLocation := r.getDumpLocation(tc.format, tc.dbname)
		assert.Equal(t, tc.expectedLocation, dumpLocation)
	}
}

const (
	contentPlain = `
-- PostgreSQL database dump

SET statement_timeout = 0;
SET default_tablespace = '';

CREATE TABLE public.pgbench_accounts (
    aid integer NOT NULL,
    bid integer,
    abalance integer,
    filler character(84)
)
WITH (fillfactor='100');`

	contentPlainWithCreateOption = `
-- PostgreSQL database dump
SET statement_timeout = 0;

CREATE DATABASE test WITH TEMPLATE = template0 ENCODING = 'UTF8' LC_COLLATE = 'en_US.utf8' LC_CTYPE = 'en_US.utf8';
ALTER DATABASE test OWNER TO postgres;

\connect test

SET statement_timeout = 0;
SET default_tablespace = '';

-- Name: pgbench_accounts; Type: TABLE; Schema: public; Owner: postgres
CREATE TABLE public.pgbench_accounts (
    aid integer NOT NULL,
    bid integer,
    abalance integer,
    filler character(84)
)
WITH (fillfactor='100');
`

	contentDumpAll = `
-- PostgreSQL database cluster dump
\connect postgres

SET statement_timeout = 0;
SET default_tablespace = '';

-- Name: abc; Type: TABLE; Schema: public; Owner: postgres

CREATE TABLE public.abc (
    a integer
);
`

	invalidDump = `
-- PostgreSQL database cluster dump
\connect 

CREATE TABLE public.abc (
    a integer
);
`
)

func TestParsingPlainFile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		dbname  string
		err     error
	}{
		{
			name:    "plain",
			content: contentPlain,
			dbname:  "",
			err:     errDBNameNotFound,
		},
		{
			name:    "plainWithCreateOption",
			content: contentPlainWithCreateOption,
			dbname:  "test",
			err:     nil,
		},
		{
			name:    "dumpAll",
			content: contentDumpAll,
			dbname:  "postgres",
			err:     nil,
		},
		{
			name:    "invalidDump",
			content: invalidDump,
			dbname:  "",
			err:     errDBNameNotFound,
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		f, err := ioutil.TempFile("", "plain_dump_*")
		require.Nil(t, err)

		// There no many test cases.
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()

		err = ioutil.WriteFile(f.Name(), []byte(tc.content), 0666)
		require.Nil(t, err)

		r := &RestoreJob{}

		dbName, err := r.parsePlainFile(f.Name())
		assert.Equal(t, tc.err, err)
		assert.Equal(t, tc.dbname, dbName)
	}
}

func TestDBNameFormatter(t *testing.T) {
	testCases := []struct {
		filename string
		dbname   string
	}{
		{
			filename: "",
			dbname:   "",
		},
		{
			filename: "testDB",
			dbname:   "testDB",
		},
		{
			filename: "test.dump",
			dbname:   "test",
		},
		{
			filename: "test-dump-2021-07.dump",
			dbname:   "test_dump_2021_07",
		},
	}

	for _, tc := range testCases {
		formattedDB := formatDBName(tc.filename)
		assert.Equal(t, tc.dbname, formattedDB)
	}

}
