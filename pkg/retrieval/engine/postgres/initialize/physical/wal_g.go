/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"fmt"

	"github.com/docker/docker/api/types/mount"
)

const (
	walgTool                = "walg"
	credentialsInternalPath = "/etc/sa/credentials.json"
	gcsStorage              = "gcs"
)

// walg defines a WAL-G as an archival restoration tool.
type walg struct {
	pgDataDir string
	options   walgOptions
}

type walgOptions struct {
	Storage         string `yaml:"storage"`
	BackupName      string `yaml:"backupName"`
	CredentialsFile string `yaml:"credentialsFile"`
}

func newWalg(pgDataDir string, options walgOptions) *walg {
	return &walg{
		pgDataDir: pgDataDir,
		options:   options,
	}
}

// GetEnvVariables returns restorer environment variables.
func (w *walg) GetEnvVariables() []string {
	envVars := []string{}

	if w.options.Storage == gcsStorage {
		envVars = append(envVars, "GOOGLE_APPLICATION_CREDENTIALS="+credentialsInternalPath)
	}

	return envVars
}

// GetMounts returns restorer volume configurations for mounting.
func (w *walg) GetMounts() []mount.Mount {
	mounts := []mount.Mount{}

	if w.options.CredentialsFile != "" {
		mounts = append(mounts,
			mount.Mount{
				Type:   mount.TypeBind,
				Source: w.options.CredentialsFile,
				Target: credentialsInternalPath,
			},
		)
	}

	return mounts
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
