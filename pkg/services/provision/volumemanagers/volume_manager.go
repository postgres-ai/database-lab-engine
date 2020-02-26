/*
2020 © Postgres.ai
*/

// Package volumemanagers provides an interface to work different volume managers.
package volumemanagers

import (
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

const (
	// VolumeManagerZFS defines "VolumeManager" option value for ZFS.
	VolumeManagerZFS = "zfs"

	// VolumeManagerLVM defines "VolumeManager" option value for LVM.
	VolumeManagerLVM = "lvm"
)

// VolumeManager defines an interface to work different volume managers.
type VolumeManager interface {
	CreateClone(name, snapshotID string) error
	DestroyClone(name string) error
	ListClonesNames() ([]string, error)

	GetSessionState(name string) (*resources.SessionState, error)
	GetDiskState() (*resources.Disk, error)
	GetSnapshots() ([]resources.Snapshot, error)
}

// VolumeManagerConfig defines volume manager config.
type VolumeManagerConfig struct {
	Pool                 string
	SnapshotFilterSuffix string
	MountDir             string
	OSUsername           string
	ClonePrefix          string
}

// NewVolumeManager defines constructor for volume managers.
func NewVolumeManager(mode string, runner runners.Runner, config VolumeManagerConfig) (VolumeManager, error) {
	var (
		manager VolumeManager
		err     error
	)

	switch mode {
	case VolumeManagerZFS:
		manager = newVolumeManagerZFS(runner, config)
	case VolumeManagerLVM:
		if manager, err = newVolumeManagerLVM(runner, config); err != nil {
			return nil, errors.Wrap(err, "failed to initialize LVM volume manager")
		}
	default:
		return nil, errors.New(fmt.Sprintf(`unsupported mode specified: "%s"`, mode))
	}

	log.Dbg(fmt.Sprintf(`Using "%s" mode.`, mode))

	return manager, nil
}
