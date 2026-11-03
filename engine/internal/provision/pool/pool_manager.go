/*
2020 © Postgres.ai
*/

package pool

import (
	"container/list"
	"fmt"
	"os"
	"path"
	"path/filepath"
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

// ErrNoPools means that there no available pools.
var ErrNoPools = errors.New("no available pools")

// Manager describes a pool manager.
type Manager struct {
	// reloadMu serializes reloads, which arrive from both SIGHUP and the admin API. It guards cfg and
	// blockDeviceTypes, which are used only while reloading.
	reloadMu         sync.Mutex
	cfg              *Config
	blockDeviceTypes map[string]string
	// mu guards fsManagerList, fsManagerPool and refreshingPool. A reload swaps the list and the map
	// together, and the elements of the list are moved under mu as well.
	mu             *sync.Mutex
	fsManagerList  *list.List
	fsManagerPool  map[string]FSManager
	refreshingPool string

	runner runners.Runner
	// fsInfo reports the pool mode of a mount directory.
	fsInfo func(path string) (string, error)
}

// updateCandidate is a pool that may be selected for a data update.
type updateCandidate struct {
	element *list.Element
	fsm     FSManager
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

// Option configures a pool manager.
type Option func(*Manager)

// WithFSInfo sets the function that reports the pool mode of a mount directory. By default the mode
// is derived from the filesystem type of the directory.
func WithFSInfo(fsInfo func(path string) (string, error)) Option {
	return func(pm *Manager) {
		pm.fsInfo = fsInfo
	}
}

// NewPoolManager creates a new pool manager.
func NewPoolManager(cfg *Config, runner runners.Runner, opts ...Option) *Manager {
	pm := &Manager{
		cfg:              cfg,
		mu:               &sync.Mutex{},
		fsManagerPool:    make(map[string]FSManager),
		runner:           runner,
		fsInfo:           getFSInfo,
		blockDeviceTypes: make(map[string]string),
		fsManagerList:    list.New(),
	}

	for _, opt := range opts {
		opt(pm)
	}

	return pm
}

// Reload reloads pool manager configuration.
func (pm *Manager) Reload(cfg Config) error {
	pm.reloadMu.Lock()
	defer pm.reloadMu.Unlock()

	*pm.cfg = cfg

	return pm.reloadPools()
}

// MakeActive marks element as active pool and moves it to the head of the pool list. A refresh can
// outlive a reload, which replaces the list, so the pool is looked up by name in the current list:
// moving an element of the replaced list would be a silent no-op.
func (pm *Manager) MakeActive(element *list.Element) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if current := pm.findElement(element.Value.(string)); current != nil {
		pm.fsManagerList.MoveToFront(current)
	}

	if first := pm.first(); first != nil {
		first.Pool().SetStatus(resources.ActivePool)
	}
}

// BeginRefresh records the pool a data refresh runs on, so a reload keeps its state.
func (pm *Manager) BeginRefresh(poolName string) {
	pm.mu.Lock()
	pm.refreshingPool = poolName
	pm.mu.Unlock()
}

// EndRefresh clears the pool recorded by BeginRefresh.
func (pm *Manager) EndRefresh() {
	pm.mu.Lock()
	pm.refreshingPool = ""
	pm.mu.Unlock()
}

// First returns the first active storage pool manager.
func (pm *Manager) First() FSManager {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.first()
}

func (pm *Manager) first() FSManager {
	active := pm.fsManagerList.Front()

	if active == nil || active.Value == nil {
		return nil
	}

	return pm.fsManagerPool[active.Value.(string)]
}

// GetPoolByName returns element by pool name.
func (pm *Manager) GetPoolByName(poolName string) *list.Element {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.findElement(poolName)
}

func (pm *Manager) findElement(poolName string) *list.Element {
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
	for _, candidate := range pm.updateCandidates() {
		clones, err := candidate.fsm.ListClonesNames()
		if err != nil {
			log.Err("failed to list clones", err)
			return nil
		}

		if len(clones) == 0 {
			return candidate.element
		}
	}

	return nil
}

// updateCandidates lists pools from the back of the queue. The result outlives the lock because
// listing clones runs an external command.
func (pm *Manager) updateCandidates() []updateCandidate {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	candidates := make([]updateCandidate, 0, pm.fsManagerList.Len())

	for element := pm.fsManagerList.Back(); element != nil; element = element.Prev() {
		if element.Value == nil {
			break
		}

		candidates = append(candidates, updateCandidate{element: element, fsm: pm.fsManagerPool[element.Value.(string)]})
	}

	return candidates
}

// GetFSManager returns a filesystem manager by name if exists.
func (pm *Manager) GetFSManager(name string) (FSManager, error) {
	pm.mu.Lock()
	fsm, ok := pm.fsManagerPool[name]
	pm.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("pool manager not found: %s", name)
	}

	return fsm, nil
}

// GetFSManagerList returns a filesystem manager list.
func (pm *Manager) GetFSManagerList() []FSManager {
	pm.mu.Lock()

	fs := make([]FSManager, 0, len(pm.fsManagerPool))

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

// GetAvailableFSManagers returns a list of non-empty (active or refreshing) filesystem managers.
func (pm *Manager) GetAvailableFSManagers() []FSManager {
	fs := []FSManager{}

	pm.mu.Lock()

	for _, fsManager := range pm.fsManagerPool {
		if pool := fsManager.Pool(); pool != nil && pool.Status() != resources.EmptyPool {
			fs = append(fs, fsManager)
		}
	}

	pm.mu.Unlock()

	return fs
}

// GetFSManagerOrderedList returns a filesystem manager list in the order of the queue.
func (pm *Manager) GetFSManagerOrderedList() []FSManager {
	fs := []FSManager{}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for element := pm.fsManagerList.Front(); element != nil; element = element.Next() {
		if element.Value == nil {
			continue
		}

		fs = append(fs, pm.fsManagerPool[element.Value.(string)])
	}

	return fs
}

// ReloadPools updates available pool managers.
func (pm *Manager) ReloadPools() error {
	pm.reloadMu.Lock()
	defer pm.reloadMu.Unlock()

	return pm.reloadPools()
}

func (pm *Manager) reloadPools() error {
	dirEntries, err := os.ReadDir(pm.cfg.MountDir)
	if err != nil {
		return err
	}

	if err := pm.reloadBlockDevices(); err != nil {
		return err
	}

	fsPools, fsManagerList := pm.examineEntries(dirEntries)

	if len(fsPools) == 0 {
		return ErrNoPools
	}

	pm.setPools(fsPools, fsManagerList)

	log.Msg("Available storage pools: ", pm.describeAvailablePools())
	log.Msg("Active pool: ", pm.First().Pool().Name)

	return nil
}

func (pm *Manager) setPools(fsPools map[string]FSManager, fsManagerList *list.List) {
	pm.mu.Lock()
	pm.fsManagerPool = fsPools
	pm.fsManagerList = fsManagerList
	pm.mu.Unlock()
}

func (pm *Manager) examineEntries(entries []os.DirEntry) (map[string]FSManager, *list.List) {
	fsManagers := make(map[string]FSManager)
	poolList := &list.List{}

	poolMappings := make(map[string]string)

	pm.mu.Lock()
	originalPools := pm.fsManagerPool
	pm.mu.Unlock()

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

		fsType, err := pm.fsInfo(dataPath)
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

		var fsm FSManager

		managerConfig := ManagerConfig{
			Pool:              pool,
			PreSnapshotSuffix: pm.cfg.PreSnapshotSuffix,
		}

		if originalFSM, ok := originalPools[pool.Name]; ok {
			pm.keepRefreshState(pool, originalFSM.Pool())
			fsm, err = BuildFromExistingManager(originalFSM, managerConfig)
		} else {
			fsm, err = NewManager(pm.runner, managerConfig)
		}

		if err != nil {
			log.Msg("failed to create clone manager:", err.Error())
			continue
		}

		if err := preparePoolDirectories(fsm); err != nil {
			log.Msg("failed to prepare pool directories:", err.Error())
			continue
		}

		fsm.RefreshSnapshotList()

		fsManagers[pool.Name] = fsm

		front := poolList.Front()
		if front == nil || front.Value == nil || fsManagers[front.Value.(string)].Pool().DataStateAt().Before(pool.DataStateAt()) {
			poolList.PushFront(fsm.Pool().Name)
			continue
		}

		poolList.PushBack(fsm.Pool().Name)
	}

	return fsManagers, poolList
}

// keepRefreshState carries the state of a pool under a running refresh over a reload. The marker
// cannot describe that pool: it reports the old data as active until the restore wipes the data
// directory, and empty afterwards. A stale refreshing status with no running refresh is not carried,
// so a reload still corrects it.
func (pm *Manager) keepRefreshState(pool, previous *resources.Pool) {
	pm.mu.Lock()
	refreshing := pm.refreshingPool == pool.Name
	pm.mu.Unlock()

	if !refreshing || previous == nil || previous.Status() != resources.RefreshingPool {
		return
	}

	pool.DSA = previous.DataStateAt()
	pool.SetStatus(resources.RefreshingPool)
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

func preparePoolDirectories(fsm FSManager) error {
	dataDir := fsm.Pool().DataDir()
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	return filepath.Walk(dataDir, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			// PGDATA dir permissions must be 0700 to avoid errors.
			err = os.Chmod(name, 0700)
		}

		return err
	})
}

func (pm *Manager) describeAvailablePools() []string {
	availablePools := []string{}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for el := pm.fsManagerList.Front(); el != nil; el = el.Next() {
		if el.Value == nil {
			log.Err("empty element: skip listing")
			continue
		}

		availablePools = append(availablePools, el.Value.(string))
	}

	return availablePools
}
