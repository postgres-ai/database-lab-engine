package diagnostic

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCleanup(t *testing.T) {
	t.Parallel()
	now := time.Now()

	dir := t.TempDir()

	for day := 0; day <= 10; day++ {
		containerName, err := uuid.NewUUID()
		assert.NoError(t, err)

		name := now.AddDate(0, 0, -1*day).In(time.UTC).Format(timeFormat)

		err = os.MkdirAll(path.Join(dir, name, containerName.String()), 0755)
		require.NoError(t, err)
	}

	err := cleanupLogsDir(dir, 5)
	require.NoError(t, err)

	// list remaining directories
	dirList, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(dirList))
}
