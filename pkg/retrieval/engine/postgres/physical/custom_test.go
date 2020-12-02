package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomRecoveryConfig(t *testing.T) {
	customTool := newCustomTool(customOptions{
		RestoreCommand: "pg_basebackup -X stream -D dataDirectory",
	})

	recoveryConfig := customTool.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command": "pg_basebackup -X stream -D dataDirectory",
		"standby_mode":    "on",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = customTool.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "pg_basebackup -X stream -D dataDirectory",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}
