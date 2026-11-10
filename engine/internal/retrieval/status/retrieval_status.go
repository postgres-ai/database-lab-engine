/*
2022 © Postgres.ai
*/

// Package status provides status of retrieval status.
package status

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	lagQuery = `
		SELECT
		  CASE WHEN pg_is_in_recovery() THEN (
			SELECT
			  CASE WHEN pg_last_wal_receive_lsn() = pg_last_wal_replay_lsn() THEN 0
			  ELSE
				coalesce(round(date_part('epoch', now() - pg_last_xact_replay_timestamp())::int8, 0), 0)
			  END)
		  ELSE 0
		  END lag_sec;
	`

	lastReplayedLsnQuery = `
		SELECT
		  pg_last_xact_replay_timestamp(),
		  pg_last_wal_replay_lsn()
		;
	`

	syncUptimeQuery = `
		SELECT
		 extract(epoch from (now() - pg_postmaster_start_time()))::int8 as uptime_sec
		FROM
		 pg_stat_database
		LIMIT 1;
	`
)

// FetchSyncMetrics - fetch synchronization status.
func FetchSyncMetrics(ctx context.Context, config *global.Config, socketPath string) (*models.Sync, error) {
	var sync = models.Sync{
		Status: models.Status{Code: models.SyncStatusOK},
	}

	conn, err := openConnection(ctx, config.Database.User(), config.Database.Name(), socketPath)
	if err != nil {
		return &models.Sync{
			Status: models.Status{Code: models.SyncStatusError},
		}, err
	}

	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Dbg("Failed to close connection", err)
		}
	}()

	if err := conn.Ping(ctx); err != nil {
		return &sync, fmt.Errorf("can't db connection %w", err)
	}

	replicationLag, err := lag(ctx, conn)
	if err != nil {
		log.Warn("Failed to fetch replication lag", err)
	} else {
		sync.ReplicationLag = replicationLag
	}

	lastReplayedLsnAt, replayedLsn, err := lastReplayedLsn(ctx, conn)
	if err != nil {
		log.Warn("Failed to fetch last replayed lsn", err)
	} else {
		sync.LastReplayedLsn = replayedLsn
		sync.LastReplayedLsnAt = lastReplayedLsnAt
	}

	uptime, err := syncUptime(ctx, conn)
	if err != nil {
		log.Warn("Failed to fetch postgres sync uptime", err)
	} else {
		sync.ReplicationUptime = uptime
	}

	return &sync, nil
}

func openConnection(ctx context.Context, username, dbname, socketPath string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, connectionString(username, dbname, socketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	return conn, nil
}

func connectionString(username, dbname, socketPath string) string {
	return fmt.Sprintf(`host='%s' port=%d user='%s' dbname='%s'`,
		db.EscapeLibpqValue(socketPath), defaults.Port, db.EscapeLibpqValue(username), db.EscapeLibpqValue(dbname))
}

func lag(ctx context.Context, conn *pgx.Conn) (int, error) {
	var lagSec int

	row := conn.QueryRow(ctx, lagQuery)

	if err := row.Scan(&lagSec); err != nil {
		return 0, fmt.Errorf("failed to read replication lag: %w", err)
	}

	return lagSec, nil
}

func lastReplayedLsn(ctx context.Context, conn *pgx.Conn) (string, string, error) {
	var timestamp, location sql.NullString

	row := conn.QueryRow(ctx, lastReplayedLsnQuery)

	if err := row.Scan(&timestamp, &location); err != nil {
		return "", "", fmt.Errorf("failed to read lsn data: %w", err)
	}

	return timestamp.String, location.String, nil
}

func syncUptime(ctx context.Context, conn *pgx.Conn) (int, error) {
	var uptime int

	row := conn.QueryRow(ctx, syncUptimeQuery)

	if err := row.Scan(&uptime); err != nil {
		return 0, fmt.Errorf("failed to read sync uptime: %w", err)
	}

	return uptime, nil
}
