/*
2020 © Postgres.ai
*/

package physical

import (
	"fmt"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
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
func (w *walg) GetRecoveryConfig(pgVersion float64) map[string]string {
	recoveryCfg := map[string]string{
		"restore_command": "wal-g wal-fetch %f %p",
	}

	if pgVersion < defaults.PGVersion12 {
		recoveryCfg["standby_mode"] = "on"
		recoveryCfg["recovery_target_timeline"] = "latest"
	}

	return recoveryCfg
}
