/*
2026 © Postgres.ai
*/

package logical

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
)

// uriSchemePrefixes are the libpq-recognised URI scheme prefixes. A connection
// string that begins with either is parsed as a URI; anything else is treated
// as keyword/value (DSN) form.
var uriSchemePrefixes = []string{"postgresql://", "postgres://"}

// withDatabase returns connStr with its database name overridden by dbName while
// preserving every other libpq parameter (sslmode, connect_timeout, options, …).
// It accepts both the URI form (postgresql://…) and the keyword/value form
// (host=… dbname=…). The rest of the string is kept opaque so no option is lost.
//
// Per-database dumps and per-database pgx connections call this so they target
// the right database without rebuilding a lossy DSN from parsed fields.
//
// An empty dbName leaves the string untouched, preserving whatever database the
// connection string already targets (or libpq's default) rather than forcing a
// blank database name.
func withDatabase(connStr, dbName string) (string, error) {
	if dbName == "" {
		return connStr, nil
	}

	if isURIConnString(connStr) {
		return withDatabaseURI(connStr, dbName)
	}

	return withDatabaseKV(connStr, dbName), nil
}

// shellQuote single-quotes s for safe inclusion in a `sh -c` command line.
// Everything between single quotes is literal to the shell, so spaces and
// metacharacters (whitespace, $(), backticks, &, |, …) in a libpq connection
// string cannot break argument boundaries or be interpreted as shell syntax.
// An embedded single quote is escaped by closing the quote, adding an escaped
// quote, and reopening it (the standard POSIX idiom).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isURIConnString reports whether s is a libpq connection URI.
func isURIConnString(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))

	for _, p := range uriSchemePrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	return false
}

// withDatabaseURI overrides the path component (the database) of a connection
// URI, leaving the query string (where sslmode and friends live) untouched.
func withDatabaseURI(connStr, dbName string) (string, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return "", fmt.Errorf("parse connection URI: %w", err)
	}

	u.Path = "/" + dbName

	return u.String(), nil
}

// withDatabaseKV appends a dbname keyword to a keyword/value DSN. libpq resolves
// duplicate keywords last-wins, so an existing dbname= earlier in the string is
// overridden. The value is libpq-escaped and single-quoted.
func withDatabaseKV(connStr, dbName string) string {
	trimmed := strings.TrimRight(connStr, " ")
	suffix := fmt.Sprintf("dbname='%s'", db.EscapeLibpqValue(dbName))

	if trimmed == "" {
		return suffix
	}

	return trimmed + " " + suffix
}

// sourcePgxConfig builds a pgx config that connects to dbName on the source.
// When connStr is non-empty the raw libpq string is parsed (so sslmode and every
// other option are preserved) with the database overridden to dbName via
// withDatabase; otherwise a DSN is built from the discrete connection fields.
// The password is taken from conn.Password and injected into the parsed config
// separately, so it never appears in the connection string itself.
func sourcePgxConfig(connStr string, conn Connection, dbName string) (*pgx.ConnConfig, error) {
	if connStr != "" {
		withDB, err := withDatabase(connStr, dbName)
		if err != nil {
			return nil, err
		}

		cfg, err := pgx.ParseConfig(withDB)
		if err != nil {
			return nil, fmt.Errorf("parse source connection string: %w", err)
		}

		cfg.Password = conn.Password

		return cfg, nil
	}

	dsn := db.ConnectionString(conn.Host, strconv.Itoa(conn.Port), conn.Username, dbName, conn.Password)

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse source DSN: %w", err)
	}

	return cfg, nil
}
