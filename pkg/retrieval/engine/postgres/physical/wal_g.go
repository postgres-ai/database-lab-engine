/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"fmt"
)

const (
	walgTool = "walg"
)

// walg defines a WAL-G as an archival restoration tool.
type walg struct {
	pgDataDir string
	options   walgOptions
}

type walgOptions struct {
	BackupName string `yaml:"backupName"`
}

func newWALG(pgDataDir string, options walgOptions) *walg {
	return &walg{
		pgDataDir: pgDataDir,
		options:   options,
	}
}

// GetRestoreCommand returns a command to restore data.
func (w *walg) GetRestoreCommand() string {
	return fmt.Sprintf("wal-g backup-fetch %s %s", w.pgDataDir, w.options.BackupName)
}

// GetRecoveryConfig returns a recovery config to restore data.
func (w *walg) GetRecoveryConfig() []byte {
	buffer := bytes.Buffer{}

	buffer.WriteString("standby_mode = 'on'\n")
	buffer.WriteString("restore_command = 'wal-g wal-fetch %f %p'\n")

	return buffer.Bytes()
}
