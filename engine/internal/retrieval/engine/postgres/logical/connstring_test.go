/*
2026 © Postgres.ai
*/

package logical

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDatabase(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		dbName  string
		want    string
	}{
		{name: "uri overrides dbname", connStr: "postgres://alice@db.example.com:5433/shop", dbName: "other", want: "postgres://alice@db.example.com:5433/other"},
		{name: "uri preserves query params", connStr: "postgres://alice@db.example.com:5433/shop?sslmode=require&connect_timeout=5", dbName: "other", want: "postgres://alice@db.example.com:5433/other?sslmode=require&connect_timeout=5"},
		{name: "uri ipv6 host", connStr: "postgres://alice@[::1]:5433/shop", dbName: "other", want: "postgres://alice@[::1]:5433/other"},
		{name: "uri missing dbname gets one", connStr: "postgres://alice@db.example.com:5433", dbName: "other", want: "postgres://alice@db.example.com:5433/other"},
		{name: "uri postgresql scheme alias", connStr: "postgresql://alice@db.example.com/shop", dbName: "other", want: "postgresql://alice@db.example.com/other"},
		{name: "kv appends dbname last-wins", connStr: "host=db.example.com port=5433 user=alice dbname=shop", dbName: "other", want: "host=db.example.com port=5433 user=alice dbname=shop dbname='other'"},
		{name: "kv preserves extra params", connStr: "host=db sslmode=require dbname=shop", dbName: "other", want: "host=db sslmode=require dbname=shop dbname='other'"},
		{name: "kv missing dbname gets one", connStr: "host=db.example.com user=alice", dbName: "other", want: "host=db.example.com user=alice dbname='other'"},
		{name: "kv escapes single quote", connStr: "host=db user=alice", dbName: "a'b", want: "host=db user=alice dbname='a''b'"},
		{name: "kv trailing space trimmed", connStr: "host=db ", dbName: "other", want: "host=db dbname='other'"},
		{name: "empty dbName leaves uri untouched", connStr: "postgres://alice@db:5432/orig?sslmode=require", dbName: "", want: "postgres://alice@db:5432/orig?sslmode=require"},
		{name: "empty dbName leaves kv untouched", connStr: "host=db dbname=orig sslmode=require", dbName: "", want: "host=db dbname=orig sslmode=require"},
		{name: "empty dbName leaves pathless uri untouched", connStr: "postgres://alice@db:5432/?sslmode=require", dbName: "", want: "postgres://alice@db:5432/?sslmode=require"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := withDatabase(tc.connStr, tc.dbName)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWithDatabase_ResultParsesToTargetDB(t *testing.T) {
	// isolate from host env so pgx default lookups are deterministic
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "")

	inputs := []string{
		"postgres://alice@db.example.com:5433/shop?sslmode=require",
		"host=db.example.com port=5433 user=alice dbname=shop sslmode=require",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			out, err := withDatabase(in, "target")
			require.NoError(t, err)

			cfg, err := pgx.ParseConfig(out)
			require.NoError(t, err)
			assert.Equal(t, "target", cfg.Database, "withDatabase output must resolve to the requested database")
		})
	}
}

func TestApplySourceConnectionString(t *testing.T) {
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "")

	t.Run("connection string wins over discrete fields, password preserved", func(t *testing.T) {
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{
			ConnectionString: "postgres://alice@db.example.com:5433/shop",
			Connection:       Connection{Host: "ignored", Port: 1111, Username: "ignored", DBName: "ignored", Password: "secret"},
		}}}

		require.NoError(t, d.applySourceConnectionString())

		assert.Equal(t, "db.example.com", d.DumpOptions.Source.Connection.Host)
		assert.Equal(t, 5433, d.DumpOptions.Source.Connection.Port)
		assert.Equal(t, "alice", d.DumpOptions.Source.Connection.Username)
		assert.Equal(t, "shop", d.DumpOptions.Source.Connection.DBName)
		assert.Equal(t, "secret", d.DumpOptions.Source.Connection.Password)
	})

	t.Run("empty connection string leaves discrete fields untouched", func(t *testing.T) {
		conn := Connection{Host: "keep", Port: 2222, Username: "keep", DBName: "keep"}
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{Connection: conn}}}

		require.NoError(t, d.applySourceConnectionString())
		assert.Equal(t, conn, d.DumpOptions.Source.Connection)
	})

	t.Run("embedded password is rejected", func(t *testing.T) {
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{
			ConnectionString: "postgres://alice:secret@db.example.com/shop",
		}}}

		require.Error(t, d.applySourceConnectionString())
	})
}

func TestReload_ConnectionStringPrecedence(t *testing.T) {
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "")

	d := &DumpJob{}
	cfg := map[string]interface{}{
		"source": map[string]interface{}{
			"connectionString": "postgres://alice@db.example.com:5433/shop",
			"connection": map[string]interface{}{
				"host":     "ignored",
				"port":     1111,
				"dbname":   "ignored",
				"username": "ignored",
			},
		},
	}

	require.NoError(t, d.Reload(cfg))

	assert.Equal(t, "postgres://alice@db.example.com:5433/shop", d.DumpOptions.Source.ConnectionString)
	assert.Equal(t, "db.example.com", d.config.db.Host)
	assert.Equal(t, 5433, d.config.db.Port)
	assert.Equal(t, "alice", d.config.db.Username)
	assert.Equal(t, "shop", d.config.db.DBName)
}

func TestDumpConnectionArgs(t *testing.T) {
	t.Run("discrete fields, password not emitted", func(t *testing.T) {
		d := &DumpJob{config: dumpJobConfig{db: Connection{Host: "h", Port: 5432, Username: "u", Password: "secret"}}}

		args, err := d.dumpConnectionArgs("shop", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"--host", "h", "--port", "5432", "--username", "u", "--dbname", "shop"}, args)
		assert.NotContains(t, strings.Join(args, " "), "secret")
	})

	t.Run("connection string overrides db per call, preserves params", func(t *testing.T) {
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{ConnectionString: "postgres://u@h:5432/orig?sslmode=require"}}}

		shopArgs, err := d.dumpConnectionArgs("shop", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"-d", "postgres://u@h:5432/shop?sslmode=require"}, shopArgs)

		blogArgs, err := d.dumpConnectionArgs("blog", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"-d", "postgres://u@h:5432/blog?sslmode=require"}, blogArgs)
	})

	t.Run("keyword/value connection string appends dbname", func(t *testing.T) {
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{ConnectionString: "host=h user=u sslmode=require"}}}

		args, err := d.dumpConnectionArgs("shop", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"-d", "host=h user=u sslmode=require dbname='shop'"}, args)
	})

	t.Run("for shell, keyword/value connection string is single-quoted", func(t *testing.T) {
		d := &DumpJob{DumpOptions: DumpOptions{Source: Source{ConnectionString: "host=h dbname='shop' sslmode=require"}}}

		args, err := d.dumpConnectionArgs("blog", true)
		require.NoError(t, err)
		require.Len(t, args, 2)
		assert.Equal(t, "-d", args[0])
		assert.Equal(t, `'host=h dbname='\''shop'\'' sslmode=require dbname='\''blog'\'''`, args[1])
	})

}

func TestShellQuote(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain value", in: "host=h", want: "'host=h'"},
		{name: "spaces preserved inside quotes", in: "host=h dbname=shop", want: "'host=h dbname=shop'"},
		{name: "metacharacters are literal", in: "x=$(touch pwned)`id`&|", want: "'x=$(touch pwned)`id`&|'"},
		{name: "embedded single quote escaped", in: "dbname='shop'", want: `'dbname='\''shop'\'''`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shellQuote(tc.in))
		})
	}
}

func TestSourcePgxConfig(t *testing.T) {
	t.Setenv("PGHOST", "")
	t.Setenv("PGPORT", "")
	t.Setenv("PGUSER", "")
	t.Setenv("PGDATABASE", "")
	t.Setenv("PGPASSWORD", "")
	t.Setenv("PGSSLMODE", "")
	t.Setenv("PGCONNECT_TIMEOUT", "")

	t.Run("discrete fields build the target connection", func(t *testing.T) {
		conn := Connection{Host: "db.example.com", Port: 5433, Username: "alice", DBName: "ignored", Password: "secret"}

		cfg, err := sourcePgxConfig("", conn, "shop")
		require.NoError(t, err)
		assert.Equal(t, "db.example.com", cfg.Host)
		assert.Equal(t, uint16(5433), cfg.Port)
		assert.Equal(t, "alice", cfg.User)
		assert.Equal(t, "shop", cfg.Database)
		assert.Equal(t, "secret", cfg.Password)
		assert.Zero(t, cfg.ConnectTimeout, "discrete path carries no connect_timeout")
	})

	t.Run("raw string preserves params, injects password, overrides db", func(t *testing.T) {
		conn := Connection{Password: "secret"}

		cfg, err := sourcePgxConfig("postgres://alice@db.example.com:5433/orig?sslmode=require&connect_timeout=7", conn, "shop")
		require.NoError(t, err)
		assert.Equal(t, "db.example.com", cfg.Host)
		assert.Equal(t, uint16(5433), cfg.Port)
		assert.Equal(t, "alice", cfg.User)
		assert.Equal(t, "shop", cfg.Database, "withDatabase must override the target db")
		assert.Equal(t, "secret", cfg.Password, "password is injected separately, not parsed from the string")
		assert.Equal(t, 7*time.Second, cfg.ConnectTimeout, "connect_timeout must survive")
		assert.NotNil(t, cfg.TLSConfig, "sslmode=require must produce a TLS config")
	})

	t.Run("invalid connection string errors", func(t *testing.T) {
		_, err := sourcePgxConfig("host=h port=notaport", Connection{}, "shop")
		require.Error(t, err)
	})
}
