/*
2025 © Postgres.ai
*/

package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

type mockStatusProvider struct {
	status *models.InstanceStatus
	uptime float64
}

func (m *mockStatusProvider) InstanceStatus() *models.InstanceStatus {
	return m.status
}

func (m *mockStatusProvider) Uptime() float64 {
	return m.uptime
}

func TestNewCollector(t *testing.T) {
	provider := &mockStatusProvider{}
	collector := NewCollector(provider)
	assert.NotNil(t, collector)
}

func TestCollector_Describe(t *testing.T) {
	provider := &mockStatusProvider{}
	collector := NewCollector(provider)

	ch := make(chan *prometheus.Desc, 100)
	collector.Describe(ch)
	close(ch)

	var descriptions []*prometheus.Desc
	for desc := range ch {
		descriptions = append(descriptions, desc)
	}

	assert.Greater(t, len(descriptions), 0, "should have at least one metric description")
}

func TestCollector_Collect_NilStatus(t *testing.T) {
	provider := &mockStatusProvider{status: nil}
	collector := NewCollector(provider)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}

	assert.Empty(t, metrics, "should not emit metrics when status is nil")
}

func TestCollector_Collect_BasicMetrics(t *testing.T) {
	now := time.Now()
	provider := &mockStatusProvider{
		uptime: 3600,
		status: &models.InstanceStatus{
			Status: &models.Status{Code: models.StatusOK, Message: "ok"},
			Engine: models.Engine{
				Version:    "3.0.0",
				Edition:    "standard",
				InstanceID: "test-instance",
			},
			Pools: []models.PoolEntry{
				{
					Name:        "dblab_pool",
					Mode:        "zfs",
					DataStateAt: models.NewLocalTime(now),
					Status:      resources.ActivePool,
					CloneList:   []string{"clone1", "clone2"},
					FileSystem: models.FileSystem{
						Mode:            "zfs",
						Size:            100 * 1024 * 1024 * 1024,
						Free:            50 * 1024 * 1024 * 1024,
						Used:            50 * 1024 * 1024 * 1024,
						DataSize:        30 * 1024 * 1024 * 1024,
						UsedBySnapshots: 10 * 1024 * 1024 * 1024,
						UsedByClones:    10 * 1024 * 1024 * 1024,
						CompressRatio:   2.5,
					},
				},
			},
			Cloning: models.Cloning{
				ExpectedCloningTime: 5.5,
				NumClones:           2,
				Clones: []*models.Clone{
					{
						ID:        "clone1",
						Branch:    "main",
						Status:    models.Status{Code: models.StatusOK},
						Protected: false,
						Metadata: models.CloneMetadata{
							CloneDiffSize: 1024,
							LogicalSize:   1024 * 1024,
							CloningTime:   2.5,
						},
					},
					{
						ID:        "clone2",
						Branch:    "feature",
						Status:    models.Status{Code: models.StatusOK},
						Protected: true,
						Metadata: models.CloneMetadata{
							CloneDiffSize: 2048,
							LogicalSize:   2 * 1024 * 1024,
							CloningTime:   3.0,
						},
					},
				},
			},
			Retrieving: models.Retrieving{
				Mode:        models.Physical,
				Status:      models.Renewed,
				LastRefresh: models.NewLocalTime(now.Add(-time.Hour)),
				NextRefresh: models.NewLocalTime(now.Add(time.Hour)),
			},
			Synchronization: &models.Sync{
				Status:            models.Status{Code: models.StatusOK},
				ReplicationLag:    5,
				ReplicationUptime: 86400,
			},
		},
	}

	collector := NewCollector(provider)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	require.NoError(t, err)
	assert.Greater(t, len(metrics), 0)

	metricNames := make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricNames = append(metricNames, m.GetName())
	}

	expectedMetrics := []string{
		"dblab_engine_info",
		"dblab_engine_uptime_seconds",
		"dblab_retrieval_mode",
		"dblab_sync_replication_lag_seconds",
		"dblab_pool_size_bytes",
		"dblab_clones_total",
	}

	for _, expected := range expectedMetrics {
		found := false
		for _, name := range metricNames {
			if name == expected {
				found = true
				break
			}
		}

		assert.True(t, found, "expected metric %s not found", expected)
	}
}

func TestCollector_EngineInfo(t *testing.T) {
	provider := &mockStatusProvider{
		uptime: 1234.5,
		status: &models.InstanceStatus{
			Status: &models.Status{Code: models.StatusOK},
			Engine: models.Engine{
				Version:    "3.1.0",
				Edition:    "enterprise",
				InstanceID: "my-instance-id",
			},
			Pools:      []models.PoolEntry{},
			Cloning:    models.Cloning{},
			Retrieving: models.Retrieving{Mode: models.Unknown, Status: models.Inactive},
		},
	}

	collector := NewCollector(provider)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	expected := `
		# HELP dblab_engine_uptime_seconds Time since Database Lab Engine started in seconds
		# TYPE dblab_engine_uptime_seconds gauge
		dblab_engine_uptime_seconds 1234.5
	`

	err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "dblab_engine_uptime_seconds")
	assert.NoError(t, err)
}

func TestCollector_ReplicationLag(t *testing.T) {
	provider := &mockStatusProvider{
		uptime: 100,
		status: &models.InstanceStatus{
			Status:     &models.Status{Code: models.StatusOK},
			Engine:     models.Engine{Version: "3.0.0", Edition: "standard", InstanceID: "test"},
			Pools:      []models.PoolEntry{},
			Cloning:    models.Cloning{},
			Retrieving: models.Retrieving{Mode: models.Physical, Status: models.Renewed},
			Synchronization: &models.Sync{
				ReplicationLag:    10,
				ReplicationUptime: 3600,
			},
		},
	}

	collector := NewCollector(provider)

	expected := `
		# HELP dblab_sync_replication_lag_seconds Replication lag in seconds (physical mode)
		# TYPE dblab_sync_replication_lag_seconds gauge
		dblab_sync_replication_lag_seconds 10
	`

	err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "dblab_sync_replication_lag_seconds")
	assert.NoError(t, err)
}

func TestCollector_ClonesTotal(t *testing.T) {
	provider := &mockStatusProvider{
		uptime: 100,
		status: &models.InstanceStatus{
			Status: &models.Status{Code: models.StatusOK},
			Engine: models.Engine{Version: "3.0.0", Edition: "standard", InstanceID: "test"},
			Pools:  []models.PoolEntry{},
			Cloning: models.Cloning{
				NumClones: 5,
				Clones:    []*models.Clone{},
			},
			Retrieving: models.Retrieving{Mode: models.Logical, Status: models.Renewed},
		},
	}

	collector := NewCollector(provider)

	expected := `
		# HELP dblab_clones_total Total number of clones
		# TYPE dblab_clones_total gauge
		dblab_clones_total 5
	`

	err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "dblab_clones_total")
	assert.NoError(t, err)
}

func TestCollector_PoolMetrics(t *testing.T) {
	provider := &mockStatusProvider{
		uptime: 100,
		status: &models.InstanceStatus{
			Status: &models.Status{Code: models.StatusOK},
			Engine: models.Engine{Version: "3.0.0", Edition: "standard", InstanceID: "test"},
			Pools: []models.PoolEntry{
				{
					Name:   "testpool",
					Mode:   "zfs",
					Status: resources.ActivePool,
					FileSystem: models.FileSystem{
						Size:            1000,
						Free:            600,
						Used:            400,
						CompressRatio:   3.0,
						UsedBySnapshots: 100,
						UsedByClones:    50,
					},
				},
			},
			Cloning:    models.Cloning{},
			Retrieving: models.Retrieving{Mode: models.Logical, Status: models.Renewed},
		},
	}

	collector := NewCollector(provider)

	expected := `
		# HELP dblab_pool_compress_ratio Compression ratio of the pool
		# TYPE dblab_pool_compress_ratio gauge
		dblab_pool_compress_ratio{pool="testpool"} 3
	`

	err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "dblab_pool_compress_ratio")
	assert.NoError(t, err)
}

func TestBranchCount(t *testing.T) {
	tests := []struct {
		name   string
		clones []*models.Clone
		want   int
	}{
		{
			name:   "empty clones",
			clones: []*models.Clone{},
			want:   0,
		},
		{
			name: "single branch",
			clones: []*models.Clone{
				{Branch: "main"},
				{Branch: "main"},
			},
			want: 1,
		},
		{
			name: "multiple branches",
			clones: []*models.Clone{
				{Branch: "main"},
				{Branch: "feature1"},
				{Branch: "feature2"},
			},
			want: 3,
		},
		{
			name: "nil clone in list",
			clones: []*models.Clone{
				{Branch: "main"},
				nil,
				{Branch: "feature"},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchCount(tt.clones)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestActiveCloneCountByStatus(t *testing.T) {
	tests := []struct {
		name   string
		clones []*models.Clone
		want   map[string]int
	}{
		{
			name:   "empty clones",
			clones: []*models.Clone{},
			want:   map[string]int{},
		},
		{
			name: "various statuses",
			clones: []*models.Clone{
				{Status: models.Status{Code: models.StatusOK}},
				{Status: models.Status{Code: models.StatusOK}},
				{Status: models.Status{Code: models.StatusCreating}},
				{Status: models.Status{Code: models.StatusDeleting}},
			},
			want: map[string]int{
				"OK":       2,
				"CREATING": 1,
				"DELETING": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActiveCloneCountByStatus(tt.clones)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPoolDataFreshness(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		pool models.PoolEntry
	}{
		{
			name: "nil data state",
			pool: models.PoolEntry{DataStateAt: nil},
		},
		{
			name: "recent data state",
			pool: models.PoolEntry{DataStateAt: models.NewLocalTime(now.Add(-time.Minute))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PoolDataFreshness(tt.pool)
			assert.GreaterOrEqual(t, result, float64(0))
		})
	}
}

func TestDiskUsagePercent(t *testing.T) {
	tests := []struct {
		name string
		fs   models.FileSystem
		want float64
	}{
		{
			name: "zero size",
			fs:   models.FileSystem{Size: 0, Used: 0},
			want: 0,
		},
		{
			name: "50 percent used",
			fs:   models.FileSystem{Size: 100, Used: 50},
			want: 50,
		},
		{
			name: "full disk",
			fs:   models.FileSystem{Size: 100, Used: 100},
			want: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiskUsagePercent(tt.fs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBoolToFloat(t *testing.T) {
	assert.Equal(t, float64(1), BoolToFloat(true))
	assert.Equal(t, float64(0), BoolToFloat(false))
}

func TestStatusCodeToFloat(t *testing.T) {
	tests := []struct {
		code string
		want float64
	}{
		{"OK", 1},
		{"CREATING", 2},
		{"DELETING", 3},
		{"RESETTING", 4},
		{"EXPORTING", 5},
		{"FATAL", 6},
		{"UNKNOWN", 0},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := StatusCodeToFloat(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBranch(t *testing.T) {
	assert.Equal(t, "main", FormatBranch(""))
	assert.Equal(t, "feature", FormatBranch("feature"))
	assert.Equal(t, "main", FormatBranch("main"))
}

func TestParseReplicationLag(t *testing.T) {
	assert.Equal(t, "0", ParseReplicationLag(nil))
	assert.Equal(t, "10", ParseReplicationLag(&models.Sync{ReplicationLag: 10}))
}

func TestNumClonesFromStatus(t *testing.T) {
	assert.Equal(t, uint64(0), NumClonesFromStatus(nil))
	assert.Equal(t, uint64(5), NumClonesFromStatus(&models.InstanceStatus{
		Cloning: models.Cloning{NumClones: 5},
	}))
}

func TestReplicationLagFromSync(t *testing.T) {
	assert.Equal(t, 0, ReplicationLagFromSync(nil))
	assert.Equal(t, 15, ReplicationLagFromSync(&models.Sync{ReplicationLag: 15}))
}
