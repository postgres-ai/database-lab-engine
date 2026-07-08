/*
2026 © Postgres.ai
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRetentionParsing(t *testing.T) {
	type root struct {
		Retention Retention `yaml:"retention"`
	}

	t.Run("present section is fully parsed", func(t *testing.T) {
		var r root

		require.NoError(t, yaml.Unmarshal([]byte(`
retention:
  unusedSnapshotMinutes: 60
  unusedBranchMinutes: 120
  checkIntervalMinutes: 5
  protectionMaxDurationMinutes: 1440
  maxDeletionsPerTick: 50
`), &r))

		assert.Equal(t, Retention{
			UnusedSnapshotMinutes:        60,
			UnusedBranchMinutes:          120,
			CheckIntervalMinutes:         5,
			ProtectionMaxDurationMinutes: 1440,
			MaxDeletionsPerTick:          50,
		}, r.Retention)
	})

	t.Run("absent section is the zero value (disabled)", func(t *testing.T) {
		var r root

		require.NoError(t, yaml.Unmarshal([]byte("server:\n  port: 2345\n"), &r))
		assert.Equal(t, Retention{}, r.Retention)
	})

	t.Run("partial section keeps unset fields zero", func(t *testing.T) {
		var r root

		require.NoError(t, yaml.Unmarshal([]byte("retention:\n  unusedSnapshotMinutes: 30\n"), &r))
		assert.Equal(t, uint(30), r.Retention.UnusedSnapshotMinutes)
		assert.Equal(t, uint(0), r.Retention.UnusedBranchMinutes)
		assert.Equal(t, uint(0), r.Retention.MaxDeletionsPerTick)
	})
}
