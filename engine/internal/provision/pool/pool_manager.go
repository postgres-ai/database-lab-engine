/*
2020 © Postgres.ai
*/

package pool

import (
	"container/list"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
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
	ObserverSubDir    string `yaml:"observerSubDir"`
	PreSnapshotSuffix string `yaml:"preSnapshotSuffix"`
	SelectedPool      string `yaml:"selectedPool"`
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

// MakeActive marks element as active pool and moves it to the head of the pool list.
func (pm *Manager) MakeActive(element *list.Element) {
	pm.fsManagerList.MoveToFront(element)

	if first := pm.First(); first != nil {
		first.Pool().SetStatus(resources.ActivePool)
	}
}

// First returns the first active storage pool manager.
func (pm *Manager) First() FSManager {
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

// GetPoolByName returns element by pool name.
func (pm *Manager) GetPoolByName(poolName string) *list.Element {
	for element := pm.fsManagerList.Front(); element != nil; element = element.Next() {
		if element.Value == nil {
			return nil
		}

		// The active pool cannot be updated as it leads to downtime.
		if element.Value.(string) == poolName {
			return element
		}
	}

	return nil
}

// GetPoolToUpdate returns element to update.
func (pm *Manager) GetPoolToUpdate() *list.Element {
	for element := pm.fsManagerList.Back(); element != nil; element = element.Prev() {
		if element.Value == nil {
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
		return nil, errors.New("pool manager not found")
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

// CollectPoolStat collects pool stats.
func (pm *Manager) CollectPoolStat() telemetry.PoolStat {
	fsManagerList := pm.GetFSManagerList()
	ps := telemetry.PoolStat{
		Number: len(fsManagerList),
	}

	for _, fsManager := range fsManagerList {
		if fsManager.Pool() == nil {
			continue
		}

		fileSystem, err := fsManager.GetFilesystemState()
		if err != nil {
			log.Err("failed to get disk stats for the pool", fsManager.Pool().Name)
			continue
		}

		if ps.FSType == "" {
			ps.FSType = fileSystem.Mode
		}

		ps.TotalSize += fileSystem.Size
		ps.TotalUsed += fileSystem.Used
	}

	return ps
}

// GetActiveFSManagers returns a list of active filesystem managers.
func (pm *Manager) GetActiveFSManagers() []FSManager {
	fs := []FSManager{}

	pm.mu.Lock()

	for _, fsManager := range pm.fsManagerPool {
		if pool := fsManager.Pool(); pool != nil && pool.Status() == resources.ActivePool {
			fs = append(fs, fsManager)
		}
	}

	pm.mu.Unlock()

	return fs
}

// GetFSManagerOrderedList returns a filesystem manager list in the order of the queue.
func (pm *Manager) GetFSManagerOrderedList() []FSManager {
	fs := []FSManager{}

	for element := pm.fsManagerList.Front(); element != nil; element = element.Next() {
		if element.Value == nil {
			continue
		}

		fs = append(fs, pm.getFSManager(element.Value.(string)))
	}

	return fs
}

// ReloadPools updates available pool managers.
func (pm *Manager) ReloadPools() error {
	dirEntries, err := os.ReadDir(pm.cfg.MountDir)
	if err != nil {
		return err
	}

	if err := pm.reloadBlockDevices(); err != nil {
		return err
	}

	fsPools, fsManagerList := pm.examineEntries(dirEntries)

	if len(fsPools) == 0 {
		return errors.New("no available pools")
	}

	pm.mu.Lock()
	pm.fsManagerPool = fsPools
	pm.fsManagerList = fsManagerList
	pm.mu.Unlock()

	log.Msg("Available storage pools: ", pm.describeAvailablePools())
	log.Msg("Active pool: ", pm.First().Pool().Name)

	return nil
}

func (pm *Manager) examineEntries(entries []os.DirEntry) (map[string]FSManager, *list.List) {
	fsManagers := make(map[string]FSManager)
	poolList := &list.List{}

	poolMappings := make(map[string]string)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dataPath := path.Join(pm.cfg.MountDir, entry.Name())

		log.Msg("Discovering: ", dataPath)

		if pm.cfg.SelectedPool != "" && pm.cfg.SelectedPool != entry.Name() {
			log.Msg(fmt.Sprintf("Skip the entry %q as it doesn't match with the selected pool %s", entry.Name(), pm.cfg.SelectedPool))
			continue
		}

		fsType, err := pm.getFSInfo(dataPath)
		if err != nil {
			log.Msg("failed to get a filesystem info: ", err.Error())
			continue
		}

		if fsType != zfs.PoolMode && fsType != lvm.PoolMode {
			log.Msg("Unsupported filesystem: ", fsType, entry.Name())
			continue
		}

		pool := &resources.Pool{
			Mode:           fsType,
			Name:           entry.Name(),
			PoolDirName:    entry.Name(),
			MountDir:       pm.cfg.MountDir,
			CloneSubDir:    pm.cfg.CloneSubDir,
			DataSubDir:     pm.cfg.DataSubDir,
			SocketSubDir:   pm.cfg.SocketSubDir,
			ObserverSubDir: pm.cfg.ObserverSubDir,
		}
		pool.SetStatus(resources.EmptyPool)

		log.Dbg("Data ", pool)

		dataStateAt, err := extractDataStateAt(pool.DataDir())
		if err != nil {
			log.Msg("failed to extract dataStateAt:", err.Error())
		}

		if dataStateAt != nil {
			pool.DSA = *dataStateAt
			pool.SetStatus(resources.ActivePool)
			log.Msg(pool.DSA.String())
		}

		// A custom pool name is not available for LVM.
		if fsType == zfs.PoolMode {
			if len(poolMappings) == 0 {
				poolMappings, err = zfs.PoolMappings(pm.runner, pm.cfg.MountDir, pm.cfg.PreSnapshotSuffix)
				if err != nil {
					log.Msg("failed to get pool mappings:", err.Error())
					continue
				}
			}

			if poolName, ok := poolMappings[entry.Name()]; ok {
				pool.Name = poolName
			}
		}

		fsm, err := NewManager(pm.runner, ManagerConfig{
			Pool:              pool,
			PreSnapshotSuffix: pm.cfg.PreSnapshotSuffix,
		})
		if err != nil {
			log.Msg("failed to create clone manager:", err.Error())
			continue
		}

		fsManagers[pool.Name] = fsm

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
		log.Msg("cannot read DBMarker configuration: ", err.Error())
		return nil, err
	}

	if dblabDescription.DataStateAt == "" {
		log.Msg("DataStateAt is empty. Data is not ready")
		return nil, err
	}

	dsa, err := time.Parse(tools.DataStateAtFormat, dblabDescription.DataStateAt)
	if err != nil {
		log.Msg("failed to parse DataStateAt: ", err.Error())
		return nil, err
	}

	return &dsa, nil
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
