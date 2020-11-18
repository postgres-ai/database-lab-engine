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
	expectedResponse11 := `
restore_command = 'pg_basebackup -X stream -D dataDirectory'
standby_mode = 'on'
`
	assert.Equal(t, []byte(expectedResponse11), recoveryConfig)

	recoveryConfig = customTool.GetRecoveryConfig(12.3)
	expectedResponse12 := `
restore_command = 'pg_basebackup -X stream -D dataDirectory'
`
	assert.Equal(t, []byte(expectedResponse12), recoveryConfig)
}
