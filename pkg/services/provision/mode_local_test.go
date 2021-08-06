package provision

import (
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

type mockPortChecker struct{}

func (m mockPortChecker) checkPortAvailability(_ string, _ uint) error {
	return nil
}

func TestPortAllocation(t *testing.T) {
	p := &Provisioner{
		mu: &sync.Mutex{},
		config: &Config{
			PortPool: PortPool{
				From: 6000,
				To:   6002,
			},
		},
		portChecker: &mockPortChecker{},
	}

	// Initialize port pool.
	require.NoError(t, p.initPortPool())

	// Allocate a new port.
	port, err := p.allocatePort()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Allocate one more port.
	_, err = p.allocatePort()
	require.NoError(t, err)

	// Impossible allocate a new port.
	_, err = p.allocatePort()
	assert.IsType(t, errors.Cause(err), &NoRoomError{})
	assert.EqualError(t, err, "session cannot be started because there is no room: no available ports")

	// Free port and allocate a new one.
	require.NoError(t, p.freePort(port))
	port, err = p.allocatePort()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Try to free a non-existing port.
	err = p.freePort(1)
	assert.EqualError(t, err, "port 1 is out of bounds of the port pool")
}

type mockFSManager struct{}

func (m mockFSManager) CreateClone(name, snapshotID string) error {
	return nil
}

func (m mockFSManager) DestroyClone(name string) error {
	return nil
}

func (m mockFSManager) ListClonesNames() ([]string, error) {
	return []string{"test_clone_0001", "test_clone_0002"}, nil
}

func (m mockFSManager) CreateSnapshot(poolSuffix, dataStateAt string) (snapshotName string, err error) {
	return "", nil
}

func (m mockFSManager) DestroySnapshot(snapshotName string) (err error) {
	return nil
}

func (m mockFSManager) CleanupSnapshots(retentionLimit int) ([]string, error) {
	return nil, nil
}

func (m mockFSManager) GetSnapshots() ([]resources.Snapshot, error) {
	return nil, nil
}

func (m mockFSManager) GetSessionState(name string) (*resources.SessionState, error) {
	return nil, nil
}

func (m mockFSManager) GetDiskState() (*resources.Disk, error) {
	return nil, nil
}

func (m mockFSManager) Pool() *resources.Pool {
	return &resources.Pool{
		Name:   "TestPool",
		Mode:   "zfs",
		DSA:    time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC),
		Status: resources.ActivePool,
	}
}

func TestBuildPoolEntry(t *testing.T) {
	testFSManager := mockFSManager{}

	expectedEntry := models.PoolEntry{
		Name:        "TestPool",
		Mode:        "zfs",
		DataStateAt: "2021-08-01 00:00:00 +0000 UTC",
		Status:      resources.ActivePool,
		CloneList:   []string{"test_clone_0001", "test_clone_0002"},
	}

	poolEntry, err := buildPoolEntry(testFSManager)
	assert.Nil(t, err)
	assert.Equal(t, expectedEntry, poolEntry)
}
