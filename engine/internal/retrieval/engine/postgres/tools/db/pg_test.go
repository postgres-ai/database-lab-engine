/*
2021 © Postgres.ai
*/

package db

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildConnStrWithSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		username string
		password string
		expected string
	}{
		{name: "password with equals sign", host: "localhost", username: "postgres", password: "pass=word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass=word'"},
		{name: "password with spaces", host: "localhost", username: "postgres", password: "pass word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass word'"},
		{name: "password with single quote", host: "localhost", username: "postgres", password: "pass'word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass\\'word'"},
		{name: "password with backslash", host: "localhost", username: "postgres", password: `pass\word`, expected: `host='localhost' port=5432 user='postgres' database='testdb' password='pass\\word'`},
		{name: "combined quote and backslash in password", host: "localhost", username: "postgres", password: `test\'name`, expected: `host='localhost' port=5432 user='postgres' database='testdb' password='test\\\'name'`},
		{name: "host with single quote", host: "host'name", username: "postgres", password: "pass", expected: "host='host\\'name' port=5432 user='postgres' database='testdb' password='pass'"},
		{name: "host with backslash", host: `host\name`, username: "postgres", password: "pass", expected: `host='host\\name' port=5432 user='postgres' database='testdb' password='pass'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConnectionString(tt.host, "5432", tt.username, "testdb", tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeLibpqValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no special chars", input: "simple", expected: "simple"},
		{name: "single quote", input: "it's", expected: `it\'s`},
		{name: "backslash", input: `path\to`, expected: `path\\to`},
		{name: "combined quote and backslash", input: `test\'name`, expected: `test\\\'name`},
		{name: "empty string", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeLibpqValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionString_ParsesBackToValues(t *testing.T) {
	values := []string{"plain", "with space", "it's", `back\slash`, `mixed\'both`, `double"quote`}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			cfg, err := pgconn.ParseConfig(ConnectionString(value, "5432", value, value, value))
			require.NoError(t, err)
			assert.Equal(t, value, cfg.Host)
			assert.Equal(t, value, cfg.User)
			assert.Equal(t, value, cfg.Database)
			assert.Equal(t, value, cfg.Password)
		})
	}
}
