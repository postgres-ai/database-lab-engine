package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWALGRecoveryConfig(t *testing.T) {
	walg := newWALG("dataDir", walgOptions{})

	recoveryConfig := walg.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command":          "wal-g wal-fetch %f %p",
		"recovery_target_timeline": "latest",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = walg.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "wal-g wal-fetch %f %p",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}
