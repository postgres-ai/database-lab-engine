/*
2025 © Postgres.ai
*/

package objbacker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// PoolManager provides integration between objbacker and ZFS pool management.
type PoolManager struct {
	manager *Manager
}

// NewPoolManager creates a new PoolManager.
func NewPoolManager(manager *Manager) *PoolManager {
	return &PoolManager{manager: manager}
}

// CreateTieredPool creates a ZFS pool with tiered storage.
// Hot data goes to localDevice, cold data goes to object storage.
func (pm *PoolManager) CreateTieredPool(ctx context.Context, poolName string, localDevice string) error {
	if !pm.manager.IsEnabled() {
		return errors.New("objbacker is not enabled")
	}

	vdev, err := pm.manager.CreateVDEV(ctx, poolName)
	if err != nil {
		return errors.Wrap(err, "failed to create objbacker VDEV")
	}

	opts := DefaultPoolCreateOptions()
	opts.LocalVDEV = localDevice
	opts.MountPoint = fmt.Sprintf("/var/lib/dblab/%s", poolName)

	args := pm.manager.BuildPoolCreateCommand(poolName, vdev.ID, opts)

	log.Msg("objbacker: creating tiered pool", poolName, "args:", args)

	cmd := exec.CommandContext(ctx, "zpool", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if destroyErr := pm.manager.DestroyVDEV(ctx, vdev.ID); destroyErr != nil {
			log.Err("objbacker: failed to cleanup VDEV after pool creation failure", destroyErr)
		}
		return errors.Wrapf(err, "failed to create pool: %s", string(output))
	}

	log.Msg("objbacker: tiered pool created successfully", poolName)

	return nil
}

// CreateObjectPool creates a ZFS pool backed entirely by object storage.
// Use with caution - performance will be limited by network latency.
func (pm *PoolManager) CreateObjectPool(ctx context.Context, poolName string, cacheDevice string) error {
	if !pm.manager.IsEnabled() {
		return errors.New("objbacker is not enabled")
	}

	vdev, err := pm.manager.CreateVDEV(ctx, poolName)
	if err != nil {
		return errors.Wrap(err, "failed to create objbacker VDEV")
	}

	opts := DefaultPoolCreateOptions()
	opts.CacheDevice = cacheDevice
	opts.MountPoint = fmt.Sprintf("/var/lib/dblab/%s", poolName)

	opts.Properties["autoexpand"] = "on"
	opts.FSProperties["recordsize"] = "1M"

	args := pm.manager.BuildPoolCreateCommand(poolName, vdev.ID, opts)

	log.Msg("objbacker: creating object-backed pool", poolName, "args:", args)

	cmd := exec.CommandContext(ctx, "zpool", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if destroyErr := pm.manager.DestroyVDEV(ctx, vdev.ID); destroyErr != nil {
			log.Err("objbacker: failed to cleanup VDEV after pool creation failure", destroyErr)
		}
		return errors.Wrapf(err, "failed to create pool: %s", string(output))
	}

	log.Msg("objbacker: object-backed pool created successfully", poolName)

	return nil
}

// MigrateSnapshotToObject migrates a ZFS snapshot to object storage.
// This is useful for archiving old snapshots to reduce local storage costs.
func (pm *PoolManager) MigrateSnapshotToObject(ctx context.Context, snapshotName string) error {
	if !pm.manager.IsEnabled() {
		return errors.New("objbacker is not enabled")
	}

	log.Msg("objbacker: migrating snapshot to object storage", snapshotName)

	sendCmd := exec.CommandContext(ctx, "zfs", "send", snapshotName)
	sendPipe, err := sendCmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to create send pipe")
	}

	objectKey := pm.manager.config.ObjectPath(fmt.Sprintf("snapshots/%s.zfs", sanitizeSnapshotName(snapshotName)))

	uploadArgs := pm.buildUploadCommand(objectKey)
	uploadCmd := exec.CommandContext(ctx, uploadArgs[0], uploadArgs[1:]...)
	uploadCmd.Stdin = sendPipe

	if err := sendCmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start zfs send")
	}

	if err := uploadCmd.Start(); err != nil {
		if killErr := sendCmd.Process.Kill(); killErr != nil {
			log.Err("objbacker: failed to kill send process", killErr)
		}
		return errors.Wrap(err, "failed to start upload")
	}

	if err := sendCmd.Wait(); err != nil {
		return errors.Wrap(err, "zfs send failed")
	}

	if err := uploadCmd.Wait(); err != nil {
		return errors.Wrap(err, "upload failed")
	}

	log.Msg("objbacker: snapshot migrated successfully", snapshotName, "object:", objectKey)

	return nil
}

// RestoreSnapshotFromObject restores a snapshot from object storage.
func (pm *PoolManager) RestoreSnapshotFromObject(ctx context.Context, snapshotName, targetPool string) error {
	if !pm.manager.IsEnabled() {
		return errors.New("objbacker is not enabled")
	}

	log.Msg("objbacker: restoring snapshot from object storage", snapshotName, "target:", targetPool)

	objectKey := pm.manager.config.ObjectPath(fmt.Sprintf("snapshots/%s.zfs", sanitizeSnapshotName(snapshotName)))

	downloadArgs := pm.buildDownloadCommand(objectKey)
	downloadCmd := exec.CommandContext(ctx, downloadArgs[0], downloadArgs[1:]...)
	downloadPipe, err := downloadCmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to create download pipe")
	}

	recvCmd := exec.CommandContext(ctx, "zfs", "receive", "-F", targetPool)
	recvCmd.Stdin = downloadPipe

	if err := downloadCmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start download")
	}

	if err := recvCmd.Start(); err != nil {
		if killErr := downloadCmd.Process.Kill(); killErr != nil {
			log.Err("objbacker: failed to kill download process", killErr)
		}
		return errors.Wrap(err, "failed to start zfs receive")
	}

	if err := downloadCmd.Wait(); err != nil {
		return errors.Wrap(err, "download failed")
	}

	if err := recvCmd.Wait(); err != nil {
		return errors.Wrap(err, "zfs receive failed")
	}

	log.Msg("objbacker: snapshot restored successfully", snapshotName)

	return nil
}

// ListArchivedSnapshots returns a list of snapshots stored in object storage.
func (pm *PoolManager) ListArchivedSnapshots(ctx context.Context) ([]ArchivedSnapshot, error) {
	if !pm.manager.IsEnabled() {
		return nil, errors.New("objbacker is not enabled")
	}

	prefix := pm.manager.config.ObjectPath("snapshots/")
	listArgs := pm.buildListCommand(prefix)

	cmd := exec.CommandContext(ctx, listArgs[0], listArgs[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list objects")
	}

	snapshots := parseSnapshotList(string(output))
	return snapshots, nil
}

// ArchivedSnapshot represents a snapshot stored in object storage.
type ArchivedSnapshot struct {
	Name         string    `json:"name"`
	ObjectKey    string    `json:"objectKey"`
	Size         uint64    `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

func (pm *PoolManager) buildUploadCommand(objectKey string) []string {
	cfg := pm.manager.config

	switch cfg.StorageType {
	case StorageTypeS3:
		args := []string{"aws", "s3", "cp", "-", fmt.Sprintf("s3://%s/%s", cfg.Bucket, objectKey)}
		if cfg.Endpoint != "" {
			args = append([]string{"aws", "--endpoint-url", cfg.Endpoint}, args[1:]...)
		}
		return args

	case StorageTypeGCS:
		return []string{"gsutil", "cp", "-", fmt.Sprintf("gs://%s/%s", cfg.Bucket, objectKey)}

	case StorageTypeAzure:
		return []string{"az", "storage", "blob", "upload",
			"--container-name", cfg.Bucket,
			"--name", objectKey,
			"--file", "-",
		}

	default:
		return []string{"cat"}
	}
}

func (pm *PoolManager) buildDownloadCommand(objectKey string) []string {
	cfg := pm.manager.config

	switch cfg.StorageType {
	case StorageTypeS3:
		args := []string{"aws", "s3", "cp", fmt.Sprintf("s3://%s/%s", cfg.Bucket, objectKey), "-"}
		if cfg.Endpoint != "" {
			args = append([]string{"aws", "--endpoint-url", cfg.Endpoint}, args[1:]...)
		}
		return args

	case StorageTypeGCS:
		return []string{"gsutil", "cp", fmt.Sprintf("gs://%s/%s", cfg.Bucket, objectKey), "-"}

	case StorageTypeAzure:
		return []string{"az", "storage", "blob", "download",
			"--container-name", cfg.Bucket,
			"--name", objectKey,
			"--file", "-",
		}

	default:
		return []string{"cat", "/dev/null"}
	}
}

func (pm *PoolManager) buildListCommand(prefix string) []string {
	cfg := pm.manager.config

	switch cfg.StorageType {
	case StorageTypeS3:
		args := []string{"aws", "s3", "ls", fmt.Sprintf("s3://%s/%s", cfg.Bucket, prefix)}
		if cfg.Endpoint != "" {
			args = append([]string{"aws", "--endpoint-url", cfg.Endpoint}, args[1:]...)
		}
		return args

	case StorageTypeGCS:
		return []string{"gsutil", "ls", "-l", fmt.Sprintf("gs://%s/%s", cfg.Bucket, prefix)}

	case StorageTypeAzure:
		return []string{"az", "storage", "blob", "list",
			"--container-name", cfg.Bucket,
			"--prefix", prefix,
			"--output", "tsv",
		}

	default:
		return []string{"echo"}
	}
}

func sanitizeSnapshotName(name string) string {
	name = strings.ReplaceAll(name, "@", "_at_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func parseSnapshotList(output string) []ArchivedSnapshot {
	var snapshots []ArchivedSnapshot
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.HasSuffix(line, ".zfs") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				name := parts[len(parts)-1]
				name = strings.TrimSuffix(name, ".zfs")
				name = strings.ReplaceAll(name, "_at_", "@")
				name = strings.ReplaceAll(name, "_", "/")

				snapshots = append(snapshots, ArchivedSnapshot{
					Name:      name,
					ObjectKey: parts[len(parts)-1],
				})
			}
		}
	}

	return snapshots
}
