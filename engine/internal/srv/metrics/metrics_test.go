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
	assert.NotNil(t, m.InstanceStatus)
	assert.NotNil(t, m.RetrievalStatus)
	assert.NotNil(t, m.DiskTotal)
	assert.NotNil(t, m.DiskFree)
	assert.NotNil(t, m.DiskUsed)
	assert.NotNil(t, m.ClonesTotal)
	assert.NotNil(t, m.CloneInfo)
	assert.NotNil(t, m.CloneAgeSeconds)
	assert.NotNil(t, m.CloneMaxAgeSeconds)
	assert.NotNil(t, m.SnapshotsTotal)
	assert.NotNil(t, m.SnapshotInfo)
	assert.NotNil(t, m.BranchesTotal)
	assert.NotNil(t, m.BranchInfo)
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

func TestMetricsReset(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)

	m.InstanceInfo.WithLabelValues("test-id", "v1.0", "CE").Set(1)
	m.DiskTotal.WithLabelValues("pool1").Set(1000)
	m.CloneInfo.WithLabelValues("clone1", "main", "snap1", "pool1", "OK", "false").Set(1)

	m.Reset()
}

func TestMetricsSetValues(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	require.NoError(t, err)

	m.InstanceInfo.WithLabelValues("instance-123", "v4.0.0", "CE").Set(1)
	m.InstanceUptime.Set(3600)
	m.InstanceStatus.WithLabelValues("OK").Set(1)

	m.DiskTotal.WithLabelValues("dblab_pool").Set(1000000000)
	m.DiskFree.WithLabelValues("dblab_pool").Set(500000000)
	m.DiskUsed.WithLabelValues("dblab_pool").Set(500000000)

	m.ClonesTotal.Set(5)
	m.CloneInfo.WithLabelValues("clone-1", "main", "snap-1", "pool1", "OK", "false").Set(1)
	m.CloneAgeSeconds.WithLabelValues("clone-1").Set(120)
	m.CloneMaxAgeSeconds.Set(3600)
	m.CloneDiffSize.WithLabelValues("clone-1").Set(1024)
	m.CloneCPUUsage.WithLabelValues("clone-1").Set(15.5)
	m.CloneMemoryUsage.WithLabelValues("clone-1").Set(256000000)

	m.SnapshotsTotal.Set(10)
	m.SnapshotInfo.WithLabelValues("snap-1", "pool1", "main").Set(1)
	m.SnapshotAgeSeconds.WithLabelValues("snap-1").Set(7200)
	m.SnapshotMaxAgeSeconds.Set(86400)
	m.SnapshotPhysicalSize.WithLabelValues("snap-1").Set(2000000000)
	m.SnapshotDataLag.WithLabelValues("snap-1").Set(300)
	m.SnapshotMaxDataLag.Set(600)

	m.BranchesTotal.Set(3)
	m.BranchInfo.WithLabelValues("main", "pool1", "snap-1").Set(1)

	m.DatasetsTotal.WithLabelValues("pool1").Set(20)
	m.DatasetsAvailable.WithLabelValues("pool1").Set(15)
}
