/*
2021 © Postgres.ai
*/

// Package db provides database helpers.
package db

import (
	"fmt"
)

// ConnectionString builds PostgreSQL connection string.
func ConnectionString(host, port, username, dbname, password string) string {
	return fmt.Sprintf("host=%s port=%s user='%s' database='%s' password='%s'", host, port, username, dbname, password)
}
