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

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// collationTestImage is a glibc-based PG15+ image whose default database
// (en_US.utf8 locale, baked into the official image) records a libc collation
// version, so datcollversion is non-empty.
const collationTestImage = "postgres:16"

func startProbePostgres(ctx context.Context, t *testing.T) (host, port string, terminate func()) {
	t.Helper()

	return startProbePostgresImage(ctx, t, tuningTestImage)
}

func startProbePostgresImage(ctx context.Context, t *testing.T, image string) (host, port string, terminate func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        image,
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

	hostStr, err := pg.Host(ctx)
	require.NoError(t, err)

	mapped, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	return hostStr, mapped.Port(), func() { _ = pg.Terminate(ctx) }
}

func TestPropose_HappyPath_Integration(t *testing.T) {
	ctx := context.Background()

	host, port, done := startProbePostgres(ctx, t)
	defer done()

	connStr := fmt.Sprintf("postgres://postgres@%s:%s/postgres?sslmode=disable", host, port)

	got, err := Propose(ctx, connStr, "probe-test-pw", nil)
	require.NoError(t, err)

	require.Equal(t, host, got.Source.Host)
	require.Equal(t, "postgres", got.Source.Username)
	require.Equal(t, "postgres", got.Source.DBName)
	require.Equal(t, []string{"postgres"}, got.Databases)
	require.Equal(t, ProviderGeneric, got.DetectedProvider, "official postgres image has no provider markers")
	require.Equal(t, "generic", got.DockerImage)
	require.Empty(t, got.DockerTag, "DockerTag is UI-resolved in 4.2")
	require.Equal(t, 14, got.PgMajorVersion, "matches the test image major version")
	require.Contains(t, got.SharedPreloadLibraries, "pg_stat_statements", "pg_stat_statements always present")
	require.NotEmpty(t, got.QueryTuning, "tuning whitelist resolved on this image")
}

func TestPropose_CollationVersion_Integration(t *testing.T) {
	ctx := context.Background()

	host, port, done := startProbePostgresImage(ctx, t, collationTestImage)
	defer done()

	connStr := fmt.Sprintf("postgres://postgres@%s:%s/postgres?sslmode=disable", host, port)

	got, err := Propose(ctx, connStr, "probe-test-pw", nil)
	require.NoError(t, err)
	require.Equal(t, 16, got.PgMajorVersion, "matches the collation test image major version")
	require.NotEmpty(t, got.CollationVersion, "a PG15+ libc database reports a recorded collation version")
}

func TestPropose_WrongPassword_Integration(t *testing.T) {
	ctx := context.Background()

	host, port, done := startProbePostgres(ctx, t)
	defer done()

	connStr := fmt.Sprintf("postgres://postgres@%s:%s/postgres?sslmode=disable", host, port)

	_, err := Propose(ctx, connStr, "definitely-not-the-password", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect to source")
}

func TestPropose_PasswordInConnString_Integration(t *testing.T) {
	// guard against regression: passwords must never be embedded in the connection string —
	// the rejection happens in ParseConnectionString before any network call.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Propose(ctx, "postgres://alice:embedded@10.255.255.1/postgres", "ignored", nil)
	require.ErrorIs(t, err, ErrPasswordInConnString)
}

func TestPropose_UnreachableHost_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// RFC 5737 TEST-NET-1 — guaranteed not to resolve to a real host.
	_, err := Propose(ctx, "postgres://postgres@192.0.2.1:5432/postgres?sslmode=disable", "any-pw", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect to source")
}
