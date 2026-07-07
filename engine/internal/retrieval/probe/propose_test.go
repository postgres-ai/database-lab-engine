/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssembleProposed_HappyPath(t *testing.T) {
	conn := Connection{Host: "db.example.com", Port: 5432, Username: "alice", DBName: "shop"}
	tuning := map[string]string{"work_mem": "4096"}

	got := assembleProposed(proposalInputs{
		conn: conn, pgMajorVersion: 15, collationVersion: "2.36",
		sourceLibs: []string{"pgaudit"}, availableExtensions: []string{"pgaudit"},
		tuning: tuning, hostMemBytes: 16 * 1024 * 1024 * 1024,
	})

	require.Equal(t, conn, got.Source)
	require.Equal(t, ProviderGeneric, got.DetectedProvider)
	require.Equal(t, "generic", got.DockerImage)
	require.Empty(t, got.DockerTag)
	require.Equal(t, 15, got.PgMajorVersion)
	require.Equal(t, "2.36", got.CollationVersion)
	require.Equal(t, []string{"shop"}, got.Databases)
	require.Equal(t, "4GB", got.SharedBuffers)
	require.True(t, got.MemoryProbed)
	require.Equal(t, "pg_stat_statements,pgaudit", got.SharedPreloadLibraries)
	require.Equal(t, tuning, got.QueryTuning)
}

func TestAssembleProposed_ProviderFromHostname(t *testing.T) {
	conn := Connection{Host: "myinst.abc.us-east-1.rds.amazonaws.com", Port: 5432, Username: "alice", DBName: "shop"}

	got := assembleProposed(proposalInputs{conn: conn, pgMajorVersion: 14, availableExtensions: []string{"rds_tools"}})

	require.Equal(t, ProviderRDS, got.DetectedProvider)
	require.Equal(t, "rds", got.DockerImage)
}

func TestAssembleProposed_AuroraFromExtension(t *testing.T) {
	conn := Connection{Host: "myinst.abc.us-east-1.rds.amazonaws.com", Port: 5432, Username: "alice", DBName: "shop"}

	got := assembleProposed(proposalInputs{conn: conn, pgMajorVersion: 14, availableExtensions: []string{"rds_tools", "aurora_stat_utils"}})

	require.Equal(t, ProviderAurora, got.DetectedProvider)
	require.Equal(t, "aurora", got.DockerImage)
}

func TestAssembleProposed_MemoryUnknown(t *testing.T) {
	conn := Connection{Host: "10.20.30.40", DBName: "x"}

	got := assembleProposed(proposalInputs{conn: conn, pgMajorVersion: 14})

	require.False(t, got.MemoryProbed)
	require.Equal(t, "1GB", got.SharedBuffers, "fallback shared_buffers when memory unknown")
}

func TestAssembleProposed_EmptyDBNameMakesEmptyDatabaseList(t *testing.T) {
	conn := Connection{Host: "10.20.30.40"}

	got := assembleProposed(proposalInputs{conn: conn, pgMajorVersion: 14, hostMemBytes: 4 * 1024 * 1024 * 1024})

	require.Empty(t, got.Databases, "no dbname → no database in proposal")
}

func TestAssembleProposed_EnsuresPgStatStatementsInPreloadLibs(t *testing.T) {
	conn := Connection{Host: "10.20.30.40", DBName: "x"}

	got := assembleProposed(proposalInputs{
		conn: conn, pgMajorVersion: 14,
		sourceLibs: []string{"pgaudit", "pg_cron"}, hostMemBytes: 4 * 1024 * 1024 * 1024,
	})

	require.Equal(t, "pg_cron,pg_stat_statements,pgaudit", got.SharedPreloadLibraries)
}

func TestQueryCollationVersion_GuardBelowPG15(t *testing.T) {
	// the guard returns before touching the connection, so a nil conn is safe
	// and proves no query is issued for older majors that lack datcollversion.
	for _, major := range []int{9, 12, 14} {
		version, err := queryCollationVersion(context.Background(), nil, major)
		require.NoError(t, err)
		require.Empty(t, version)
	}
}
