/*
2019 © Postgres.ai
*/

package volumemanagers

import (
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/volumemanagers/zfs"
)

const (
	defaultSessionCloneSize = 10
)

type volumeManagerZFS struct {
	runner runners.Runner
	config VolumeManagerConfig
}

// newVolumeManagerZFS creates a new VolumeManager instance for ZFS.
func newVolumeManagerZFS(runner runners.Runner, config VolumeManagerConfig) *volumeManagerZFS {
	m := volumeManagerZFS{}

	m.runner = runner
	m.config = config

	return &m
}

func (m *volumeManagerZFS) CreateClone(name, snapshotID string) error {
	return zfs.CreateClone(m.runner, m.config.Pool, name, snapshotID, m.config.MountDir, m.config.OSUsername)
}

func (m *volumeManagerZFS) DestroyClone(name string) error {
	return zfs.DestroyClone(m.runner, m.config.Pool, name)
}

func (m *volumeManagerZFS) ListClonesNames() ([]string, error) {
	return zfs.ListClones(m.runner, m.config.ClonePrefix)
}

func (m *volumeManagerZFS) GetSessionState(name string) (*resources.SessionState, error) {
	state := &resources.SessionState{
		CloneSize: defaultSessionCloneSize,
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

	state.CloneSize = sEntry.Used

	return state, nil
}

func (m *volumeManagerZFS) GetDiskState() (*resources.Disk, error) {
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
		DataSize: poolEntry.LogicalReferenced,
	}

	return disk, nil
}

func (m *volumeManagerZFS) GetSnapshots() ([]resources.Snapshot, error) {
	entries, err := zfs.ListSnapshots(m.runner, m.config.Pool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list snapshots")
	}

	snapshots := make([]resources.Snapshot, 0, len(entries))

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name, m.config.SnapshotFilterSuffix) {
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
