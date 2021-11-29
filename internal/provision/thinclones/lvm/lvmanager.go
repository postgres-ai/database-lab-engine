/*
2019 © Postgres.ai
*/

package lvm

import (
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	poolPartsLen = 2
)

// LVManager describes an LVM2 filesystem manager.
type LVManager struct {
	runner        runners.Runner
	pool          *resources.Pool
	volumeGroup   string
	logicalVolume string
}

// NewFSManager creates a new Manager instance for LVM.
func NewFSManager(runner runners.Runner, pool *resources.Pool) (*LVManager, error) {
	m := LVManager{
		runner: runner,
		pool:   pool,
	}

	if err := m.parsePool(); err != nil {
		return nil, err
	}

	return &m, nil
}

// Pool gets a storage pool.
func (m *LVManager) Pool() *resources.Pool {
	return m.pool
}

// CreateClone creates a new volume.
func (m *LVManager) CreateClone(name, _ string) error {
	return CreateVolume(m.runner, m.volumeGroup, m.logicalVolume, name, m.pool.ClonesDir())
}

// DestroyClone destroys volumes.
func (m *LVManager) DestroyClone(name string) error {
	return RemoveVolume(m.runner, m.volumeGroup, m.logicalVolume, name, m.pool.ClonesDir())
}

// ListClonesNames returns a list of clone names.
func (m *LVManager) ListClonesNames() ([]string, error) {
	volumes, err := ListVolumes(m.runner, m.volumeGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list LVM volumes")
	}

	volumesNames := make([]string, 0, len(volumes))

	for _, volume := range volumes {
		volumesNames = append(volumesNames, volume.Name)
	}

	return volumesNames, nil
}

func (m *LVManager) parsePool() error {
	parts := strings.SplitN(m.pool.Name, "-", poolPartsLen)
	if len(parts) < poolPartsLen {
		return errors.Errorf("failed to extract volume group and logical volume from %q", m.pool.Name)
	}

	m.volumeGroup = parts[0]
	m.logicalVolume = parts[1]

	return nil
}

// CreateSnapshot is not supported in LVM mode.
func (m *LVManager) CreateSnapshot(_, _ string) (string, error) {
	log.Msg("Creating a snapshot is not supported in LVM mode. Skip the operation.")

	return "", nil
}

// DestroySnapshot is not supported in LVM mode.
func (m *LVManager) DestroySnapshot(_ string) error {
	log.Msg("Destroying a snapshot is not supported in LVM mode. Skip the operation.")

	return nil
}

// CleanupSnapshots is not supported in LVM mode.
func (m *LVManager) CleanupSnapshots(_ int) ([]string, error) {
	log.Msg("Cleanup snapshots is not supported in LVM mode. Skip the operation.")

	return nil, nil
}

// GetSnapshots is not implemented.
func (m *LVManager) GetSnapshots() ([]resources.Snapshot, error) {
	// TODO(anatoly): Not supported in LVM mode warning.
	return []resources.Snapshot{
		{
			ID:          "TechnicalSnapshot",
			CreatedAt:   time.Now(),
			DataStateAt: time.Now(),
			Pool:        m.pool.Name,
		},
	}, nil
}

// GetSessionState is not implemented.
func (m *LVManager) GetSessionState(_ string) (*resources.SessionState, error) {
	// TODO(anatoly): Implement.
	return &resources.SessionState{}, nil
}

// GetDiskState is not implemented.
func (m *LVManager) GetFilesystemState() (models.FileSystem, error) {
	// TODO(anatoly): Implement.
	return models.FileSystem{Mode: PoolMode}, nil
}
