//go:build integration
// +build integration

/*
2026 © Postgres.ai
*/

package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const probeSourceTestImage = "postgres:14"

func startProbeSourcePostgres(ctx context.Context, t *testing.T) (host, port string, terminate func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        probeSourceTestImage,
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "probe-handler-pw"},
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

func TestProbeSource_HappyPath_Integration(t *testing.T) {
	ctx := context.Background()

	host, port, done := startProbeSourcePostgres(ctx, t)
	defer done()

	srv := newProbeTestServer(t, false)

	body := fmt.Sprintf(`{"url":"postgres://postgres@%s:%s/postgres?sslmode=disable","password":"probe-handler-pw"}`, host, port)
	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got models.ProposedConfig
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Equal(t, host, got.Source.Host)
	require.Equal(t, "postgres", got.Source.Username)
	require.Equal(t, "postgres", got.Source.DBName)
	require.Equal(t, "generic", got.DetectedProvider)
	require.Equal(t, 14, got.PgMajorVersion)
	require.Contains(t, got.SharedPreloadLibraries, "pg_stat_statements")
	require.NotEmpty(t, got.QueryTuning)
}

func TestProbeSource_WrongPassword_Integration(t *testing.T) {
	ctx := context.Background()

	host, port, done := startProbeSourcePostgres(ctx, t)
	defer done()

	srv := newProbeTestServer(t, false)

	body := fmt.Sprintf(`{"url":"postgres://postgres@%s:%s/postgres?sslmode=disable","password":"wrong"}`, host, port)
	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, rec.Body.String(), "wrong", "user-supplied password must not be reflected in the response")
}

func TestProbeSource_UnreachableHost_Integration(t *testing.T) {
	srv := newProbeTestServer(t, false)

	// RFC 5737 TEST-NET-1 — guaranteed not to resolve to a real host.
	body := `{"url":"postgres://postgres@192.0.2.1:5432/postgres?sslmode=disable","password":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
