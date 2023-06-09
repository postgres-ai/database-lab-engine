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
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
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

// UpdateConfig updates the manager's pool.
func (m *LVManager) UpdateConfig(pool *resources.Pool) {
	m.pool = pool
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

// SnapshotList is not implemented.
func (m *LVManager) SnapshotList() []resources.Snapshot {
	// TODO(anatoly): Not supported in LVM mode warning.
	return []resources.Snapshot{
		{
			ID:          "TechnicalSnapshot",
			CreatedAt:   time.Now(),
			DataStateAt: time.Now(),
			Pool:        m.pool.Name,
		},
	}
}

// RefreshSnapshotList is not supported in LVM mode.
func (m *LVManager) RefreshSnapshotList() {
	log.Msg("RefreshSnapshotList is not supported for LVM. Skip the operation")
}

// GetSessionState is not implemented.
func (m *LVManager) GetSessionState(_ string) (*resources.SessionState, error) {
	// TODO(anatoly): Implement.
	return &resources.SessionState{}, nil
}

// GetFilesystemState is not implemented.
func (m *LVManager) GetFilesystemState() (models.FileSystem, error) {
	// TODO(anatoly): Implement.
	return models.FileSystem{Mode: PoolMode}, nil
}

// InitBranching inits data branching.
func (m *LVManager) InitBranching() error {
	log.Msg("InitBranching is not supported for LVM. Skip the operation")

	return nil
}

// VerifyBranchMetadata checks snapshot metadata.
func (m *LVManager) VerifyBranchMetadata() error {
	log.Msg("VerifyBranchMetadata is not supported for LVM. Skip the operation")

	return nil
}

// CreateBranch clones data as a new branch.
func (m *LVManager) CreateBranch(_, _ string) error {
	log.Msg("CreateBranch is not supported for LVM. Skip the operation")

	return nil
}

// Snapshot takes a snapshot of the current data state.
func (m *LVManager) Snapshot(_ string) error {
	log.Msg("Snapshot is not supported for LVM. Skip the operation")

	return nil
}

// Reset rollbacks data to ZFS snapshot.
func (m *LVManager) Reset(_ string, _ thinclones.ResetOptions) error {
	log.Msg("Reset is not supported for LVM. Skip the operation")

	return nil
}

// ListBranches lists data pool branches.
func (m *LVManager) ListBranches() (map[string]string, error) {
	log.Msg("ListBranches is not supported for LVM. Skip the operation")

	return nil, nil
}

// AddBranchProp adds branch to snapshot property.
func (m *LVManager) AddBranchProp(_, _ string) error {
	log.Msg("AddBranchProp is not supported for LVM. Skip the operation")

	return nil
}

// DeleteBranchProp deletes branch from snapshot property.
func (m *LVManager) DeleteBranchProp(_, _ string) error {
	log.Msg("DeleteBranchProp is not supported for LVM. Skip the operation")

	return nil
}

// DeleteChildProp deletes child from snapshot property.
func (m *LVManager) DeleteChildProp(_, _ string) error {
	log.Msg("DeleteChildProp is not supported for LVM. Skip the operation")

	return nil
}

// DeleteRootProp deletes root from snapshot property.
func (m *LVManager) DeleteRootProp(_, _ string) error {
	log.Msg("DeleteRootProp is not supported for LVM. Skip the operation")

	return nil
}

// SetRelation sets relation between snapshots.
func (m *LVManager) SetRelation(_, _ string) error {
	log.Msg("SetRelation is not supported for LVM. Skip the operation")

	return nil
}

// SetRoot marks snapshot as a root of branch.
func (m *LVManager) SetRoot(_, _ string) error {
	log.Msg("SetRoot is not supported for LVM. Skip the operation")

	return nil
}

// GetRepo provides data repository details.
func (m *LVManager) GetRepo() (*models.Repo, error) {
	log.Msg("GetRepo is not supported for LVM. Skip the operation")

	return nil, nil
}

// SetDSA sets value of DataStateAt to snapshot.
func (m *LVManager) SetDSA(_, snapshotName string) error {
	log.Msg("SetDSA is not supported for LVM. Skip the operation")

	return nil
}

// SetMessage sets commit message to snapshot.
func (m *LVManager) SetMessage(_, snapshotName string) error {
	log.Msg("SetMessage is not supported for LVM. Skip the operation")

	return nil
}

// SetMountpoint sets clone mount point.
func (m *LVManager) SetMountpoint(_, _ string) error {
	log.Msg("SetMountpoint is not supported for LVM. Skip the operation")

	return nil
}

// Rename renames clone.
func (m *LVManager) Rename(_, _ string) error {
	log.Msg("Rename is not supported for LVM. Skip the operation")

	return nil
}

// DeleteBranch deletes branch.
func (m *LVManager) DeleteBranch(_ string) error {
	log.Msg("DeleteBranch is not supported for LVM. Skip the operation")

	return nil
}
