// +build integration

/*
2021 © Postgres.ai
*/

package snapshot

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/client"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	initialScript96 = `
-- SCHEMA
begin;
create table timezones
(
  id         serial PRIMARY KEY,
  created    timestamptz DEFAULT now() NOT NULL,
  modified   timestamptz DEFAULT now() NOT NULL,
  name       text                      NOT NULL,
  timeoffset smallint                  NOT NULL,
  identifier text                      NOT NULL
);
commit;
select pg_switch_xlog();

-- SEED
begin;
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (1, 'eastern', '-5', 'est');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (2, 'central', '-6', 'cst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (3, 'mountain', '-7', 'mst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (4, 'pacific', '-8', 'pst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (5, 'alaska', '-9', 'ast');
alter sequence timezones_id_seq restart with 6;
commit;
select pg_switch_xlog();
`

	initialScript = `
-- SCHEMA
begin;
create table timezones
(
  id         serial PRIMARY KEY,
  created    timestamptz DEFAULT now() NOT NULL,
  modified   timestamptz DEFAULT now() NOT NULL,
  name       text                      NOT NULL,
  timeoffset smallint                  NOT NULL,
  identifier text                      NOT NULL
);
commit;
select pg_switch_wal();

-- SEED
begin;
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (1, 'eastern', '-5', 'est');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (2, 'central', '-6', 'cst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (3, 'mountain', '-7', 'mst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (4, 'pacific', '-8', 'pst');
INSERT INTO timezones  (id, name, timeoffset, identifier) VALUES  (5, 'alaska', '-9', 'ast');
alter sequence timezones_id_seq restart with 6;
commit;
select pg_switch_wal();
`
)

const (
	port         = "5432/tcp"
	dbname       = "postgres"
	user         = "postgres"
	testPassword = "password"
	pgdata       = "/var/lib/postgresql/data/"
)

func TestParsingWAL96(t *testing.T) {
	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal("Failed to create a Docker client:", err)
	}

	testWALParsing(t, dockerCLI, 9.6, initialScript96)
}

func TestParsingWAL(t *testing.T) {
	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal("Failed to create a Docker client:", err)
	}

	postgresVersions := []float64{10, 11, 12, 13, 14}

	for _, pgVersion := range postgresVersions {
		testWALParsing(t, dockerCLI, pgVersion, initialScript)
	}
}

func testWALParsing(t *testing.T, dockerCLI *client.Client, pgVersion float64, initialSQL string) {
	ctx := context.Background()

	pgVersionString := fmt.Sprintf("%g", pgVersion)

	// Create a temporary directory to store PGDATA.
	dir, err := os.MkdirTemp("", "pg_test_"+pgVersionString+"_")
	require.Nil(t, err)

	defer os.Remove(dir)

	// Run a test container.
	logStrategyForAcceptingConnections := wait.NewLogStrategy("database system is ready to accept connections")
	logStrategyForAcceptingConnections.Occurrence = 2

	req := testcontainers.ContainerRequest{
		Name:         "pg_test_" + pgVersionString,
		Image:        "postgres:" + pgVersionString,
		ExposedPorts: []string{port},
		WaitingFor: wait.ForAll(
			logStrategyForAcceptingConnections,
			wait.ForLog("PostgreSQL init process complete; ready for start up."),
		),
		Env: map[string]string{
			"POSTGRES_PASSWORD": testPassword,
			"PGDATA":            pgdata,
		},
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.Nil(t, err)

	defer func() { _ = postgresContainer.Terminate(ctx) }()

	// Prepare test data.
	code, err := postgresContainer.Exec(ctx, []string{"psql", "-U", user, "-d", dbname, "-XAtc", initialSQL})
	require.Nil(t, err)
	assert.Equal(t, 0, code)

	p := &PhysicalInitial{
		dockerClient: dockerCLI,
	}

	// Check WAL parsing.
	dsa, err := p.getDSAFromWAL(ctx, pgVersion, postgresContainer.GetContainerID(), pgdata)
	assert.Nil(t, err)
	assert.NotEmpty(t, dsa)

	t.Log("DSA: ", dsa)
}
