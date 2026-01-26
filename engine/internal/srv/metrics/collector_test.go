/*
2025 © Postgres.ai
*/

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

type mockCloningService struct{}

func (m *mockCloningService) GetClones() []*models.Clone { return nil }

func (m *mockCloningService) GetSnapshots() ([]models.Snapshot, error) { return nil, nil }

type mockRetrievalService struct{}

func (m *mockRetrievalService) GetRetrievalMode() models.RetrievalMode { return models.Physical }

func (m *mockRetrievalService) GetRetrievalStatus() models.RetrievalStatus { return models.Inactive }

type mockPoolService struct{}

func (m *mockPoolService) GetFSManagerList() []pool.FSManager { return nil }

func newTestCollector(m *Metrics) *Collector {
	return &Collector{
		metrics:      m,
		prevCPUStats: make(map[string]containerCPUState),
	}
}

func TestNewCollector(t *testing.T) {
	m := NewMetrics()
	engProps := &global.EngineProps{InstanceID: "test-instance"}
	startedAt := time.Now()

	c, err := NewCollector(m, &mockCloningService{}, &mockRetrievalService{}, &mockPoolService{}, engProps, nil, startedAt)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.metrics)
	assert.NotNil(t, c.prevCPUStats)
	assert.Equal(t, engProps, c.engProps)
	assert.Equal(t, startedAt, c.startedAt)
}

func TestNewCollector_ValidationErrors(t *testing.T) {
	m := NewMetrics()
	cloning := &mockCloningService{}
	retrieval := &mockRetrievalService{}
	pm := &mockPoolService{}
	engProps := &global.EngineProps{InstanceID: "test-instance"}
	startedAt := time.Now()

	tests := []struct {
		name      string
		metrics   *Metrics
		cloning   CloningService
		retrieval RetrievalService
		pm        PoolService
		engProps  *global.EngineProps
		wantErr   string
	}{
		{name: "nil metrics", metrics: nil, cloning: cloning, retrieval: retrieval, pm: pm, engProps: engProps, wantErr: "metrics is required"},
		{name: "nil cloning", metrics: m, cloning: nil, retrieval: retrieval, pm: pm, engProps: engProps, wantErr: "cloning is required"},
		{name: "nil retrieval", metrics: m, cloning: cloning, retrieval: nil, pm: pm, engProps: engProps, wantErr: "retrieval is required"},
		{name: "nil pool manager", metrics: m, cloning: cloning, retrieval: retrieval, pm: nil, engProps: engProps, wantErr: "pool manager is required"},
		{name: "nil engine props", metrics: m, cloning: cloning, retrieval: retrieval, pm: pm, engProps: nil, wantErr: "engine props is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCollector(tt.metrics, tt.cloning, tt.retrieval, tt.pm, tt.engProps, nil, startedAt)
			assert.Nil(t, c)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCalculateCPUPercent_NoPreviousStats(t *testing.T) {
	c := newTestCollector(NewMetrics())

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1000000000},
			SystemUsage: 5000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Equal(t, cpuNoData, result)

	_, exists := c.prevCPUStats["clone-1"]
	assert.True(t, exists)
}

func TestCalculateCPUPercent_WithPreviousStats(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{
		totalUsage:  1000000000,
		systemUsage: 5000000000,
		timestamp:   time.Now().Add(-2 * time.Second),
	}

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2000000000},
			SystemUsage: 10000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Greater(t, result, float64(0))
}

func TestCalculateCPUPercent_RapidScrape(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{
		totalUsage:  1000000000,
		systemUsage: 5000000000,
		timestamp:   time.Now().Add(-100 * time.Millisecond),
	}

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2000000000},
			SystemUsage: 10000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Equal(t, cpuNoData, result)
}

func TestCalculateCPUPercent_CounterRollover(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{
		totalUsage:  5000000000,
		systemUsage: 10000000000,
		timestamp:   time.Now().Add(-2 * time.Second),
	}

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1000000000},
			SystemUsage: 2000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Equal(t, float64(0), result)
}

func TestCalculateCPUPercent_ZeroSystemDelta(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{
		totalUsage:  1000000000,
		systemUsage: 5000000000,
		timestamp:   time.Now().Add(-2 * time.Second),
	}

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2000000000},
			SystemUsage: 5000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Equal(t, float64(0), result)
}

func TestCalculateCPUPercent_FallbackCPUCount(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{
		totalUsage:  1000000000,
		systemUsage: 5000000000,
		timestamp:   time.Now().Add(-2 * time.Second),
	}

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage:  2000000000,
				PercpuUsage: []uint64{500000000, 500000000, 500000000, 500000000},
			},
			SystemUsage: 10000000000,
			OnlineCPUs:  0,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Greater(t, result, float64(0))
}

func TestCleanupStaleCPUStats(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{totalUsage: 100}
	c.prevCPUStats["clone-2"] = containerCPUState{totalUsage: 200}
	c.prevCPUStats["clone-3"] = containerCPUState{totalUsage: 300}

	activeCloneIDs := map[string]struct{}{"clone-1": {}, "clone-3": {}}

	c.cleanupStaleCPUStats(activeCloneIDs)

	assert.Len(t, c.prevCPUStats, 2)
	_, exists1 := c.prevCPUStats["clone-1"]
	_, exists2 := c.prevCPUStats["clone-2"]
	_, exists3 := c.prevCPUStats["clone-3"]

	assert.True(t, exists1)
	assert.False(t, exists2)
	assert.True(t, exists3)
}

func TestCleanupStaleCPUStats_EmptyActive(t *testing.T) {
	c := newTestCollector(NewMetrics())

	c.prevCPUStats["clone-1"] = containerCPUState{totalUsage: 100}
	c.prevCPUStats["clone-2"] = containerCPUState{totalUsage: 200}

	c.cleanupStaleCPUStats(map[string]struct{}{})

	assert.Len(t, c.prevCPUStats, 0)
}

func TestCollectAndServe_Concurrency(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	err := m.Register(reg)
	require.NoError(t, err)

	engProps := &global.EngineProps{InstanceID: "test-instance"}

	c, err := NewCollector(m, &mockCloningService{}, &mockRetrievalService{}, &mockPoolService{}, engProps, nil, time.Now())
	require.NoError(t, err)

	c.Collect(context.Background())

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			rec := httptest.NewRecorder()

			c.CollectAndServe(handler, rec, req)

			if rec.Code != http.StatusOK {
				errors <- fmt.Errorf("unexpected status code: %d", rec.Code)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent request failed: %v", err)
	}
}

func TestGetContainerStats_NilDockerClient(t *testing.T) {
	c := newTestCollector(NewMetrics())

	result := c.getContainerStats(context.Background(), nil)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestFilterActiveClones(t *testing.T) {
	tests := []struct {
		name     string
		clones   []*models.Clone
		expected int
	}{
		{
			name:     "empty list",
			clones:   []*models.Clone{},
			expected: 0,
		},
		{
			name:     "nil list",
			clones:   nil,
			expected: 0,
		},
		{
			name: "all active",
			clones: []*models.Clone{
				{ID: "clone-1", Status: models.Status{Code: models.StatusOK}},
				{ID: "clone-2", Status: models.Status{Code: models.StatusOK}},
			},
			expected: 2,
		},
		{
			name: "mixed statuses",
			clones: []*models.Clone{
				{ID: "clone-1", Status: models.Status{Code: models.StatusOK}},
				{ID: "clone-2", Status: models.Status{Code: models.StatusCreating}},
				{ID: "clone-3", Status: models.Status{Code: models.StatusOK}},
				nil,
			},
			expected: 2,
		},
		{
			name: "none active",
			clones: []*models.Clone{
				{ID: "clone-1", Status: models.Status{Code: models.StatusCreating}},
				{ID: "clone-2", Status: models.Status{Code: models.StatusDeleting}},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterActiveClones(tt.clones)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestStartBackgroundCollection(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	err := m.Register(reg)
	require.NoError(t, err)

	engProps := &global.EngineProps{InstanceID: "test-instance"}

	c, err := NewCollector(m, &mockCloningService{}, &mockRetrievalService{}, &mockPoolService{}, engProps, nil, time.Now())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.StartBackgroundCollection(ctx, 50*time.Millisecond)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("background collection did not stop")
	}
}

func TestResetDynamic(t *testing.T) {
	m := NewMetrics()

	m.InstanceInfo.WithLabelValues("id", "v1", "ce").Set(1)
	m.ClonesByStatus.WithLabelValues("OK").Set(5)
	m.SnapshotsByPool.WithLabelValues("pool1").Set(3)

	m.ResetDynamic()

	ch := make(chan prometheus.Metric, 10)

	m.InstanceInfo.Collect(ch)

	select {
	case <-ch:
		t.Error("expected InstanceInfo to be reset")
	default:
	}

	m.ClonesByStatus.Collect(ch)

	select {
	case <-ch:
		t.Error("expected ClonesByStatus to be reset")
	default:
	}
}

func TestCalculateCPUPercent_Concurrent(t *testing.T) {
	c := newTestCollector(NewMetrics())

	var wg sync.WaitGroup
	cloneCount := 100

	for i := 0; i < cloneCount; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			cloneID := fmt.Sprintf("clone-%d", id)
			stats := &container.StatsResponse{
				CPUStats: container.CPUStats{
					CPUUsage:    container.CPUUsage{TotalUsage: uint64(1000000000 + id*1000)},
					SystemUsage: uint64(5000000000 + id*1000),
					OnlineCPUs:  4,
				},
			}

			c.calculateCPUPercent(cloneID, stats)
			c.calculateCPUPercent(cloneID, stats)
		}(i)
	}

	wg.Wait()

	assert.Len(t, c.prevCPUStats, cloneCount)
}
