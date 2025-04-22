/*
2020 © Postgres.ai
*/

package physical

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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
)

// walg defines a WAL-G as an archival restoration tool.
type walg struct {
	dockerClient     *client.Client
	pgDataDir        string
	parsedBackupName string
	options          walgOptions
}

type walgOptions struct {
	BackupName string `yaml:"backupName"`
}

func newWALG(dockerClient *client.Client, pgDataDir string, options walgOptions) *walg {
	walg := &walg{
		dockerClient:     dockerClient,
		pgDataDir:        pgDataDir,
		options:          options,
		parsedBackupName: options.BackupName,
	}

	return walg
}

// GetRestoreCommand returns a command to restore data.
func (w *walg) GetRestoreCommand() string {
	return fmt.Sprintf("wal-g backup-fetch %s %s", w.pgDataDir, w.parsedBackupName)
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
	name, err := getLastBackupName(ctx, w.dockerClient, containerID)

	if err != nil {
		return err
	}

	if name != "" {
		w.parsedBackupName = name
		return nil
	}

	return fmt.Errorf("failed to fetch last backup name from wal-g")
}

// getLastBackupName returns the name of the latest backup from the wal-g backup list.
func getLastBackupName(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	walgVersion, err := getWalgVersion(ctx, dockerClient, containerID)

	if err != nil {
		return "", err
	}

	result := semver.Compare(walgVersion, walg11Version)

	// Try to fetch the latest backup with the details command for WAL-G 1.1 and higher
	if result >= 0 {
		output, err := parseLastBackupFromDetails(ctx, dockerClient, containerID)

		if err == nil {
			return output, err
		}

		// fallback to fetching last backup from list
		log.Err("failed to parse last backup from wal-g details", err)
	}

	return parseLastBackupFromList(ctx, dockerClient, containerID)
}

// parseLastBackupFromList parses the name of the latest backup from "wal-g backup-list" output.
func parseLastBackupFromList(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	output, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Cmd: []string{"bash", "-c", "wal-g backup-list | grep base | sort -nk1 | tail -1 | awk '{print $1}'"},
	})
	if err != nil {
		return "", err
	}

	log.Dbg("The latest WAL-G backup from the list", output)

	return output, nil
}

// parseLastBackupFromDetails parses the name of the latest backup from "wal-g backup-list --detail" output.
func parseLastBackupFromDetails(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	output, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Cmd: []string{"bash", "-c", "wal-g backup-list --detail | tail -1 | awk '{print $1}'"},
	})
	if err != nil {
		return "", err
	}

	log.Dbg("The latest WAL-G backup from list details", output)

	return output, nil
}

// getWalgVersion fetches the WAL-G version installed in the provided container.
func getWalgVersion(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	output, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Cmd: []string{"bash", "-c", "wal-g --version"},
	})
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
