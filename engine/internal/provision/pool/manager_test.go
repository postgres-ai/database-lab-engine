package pool

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
)

func TestBuildFromExistingManager_ServesReloadedPool(t *testing.T) {
	reloadedDSA := time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC)

	testCases := []struct {
		name  string
		mode  string
		build func(t *testing.T, pool *resources.Pool) FSManager
	}{
		{
			name: "zfs", mode: zfs.PoolMode,
			build: func(_ *testing.T, pool *resources.Pool) FSManager {
				return zfs.NewFSManager(nil, zfs.Config{Pool: pool})
			},
		},
		{
			name: "lvm", mode: lvm.PoolMode,
			build: func(t *testing.T, pool *resources.Pool) FSManager {
				fsm, err := lvm.NewFSManager(nil, pool)
				require.NoError(t, err)

				return fsm
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startupPool := &resources.Pool{Name: "dblab_vg-dblab_lv", Mode: tc.mode, MountDir: "/var/lib/dblab", DataSubDir: "data"}
			startupPool.SetStatus(resources.ActivePool)

			reloadedPool := &resources.Pool{
				Name: "dblab_vg-dblab_lv", Mode: tc.mode, MountDir: "/var/lib/dblab", DataSubDir: "pgdata", DSA: reloadedDSA,
			}
			reloadedPool.SetStatus(resources.EmptyPool)

			fsm := tc.build(t, startupPool)

			reloaded, err := BuildFromExistingManager(fsm, ManagerConfig{Pool: reloadedPool})
			require.NoError(t, err)

			assert.Same(t, fsm, reloaded)
			assert.Same(t, reloadedPool, reloaded.Pool())
			assert.Equal(t, "pgdata", reloaded.Pool().DataSubDir)
			assert.Equal(t, reloadedDSA, reloaded.Pool().DSA)
			assert.Equal(t, resources.EmptyPool, reloaded.Pool().Status())
			assert.Equal(t, "data", startupPool.DataSubDir)
			assert.Equal(t, resources.ActivePool, startupPool.Status())
		})
	}
}

func TestBuildFromExistingManager_ReloadRacesWithPoolReaders(t *testing.T) {
	t.Parallel()

	fsm := zfs.NewFSManager(nil, zfs.Config{Pool: &resources.Pool{Name: "dblab_pool", Mode: zfs.PoolMode, DataSubDir: "data"}})

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			reloadedPool := &resources.Pool{Name: "dblab_pool", Mode: zfs.PoolMode, DataSubDir: fmt.Sprintf("data_%d", i)}

			_, err := BuildFromExistingManager(fsm, ManagerConfig{Pool: reloadedPool})
			assert.NoError(t, err)
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			pool := fsm.Pool()
			_ = pool.Name
			_ = pool.DataDir()
			_ = pool.DSA
		}
	}()

	wg.Wait()
}
