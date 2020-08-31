/*
2019 © Postgres.ai
*/

package thinclones

import (
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones/zfs"
)

const (
	defaultSessionCloneSize = 10
)

type managerZFS struct {
	runner runners.Runner
	config ManagerConfig
}

// newManagerZFS creates a new Manager instance for ZFS.
func newManagerZFS(runner runners.Runner, config ManagerConfig) *managerZFS {
	m := managerZFS{}

	m.runner = runner
	m.config = config

	return &m
}

// CreateSnapshot creates a new snapshot.
func (m *managerZFS) CreateSnapshot(poolSuffix, dataStateAt string) (string, error) {
	pool := m.config.Pool

	if poolSuffix != "" {
		pool += "/" + poolSuffix
	}

	return zfs.CreateSnapshot(m.runner, pool, dataStateAt, m.config.PreSnapshotSuffix)
}

// DestroySnapshot destroys the snapshot.
func (m *managerZFS) DestroySnapshot(snapshotName string) error {
	return zfs.DestroySnapshot(m.runner, snapshotName)
}

// CleanupSnapshots destroys old snapshots considering retention limit.
func (m *managerZFS) CleanupSnapshots(retentionLimit int) ([]string, error) {
	return zfs.CleanupSnapshots(m.runner, m.config.Pool, retentionLimit)
}

func (m *managerZFS) CreateClone(name, snapshotID string) error {
	return zfs.CreateClone(m.runner, m.config.Pool, name, snapshotID, m.config.ClonesMountDir, m.config.OSUsername)
}

func (m *managerZFS) DestroyClone(name string) error {
	return zfs.DestroyClone(m.runner, m.config.Pool, name)
}

func (m *managerZFS) ListClonesNames() ([]string, error) {
	return zfs.ListClones(m.runner, m.config.Pool, m.config.ClonePrefix)
}

func (m *managerZFS) GetSessionState(name string) (*resources.SessionState, error) {
	state := &resources.SessionState{
		CloneDiffSize: defaultSessionCloneSize,
	}

	entries, err := zfs.ListFilesystems(m.runner, m.config.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	var sEntry *zfs.ListEntry

	entryName := m.config.Pool + "/" + name

	for _, entry := range entries {
		if entry.Name == entryName {
			sEntry = entry
			break
		}
	}

	if sEntry == nil {
		return nil, errors.New("cannot get session state: specified ZFS pool does not exist")
	}

	state.CloneDiffSize = sEntry.Used

	return state, nil
}

func (m *managerZFS) GetDiskState() (*resources.Disk, error) {
	parts := strings.SplitN(m.config.Pool, "/", 2)
	parentPool := parts[0]

	entries, err := zfs.ListFilesystems(m.runner, parentPool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	var parentPoolEntry, poolEntry *zfs.ListEntry

	for _, entry := range entries {
		if entry.Name == parentPool {
			parentPoolEntry = entry
		}

		if entry.Name == m.config.Pool {
			poolEntry = entry
		}

		if parentPoolEntry != nil && poolEntry != nil {
			break
		}
	}

	if parentPoolEntry == nil || poolEntry == nil {
		return nil, errors.New("cannot get disk state: pool entries not found")
	}

	disk := &resources.Disk{
		Size:     parentPoolEntry.Available + parentPoolEntry.Used,
		Free:     parentPoolEntry.Available,
		Used:     parentPoolEntry.Used,
		DataSize: poolEntry.LogicalReferenced,
	}

	return disk, nil
}

func (m *managerZFS) GetSnapshots() ([]resources.Snapshot, error) {
	entries, err := zfs.ListSnapshots(m.runner, m.config.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list snapshots")
	}

	snapshots := make([]resources.Snapshot, 0, len(entries))

	for _, entry := range entries {
		// Filter pre-snapshots, they will not be allowed to be used for cloning.
		if strings.HasSuffix(entry.Name, m.config.PreSnapshotSuffix) {
			continue
		}

		snapshot := resources.Snapshot{
			ID:          entry.Name,
			CreatedAt:   entry.Creation,
			DataStateAt: entry.DataStateAt,
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}
