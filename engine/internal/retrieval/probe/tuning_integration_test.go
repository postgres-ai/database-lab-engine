//go:build integration
// +build integration

/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// pinned to the minimum supported version per the simplified-install plan:
// every GUC in tuningParamNames must resolve here. Bumping this image to a
// newer major silently masks a regression where a pg13+-only GUC creeps into
// the whitelist.
const tuningTestImage = "postgres:14"

func TestCollectTuningParams_Integration(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        tuningTestImage,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "probe-test-pw",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	defer func() { _ = pg.Terminate(ctx) }()

	host, err := pg.Host(ctx)
	require.NoError(t, err)

	mapped, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	connStr := fmt.Sprintf("postgres://postgres:probe-test-pw@%s:%s/postgres?sslmode=disable", host, mapped.Port())

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(t, err)

	defer func() { _ = conn.Close(ctx) }()

	params, err := CollectTuningParams(ctx, conn)
	require.NoError(t, err)

	// every whitelisted parameter must resolve against the minimum supported pg version
	require.Len(t, params, len(tuningParamNames), "all whitelisted params should be present on %s", tuningTestImage)

	for _, name := range tuningParamNames {
		_, ok := params[name]
		require.True(t, ok, "missing parameter %q in result", name)
	}

	require.NotEmpty(t, params["work_mem"], "work_mem should have a non-empty setting")
}

func TestCollectTuningParams_ClosedConn_Integration(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        tuningTestImage,
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "probe-test-pw"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	defer func() { _ = pg.Terminate(ctx) }()

	host, err := pg.Host(ctx)
	require.NoError(t, err)

	mapped, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	connStr := fmt.Sprintf("postgres://postgres:probe-test-pw@%s:%s/postgres?sslmode=disable", host, mapped.Port())

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(t, err)

	require.NoError(t, conn.Close(ctx))

	_, err = CollectTuningParams(ctx, conn)
	require.Error(t, err, "querying a closed connection should return a wrapped error")
	require.Contains(t, err.Error(), "query pg_settings", "error should be wrapped with probe context")
}
