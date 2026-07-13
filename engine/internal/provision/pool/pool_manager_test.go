/*
2020 © Postgres.ai
*/

package pool

import (
	"container/list"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
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
