/*
2026 © Postgres.ai
*/

package probe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConnectionString(t *testing.T) {
	// isolate from host env so pgx default-port and default-user lookups are deterministic
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "")

	tests := []struct {
		name    string
		input   string
		want    Connection
		wantErr error
	}{
		{name: "uri happy path", input: "postgres://alice@db.example.com:5433/shop", want: Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "shop"}},
		{name: "postgresql scheme alias", input: "postgresql://alice@db.example.com:5433/shop", want: Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "shop"}},
		{name: "dsn happy path", input: "host=db.example.com port=5433 user=alice dbname=shop", want: Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "shop"}},
		{name: "uri missing port", input: "postgres://alice@db.example.com/shop", want: Connection{Host: "db.example.com", Port: 5432, Username: "alice", DBName: "shop"}},
		{name: "uri missing dbname leaves dbname empty", input: "postgres://alice@db.example.com:5433", want: Connection{Host: "db.example.com", Port: 5433, Username: "alice"}},
		{name: "ipv6 host", input: "postgres://alice@[::1]:5433/shop", want: Connection{Host: "::1", Port: 5433, Username: "alice", DBName: "shop"}},
		{name: "password in uri userinfo", input: "postgres://alice:secret@db.example.com/shop", wantErr: ErrPasswordInConnString},
		{name: "password in dsn key", input: "host=db.example.com user=alice dbname=shop password=secret", wantErr: ErrPasswordInConnString},
		{name: "multi-host uri", input: "postgres://alice@db1.example.com,db2.example.com/shop", wantErr: ErrMultiHostConnString},
		{name: "malformed input", input: "::::not a connection string::::", wantErr: errParseFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConnectionString(tc.input)
			if tc.wantErr != nil {
				require.Error(t, err)
				if !errors.Is(tc.wantErr, errParseFailure) {
					require.ErrorIs(t, err, tc.wantErr)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// errParseFailure is a sentinel used only by the test table to flag rows where
// any non-nil error from pgx is acceptable (the wrapped error message is not
// stable enough to assert on).
var errParseFailure = errors.New("parse failure")

func TestParseConnectionString_PGPASSWORDInEnv(t *testing.T) {
	// the engine may legitimately run with PGPASSWORD set (the documented way to
	// supply the source password); pgx.ParseConfig folds it into cfg.Password,
	// but a password-less string must NOT be rejected on that basis.
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "from-env-secret")

	t.Run("uri without embedded password is accepted", func(t *testing.T) {
		got, err := ParseConnectionString("postgres://alice@db.example.com:5433/shop?sslmode=require")
		require.NoError(t, err)
		require.Equal(t, Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "shop"}, got)
	})

	t.Run("dsn without embedded password is accepted", func(t *testing.T) {
		got, err := ParseConnectionString("host=db.example.com port=5433 user=alice dbname=shop sslmode=require")
		require.NoError(t, err)
		require.Equal(t, Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "shop"}, got)
	})

	t.Run("embedded password is still rejected with PGPASSWORD set", func(t *testing.T) {
		_, err := ParseConnectionString("host=db.example.com user=alice dbname=shop password=embedded")
		require.ErrorIs(t, err, ErrPasswordInConnString)
	})

	t.Run("password inside another quoted option value is not a false positive", func(t *testing.T) {
		got, err := ParseConnectionString("host=db.example.com user=alice dbname=shop options='-c password=x'")
		require.NoError(t, err)
		require.Equal(t, "alice", got.Username)
	})

	t.Run("password in uri query is rejected", func(t *testing.T) {
		_, err := ParseConnectionString("postgres://alice@db.example.com/shop?password=embedded")
		require.ErrorIs(t, err, ErrPasswordInConnString)
	})
}
