/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitParamsExtraction(t *testing.T) {
	controlDataOutput := bytes.NewBufferString(`
wal_level setting:                    logical
wal_log_hints setting:                on
max_connections setting:              500
max_worker_processes setting:         8
max_prepared_xacts setting:           0
max_locks_per_xact setting:           128
track_commit_timestamp setting:       off
`)

	expectedSettings := map[string]string{
		"max_connections":           "500",
		"max_locks_per_transaction": "128",
		"max_prepared_transactions": "0",
		"max_worker_processes":      "8",
		"track_commit_timestamp":    "off",
	}

	settings, err := extractControlDataParams(context.Background(), controlDataOutput)

	require.Nil(t, err)
	assert.EqualValues(t, settings, expectedSettings)
}
