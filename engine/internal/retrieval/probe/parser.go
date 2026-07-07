/*
2026 © Postgres.ai
*/

package probe

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ParseConnectionString parses a libpq DSN or PostgreSQL URI into a Connection.
// It wraps pgx.ParseConfig so it handles URI, DSN, IPv6 hosts and quoted
// values consistently with how the engine later connects.
//
// The parser rejects connection strings that embed a password (via either URI
// userinfo or DSN password= key) — the password is always supplied as a
// separate argument so it never appears in logs, projection writes, or
// telemetry. It also rejects multi-host configs: the simplified-install flow
// targets a single source.
func ParseConnectionString(s string) (Connection, error) {
	cfg, err := pgx.ParseConfig(s)
	if err != nil {
		return Connection{}, fmt.Errorf("parse connection string: %w", err)
	}

	embedsPassword, err := connStringEmbedsPassword(s)
	if err != nil {
		return Connection{}, fmt.Errorf("parse connection string: %w", err)
	}

	if embedsPassword {
		return Connection{}, ErrPasswordInConnString
	}

	// pgx populates Fallbacks with alternative TLS modes for the same host, so
	// the presence of fallbacks alone does not imply multiple targets. A
	// connection string is multi-host only when a fallback advertises a host
	// or port different from the primary.
	for _, fb := range cfg.Fallbacks {
		if fb.Host != cfg.Host || fb.Port != cfg.Port {
			return Connection{}, ErrMultiHostConnString
		}
	}

	return Connection{
		Host:     cfg.Host,
		Port:     int(cfg.Port),
		Username: cfg.User,
		DBName:   cfg.Database,
	}, nil
}

// connStringEmbedsPassword reports whether the connection string literally
// carries a password — in the URI userinfo, the URI password= query parameter,
// or a keyword/value password= key.
//
// It inspects the raw string rather than the parsed config's Password field:
// pgx.ParseConfig merges PGPASSWORD/PGPASSFILE from the process environment into
// that field, so trusting it would reject a password-less string whenever the
// engine runs with PGPASSWORD set (the documented way to supply the source
// password).
func connStringEmbedsPassword(s string) (bool, error) {
	trimmed := strings.TrimSpace(s)

	if isURIConnString(trimmed) {
		u, err := url.Parse(trimmed)
		if err != nil {
			return false, err
		}

		if u.User != nil {
			if _, ok := u.User.Password(); ok {
				return true, nil
			}
		}

		return u.Query().Get("password") != "", nil
	}

	return dsnEmbedsPassword(trimmed), nil
}

// isURIConnString reports whether s is a libpq connection URI.
func isURIConnString(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "postgresql://") || strings.HasPrefix(lower, "postgres://")
}

// dsnEmbedsPassword scans a keyword/value DSN for a "password" key, honouring
// libpq single-quoted values (with \' and \\ escapes) so a password appearing
// inside another option's value is not mistaken for the key.
func dsnEmbedsPassword(dsn string) bool {
	i, n := 0, len(dsn)

	for i < n {
		for i < n && isDSNSpace(dsn[i]) {
			i++
		}

		keyStart := i

		for i < n && dsn[i] != '=' && !isDSNSpace(dsn[i]) {
			i++
		}

		keyword := dsn[keyStart:i]

		for i < n && isDSNSpace(dsn[i]) {
			i++
		}

		if i >= n || dsn[i] != '=' {
			break // malformed token; stop scanning
		}

		i++ // consume '='

		if strings.EqualFold(keyword, "password") {
			return true
		}

		i = skipDSNValue(dsn, i)
	}

	return false
}

// skipDSNValue advances past a DSN value starting at i, handling a single-quoted
// value with backslash escapes or an unquoted whitespace-terminated value.
func skipDSNValue(dsn string, i int) int {
	n := len(dsn)

	for i < n && isDSNSpace(dsn[i]) {
		i++
	}

	if i < n && dsn[i] == '\'' {
		i++

		for i < n {
			if dsn[i] == '\\' && i+1 < n {
				i += 2
				continue
			}

			if dsn[i] == '\'' {
				return i + 1
			}

			i++
		}

		return i
	}

	for i < n && !isDSNSpace(dsn[i]) {
		i++
	}

	return i
}

func isDSNSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
