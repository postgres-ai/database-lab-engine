/*
2025 © Postgres.ai
*/

// Package metrics provides Prometheus metrics for the Database Lab Engine.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "dblab"
)

// Metrics contains all Prometheus metrics for DBLab.
type Metrics struct {
	// instance metrics
	InstanceInfo    *prometheus.GaugeVec
	InstanceUptime  prometheus.Gauge
	InstanceStatus  *prometheus.GaugeVec
	RetrievalStatus *prometheus.GaugeVec

	// pool/filesystem metrics
	DiskTotal           *prometheus.GaugeVec
	DiskFree            *prometheus.GaugeVec
	DiskUsed            *prometheus.GaugeVec
	DiskUsedBySnapshots *prometheus.GaugeVec
	DiskUsedByClones    *prometheus.GaugeVec
	DiskDataSize        *prometheus.GaugeVec
	DiskCompressRatio   *prometheus.GaugeVec
	PoolStatus          *prometheus.GaugeVec

	// clone metrics
	ClonesTotal        prometheus.Gauge
	CloneInfo          *prometheus.GaugeVec
	CloneAgeSeconds    *prometheus.GaugeVec
	CloneMaxAgeSeconds prometheus.Gauge
	CloneDiffSize      *prometheus.GaugeVec
	CloneLogicalSize   *prometheus.GaugeVec
	CloneCPUUsage      *prometheus.GaugeVec
	CloneMemoryUsage   *prometheus.GaugeVec
	CloneMemoryLimit   *prometheus.GaugeVec

	// snapshot metrics
	SnapshotsTotal        prometheus.Gauge
	SnapshotInfo          *prometheus.GaugeVec
	SnapshotAgeSeconds    *prometheus.GaugeVec
	SnapshotMaxAgeSeconds prometheus.Gauge
	SnapshotPhysicalSize  *prometheus.GaugeVec
	SnapshotLogicalSize   *prometheus.GaugeVec
	SnapshotDataLag       *prometheus.GaugeVec
	SnapshotMaxDataLag    prometheus.Gauge
	SnapshotNumClones     *prometheus.GaugeVec

	// branch metrics
	BranchesTotal prometheus.Gauge
	BranchInfo    *prometheus.GaugeVec

	// dataset metrics (for logical mode - non-busy slots)
	DatasetsTotal     *prometheus.GaugeVec
	DatasetsAvailable *prometheus.GaugeVec
	DatasetInfo       *prometheus.GaugeVec
}

// NewMetrics creates a new Metrics instance with all Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		// instance metrics
		InstanceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instance_info",
				Help:      "Information about the DBLab instance",
			},
			[]string{"instance_id", "version", "edition"},
		),
		InstanceUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instance_uptime_seconds",
				Help:      "Time in seconds since the DBLab instance started",
			},
		),
		InstanceStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instance_status",
				Help:      "Status of the DBLab instance (1=OK, 0=not OK)",
			},
			[]string{"status_code"},
		),
		RetrievalStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "retrieval_status",
				Help:      "Status of data retrieval (1=active for status)",
			},
			[]string{"mode", "status"},
		),

		// pool/filesystem metrics
		DiskTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_total_bytes",
				Help:      "Total disk space in bytes",
			},
			[]string{"pool"},
		),
		DiskFree: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_free_bytes",
				Help:      "Free disk space in bytes",
			},
			[]string{"pool"},
		),
		DiskUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_used_bytes",
				Help:      "Used disk space in bytes",
			},
			[]string{"pool"},
		),
		DiskUsedBySnapshots: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_used_by_snapshots_bytes",
				Help:      "Disk space used by snapshots in bytes",
			},
			[]string{"pool"},
		),
		DiskUsedByClones: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_used_by_clones_bytes",
				Help:      "Disk space used by clones in bytes",
			},
			[]string{"pool"},
		),
		DiskDataSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_data_size_bytes",
				Help:      "Size of the data directory in bytes",
			},
			[]string{"pool"},
		),
		DiskCompressRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "disk_compress_ratio",
				Help:      "Compression ratio of the filesystem",
			},
			[]string{"pool"},
		),
		PoolStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "pool_status",
				Help:      "Status of the pool (1=active for status)",
			},
			[]string{"pool", "mode", "status"},
		),

		// clone metrics
		ClonesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clones_total",
				Help:      "Total number of clones",
			},
		),
		CloneInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_info",
				Help:      "Information about a clone (always 1)",
			},
			[]string{"clone_id", "branch", "snapshot_id", "pool", "status", "protected"},
		),
		CloneAgeSeconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_age_seconds",
				Help:      "Age of the clone in seconds since creation",
			},
			[]string{"clone_id"},
		),
		CloneMaxAgeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_max_age_seconds",
				Help:      "Maximum age of any clone in seconds",
			},
		),
		CloneDiffSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_diff_size_bytes",
				Help:      "Extra disk space used by the clone (diff from snapshot)",
			},
			[]string{"clone_id"},
		),
		CloneLogicalSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_logical_size_bytes",
				Help:      "Logical size of the clone data",
			},
			[]string{"clone_id"},
		),
		CloneCPUUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_cpu_usage_percent",
				Help:      "CPU usage percentage of the clone container",
			},
			[]string{"clone_id"},
		),
		CloneMemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_memory_usage_bytes",
				Help:      "Memory usage in bytes of the clone container",
			},
			[]string{"clone_id"},
		),
		CloneMemoryLimit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_memory_limit_bytes",
				Help:      "Memory limit in bytes of the clone container",
			},
			[]string{"clone_id"},
		),

		// snapshot metrics
		SnapshotsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshots_total",
				Help:      "Total number of snapshots",
			},
		),
		SnapshotInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_info",
				Help:      "Information about a snapshot (always 1)",
			},
			[]string{"snapshot_id", "pool", "branch"},
		),
		SnapshotAgeSeconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_age_seconds",
				Help:      "Age of the snapshot in seconds since creation",
			},
			[]string{"snapshot_id"},
		),
		SnapshotMaxAgeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_max_age_seconds",
				Help:      "Maximum age of any snapshot in seconds",
			},
		),
		SnapshotPhysicalSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_physical_size_bytes",
				Help:      "Physical disk space used by the snapshot",
			},
			[]string{"snapshot_id"},
		),
		SnapshotLogicalSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_logical_size_bytes",
				Help:      "Logical size of the snapshot data",
			},
			[]string{"snapshot_id"},
		),
		SnapshotDataLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_data_lag_seconds",
				Help:      "Time difference between snapshot data state and now in seconds",
			},
			[]string{"snapshot_id"},
		),
		SnapshotMaxDataLag: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_max_data_lag_seconds",
				Help:      "Maximum data lag of any snapshot in seconds",
			},
		),
		SnapshotNumClones: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_num_clones",
				Help:      "Number of clones using this snapshot",
			},
			[]string{"snapshot_id"},
		),

		// branch metrics
		BranchesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "branches_total",
				Help:      "Total number of branches",
			},
		),
		BranchInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "branch_info",
				Help:      "Information about a branch (always 1)",
			},
			[]string{"branch_name", "pool", "snapshot_id"},
		),

		// dataset metrics
		DatasetsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "datasets_total",
				Help:      "Total number of datasets (slots) in the pool",
			},
			[]string{"pool"},
		),
		DatasetsAvailable: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "datasets_available",
				Help:      "Number of available (non-busy) dataset slots for reuse",
			},
			[]string{"pool"},
		),
		DatasetInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "dataset_info",
				Help:      "Information about a dataset (1=busy, 0=available)",
			},
			[]string{"pool", "dataset_name"},
		),
	}
}

// Register registers all metrics with the Prometheus registry.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.InstanceInfo,
		m.InstanceUptime,
		m.InstanceStatus,
		m.RetrievalStatus,
		m.DiskTotal,
		m.DiskFree,
		m.DiskUsed,
		m.DiskUsedBySnapshots,
		m.DiskUsedByClones,
		m.DiskDataSize,
		m.DiskCompressRatio,
		m.PoolStatus,
		m.ClonesTotal,
		m.CloneInfo,
		m.CloneAgeSeconds,
		m.CloneMaxAgeSeconds,
		m.CloneDiffSize,
		m.CloneLogicalSize,
		m.CloneCPUUsage,
		m.CloneMemoryUsage,
		m.CloneMemoryLimit,
		m.SnapshotsTotal,
		m.SnapshotInfo,
		m.SnapshotAgeSeconds,
		m.SnapshotMaxAgeSeconds,
		m.SnapshotPhysicalSize,
		m.SnapshotLogicalSize,
		m.SnapshotDataLag,
		m.SnapshotMaxDataLag,
		m.SnapshotNumClones,
		m.BranchesTotal,
		m.BranchInfo,
		m.DatasetsTotal,
		m.DatasetsAvailable,
		m.DatasetInfo,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// Reset clears all vector metrics to remove stale data.
func (m *Metrics) Reset() {
	m.InstanceInfo.Reset()
	m.InstanceStatus.Reset()
	m.RetrievalStatus.Reset()
	m.DiskTotal.Reset()
	m.DiskFree.Reset()
	m.DiskUsed.Reset()
	m.DiskUsedBySnapshots.Reset()
	m.DiskUsedByClones.Reset()
	m.DiskDataSize.Reset()
	m.DiskCompressRatio.Reset()
	m.PoolStatus.Reset()
	m.CloneInfo.Reset()
	m.CloneAgeSeconds.Reset()
	m.CloneDiffSize.Reset()
	m.CloneLogicalSize.Reset()
	m.CloneCPUUsage.Reset()
	m.CloneMemoryUsage.Reset()
	m.CloneMemoryLimit.Reset()
	m.SnapshotInfo.Reset()
	m.SnapshotAgeSeconds.Reset()
	m.SnapshotPhysicalSize.Reset()
	m.SnapshotLogicalSize.Reset()
	m.SnapshotDataLag.Reset()
	m.SnapshotNumClones.Reset()
	m.BranchInfo.Reset()
	m.DatasetsTotal.Reset()
	m.DatasetsAvailable.Reset()
	m.DatasetInfo.Reset()
}
