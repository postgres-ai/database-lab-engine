package backup

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackupTimestamps(t *testing.T) {
	now = mockNow()
	r := require.New(t)
	tmp := os.TempDir()

	configFile := path.Join(tmp, "backed.yaml")
	cleanup, err := filepath.Glob(configFile + "*")
	r.NoError(err)
	for _, filename := range cleanup {
		err := os.Remove(filename)
		r.NoError(err)
	}

	err = os.WriteFile(configFile, []byte(""), 0644)
	r.NoError(err)

	backups, err := NewBackupCollection(configFile)
	r.NoError(err)
	r.Len(backups.backups, 0)

	err = backups.Backup()
	backup1 := backups.backups[0]
	r.NoError(err)
	r.NotNil(backup1)

	err = backups.Backup()
	backup2 := backups.backups[1]
	r.NoError(err)
	r.NotNil(backup2)

	err = backups.Backup()
	backup3 := backups.backups[2]
	r.NoError(err)
	r.NotNil(backup3)

	err = backups.EnsureMaxBackups(2)
	r.NoError(err)

	backups, err = NewBackupCollection(configFile)
	r.NoError(err)
	r.Len(backups.backups, 2)

	r.EqualValues(backup2.Filename, backups.backups[0].Filename)
	r.EqualValues(backup3.Filename, backups.backups[1].Filename)
}

func mockNow() func() time.Time {
	seed := time.Now()
	return func() time.Time {
		seed = seed.Add(time.Second)
		return seed
	}
}
