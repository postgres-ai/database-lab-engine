/*
2020 © Postgres.ai
*/

package observer

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" //nolint
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

func initConnection(clone *models.Clone, sslMode string) (*sql.DB, error) {
	db, err := sql.Open("postgres", buildConnectionString(clone, sslMode))
	if err != nil {
		return nil, errors.Wrap(err, "cannot init connection")
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, errors.Wrap(err, "cannot init connection")
	}

	return db, nil
}

func runQuery(db *sql.DB, query string, args ...interface{}) (string, error) {
	var result = ""

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Err("DB query:", err)
		return "", err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Err("Error when closing:", err)
		}
	}()

	for rows.Next() {
		var s string

		if err := rows.Scan(&s); err != nil {
			log.Err("DB query traversal:", err)
			return s, err
		}

		result += s + "\n"
	}

	if err := rows.Err(); err != nil {
		log.Err("DB query traversal:", err)
		return result, err
	}

	return result, nil
}

func buildConnectionString(clone *models.Clone, sslMode string) string {
	db := clone.DB
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		db.Host, db.Port, db.Username, db.Password, sslMode)
}
