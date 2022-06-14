package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWALGRecoveryConfig(t *testing.T) {
	walg := newWALG(nil, "dataDir", walgOptions{})

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

func TestWALGVersionParse(t *testing.T) {
	version, err := parseWalGVersion("wal-g version v2.0.0\t1eb88a5\t2022.05.20_10:45:57\tPostgreSQL")
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Equal(t, "v2.0.0", version)
}
