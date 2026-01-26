/*
2025 © Postgres.ai
*/

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	require.NotNil(t, m)

	assert.NotNil(t, m.InstanceInfo)
	assert.NotNil(t, m.InstanceUptime)
	assert.NotNil(t, m.InstanceStatusCode)
	assert.NotNil(t, m.RetrievalStatus)
	assert.NotNil(t, m.DiskTotal)
	assert.NotNil(t, m.DiskFree)
	assert.NotNil(t, m.DiskUsed)
	assert.NotNil(t, m.ClonesTotal)
	assert.NotNil(t, m.ClonesByStatus)
	assert.NotNil(t, m.CloneMaxAgeSeconds)
	assert.NotNil(t, m.CloneTotalDiffSize)
	assert.NotNil(t, m.CloneTotalCPUUsage)
	assert.NotNil(t, m.CloneAvgCPUUsage)
	assert.NotNil(t, m.SnapshotsTotal)
	assert.NotNil(t, m.SnapshotsByPool)
	assert.NotNil(t, m.SnapshotMaxAgeSeconds)
	assert.NotNil(t, m.BranchesTotal)
	assert.NotNil(t, m.ScrapeSuccessTimestamp)
	assert.NotNil(t, m.ScrapeDurationSeconds)
	assert.NotNil(t, m.ScrapeErrorsTotal)
}

func TestMetricsRegister(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)
}

func TestMetricsRegisterDuplicate(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)

	err = m.Register(reg)
	assert.Error(t, err)
}

func TestMetricsResetDynamic(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)

	m.InstanceInfo.WithLabelValues("test-id", "v1.0", "CE").Set(1)
	m.DiskTotal.WithLabelValues("pool1").Set(1000)
	m.ClonesByStatus.WithLabelValues("OK").Set(5)

	m.ResetDynamic()
}

func TestMetricsSetValues(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)

	m.InstanceInfo.WithLabelValues("instance-123", "v4.0.0", "CE").Set(1)
	m.InstanceUptime.Set(3600)
	m.InstanceStatusCode.Set(StatusCodeOK)

	m.DiskTotal.WithLabelValues("dblab_pool").Set(1000000000)
	m.DiskFree.WithLabelValues("dblab_pool").Set(500000000)
	m.DiskUsed.WithLabelValues("dblab_pool").Set(500000000)

	m.ClonesTotal.Set(5)
	m.ClonesByStatus.WithLabelValues("OK").Set(3)
	m.ClonesByStatus.WithLabelValues("Creating").Set(2)
	m.CloneMaxAgeSeconds.Set(3600)
	m.CloneTotalDiffSize.Set(1024000)
	m.CloneTotalLogicalSize.Set(5000000)
	m.CloneTotalCPUUsage.Set(45.5)
	m.CloneAvgCPUUsage.Set(15.2)
	m.CloneTotalMemoryUsage.Set(768000000)
	m.CloneTotalMemoryLimit.Set(1024000000)
	m.CloneProtectedCount.Set(2)

	m.SnapshotsTotal.Set(10)
	m.SnapshotsByPool.WithLabelValues("pool1").Set(10)
	m.SnapshotMaxAgeSeconds.Set(86400)
	m.SnapshotTotalPhysicalSize.Set(20000000000)
	m.SnapshotTotalLogicalSize.Set(50000000000)
	m.SnapshotMaxDataLag.Set(600)
	m.SnapshotTotalNumClones.Set(5)

	m.BranchesTotal.Set(3)

	m.DatasetsTotal.WithLabelValues("pool1").Set(20)
	m.DatasetsAvailable.WithLabelValues("pool1").Set(15)

	m.ScrapeSuccessTimestamp.Set(1700000000)
	m.ScrapeDurationSeconds.Set(0.5)
	m.ScrapeErrorsTotal.Add(0)
}

func TestStatusCodes(t *testing.T) {
	assert.Equal(t, 0, StatusCodeOK)
	assert.Equal(t, 1, StatusCodeWarning)
	assert.Equal(t, 2, StatusCodeBad)
}
