package provision

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestPortAllocation(t *testing.T) {
	cfg := &Config{
		PortPool: PortPool{
			From: 6000,
			To:   6002,
		},
	}

	p, err := New(context.Background(), cfg, &resources.DB{}, &client.Client{}, &pool.Manager{}, "instanceID", "networkID", "")
	require.NoError(t, err)

	// Allocate a new port.
	port, err := p.allocatePort()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Allocate one more port.
	_, err = p.allocatePort()
	require.NoError(t, err)

	// Allocate one more port.
	_, err = p.allocatePort()
	require.NoError(t, err)

	// Impossible allocate a new port.
	_, err = p.allocatePort()
	assert.IsType(t, errors.Cause(err), &NoRoomError{})
	assert.EqualError(t, err, "session cannot be started because there is no room: no available ports")

	// Free port and allocate a new one.
	require.NoError(t, p.FreePort(port))
	port, err = p.allocatePort()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Try to free a non-existing port.
	err = p.FreePort(1)
	assert.EqualError(t, err, "port 1 is out of bounds of the port pool")
}

type mockFSManager struct {
	pool      *resources.Pool
	cloneList []string
}

func (m mockFSManager) CreateClone(_, _ string) error {
	return nil
}

func (m mockFSManager) DestroyClone(_ string) error {
	return nil
}

func (m mockFSManager) ListClonesNames() ([]string, error) {
	return m.cloneList, nil
}

func (m mockFSManager) CreateSnapshot(_, _ string) (snapshotName string, err error) {
	return "", nil
}

func (m mockFSManager) DestroySnapshot(_ string) (err error) {
	return nil
}

func (m mockFSManager) CleanupSnapshots(_ int) ([]string, error) {
	return nil, nil
}

func (m mockFSManager) SnapshotList() []resources.Snapshot {
	return nil
}

func (m mockFSManager) RefreshSnapshotList() {
}

func (m mockFSManager) GetSessionState(_ string) (*resources.SessionState, error) {
	return nil, nil
}

func (m mockFSManager) GetFilesystemState() (models.FileSystem, error) {
	return models.FileSystem{Mode: "zfs"}, nil
}

func (m mockFSManager) Pool() *resources.Pool {
	return m.pool
}

func (m mockFSManager) InitBranching() error {
	return nil
}

func (m mockFSManager) VerifyBranchMetadata() error {
	return nil
}

func (m mockFSManager) CreateBranch(_, _ string) error {
	return nil
}

func (m mockFSManager) Snapshot(_ string) error {
	return nil
}

func (m mockFSManager) Reset(_ string, _ thinclones.ResetOptions) error {
	return nil
}

func (m mockFSManager) ListBranches() (map[string]string, error) {
	return nil, nil
}

func (m mockFSManager) AddBranchProp(_, _ string) error {
	return nil
}

func (m mockFSManager) DeleteBranchProp(_, _ string) error {
	return nil
}

func (m mockFSManager) SetRelation(_, _ string) error {
	return nil
}

func (m mockFSManager) SetRoot(_, _ string) error {
	return nil
}

func (m mockFSManager) GetRepo() (*models.Repo, error) {
	return nil, nil
}

func (m mockFSManager) SetDSA(_, _ string) error {
	return nil
}

func (m mockFSManager) SetMessage(_, _ string) error {
	return nil
}

func (m mockFSManager) SetMountpoint(_, _ string) error {
	return nil
}

func (m mockFSManager) Rename(_, _ string) error {
	return nil
}

func (m mockFSManager) DeleteBranch(_ string) error {
	return nil
}

func (m mockFSManager) DeleteChildProp(_, _ string) error {
	return nil
}

func (m mockFSManager) DeleteRootProp(_, _ string) error {
	return nil
}

func (m mockFSManager) HasDependentEntity(_ string) error {
	return nil
}

func TestBuildPoolEntry(t *testing.T) {
	testCases := []struct {
		pool          *resources.Pool
		poolStatus    resources.PoolStatus
		cloneList     []string
		expectedEntry models.PoolEntry
	}{
		{
			pool: &resources.Pool{
				Name: "TestPool",
				Mode: "zfs",
				DSA:  time.Date(2021, 8, 1, 0, 0, 0, 0, time.UTC),
			},
			poolStatus: resources.ActivePool,
			cloneList:  []string{"test_clone_0001", "test_clone_0002"},
			expectedEntry: models.PoolEntry{
				Name:        "TestPool",
				Mode:        "zfs",
				DataStateAt: &models.LocalTime{Time: time.Date(2021, 8, 01, 0, 0, 0, 0, time.UTC)},
				Status:      resources.ActivePool,
				CloneList:   []string{"test_clone_0001", "test_clone_0002"},
				FileSystem:  models.FileSystem{Mode: "zfs"},
			},
		},
		{
			pool: &resources.Pool{
				Name: "TestPoolWithoutDSA",
				Mode: "zfs",
			},
			poolStatus: resources.EmptyPool,
			cloneList:  []string{},
			expectedEntry: models.PoolEntry{
				Name:        "TestPoolWithoutDSA",
				Mode:        "zfs",
				DataStateAt: &models.LocalTime{},
				Status:      resources.EmptyPool,
				CloneList:   []string{},
				FileSystem:  models.FileSystem{Mode: "zfs"},
			},
		},
	}

	for _, tc := range testCases {
		p := tc.pool
		p.SetStatus(tc.poolStatus)

		testFSManager := mockFSManager{
			pool:      p,
			cloneList: tc.cloneList,
		}

		poolEntry, err := buildPoolEntry(testFSManager)
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedEntry, poolEntry)
	}
}

func TestParsingDockerImage(t *testing.T) {
	t.Run("Parse PostgreSQL version from tags of a Docker image", func(t *testing.T) {
		testCases := []struct {
			image           string
			expectedVersion string
		}{
			{
				image:           "postgresai/extended-postgres:11",
				expectedVersion: "11",
			},
			{
				image:           "postgresai/extended-postgres:11-alpine",
				expectedVersion: "11",
			},
			{
				image:           "postgresai/extended-postgres:alpine",
				expectedVersion: "",
			},
			{
				image:           "internal.example.com:5000/pg:9.6-ext",
				expectedVersion: "9.6",
			},
		}

		for _, tc := range testCases {
			version := parseImageVersion(tc.image)
			assert.Equal(t, tc.expectedVersion, version)
		}
	})
}

func TestLatestSnapshot(t *testing.T) {
	t.Run("Test selecting the latest snapshot ID", func(t *testing.T) {
		dateTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

		testCases := []struct {
			snapshots  []resources.Snapshot
			expectedID string
			err        error
		}{
			{
				err: errors.New("no snapshots available"),
			},
			{
				snapshots: []resources.Snapshot{
					{ID: "test1", CreatedAt: dateTime},
					{ID: "test2", CreatedAt: dateTime.Add(time.Hour)},
					{ID: "test3", CreatedAt: dateTime.Add(-time.Hour)},
				},
				expectedID: "test2",
			},
			{
				snapshots: []resources.Snapshot{
					{ID: "test1", DataStateAt: dateTime},
					{ID: "test2", DataStateAt: dateTime.Add(time.Hour)},
					{ID: "test3", DataStateAt: dateTime.Add(2 * time.Hour)},
				},
				expectedID: "test3",
			},
			{
				snapshots: []resources.Snapshot{
					{ID: "test1", CreatedAt: dateTime, DataStateAt: dateTime},
					{ID: "test2", CreatedAt: dateTime.Add(time.Hour), DataStateAt: dateTime.Add(time.Hour)},
					{ID: "test3", CreatedAt: dateTime.Add(-time.Hour), DataStateAt: dateTime.Add(2 * time.Hour)},
				},
				expectedID: "test3",
			},
		}

		for _, tc := range testCases {
			latest, err := getLatestSnapshot(tc.snapshots)
			if err != nil {
				assert.EqualError(t, err, tc.err.Error())
				continue
			}

			assert.Equal(t, tc.expectedID, latest.ID)
		}
	})
}

func TestDetectLogsTimeZone(t *testing.T) {
	tempDir := path.Join(os.TempDir(), "dle_logs_tz")
	defer os.RemoveAll(tempDir)

	const (
		layout         = "2006-01-02 15:04:05.000 MST"
		datetime       = "2023-04-28 12:50:10.779 CEST"
		emptyContent   = `# PostgreSQL configuration file`
		invalidContent = `# PostgreSQL configuration file
# TimeZone setting
log_timezone = 'America/Stockholm'
`
		validContent = `# PostgreSQL configuration file
# TimeZone setting
log_timezone = 'Europe/Stockholm'
`
	)

	tests := []struct {
		name        string
		dataDir     string
		fileName    string
		content     string
		expectedLoc *time.Location
	}{
		{
			name:        "no config file",
			dataDir:     "/path/to/missing/config",
			fileName:    "missing_config",
			expectedLoc: time.UTC,
		},
		{
			name:        "config file without timezone",
			dataDir:     "empty_config",
			fileName:    "postgresql.dblab.user_defined.conf",
			content:     emptyContent,
			expectedLoc: time.UTC,
		},
		{
			name:        "config file with invalid timezone",
			dataDir:     "invalid_dir",
			fileName:    "postgresql.dblab.user_defined.conf",
			content:     invalidContent,
			expectedLoc: time.UTC,
		},
		{
			name:        "config file with valid timezone",
			dataDir:     "valid_dir",
			fileName:    "postgresql.dblab.user_defined.conf",
			content:     validContent,
			expectedLoc: time.FixedZone("CEST", 2*60*60), // CEST (+2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCaseDir := path.Join(tempDir, tt.dataDir)
			err := createTempConfigFile(testCaseDir, tt.fileName, tt.content)
			require.NoError(t, err)

			loc := detectLogsTimeZone(testCaseDir)

			expectedTime, err := time.ParseInLocation(layout, datetime, tt.expectedLoc)
			require.NoError(t, err)

			locationTime, err := time.ParseInLocation(layout, datetime, loc)
			require.NoError(t, err)

			require.Truef(t, locationTime.UTC().Equal(expectedTime.UTC()), "detectLogsTimeZone(%s) returned unexpected location time. Expected %s, but got %s.", tt.dataDir, expectedTime, locationTime)
		})
	}
}

func createTempConfigFile(testCaseDir, fileName string, content string) error {
	err := os.MkdirAll(testCaseDir, 0777)
	if err != nil {
		return err
	}

	fn := path.Join(testCaseDir, fileName)

	return os.WriteFile(fn, []byte(content), 0666)
}

func TestProvisionHosts(t *testing.T) {
	tests := []struct {
		name          string
		udAddresses   string
		gateway       string
		expectedHosts string
	}{
		{
			name:          "Empty fields",
			udAddresses:   "",
			gateway:       "",
			expectedHosts: "",
		},
		{
			name:          "Empty user-defined address",
			udAddresses:   "",
			gateway:       "172.20.0.1",
			expectedHosts: "",
		},
		{
			name:          "Wildcard IP",
			udAddresses:   "0.0.0.0",
			gateway:       "172.20.0.1",
			expectedHosts: "0.0.0.0",
		},
		{
			name:          "User-defined address",
			udAddresses:   "192.168.1.1",
			gateway:       "172.20.0.1",
			expectedHosts: "172.20.0.1,192.168.1.1",
		},
		{
			name:          "Multiple user-defined addresses",
			udAddresses:   "192.168.1.1,10.0.58.1",
			gateway:       "172.20.0.1",
			expectedHosts: "172.20.0.1,192.168.1.1,10.0.58.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			p := Provisioner{
				config: &Config{
					CloneAccessAddresses: tt.udAddresses,
				},
				gateway: tt.gateway,
			}

			assert.Equal(t, tt.expectedHosts, p.getProvisionHosts())
		})
	}
}
