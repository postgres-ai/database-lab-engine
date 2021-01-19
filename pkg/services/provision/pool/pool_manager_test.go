package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
)

type MockFSManager struct {
	pool *resources.Pool
}

func (m MockFSManager) CreateClone(name, snapshotID string) error { return nil }

func (m MockFSManager) DestroyClone(name string) error { return nil }

func (m MockFSManager) ListClonesNames() ([]string, error) { return nil, nil }

func (m MockFSManager) CreateSnapshot(poolSuffix, dataStateAt string) (snapshotName string, err error) {
	return "", nil
}

func (m MockFSManager) DestroySnapshot(snapshotName string) (err error) { return nil }

func (m MockFSManager) CleanupSnapshots(retentionLimit int) ([]string, error) { return nil, nil }

func (m MockFSManager) GetSnapshots() ([]resources.Snapshot, error) { return nil, nil }

func (m MockFSManager) GetSessionState(name string) (*resources.SessionState, error) { return nil, nil }

func (m MockFSManager) GetDiskState() (*resources.Disk, error) { return nil, nil }

func (m MockFSManager) Pool() *resources.Pool { return m.pool }

func TestLoad(t *testing.T) {
	pm := NewPoolManager(&Config{}, nil)
	startTime := time.Now()

	alpha := MockFSManager{
		pool: &resources.Pool{DSA: time.Time{}},
	}

	fsPool := map[string]FSManager{
		"alpha": alpha,
	}

	activePool, oldestPool := pm.detectWorkingPools(fsPool)

	assert.Equal(t, activePool, alpha)
	assert.Nil(t, oldestPool)

	beta := MockFSManager{
		pool: &resources.Pool{DSA: startTime},
	}
	fsPool["beta"] = beta
	activePool, oldestPool = pm.detectWorkingPools(fsPool)

	assert.Equal(t, activePool, beta)
	assert.Equal(t, oldestPool, alpha)

	gamma := MockFSManager{
		pool: &resources.Pool{DSA: startTime.Add(time.Hour)},
	}
	fsPool["gamma"] = gamma
	activePool, oldestPool = pm.detectWorkingPools(fsPool)

	assert.Equal(t, activePool, gamma)
	assert.Equal(t, oldestPool, alpha)
}
