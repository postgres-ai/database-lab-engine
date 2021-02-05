/*
2020 © Postgres.ai
*/

package pool

import (
	"io/ioutil"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
)

const (
	// ZFS defines the zfs filesystem name.
	ZFS = "zfs"
	// LVM defines the lvm filesystem name.
	LVM = "lvm"
	// ext4 defines the ext4 filesystem name.
	ext4 = "ext4"
)

// Manager describes a pool manager.
type Manager struct {
	cfg              *Config
	mu               *sync.Mutex
	fsManagerPool    map[string]FSManager
	fsManager        FSManager
	oldFsManager     FSManager
	runner           runners.Runner
	blockDeviceTypes map[string]string
}

// Config defines a config of a pool manager.
type Config struct {
	MountDir          string `yaml:"mountDir"`
	CloneSubDir       string `yaml:"clonesMountSubDir"`
	DataSubDir        string `yaml:"dataSubDir"`
	SocketSubDir      string `yaml:"socketSubDir"`
	PreSnapshotSuffix string `yaml:"preSnapshotSuffix"`
}

// NewPoolManager creates a new pool manager.
func NewPoolManager(cfg *Config, runner runners.Runner) *Manager {
	return &Manager{
		cfg:              cfg,
		mu:               &sync.Mutex{},
		fsManagerPool:    make(map[string]FSManager),
		runner:           runner,
		blockDeviceTypes: make(map[string]string),
	}
}

// Reload reloads pool manager configuration.
func (pm *Manager) Reload(cfg Config) error {
	*pm.cfg = cfg

	return pm.ReloadPools()
}

// Active returns the active filesystem pool manager.
func (pm *Manager) Active() FSManager {
	return pm.fsManager
}

// SetActive sets a new active pool manager.
func (pm *Manager) SetActive(active FSManager) {
	pm.fsManager = active
}

// Oldest returns the oldest filesystem pool manager.
func (pm *Manager) Oldest() FSManager {
	return pm.oldFsManager
}

// SetOldest sets a pool manager to update.
func (pm *Manager) SetOldest(pool FSManager) {
	pm.oldFsManager = pool
}

// GetFSManager returns a filesystem manager by name if exists.
func (pm *Manager) GetFSManager(name string) (FSManager, error) {
	pm.mu.Lock()
	fsm, ok := pm.fsManagerPool[name]
	pm.mu.Unlock()

	if !ok {
		return nil, errors.New("clone manager not found")
	}

	return fsm, nil
}

// GetFSManagerList returns a filesystem manager list.
func (pm *Manager) GetFSManagerList() []FSManager {
	fs := []FSManager{}

	pm.mu.Lock()

	for _, pool := range pm.fsManagerPool {
		fs = append(fs, pool)
	}

	pm.mu.Unlock()

	return fs
}

// ReloadPools updates available pool managers.
func (pm *Manager) ReloadPools() error {
	entries, err := ioutil.ReadDir(pm.cfg.MountDir)
	if err != nil {
		return err
	}

	if err := pm.reloadBlockDevices(); err != nil {
		return err
	}

	fsPools := pm.examineEntries(entries)

	if len(fsPools) == 0 {
		return errors.New("no available filesystem pools")
	}

	active, old := pm.detectWorkingPools(fsPools)

	if active == nil {
		return errors.New("active pool not found: make sure it exists")
	}

	pm.mu.Lock()

	pm.fsManagerPool = fsPools
	pm.SetActive(active)
	pm.SetOldest(old)

	pm.mu.Unlock()

	log.Msg("Available FS pools: ", pm.describeAvailablePools())
	log.Msg("Active pool: ", pm.Active().Pool().Name)

	return nil
}

func (pm *Manager) examineEntries(entries []os.FileInfo) map[string]FSManager {
	fsManagers := make(map[string]FSManager)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dataPath := path.Join(pm.cfg.MountDir, entry.Name())

		log.Msg("Discovering: ", dataPath)

		fsType, err := pm.getFSInfo(dataPath)
		if err != nil {
			log.Msg("failed to get a filesystem info: ", err.Error())
			continue
		}

		if fsType != ZFS && fsType != LVM {
			log.Msg("Unsupported filesystem: ", fsType, entry.Name())
			continue
		}

		pool := &resources.Pool{
			Mode:         fsType,
			Name:         entry.Name(),
			MountDir:     pm.cfg.MountDir,
			CloneSubDir:  pm.cfg.CloneSubDir,
			DataSubDir:   pm.cfg.DataSubDir,
			SocketSubDir: pm.cfg.SocketSubDir,
		}

		log.Dbg("Data ", pool)

		dataStateAt, err := extractDataStateAt(pool.DataDir())
		if err != nil {
			log.Msg("failed to extract dataStateAt:", err.Error())
		}

		if dataStateAt != nil {
			pool.DSA = *dataStateAt
			log.Msg(pool.DSA.String())
		}

		fsm, err := NewManager(pm.runner, ManagerConfig{
			Pool:              pool,
			PreSnapshotSuffix: pm.cfg.PreSnapshotSuffix,
		})
		if err != nil {
			log.Msg("failed to create clone manager:", err.Error())
			continue
		}

		// TODO(akartasov): extract pool name.
		fsManagers[entry.Name()] = fsm
	}

	return fsManagers
}

// reloadBlockDevices gets filesystem types of block devices.
// Temporarily switched off because cannot detect LVM types inside a container.
func (pm *Manager) reloadBlockDevices() error {
	blockDeviceTypes, err := getBlockDeviceTypes()
	if err != nil {
		return err
	}

	pm.blockDeviceTypes = blockDeviceTypes

	return nil
}

func (pm *Manager) detectWorkingPools(fsm map[string]FSManager) (FSManager, FSManager) {
	var fsManager, old FSManager

	for _, manager := range fsm {
		if fsManager == nil {
			fsManager = manager
			continue
		}

		if fsManager.Pool().DSA.Before(manager.Pool().DSA) {
			if old == nil {
				old = fsManager
			}

			fsManager = manager

			continue
		}

		if old == nil || manager.Pool().DSA.Before(old.Pool().DSA) {
			old = manager
		}
	}

	return fsManager, old
}

func extractDataStateAt(dataPath string) (*time.Time, error) {
	marker := dbmarker.NewMarker(dataPath)

	dblabDescription, err := marker.GetConfig()
	if err != nil {
		log.Msg("DBMarker error:", err.Error())
		return nil, err
	}

	if dblabDescription.DataStateAt == "" {
		log.Msg("DataStateAt is empty. Data is not ready")
		return nil, err
	}

	dsa, err := time.Parse(tools.DataStateAtFormat, dblabDescription.DataStateAt)
	if err != nil {
		log.Msg("failed to parse DataStateAt")
		return nil, err
	}

	return &dsa, nil
}

func (pm *Manager) getFSInfo(path string) (string, error) {
	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(path, &fs); err != nil {
		return "", err
	}

	fsType := detectFSType(fs.Type)
	if fsType == ext4 {
		// cannot detect LVM checking the blockDeviceTypes map.
		return LVM, nil
	}

	return fsType, nil
}

func (pm *Manager) describeAvailablePools() []string {
	availablePools := []string{}

	pm.mu.Lock()

	for _, fsm := range pm.fsManagerPool {
		availablePools = append(availablePools, fsm.Pool().DataDir())
	}

	pm.mu.Unlock()

	return availablePools
}
