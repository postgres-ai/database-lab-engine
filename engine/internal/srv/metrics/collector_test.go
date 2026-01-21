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

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
)

func TestNewCollector(t *testing.T) {
	m := NewMetrics()
	engProps := &global.EngineProps{InstanceID: "test-instance"}
	startedAt := time.Now()

	c := NewCollector(m, nil, nil, nil, engProps, nil, startedAt)

	require.NotNil(t, c)
	assert.NotNil(t, c.metrics)
	assert.NotNil(t, c.prevCPUStats)
	assert.Equal(t, engProps, c.engProps)
	assert.Equal(t, startedAt, c.startedAt)
}

func TestCalculateCPUPercent_NoPreviousStats(t *testing.T) {
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

	stats := &container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1000000000},
			SystemUsage: 5000000000,
			OnlineCPUs:  4,
		},
	}

	result := c.calculateCPUPercent("clone-1", stats)
	assert.Equal(t, float64(0), result)

	_, exists := c.prevCPUStats["clone-1"]
	assert.True(t, exists)
}

func TestCalculateCPUPercent_WithPreviousStats(t *testing.T) {
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	assert.Equal(t, float64(0), result)
}

func TestCalculateCPUPercent_CounterRollover(t *testing.T) {
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

	c.prevCPUStats["clone-1"] = containerCPUState{totalUsage: 100}
	c.prevCPUStats["clone-2"] = containerCPUState{totalUsage: 200}
	c.prevCPUStats["clone-3"] = containerCPUState{totalUsage: 300}

	activeCloneIDs := map[string]struct{}{
		"clone-1": {},
		"clone-3": {},
	}

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

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
	c := NewCollector(m, nil, nil, nil, engProps, nil, time.Now())

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			rec := httptest.NewRecorder()

			c.CollectAndServe(context.Background(), handler, rec, req)

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
	m := NewMetrics()
	c := NewCollector(m, nil, nil, nil, &global.EngineProps{}, nil, time.Now())

	result := c.getContainerStats(context.Background(), nil)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}
