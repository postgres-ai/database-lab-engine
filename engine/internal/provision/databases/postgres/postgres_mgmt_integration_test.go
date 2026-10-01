//go:build integration
// +build integration

/*
2026 © Postgres.ai
*/

package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
)

const (
	restrictionTestAdmin = "postgres"
	restrictionTestUser  = "restricted_user"

	// measurements_1 owns a sequence of its own: a partition that keeps its previous owner
	// also blocks the ownership change of every sequence linked to it.
	restrictionTestSchema = `
create table measurements (id int, region int) partition by list (region);
create table measurements_1 (id serial, region int);
alter table measurements attach partition measurements_1 for values in (1);
create table measurements_2 partition of measurements for values in (2);`

	restrictionTestNotOwnedQuery = `
select count(*) from pg_class
where relname like 'measurements%' and relkind in ('r', 'p', 'S') and relowner <> $1::regrole`

	restrictionTestAttachedQuery = `select count(*) from pg_inherits where inhparent = 'measurements'::regclass`

	restrictionTestPendingQuery = `select inhdetachpending from pg_inherits where inhrelid = 'measurements_1'::regclass`
)

func startRestrictionPostgres(ctx context.Context, t *testing.T, image string) *resources.AppConfig {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_HOST_AUTH_METHOD": "trust"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	host, err := pg.Host(ctx)
	require.NoError(t, err)

	mapped, err := pg.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	port, err := strconv.ParseUint(mapped.Port(), 10, 32)
	require.NoError(t, err)

	return &resources.AppConfig{
		Host: host,
		Port: uint(port),
		DB:   &resources.DB{Username: restrictionTestAdmin, DBName: restrictionTestAdmin},
	}
}

// leavePendingDetach cancels a concurrent detach while it waits for a transaction that still uses
// the partitioned table. The pending flag is committed before that wait, so it survives the cancel.
func leavePendingDetach(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	holder, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = holder.ExecContext(ctx, "select count(*) from measurements")
	require.NoError(t, err)

	conn, err := db.Conn(ctx)
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	_, err = conn.ExecContext(ctx, "set statement_timeout = '1s'")
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, "alter table measurements detach partition measurements_1 concurrently")
	require.ErrorContains(t, err, "statement timeout")
	require.NoError(t, holder.Rollback())

	var pending bool

	require.NoError(t, db.QueryRowContext(ctx, restrictionTestPendingQuery).Scan(&pending))
	require.True(t, pending, "the partition must be left with an incomplete detach")
}

func TestCreateUser_RestrictedOwnsPartitions_Integration(t *testing.T) {
	testCases := []struct {
		name          string
		image         string
		pendingDetach bool
		attached      int
	}{
		{name: "version without concurrent detach", image: "postgres:13", pendingDetach: false, attached: 2},
		{name: "oldest version with concurrent detach", image: "postgres:14", pendingDetach: true, attached: 1},
		{name: "recent version with concurrent detach", image: "postgres:17", pendingDetach: true, attached: 1},
	}

	// production reaches a clone through its unix socket, where lib/pq never negotiates ssl.
	t.Setenv("PGSSLMODE", "disable")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			appConfig := startRestrictionPostgres(ctx, t, tc.image)

			connStr := getPgConnStr(appConfig.Host, appConfig.DB.DBName, appConfig.DB.Username, appConfig.Port)

			db, err := sql.Open("postgres", connStr)
			require.NoError(t, err)

			defer func() { _ = db.Close() }()

			_, err = db.ExecContext(ctx, restrictionTestSchema)
			require.NoError(t, err)

			if tc.pendingDetach {
				leavePendingDetach(ctx, t, db)
			}

			user := resources.EphemeralUser{Name: restrictionTestUser, Password: "restriction-test-pw", Restricted: true}
			require.NoError(t, CreateUser(appConfig, user))

			var notOwned, attached int

			require.NoError(t, db.QueryRowContext(ctx, restrictionTestNotOwnedQuery, restrictionTestUser).Scan(&notOwned))
			assert.Zero(t, notOwned, "every table and sequence must belong to the restricted user")

			require.NoError(t, db.QueryRowContext(ctx, restrictionTestAttachedQuery).Scan(&attached))
			assert.Equal(t, tc.attached, attached)
		})
	}
}
