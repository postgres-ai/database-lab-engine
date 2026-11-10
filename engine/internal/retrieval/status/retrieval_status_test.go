package status

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionString_ParsesBackToValues(t *testing.T) {
	values := []string{"plain", "with space", "it's", `back\slash`, `double"quote`}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			cfg, err := pgconn.ParseConfig(connectionString(value, value, "/sockets/"+value))
			require.NoError(t, err)
			assert.Equal(t, "/sockets/"+value, cfg.Host)
			assert.Equal(t, value, cfg.User)
			assert.Equal(t, value, cfg.Database)
		})
	}
}
