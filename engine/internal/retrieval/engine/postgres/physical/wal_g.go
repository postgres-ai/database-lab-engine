/*
2020 © Postgres.ai
*/

package physical

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/moby/moby/client"
	"golang.org/x/mod/semver"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	walgTool       = "walg"
	walgSplitCount = 3
	walg11Version  = "v1.1"
	latestBackup   = "LATEST"

	backupNamePrefix = "base_"
)

// containerExec runs a shell command in a container and returns its output.
type containerExec func(ctx context.Context, containerID, cmd string) (string, error)

// walg defines a WAL-G as an archival restoration tool.
type walg struct {
	exec             containerExec
	pgDataDir        string
	parsedBackupName string
	options          walgOptions
}

type walgOptions struct {
	BackupName    string   `yaml:"backupName"`
	CustomOptions []string `yaml:"customOptions"`
}

func newWALG(dockerClient *client.Client, pgDataDir string, options walgOptions) *walg {
	walg := &walg{
		exec:             dockerExec(dockerClient),
		pgDataDir:        pgDataDir,
		options:          options,
		parsedBackupName: options.BackupName,
	}

	return walg
}

// dockerExec returns a containerExec that runs the command with bash in a Docker container.
func dockerExec(dockerClient *client.Client) containerExec {
	return func(ctx context.Context, containerID, cmd string) (string, error) {
		return tools.ExecCommandWithOutput(ctx, dockerClient, containerID, client.ExecCreateOptions{
			Cmd: []string{"bash", "-c", cmd},
		})
	}
}

// GetRestoreCommand returns a command to restore data.
func (w *walg) GetRestoreCommand() string {
	restoreCmd := fmt.Sprintf("wal-g backup-fetch %s %s", w.pgDataDir, w.parsedBackupName)

	if len(w.options.CustomOptions) > 0 {
		restoreCmd += " " + strings.Join(w.options.CustomOptions, " ")
	}

	return restoreCmd
}

// GetRecoveryConfig returns a recovery config to restore data.
func (w *walg) GetRecoveryConfig(pgVersion float64) map[string]string {
	recoveryCfg := map[string]string{
		"restore_command": "wal-g wal-fetch %f %p",
	}

	if pgVersion < defaults.PGVersion12 {
		recoveryCfg["recovery_target_timeline"] = "latest"
	}

	return recoveryCfg
}

// Init initializes the wal-g tool to run in the provided container.
func (w *walg) Init(ctx context.Context, containerID string) error {
	if strings.ToUpper(w.options.BackupName) != latestBackup {
		return nil
	}
	// workaround for issue with identification of last backup
	// https://gitlab.com/postgres-ai/database-lab/-/issues/365
	name, err := w.getLastBackupName(ctx, containerID)

	if err != nil {
		return err
	}

	if name != "" {
		w.parsedBackupName = name
		return nil
	}

	return fmt.Errorf("failed to fetch last backup name from wal-g: no backups found in the configured storage")
}

// getLastBackupName returns the name of the latest backup from the wal-g backup list.
func (w *walg) getLastBackupName(ctx context.Context, containerID string) (string, error) {
	walgVersion, err := w.getWalgVersion(ctx, containerID)

	if err != nil {
		return "", err
	}

	result := semver.Compare(walgVersion, walg11Version)

	// Try to fetch the latest backup with the details command for WAL-G 1.1 and higher
	if result >= 0 {
		output, err := w.runBackupList(ctx, containerID, "--detail")

		if err == nil {
			name := lastListedBackup(output)
			log.Dbg("The latest WAL-G backup from list details", name)

			return name, nil
		}

		// fallback to fetching last backup from list
		log.Err("failed to list wal-g backups with details", err)
	}

	output, err := w.runBackupList(ctx, containerID)
	if err != nil {
		return "", err
	}

	name := highestBackupName(output)
	log.Dbg("The latest WAL-G backup from the list", name)

	return name, nil
}

// runBackupList runs "wal-g backup-list" without a pipeline so that its exit code is not masked,
// and merges stderr into the output so that a failure carries the wal-g error message.
func (w *walg) runBackupList(ctx context.Context, containerID string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{"wal-g", "backup-list"}, args...), " ") + " 2>&1"

	output, err := w.exec(ctx, containerID, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to list wal-g backups: %w: %s", err, output)
	}

	return output, nil
}

// lastListedBackup returns the last backup name in the "wal-g backup-list --detail" output, which lists backups by creation time.
func lastListedBackup(output string) string {
	names := backupNames(output)
	if len(names) == 0 {
		return ""
	}

	return names[len(names)-1]
}

// highestBackupName returns the lexicographically highest backup name in the "wal-g backup-list" output.
func highestBackupName(output string) string {
	names := backupNames(output)
	if len(names) == 0 {
		return ""
	}

	return slices.Max(names)
}

// backupNames extracts backup names from the "wal-g backup-list" output, skipping the table header and wal-g log lines.
func backupNames(output string) []string {
	var names []string

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)

		if len(fields) == 0 || !strings.HasPrefix(fields[0], backupNamePrefix) {
			continue
		}

		names = append(names, fields[0])
	}

	return names
}

// getWalgVersion fetches the WAL-G version installed in the provided container.
func (w *walg) getWalgVersion(ctx context.Context, containerID string) (string, error) {
	output, err := w.exec(ctx, containerID, "wal-g --version")
	if err != nil {
		return "", err
	}

	log.Dbg(output)

	return parseWalGVersion(output)
}

// parseWalGVersion extracts the version from the 'wal-g --version' output.
// For example, "wal-g version v2.0.0	1eb88a5	2022.05.20_10:45:57	PostgreSQL".
func parseWalGVersion(output string) (string, error) {
	walgVersion := strings.Split(output, "\t")
	versionParts := strings.Split(walgVersion[0], " ")

	if len(versionParts) < walgSplitCount {
		return "", fmt.Errorf("failed to extract wal-g version number")
	}

	return versionParts[walgSplitCount-1], nil
}
