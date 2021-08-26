package pgconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	controlData := `
restore_command = 'wal-g wal-fetch %f %p'
standby_mode = 'on'
recovery_target_timeline = latest
`
	expected := map[string]string{
		"restore_command":          "wal-g wal-fetch %f %p",
		"standby_mode":             "on",
		"recovery_target_timeline": "latest",
	}

	f, err := os.CreateTemp("", "readPGConfig*")
	require.Nil(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	err = os.WriteFile(f.Name(), []byte(controlData), 0644)
	require.Nil(t, err)

	fileConfig, err := readConfig(f.Name())
	require.Nil(t, err)

	assert.Equal(t, len(expected), len(fileConfig))
	assert.Equal(t, expected["restore_command"], fileConfig["restore_command"])
	assert.Equal(t, expected["standby_mode"], fileConfig["standby_mode"])
	assert.Equal(t, expected["recovery_target_timeline"], fileConfig["recovery_target_timeline"])
}
