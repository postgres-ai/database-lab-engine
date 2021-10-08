/*
2020 © Postgres.ai
*/

package observer

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

// InitConnection creates a new connection to the clone database.
func InitConnection(clone *models.Clone, socketDir string) (*pgx.Conn, error) {
	host, err := unixSocketDir(socketDir, clone.DB.Port)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse clone port")
	}

	connectionStr := buildConnectionString(clone, host)

	conn, err := pgx.Connect(context.Background(), connectionStr)
	if err != nil {
		log.Err("DB connection:", err)
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot init connection")
	}

	return conn, nil
}

func runQuery(ctx context.Context, db *pgx.Conn, query string, args ...interface{}) (string, error) {
	result := strings.Builder{}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		log.Err("DB query:", err)
		return "", err
	}

	defer rows.Close()

	for rows.Next() {
		var s string

		if err := rows.Scan(&s); err != nil {
			log.Err("DB query traversal:", err)
			return s, err
		}

		result.WriteString(s)
		result.WriteString("\n")
	}

	if err := rows.Err(); err != nil {
		log.Err("DB query traversal:", err)
		return result.String(), err
	}

	return result.String(), nil
}

func unixSocketDir(socketDir, portStr string) (string, error) {
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return "", err
	}

	return path.Join(socketDir, util.GetCloneName(uint(port))), nil
}

func buildConnectionString(clone *models.Clone, socketDir string) string {
	db := clone.DB

	if db.DBName == "" {
		db.DBName = defaults.DBName
	}

	return fmt.Sprintf(`host=%s port=%s user=%s database='%s' application_name='%s'`,
		socketDir,
		db.Port,
		db.Username,
		db.DBName,
		observerApplicationName)
}
