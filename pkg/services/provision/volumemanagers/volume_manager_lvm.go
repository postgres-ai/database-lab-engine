/*
2019 © Postgres.ai
*/

package volumemanagers

import (
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/volumemanagers/lvm"
)

const (
	poolPartsLen = 2
)

type volumeManagerLVM struct {
	runner      runners.Runner
	config      VolumeManagerConfig
	volumeGroup string
	logicVolume string
}

// newVolumeManagerLVM creates a new VolumeManager instance for LVM.
func newVolumeManagerLVM(runner runners.Runner, config VolumeManagerConfig) (*volumeManagerLVM, error) {
	m := volumeManagerLVM{}

	m.runner = runner
	m.config = config

	if err := m.parsePool(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *volumeManagerLVM) CreateClone(name, snapshotID string) error {
	return lvm.CreateVolume(m.runner, m.volumeGroup, m.logicVolume, name, m.config.MountDir)
}

func (m *volumeManagerLVM) DestroyClone(name string) error {
	return lvm.RemoveVolume(m.runner, m.volumeGroup, m.logicVolume, name, m.config.MountDir)
}

func (m *volumeManagerLVM) ListClonesNames() ([]string, error) {
	volumes, err := lvm.ListVolumes(m.runner, m.volumeGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list LVM volumes")
	}

	volumesNames := make([]string, 0, len(volumes))

	for _, volume := range volumes {
		volumesNames = append(volumesNames, volume.Name)
	}

	return volumesNames, nil
}

func (m *volumeManagerLVM) GetSessionState(name string) (*resources.SessionState, error) {
	// TODO(anatoly): Implement.
	return &resources.SessionState{}, nil
}

func (m *volumeManagerLVM) GetDiskState() (*resources.Disk, error) {
	// TODO(anatoly): Implement.
	return &resources.Disk{}, nil
}

func (m *volumeManagerLVM) GetSnapshots() ([]resources.Snapshot, error) {
	// TODO(anatoly): Not supported in LVM mode warning.
	return []resources.Snapshot{resources.Snapshot{
		ID: "default",
	}}, nil
}

func (m *volumeManagerLVM) parsePool() error {
	parts := strings.SplitN(m.config.Pool, "/", poolPartsLen)
	if len(parts) < poolPartsLen {
		return errors.Errorf(`failed to parse "pool" value from config`)
	}

	m.volumeGroup = parts[0]
	m.logicVolume = parts[1]

	return nil
}
