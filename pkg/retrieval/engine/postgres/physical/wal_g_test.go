package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWALGRecoveryConfig(t *testing.T) {
	walg := newWALG("dataDir", walgOptions{})

	recoveryConfig := walg.GetRecoveryConfig(11.7)
	expectedResponse11 := `
restore_command = 'wal-g wal-fetch %f %p'
standby_mode = 'on'
`
	assert.Equal(t, []byte(expectedResponse11), recoveryConfig)

	recoveryConfig = walg.GetRecoveryConfig(12.3)
	expectedResponse12 := `
restore_command = 'wal-g wal-fetch %f %p'
`
	assert.Equal(t, []byte(expectedResponse12), recoveryConfig)
}
