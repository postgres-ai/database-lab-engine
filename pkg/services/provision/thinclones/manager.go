/*
2020 © Postgres.ai
*/

// Package thinclones provides an interface to work different thin-clone managers.
package thinclones

import (
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

const (
	// ManagerZFS defines "Manager" option value for ZFS.
	ManagerZFS = "zfs"

	// ManagerLVM defines "Manager" option value for LVM.
	ManagerLVM = "lvm"
)

// Manager defines an interface to work different thin-clone managers.
type Manager interface {
	CreateClone(name, snapshotID string) error
	DestroyClone(name string) error
	ListClonesNames() ([]string, error)

	GetSessionState(name string) (*resources.SessionState, error)
	GetDiskState() (*resources.Disk, error)
	CreateSnapshot(dataStateAt string) (snapshotName string, err error)
	DestroySnapshot(snapshotName string) (err error)
	GetSnapshots() ([]resources.Snapshot, error)
}

// ManagerConfig defines thin-clone manager config.
type ManagerConfig struct {
	Pool                 string
	SnapshotFilterSuffix string
	MountDir             string
	OSUsername           string
	ClonePrefix          string
}

// NewManager defines constructor for thin-clone managers.
func NewManager(mode string, runner runners.Runner, config ManagerConfig) (Manager, error) {
	var (
		manager Manager
		err     error
	)

	switch mode {
	case ManagerZFS:
		manager = newManagerZFS(runner, config)
	case ManagerLVM:
		if manager, err = newManagerLVM(runner, config); err != nil {
			return nil, errors.Wrap(err, "failed to initialize LVM thin-clone manager")
		}
	default:
		return nil, errors.New(fmt.Sprintf(`unsupported thin-clone manager specified: "%s"`, mode))
	}

	log.Dbg(fmt.Sprintf(`Using "%s" thin-clone manager.`, mode))

	return manager, nil
}
