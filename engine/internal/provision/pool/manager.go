/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

import (
	"fmt"
	"os/user"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// FSManager defines an interface to work different thin-clone managers.
type FSManager interface {
	Cloner
	Snapshotter
	StateReporter
	Pooler
	Branching
}

// Cloner describes methods of clone management.
type Cloner interface {
	CreateClone(branch, name, snapshotID string, revision int) error
	DestroyClone(branch, name string, revision int) error
	ListClonesNames() ([]string, error)
}

// StateReporter describes methods of state reporting.
type StateReporter interface {
	GetSessionState(branch, name string) (*resources.SessionState, error)
	GetFilesystemState() (models.FileSystem, error)
}

// Snapshotter describes methods of snapshot management.
type Snapshotter interface {
	CreateSnapshot(poolSuffix, dataStateAt string) (snapshotName string, err error)
	DestroySnapshot(snapshotName string, options thinclones.DestroyOptions) (err error)
	CleanupSnapshots(retentionLimit int) ([]string, error)
	SnapshotList() []resources.Snapshot
	RefreshSnapshotList()
}

// Branching describes methods for data branching.
type Branching interface {
	InitBranching() error
	VerifyBranchMetadata() error
	CreateDataset(datasetName string) error
	CreateBranch(branchName, snapshotID string) error
	DestroyDataset(branchName string) (err error)
	ListBranches() (map[string]string, error)
	ListAllBranches() ([]models.BranchEntity, error)
	GetRepo() (*models.Repo, error)
	GetAllRepo() (*models.Repo, error)
	SetRelation(parent, snapshotName string) error
	Snapshot(snapshotName string) error
	Move(baseSnap, currentSnap, target string) error
	SetMountpoint(path, branch string) error
	Rename(oldName, branch string) error
	GetSnapshotProperties(snapshotName string) (thinclones.SnapshotProperties, error)
	AddBranchProp(branch, snapshotName string) error
	DeleteBranchProp(branch, snapshotName string) error
	DeleteChildProp(childSnapshot, snapshotName string) error
	DeleteRootProp(branch, snapshotName string) error
	SetRoot(branch, snapshotName string) error
	SetDSA(dsa, snapshotName string) error
	SetMessage(message, snapshotName string) error
	Reset(snapshotID string, options thinclones.ResetOptions) error
	HasDependentEntity(snapshotName string) ([]string, error)
	KeepRelation(snapshotName string) error
}

// Pooler describes methods for Pool providing.
type Pooler interface {
	Pool() *resources.Pool
}

// ManagerConfig defines thin-clone manager config.
type ManagerConfig struct {
	Pool              *resources.Pool
	PreSnapshotSuffix string
}

// NewManager defines constructor for thin-clone managers.
func NewManager(runner runners.Runner, config ManagerConfig) (FSManager, error) {
	var (
		manager FSManager
		err     error
	)

	switch config.Pool.Mode {
	case zfs.PoolMode:
		zfsConfig, err := buildZFSConfig(config)
		if err != nil {
			return nil, err
		}

		manager = zfs.NewFSManager(runner, zfsConfig)

	case lvm.PoolMode:
		if manager, err = lvm.NewFSManager(runner, config.Pool); err != nil {
			return nil, errors.Wrap(err, "failed to initialize LVM thin-clone manager")
		}

	default:
		return nil, fmt.Errorf(`unsupported thin-clone manager specified: "%s"`, config.Pool.Mode)
	}

	log.Dbg(fmt.Sprintf(`Using "%s" thin-clone manager.`, config.Pool.Mode))

	return manager, nil
}

// BuildFromExistingManager prepares FSManager from an existing one.
func BuildFromExistingManager(fsm FSManager, config ManagerConfig) (FSManager, error) {
	switch manager := fsm.(type) {
	case *zfs.Manager:
		zfsConfig, err := buildZFSConfig(config)
		if err != nil {
			return nil, err
		}

		manager.UpdateConfig(zfsConfig)

		fsm = manager

	case *lvm.LVManager:
		manager.UpdateConfig(config.Pool)

		fsm = manager

	default:
		return nil, fmt.Errorf(`unsupported thin-clone manager: %T`, manager)
	}

	return fsm, nil
}

func buildZFSConfig(config ManagerConfig) (zfs.Config, error) {
	osUser, err := user.Current()
	if err != nil {
		return zfs.Config{}, fmt.Errorf("failed to get current user: %w", err)
	}

	return zfs.Config{
		Pool:              config.Pool,
		PreSnapshotSuffix: config.PreSnapshotSuffix,
		OSUsername:        osUser.Username,
	}, nil
}
