/*
2020 © Postgres.ai
*/

package pool

import (
	"container/list"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

type mockFSManager struct {
	pool       *resources.Pool
	clones     []string
	clonesErr  error
	filesystem models.FileSystem
	fsErr      error
}

func (m *mockFSManager) CreateClone(_, _, _ string, _ int) error { return nil }
func (m *mockFSManager) DestroyClone(_, _ string, _ int) error   { return nil }
func (m *mockFSManager) ListClonesNames() ([]string, error)      { return m.clones, m.clonesErr }
func (m *mockFSManager) ListCloneDatasets() ([]thinclones.CloneDataset, error) {
	return nil, m.clonesErr
}
func (m *mockFSManager) GetSessionState(_, _ string) (*resources.SessionState, error) {
	return nil, nil
}
func (m *mockFSManager) GetBatchSessionState(_ []resources.SessionStateRequest) (map[string]resources.SessionState, error) {
	return nil, nil
}
func (m *mockFSManager) GetFilesystemState() (models.FileSystem, error)              { return m.filesystem, m.fsErr }
func (m *mockFSManager) EnsureDataOwnership(_ string) error                          { return nil }
func (m *mockFSManager) CreateSnapshot(_, _ string) (string, error)                  { return "", nil }
func (m *mockFSManager) DestroySnapshot(_ string, _ thinclones.DestroyOptions) error { return nil }
func (m *mockFSManager) CleanupSnapshots(_ int, _ models.RetrievalMode) ([]string, error) {
	return nil, nil
}
func (m *mockFSManager) SnapshotList() []resources.Snapshot       { return nil }
func (m *mockFSManager) RefreshSnapshotList()                     {}
func (m *mockFSManager) Pool() *resources.Pool                    { return m.pool }
func (m *mockFSManager) InitBranching() error                     { return nil }
func (m *mockFSManager) VerifyBranchMetadata() error              { return nil }
func (m *mockFSManager) CreateDataset(_ string) error             { return nil }
func (m *mockFSManager) CreateBranch(_, _ string) error           { return nil }
func (m *mockFSManager) DestroyDataset(_ string) error            { return nil }
func (m *mockFSManager) ListBranches() (map[string]string, error) { return nil, nil }
func (m *mockFSManager) ListAllBranches(_ []string) ([]models.BranchEntity, error) {
	return nil, nil
}
func (m *mockFSManager) GetRepo() (*models.Repo, error)    { return nil, nil }
func (m *mockFSManager) GetAllRepo() (*models.Repo, error) { return nil, nil }
func (m *mockFSManager) SetRelation(_, _ string) error     { return nil }
func (m *mockFSManager) Snapshot(_ string) error           { return nil }
func (m *mockFSManager) Move(_, _, _ string) error         { return nil }
func (m *mockFSManager) SetMountpoint(_, _ string) error   { return nil }
func (m *mockFSManager) Rename(_, _ string) error          { return nil }
func (m *mockFSManager) GetSnapshotProperties(_ string) (thinclones.SnapshotProperties, error) {
	return thinclones.SnapshotProperties{}, nil
}
func (m *mockFSManager) AddBranchProp(_, _ string) error                 { return nil }
func (m *mockFSManager) DeleteBranchProp(_, _ string) error              { return nil }
func (m *mockFSManager) DeleteChildProp(_, _ string) error               { return nil }
func (m *mockFSManager) DeleteRootProp(_, _ string) error                { return nil }
func (m *mockFSManager) SetRoot(_, _ string) error                       { return nil }
func (m *mockFSManager) SetDSA(_, _ string) error                        { return nil }
func (m *mockFSManager) SetMessage(_, _ string) error                    { return nil }
func (m *mockFSManager) Reset(_ string, _ thinclones.ResetOptions) error { return nil }
func (m *mockFSManager) HasDependentEntity(_ string) ([]string, error)   { return nil, nil }
func (m *mockFSManager) KeepRelation(_ string) error                     { return nil }
func (m *mockFSManager) GetDatasetOrigins(_ string) []string             { return nil }
func (m *mockFSManager) GetActiveDatasets(_ string) ([]string, error)    { return nil, nil }
func (m *mockFSManager) SetProtectedTill(_, _ string) error              { return nil }
func (m *mockFSManager) SetDeleteAt(_, _ string) error                   { return nil }
func (m *mockFSManager) GetProtection(_ string) (thinclones.ProtectionProperties, error) {
	return thinclones.ProtectionProperties{}, nil
}
func (m *mockFSManager) ListProtection() (map[string]thinclones.ProtectionProperties, error) {
	return nil, nil
}
func (m *mockFSManager) DestroyBranchDataset(_ string) error { return nil }

func newTestManager(pools map[string]FSManager, poolList *list.List) *Manager {
	return &Manager{
		cfg:           &Config{},
		mu:            &sync.Mutex{},
		fsManagerPool: pools,
		fsManagerList: poolList,
	}
}

// newReloadTestManager serves LVM pools from a temporary mount directory. The pool mode is fixed
// because the test directory is on tmpfs or overlayfs, which the manager does not support.
func newReloadTestManager(t *testing.T, poolNames ...string) *Manager {
	t.Helper()

	mountDir := t.TempDir()

	for _, name := range poolNames {
		require.NoError(t, os.MkdirAll(filepath.Join(mountDir, name, "data"), 0700))
	}

	return NewPoolManager(&Config{MountDir: mountDir, DataSubDir: "data"}, nil,
		WithFSInfo(func(string) (string, error) { return lvm.PoolMode, nil }))
}

func writeDataStateAt(t *testing.T, dataDir string, dsa time.Time) {
	t.Helper()

	marker := dbmarker.NewMarker(dataDir)
	require.NoError(t, marker.CreateConfig())
	require.NoError(t, marker.SaveConfig(&dbmarker.Config{DataStateAt: dsa.Format(tools.DataStateAtFormat)}))
}

func buildPoolList(names ...string) *list.List {
	l := list.New()
	for _, name := range names {
		l.PushBack(name)
	}

	return l
}

func TestManager_First(t *testing.T) {
	t.Run("returns nil when list is empty", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		assert.Nil(t, pm.First())
	})

	t.Run("returns nil when front element value is nil", func(t *testing.T) {
		l := list.New()
		l.PushBack(nil)
		pm := newTestManager(map[string]FSManager{}, l)
		assert.Nil(t, pm.First())
	})

	t.Run("returns first fs manager", func(t *testing.T) {
		pool := &resources.Pool{Name: "pool1"}
		mock := &mockFSManager{pool: pool}
		pm := newTestManager(map[string]FSManager{"pool1": mock}, buildPoolList("pool1"))
		first := pm.First()
		require.NotNil(t, first)
		assert.Equal(t, "pool1", first.Pool().Name)
	})

	t.Run("returns first when multiple pools exist", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))
		first := pm.First()
		require.NotNil(t, first)
		assert.Equal(t, "pool1", first.Pool().Name)
	})
}

func TestManager_MakeActive(t *testing.T) {
	t.Run("moves element to front and sets active status", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

		elem := pm.fsManagerList.Back()
		pm.MakeActive(elem)

		first := pm.First()
		require.NotNil(t, first)
		assert.Equal(t, "pool2", first.Pool().Name)
		assert.Equal(t, resources.ActivePool, first.Pool().Status())
	})

	t.Run("activates the pool of an element held across a reload", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

		elem := pm.GetPoolByName("pool2")
		pm.setPools(pools, buildPoolList("pool1", "pool2"))
		pm.MakeActive(elem)

		first := pm.First()
		require.NotNil(t, first)
		assert.Equal(t, "pool2", first.Pool().Name)
		assert.Equal(t, resources.ActivePool, first.Pool().Status())
	})

	t.Run("keeps the current order when the pool is gone after a reload", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		pm := newTestManager(map[string]FSManager{"pool1": mock1, "pool2": mock2}, buildPoolList("pool1", "pool2"))

		elem := pm.GetPoolByName("pool2")
		pm.setPools(map[string]FSManager{"pool1": mock1}, buildPoolList("pool1"))
		pm.MakeActive(elem)

		first := pm.First()
		require.NotNil(t, first)
		assert.Equal(t, "pool1", first.Pool().Name)
		assert.Equal(t, resources.ActivePool, first.Pool().Status())
	})
}

func TestManager_keepRefreshState(t *testing.T) {
	dsa := time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC)

	testCases := []struct {
		name           string
		refreshingPool string
		previousStatus resources.PoolStatus
		wantStatus     resources.PoolStatus
		wantDSA        time.Time
	}{
		{name: "keeps the state of a running refresh", refreshingPool: "pool1", previousStatus: resources.RefreshingPool,
			wantStatus: resources.RefreshingPool, wantDSA: dsa},
		{name: "drops a stale status with no running refresh", previousStatus: resources.RefreshingPool, wantStatus: resources.EmptyPool},
		{name: "ignores a refresh of another pool", refreshingPool: "pool2", previousStatus: resources.RefreshingPool,
			wantStatus: resources.EmptyPool},
		{name: "ignores a pool whose refresh jobs have not started", refreshingPool: "pool1", previousStatus: resources.ActivePool,
			wantStatus: resources.EmptyPool},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := newTestManager(map[string]FSManager{}, list.New())
			pm.BeginRefresh(tc.refreshingPool)

			previous := &resources.Pool{Name: "pool1", DSA: dsa}
			previous.SetStatus(tc.previousStatus)

			pool := resources.NewPool("pool1")
			pm.keepRefreshState(pool, previous)

			assert.Equal(t, tc.wantStatus, pool.Status())
			assert.Equal(t, tc.wantDSA, pool.DataStateAt())
		})
	}

	t.Run("drops the state once the refresh has ended", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		pm.BeginRefresh("pool1")
		pm.EndRefresh()

		previous := resources.NewPool("pool1")
		previous.SetStatus(resources.RefreshingPool)

		pool := resources.NewPool("pool1")
		pm.keepRefreshState(pool, previous)

		assert.Equal(t, resources.EmptyPool, pool.Status())
	})

	t.Run("handles a manager without a pool", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		pm.BeginRefresh("pool1")

		pool := resources.NewPool("pool1")
		pm.keepRefreshState(pool, nil)

		assert.Equal(t, resources.EmptyPool, pool.Status())
	})
}

func TestManager_ReloadDuringRefresh(t *testing.T) {
	activeDSA := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	refreshedDSA := time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC)

	testCases := []struct {
		name       string
		refreshing bool
		wantStatus resources.PoolStatus
		wantDSA    time.Time
	}{
		{name: "keeps the state of the pool under refresh", refreshing: true, wantStatus: resources.RefreshingPool,
			wantDSA: refreshedDSA},
		{name: "rebuilds a pool with no running refresh from its marker", wantStatus: resources.EmptyPool},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := newReloadTestManager(t, "vg-lv1", "vg-lv2")
			writeDataStateAt(t, filepath.Join(pm.cfg.MountDir, "vg-lv1", "data"), activeDSA)
			require.NoError(t, pm.ReloadPools())

			refreshed, err := pm.GetFSManager("vg-lv2")
			require.NoError(t, err)

			if tc.refreshing {
				pm.BeginRefresh("vg-lv2")
			}

			refreshed.Pool().SetStatus(resources.RefreshingPool)
			refreshed.Pool().SetDSA(refreshedDSA)

			element := pm.GetPoolByName("vg-lv2")
			require.NotNil(t, element)
			require.NoError(t, pm.ReloadPools())

			reloaded, err := pm.GetFSManager("vg-lv2")
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, reloaded.Pool().Status())
			assert.Equal(t, tc.wantDSA, reloaded.Pool().DataStateAt())

			pm.MakeActive(element)

			first := pm.First()
			require.NotNil(t, first)
			assert.Equal(t, "vg-lv2", first.Pool().Name)
			assert.Equal(t, resources.ActivePool, first.Pool().Status())
			assert.Equal(t, tc.wantDSA, first.Pool().DataStateAt())
		})
	}
}

func TestManager_ReloadRacesWithPoolReload(t *testing.T) {
	pm := newReloadTestManager(t, "vg-lv1")
	require.NoError(t, pm.ReloadPools())

	cfg := *pm.cfg

	const iterations = 20

	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			assert.NoError(t, pm.Reload(cfg))
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			assert.NoError(t, pm.ReloadPools())
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			pm.MakeActive(pm.GetPoolByName("vg-lv1"))
		}
	}()

	wg.Wait()
}

func TestManager_ReloadRacesWithListReaders(t *testing.T) {
	t.Parallel()

	mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
	mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
	pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
	pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

	const iterations = 1000

	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			pm.setPools(pools, buildPoolList("pool1", "pool2"))
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			pm.MakeActive(pm.GetPoolByName("pool2"))
			pm.MakeActive(pm.GetPoolByName("pool1"))
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			assert.NotNil(t, pm.First())
			assert.NotNil(t, pm.GetPoolByName("pool2"))
			_, err := pm.GetFSManager("pool1")
			assert.NoError(t, err)
			assert.Len(t, pm.GetFSManagerOrderedList(), 2)
			assert.Len(t, pm.describeAvailablePools(), 2)
			assert.NotNil(t, pm.GetPoolToUpdate())
		}
	}()

	wg.Wait()
}

func TestManager_GetPoolByName(t *testing.T) {
	mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
	mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
	pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
	pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

	t.Run("finds existing pool", func(t *testing.T) {
		elem := pm.GetPoolByName("pool2")
		require.NotNil(t, elem)
		assert.Equal(t, "pool2", elem.Value.(string))
	})

	t.Run("returns nil for non-existent pool", func(t *testing.T) {
		elem := pm.GetPoolByName("pool3")
		assert.Nil(t, elem)
	})

	t.Run("returns nil when element value is nil", func(t *testing.T) {
		l := list.New()
		l.PushBack(nil)
		pm := newTestManager(map[string]FSManager{}, l)
		assert.Nil(t, pm.GetPoolByName("any"))
	})
}

func TestManager_GetPoolToUpdate(t *testing.T) {
	t.Run("returns pool with no clones from back", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}, clones: []string{"clone1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}, clones: []string{}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

		elem := pm.GetPoolToUpdate()
		require.NotNil(t, elem)
		assert.Equal(t, "pool2", elem.Value.(string))
	})

	t.Run("returns nil when all pools have clones", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}, clones: []string{"c1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}, clones: []string{"c2"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))
		assert.Nil(t, pm.GetPoolToUpdate())
	})

	t.Run("returns nil when list clone fails", func(t *testing.T) {
		mock := &mockFSManager{pool: &resources.Pool{Name: "pool1"}, clonesErr: fmt.Errorf("error")}
		pools := map[string]FSManager{"pool1": mock}
		pm := newTestManager(pools, buildPoolList("pool1"))
		assert.Nil(t, pm.GetPoolToUpdate())
	})

	t.Run("returns nil on empty list", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		assert.Nil(t, pm.GetPoolToUpdate())
	})

	t.Run("returns nil when element value is nil", func(t *testing.T) {
		l := list.New()
		l.PushBack(nil)
		pm := newTestManager(map[string]FSManager{}, l)
		assert.Nil(t, pm.GetPoolToUpdate())
	})
}

func TestManager_GetFSManager(t *testing.T) {
	mock := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
	pm := newTestManager(map[string]FSManager{"pool1": mock}, buildPoolList("pool1"))

	t.Run("returns manager when found", func(t *testing.T) {
		fsm, err := pm.GetFSManager("pool1")
		require.NoError(t, err)
		assert.Equal(t, "pool1", fsm.Pool().Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := pm.GetFSManager("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pool manager not found")
	})
}

func TestManager_GetFSManagerList(t *testing.T) {
	t.Run("returns empty list when no pools", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		result := pm.GetFSManagerList()
		assert.Empty(t, result)
	})

	t.Run("returns all managers", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))
		result := pm.GetFSManagerList()
		assert.Len(t, result, 2)
	})
}

func TestManager_GetAvailableFSManagers(t *testing.T) {
	t.Run("excludes empty pools", func(t *testing.T) {
		activePool := &resources.Pool{Name: "active"}
		activePool.SetStatus(resources.ActivePool)
		emptyPool := &resources.Pool{Name: "empty"}
		emptyPool.SetStatus(resources.EmptyPool)

		pools := map[string]FSManager{
			"active": &mockFSManager{pool: activePool},
			"empty":  &mockFSManager{pool: emptyPool},
		}
		pm := newTestManager(pools, buildPoolList("active", "empty"))
		result := pm.GetAvailableFSManagers()
		require.Len(t, result, 1)
		assert.Equal(t, "active", result[0].Pool().Name)
	})

	t.Run("includes refreshing pools", func(t *testing.T) {
		refreshing := &resources.Pool{Name: "refreshing"}
		refreshing.SetStatus(resources.RefreshingPool)
		pools := map[string]FSManager{"refreshing": &mockFSManager{pool: refreshing}}
		pm := newTestManager(pools, buildPoolList("refreshing"))
		result := pm.GetAvailableFSManagers()
		require.Len(t, result, 1)
	})

	t.Run("skips nil pools", func(t *testing.T) {
		pools := map[string]FSManager{"nil_pool": &mockFSManager{pool: nil}}
		pm := newTestManager(pools, buildPoolList("nil_pool"))
		result := pm.GetAvailableFSManagers()
		assert.Empty(t, result)
	})

	t.Run("returns empty when no pools", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		assert.Empty(t, pm.GetAvailableFSManagers())
	})
}

func TestManager_GetFSManagerOrderedList(t *testing.T) {
	t.Run("returns managers in list order", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		mock2 := &mockFSManager{pool: &resources.Pool{Name: "pool2"}}
		mock3 := &mockFSManager{pool: &resources.Pool{Name: "pool3"}}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2, "pool3": mock3}
		pm := newTestManager(pools, buildPoolList("pool2", "pool3", "pool1"))

		result := pm.GetFSManagerOrderedList()
		require.Len(t, result, 3)
		assert.Equal(t, "pool2", result[0].Pool().Name)
		assert.Equal(t, "pool3", result[1].Pool().Name)
		assert.Equal(t, "pool1", result[2].Pool().Name)
	})

	t.Run("returns empty list when no pools", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		assert.Empty(t, pm.GetFSManagerOrderedList())
	})

	t.Run("skips nil elements", func(t *testing.T) {
		mock := &mockFSManager{pool: &resources.Pool{Name: "pool1"}}
		l := list.New()
		l.PushBack(nil)
		l.PushBack("pool1")
		pm := newTestManager(map[string]FSManager{"pool1": mock}, l)
		result := pm.GetFSManagerOrderedList()
		require.Len(t, result, 1)
		assert.Equal(t, "pool1", result[0].Pool().Name)
	})
}

func TestManager_CollectPoolStat(t *testing.T) {
	t.Run("collects stats from all pools", func(t *testing.T) {
		mock1 := &mockFSManager{
			pool:       &resources.Pool{Name: "pool1"},
			filesystem: models.FileSystem{Mode: "zfs", Size: 100, Used: 40},
		}
		mock2 := &mockFSManager{
			pool:       &resources.Pool{Name: "pool2"},
			filesystem: models.FileSystem{Mode: "zfs", Size: 200, Used: 80},
		}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))

		stat := pm.CollectPoolStat()
		assert.Equal(t, 2, stat.Number)
		assert.Equal(t, "zfs", stat.FSType)
		assert.Equal(t, uint64(300), stat.TotalSize)
		assert.Equal(t, uint64(120), stat.TotalUsed)
	})

	t.Run("returns empty stats when no pools", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		stat := pm.CollectPoolStat()
		assert.Equal(t, telemetry.PoolStat{}, stat)
	})

	t.Run("skips pools with nil pool reference", func(t *testing.T) {
		mock := &mockFSManager{pool: nil}
		pools := map[string]FSManager{"pool1": mock}
		pm := newTestManager(pools, buildPoolList("pool1"))
		stat := pm.CollectPoolStat()
		assert.Equal(t, 1, stat.Number)
		assert.Equal(t, uint64(0), stat.TotalSize)
	})

	t.Run("skips pools with filesystem error", func(t *testing.T) {
		mock := &mockFSManager{pool: &resources.Pool{Name: "pool1"}, fsErr: fmt.Errorf("disk error")}
		pools := map[string]FSManager{"pool1": mock}
		pm := newTestManager(pools, buildPoolList("pool1"))
		stat := pm.CollectPoolStat()
		assert.Equal(t, 1, stat.Number)
		assert.Equal(t, uint64(0), stat.TotalSize)
	})

	t.Run("sets fs type from first successful pool", func(t *testing.T) {
		mock1 := &mockFSManager{pool: &resources.Pool{Name: "pool1"}, fsErr: fmt.Errorf("error")}
		mock2 := &mockFSManager{
			pool:       &resources.Pool{Name: "pool2"},
			filesystem: models.FileSystem{Mode: "lvm", Size: 50, Used: 10},
		}
		pools := map[string]FSManager{"pool1": mock1, "pool2": mock2}
		pm := newTestManager(pools, buildPoolList("pool1", "pool2"))
		stat := pm.CollectPoolStat()
		assert.Equal(t, "lvm", stat.FSType)
	})
}

func TestManager_describeAvailablePools(t *testing.T) {
	t.Run("returns pool names in order", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, buildPoolList("alpha", "beta", "gamma"))
		result := pm.describeAvailablePools()
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, result)
	})

	t.Run("returns empty slice when no pools", func(t *testing.T) {
		pm := newTestManager(map[string]FSManager{}, list.New())
		result := pm.describeAvailablePools()
		assert.Empty(t, result)
	})

	t.Run("skips nil elements", func(t *testing.T) {
		l := list.New()
		l.PushBack("pool1")
		l.PushBack(nil)
		l.PushBack("pool2")
		pm := newTestManager(map[string]FSManager{}, l)
		result := pm.describeAvailablePools()
		assert.Equal(t, []string{"pool1", "pool2"}, result)
	})
}

func TestNewPoolManager(t *testing.T) {
	t.Run("creates manager with initialized fields", func(t *testing.T) {
		cfg := &Config{MountDir: "/mnt/data"}
		pm := NewPoolManager(cfg, nil)
		require.NotNil(t, pm)
		assert.NotNil(t, pm.fsManagerPool)
		assert.NotNil(t, pm.fsManagerList)
		assert.NotNil(t, pm.mu)
		assert.NotNil(t, pm.blockDeviceTypes)
		assert.Equal(t, "/mnt/data", pm.cfg.MountDir)
	})
}
