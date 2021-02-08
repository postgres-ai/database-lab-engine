/*
2020 © Postgres.ai
*/

package pool

import (
	"container/list"
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
	fsManagerList    *list.List
	fsManagerPool    map[string]FSManager
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
		fsManagerList:    list.New(),
	}
}

// Reload reloads pool manager configuration.
func (pm *Manager) Reload(cfg Config) error {
	*pm.cfg = cfg

	return pm.ReloadPools()
}

// SetActive sets a new active pool manager element.
func (pm *Manager) SetActive(element *list.Element) {
	pm.fsManagerList.MoveToFront(element)
}

// Active returns the active storage pool manager.
func (pm *Manager) Active() FSManager {
	active := pm.fsManagerList.Front()

	if active == nil || active.Value == nil {
		return nil
	}

	return pm.getFSManager(active.Value.(string))
}

func (pm *Manager) getFSManager(pool string) FSManager {
	pm.mu.Lock()
	fsm := pm.fsManagerPool[pool]
	pm.mu.Unlock()

	return fsm
}

// GetPoolToUpdate returns the element to update.
func (pm *Manager) GetPoolToUpdate() *list.Element {
	for element := pm.fsManagerList.Back(); element != nil; element = element.Prev() {
		if element.Value == nil {
			return nil
		}

		// The active pool cannot be updated as it leads to downtime.
		if element == pm.fsManagerList.Front() {
			return nil
		}

		fsm := pm.getFSManager(element.Value.(string))

		clones, err := fsm.ListClonesNames()
		if err != nil {
			log.Err("failed to list clones", err)
			return nil
		}

		if len(clones) == 0 {
			return element
		}
	}

	return nil
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

	fsPools, fsManagerList := pm.examineEntries(entries)

	if len(fsPools) == 0 {
		return errors.New("no available pools")
	}

	pm.mu.Lock()
	pm.fsManagerPool = fsPools
	pm.fsManagerList = fsManagerList
	pm.mu.Unlock()

	log.Msg("Available storage pools: ", pm.describeAvailablePools())
	log.Msg("Active pool: ", pm.Active().Pool().Name)

	return nil
}

func (pm *Manager) examineEntries(entries []os.FileInfo) (map[string]FSManager, *list.List) {
	fsManagers := make(map[string]FSManager)
	poolList := &list.List{}

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

		front := poolList.Front()
		if front == nil || front.Value == nil || fsManagers[front.Value.(string)].Pool().DSA.Before(pool.DSA) {
			poolList.PushFront(fsm.Pool().Name)
			continue
		}

		poolList.PushBack(fsm.Pool().Name)
	}

	return fsManagers, poolList
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

	for el := pm.fsManagerList.Front(); el != nil; el = el.Next() {
		if el.Value == nil {
			log.Err("empty element: skip listing")
			continue
		}

		availablePools = append(availablePools, el.Value.(string))
	}

	return availablePools
}
