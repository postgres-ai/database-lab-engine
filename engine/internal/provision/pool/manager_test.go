package pool

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

type recordingRunner struct {
	mu       sync.Mutex
	commands []string
}

func (r *recordingRunner) Run(cmd string, _ ...bool) (string, error) {
	r.mu.Lock()
	r.commands = append(r.commands, cmd)
	r.mu.Unlock()

	return "", nil
}

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

func TestBuildFromExistingManager_LVMClonesUseReloadedPool(t *testing.T) {
	runner := &recordingRunner{}
	startupPool := &resources.Pool{Name: "dblab_vg-dblab_lv", MountDir: "/var/lib/dblab", PoolDirName: "startup"}

	fsm, err := lvm.NewFSManager(runner, startupPool)
	require.NoError(t, err)

	reloadedPool := &resources.Pool{Name: "dblab_vg-dblab_lv", MountDir: "/var/lib/dblab", PoolDirName: "reloaded"}

	_, err = BuildFromExistingManager(fsm, ManagerConfig{Pool: reloadedPool})
	require.NoError(t, err)

	require.NoError(t, fsm.CreateClone(branching.DefaultBranch, "clone_1", "", 0))
	require.NoError(t, fsm.DestroyClone(branching.DefaultBranch, "clone_1", 0))

	commands := strings.Join(runner.commands, "\n")
	assert.Contains(t, commands, "mkdir -p "+reloadedPool.ClonesDir(branching.DefaultBranch))
	assert.Contains(t, commands, "umount "+reloadedPool.ClonesDir(branching.DefaultBranch))
	assert.NotContains(t, commands, "/startup/")
}

func TestBuildFromExistingManager_ReloadRacesWithPoolReaders(t *testing.T) {
	testCases := []struct {
		name       string
		mode       string
		withClones bool
		build      func(t *testing.T, pool *resources.Pool) FSManager
	}{
		{
			name: "zfs", mode: zfs.PoolMode,
			build: func(_ *testing.T, pool *resources.Pool) FSManager {
				return zfs.NewFSManager(nil, zfs.Config{Pool: pool})
			},
		},
		{
			name: "lvm", mode: lvm.PoolMode, withClones: true,
			build: func(t *testing.T, pool *resources.Pool) FSManager {
				fsm, err := lvm.NewFSManager(&recordingRunner{}, pool)
				require.NoError(t, err)

				return fsm
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsm := tc.build(t, &resources.Pool{Name: "dblab_vg-dblab_lv", Mode: tc.mode, DataSubDir: "data"})

			const iterations = 200

			var wg sync.WaitGroup

			wg.Add(3)

			go func() {
				defer wg.Done()

				for i := 0; i < iterations; i++ {
					reloadedPool := &resources.Pool{Name: "dblab_vg-dblab_lv", Mode: tc.mode, DataSubDir: fmt.Sprintf("data_%d", i)}

					_, err := BuildFromExistingManager(fsm, ManagerConfig{Pool: reloadedPool})
					assert.NoError(t, err)
				}
			}()

			go func() {
				defer wg.Done()

				for i := 0; i < iterations; i++ {
					fsm.Pool().SetDSA(time.Now())
				}
			}()

			go func() {
				defer wg.Done()

				for i := 0; i < iterations; i++ {
					pool := fsm.Pool()
					_ = pool.Name
					_ = pool.DataDir()
					_ = pool.DataStateAt()
					_ = fsm.SnapshotList()

					if tc.withClones {
						assert.NoError(t, fsm.CreateClone(branching.DefaultBranch, "clone_1", "", 0))
						assert.NoError(t, fsm.DestroyClone(branching.DefaultBranch, "clone_1", 0))
					}
				}
			}()

			wg.Wait()
		})
	}
}
