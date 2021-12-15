/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
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
		copyOptions       RestoreOptions
		command           []string
		isDumpLocationDir bool
	}{
		{
			copyOptions: RestoreOptions{
				ParallelJobs: 1,
				ForceInit:    false,
				Databases: map[string]DumpDefinition{
					"testDB": {
						Format: customFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--exit-on-error", "--create", "--jobs", "1", "/tmp/db.dump"},
		},
		{
			copyOptions: RestoreOptions{
				ParallelJobs: 4,
				ForceInit:    true,
			},
			command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--exit-on-error", "--create", "--clean", "--if-exists", "--jobs", "4", ""},
		},
		{
			copyOptions: RestoreOptions{
				ParallelJobs: 2,
				ForceInit:    false,
				Databases:    map[string]DumpDefinition{"testDB": {}},
				DumpLocation: "/tmp/db.dump",
			},
			command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--exit-on-error", "--create", "--jobs", "2", "/tmp/db.dump/testDB"},
		},
		{
			copyOptions: RestoreOptions{
				ParallelJobs: 1,
				Databases: map[string]DumpDefinition{
					"testDB": {
						Tables: []string{"test", "users"},
						Format: directoryFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			command: []string{"pg_restore", "--username", "john", "--dbname", "postgres", "--no-privileges", "--no-owner", "--exit-on-error", "--create", "--jobs", "1", "--table", "test", "--table", "users", "/tmp/db.dump/testDB"},
		},
		{
			copyOptions: RestoreOptions{
				Databases: map[string]DumpDefinition{
					"testDB.dump": {
						Format: plainFormat,
						dbName: "testDB",
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			isDumpLocationDir: true,
			command:           []string{"sh", "-c", "cat /tmp/db.dump/testDB.dump | psql --username john --dbname postgres"},
		},
		{
			copyOptions: RestoreOptions{
				Databases: map[string]DumpDefinition{
					"testDB.dump": {
						Format: plainFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			isDumpLocationDir: true,
			command:           []string{"sh", "-c", "cat /tmp/db.dump/testDB.dump | psql --username john --dbname testDB"},
		},
		{
			copyOptions: RestoreOptions{
				Databases: map[string]DumpDefinition{
					"testDB.dump": {
						Format: plainFormat,
					},
				},
				DumpLocation: "/tmp/db.dump",
			},
			isDumpLocationDir: false,
			command:           []string{"sh", "-c", "cat /tmp/db.dump | psql --username john --dbname testDB"},
		},
	}

	for _, tc := range testCases {
		logicalJob.RestoreOptions = tc.copyOptions
		logicalJob.isDumpLocationDir = tc.isDumpLocationDir
		for dbName, definition := range tc.copyOptions.Databases {
			restoreCommand := logicalJob.buildLogicalRestoreCommand(dbName, definition)
			assert.Equal(t, restoreCommand, tc.command)
		}
	}
}

func TestDiscoverDumpDirectories(t *testing.T) {
	t.Skip("docker client is required")

	tmpDirRoot, err := os.MkdirTemp("", "dblab_test_restore_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirRoot) }()

	tmpDirDB1, err := os.MkdirTemp(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB1) }()

	tmpTOCFile1, err := os.Create(path.Join(tmpDirDB1, "toc.dat"))
	require.Nil(t, err)
	err = tmpTOCFile1.Close()
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpTOCFile1.Name()) }()

	tmpDirDB2, err := os.MkdirTemp(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB2) }()

	tmpTOCFile2, err := os.Create(path.Join(tmpDirDB2, "toc.dat"))
	require.Nil(t, err)
	err = tmpTOCFile2.Close()
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpTOCFile2.Name()) }()

	tmpDirDB3, err := os.MkdirTemp(tmpDirRoot, "db_")
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpDirDB3) }()

	tmpTOCFile3, err := os.Create(path.Join(tmpDirDB3, "toc.dat"))
	require.Nil(t, err)
	err = tmpTOCFile3.Close()
	require.Nil(t, err)
	defer func() { _ = os.Remove(tmpTOCFile3.Name()) }()

	expectedMap := map[string]DumpDefinition{
		path.Base(tmpDirDB1): {Format: directoryFormat},
		path.Base(tmpDirDB2): {Format: directoryFormat},
		path.Base(tmpDirDB3): {Format: directoryFormat},
	}

	r := &RestoreJob{
		RestoreOptions: RestoreOptions{
			DumpLocation: tmpDirRoot,
		},
	}

	dumpDirectories, err := r.discoverDumpLocation(context.Background(), "contID")
	assert.Nil(t, err)

	assert.Equal(t, expectedMap, dumpDirectories)
}

func TestDumpLocation(t *testing.T) {
	testCases := []struct {
		format            string
		isDumpLocationDir bool
		dbname            string
		expectedLocation  string
	}{
		{format: directoryFormat, dbname: "postgresDir", expectedLocation: "/tmp/dblab_test/postgresDir"},
		{format: customFormat, dbname: "postgresCustom", isDumpLocationDir: true, expectedLocation: "/tmp/dblab_test/postgresCustom"},
		{format: customFormat, dbname: "postgresCustom", expectedLocation: "/tmp/dblab_test"},
		{format: plainFormat, dbname: "postgresPlain", isDumpLocationDir: true, expectedLocation: "/tmp/dblab_test/postgresPlain"},
		{format: plainFormat, dbname: "postgresPlain", expectedLocation: "/tmp/dblab_test"},
	}

	for _, tc := range testCases {
		r := &RestoreJob{}
		r.RestoreOptions.DumpLocation = "/tmp/dblab_test"
		r.isDumpLocationDir = tc.isDumpLocationDir
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
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		f, err := os.CreateTemp("", "plain_dump_*")
		require.Nil(t, err)

		// There no many test cases.
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()

		err = os.WriteFile(f.Name(), []byte(tc.content), 0666)
		require.Nil(t, err)

		r := &RestoreJob{}

		dbName, err := r.parsePlainFile(f.Name())
		assert.Equal(t, tc.err, err)
		assert.Equal(t, tc.dbname, dbName)
	}
}

func TestParsingInvalidPlainFile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		dbname  string
	}{
		{
			name:    "invalid",
			content: invalidDump,
			dbname:  "",
		},
	}

	for _, tc := range testCases {
		t.Log(tc.name)

		f, err := os.CreateTemp("", "plain_dump_*")
		require.Nil(t, err)

		// There no many test cases.
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()

		err = os.WriteFile(f.Name(), []byte(tc.content), 0666)
		require.Nil(t, err)

		r := &RestoreJob{}

		dbName, err := r.parsePlainFile(f.Name())
		assert.Error(t, err)
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
