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

// Status codes for numeric status gauges.
const (
	StatusCodeOK      = 0
	StatusCodeWarning = 1
	StatusCodeBad     = 2
)

// Metrics contains all Prometheus metrics for DBLab.
type Metrics struct {
	// instance metrics
	InstanceInfo       *prometheus.GaugeVec
	InstanceUptime     prometheus.Gauge
	InstanceStatusCode prometheus.Gauge
	RetrievalStatus    *prometheus.GaugeVec

	// pool/filesystem metrics
	DiskTotal           *prometheus.GaugeVec
	DiskFree            *prometheus.GaugeVec
	DiskUsed            *prometheus.GaugeVec
	DiskUsedBySnapshots *prometheus.GaugeVec
	DiskUsedByClones    *prometheus.GaugeVec
	DiskDataSize        *prometheus.GaugeVec
	DiskCompressRatio   *prometheus.GaugeVec
	PoolStatus          *prometheus.GaugeVec

	// clone aggregate metrics
	ClonesTotal           prometheus.Gauge
	ClonesByStatus        *prometheus.GaugeVec
	CloneMaxAgeSeconds    prometheus.Gauge
	CloneTotalDiffSize    prometheus.Gauge
	CloneTotalLogicalSize prometheus.Gauge
	CloneTotalCPUUsage    prometheus.Gauge
	CloneTotalMemoryUsage prometheus.Gauge
	CloneTotalMemoryLimit prometheus.Gauge
	CloneAvgCPUUsage      prometheus.Gauge
	CloneProtectedCount   prometheus.Gauge

	// snapshot aggregate metrics
	SnapshotsTotal            prometheus.Gauge
	SnapshotsByPool           *prometheus.GaugeVec
	SnapshotMaxAgeSeconds     prometheus.Gauge
	SnapshotTotalPhysicalSize prometheus.Gauge
	SnapshotTotalLogicalSize  prometheus.Gauge
	SnapshotMaxDataLag        prometheus.Gauge
	SnapshotTotalNumClones    prometheus.Gauge

	// branch metrics
	BranchesTotal prometheus.Gauge

	// dataset metrics (for logical mode - non-busy slots)
	DatasetsTotal     *prometheus.GaugeVec
	DatasetsAvailable *prometheus.GaugeVec

	// observability metrics
	ScrapeSuccessTimestamp prometheus.Gauge
	ScrapeDurationSeconds  prometheus.Gauge
	ScrapeErrorsTotal      prometheus.Counter
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
		InstanceStatusCode: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instance_status_code",
				Help:      "Status code of the DBLab instance (0=OK, 1=Warning, 2=Bad)",
			},
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

		// clone aggregate metrics
		ClonesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clones_total",
				Help:      "Total number of clones",
			},
		),
		ClonesByStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clones_by_status",
				Help:      "Number of clones by status",
			},
			[]string{"status"},
		),
		CloneMaxAgeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_max_age_seconds",
				Help:      "Maximum age of any clone in seconds",
			},
		),
		CloneTotalDiffSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_total_diff_size_bytes",
				Help:      "Total extra disk space used by all clones (sum of diffs from snapshots)",
			},
		),
		CloneTotalLogicalSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_total_logical_size_bytes",
				Help:      "Total logical size of all clone data",
			},
		),
		CloneTotalCPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_total_cpu_usage_percent",
				Help:      "Total CPU usage percentage across all clone containers",
			},
		),
		CloneTotalMemoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_total_memory_usage_bytes",
				Help:      "Total memory usage in bytes across all clone containers",
			},
		),
		CloneTotalMemoryLimit: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_total_memory_limit_bytes",
				Help:      "Total memory limit in bytes across all clone containers",
			},
		),
		CloneAvgCPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_avg_cpu_usage_percent",
				Help:      "Average CPU usage percentage across all clone containers with valid data",
			},
		),
		CloneProtectedCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "clone_protected_count",
				Help:      "Number of protected clones",
			},
		),

		// snapshot aggregate metrics
		SnapshotsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshots_total",
				Help:      "Total number of snapshots",
			},
		),
		SnapshotsByPool: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshots_by_pool",
				Help:      "Number of snapshots by pool",
			},
			[]string{"pool"},
		),
		SnapshotMaxAgeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_max_age_seconds",
				Help:      "Maximum age of any snapshot in seconds",
			},
		),
		SnapshotTotalPhysicalSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_total_physical_size_bytes",
				Help:      "Total physical disk space used by all snapshots",
			},
		),
		SnapshotTotalLogicalSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_total_logical_size_bytes",
				Help:      "Total logical size of all snapshot data",
			},
		),
		SnapshotMaxDataLag: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_max_data_lag_seconds",
				Help:      "Maximum data lag of any snapshot in seconds",
			},
		),
		SnapshotTotalNumClones: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "snapshot_total_num_clones",
				Help:      "Total number of clones across all snapshots",
			},
		),

		// branch metrics
		BranchesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "branches_total",
				Help:      "Total number of branches",
			},
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

		// observability metrics
		ScrapeSuccessTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "scrape_success_timestamp",
				Help:      "Unix timestamp of last successful metrics collection",
			},
		),
		ScrapeDurationSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "scrape_duration_seconds",
				Help:      "Duration of last metrics collection in seconds",
			},
		),
		ScrapeErrorsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "scrape_errors_total",
				Help:      "Total number of errors during metrics collection",
			},
		),
	}
}

// Register registers all metrics with the Prometheus registry.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.InstanceInfo,
		m.InstanceUptime,
		m.InstanceStatusCode,
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
		m.ClonesByStatus,
		m.CloneMaxAgeSeconds,
		m.CloneTotalDiffSize,
		m.CloneTotalLogicalSize,
		m.CloneTotalCPUUsage,
		m.CloneTotalMemoryUsage,
		m.CloneTotalMemoryLimit,
		m.CloneAvgCPUUsage,
		m.CloneProtectedCount,
		m.SnapshotsTotal,
		m.SnapshotsByPool,
		m.SnapshotMaxAgeSeconds,
		m.SnapshotTotalPhysicalSize,
		m.SnapshotTotalLogicalSize,
		m.SnapshotMaxDataLag,
		m.SnapshotTotalNumClones,
		m.BranchesTotal,
		m.DatasetsTotal,
		m.DatasetsAvailable,
		m.ScrapeSuccessTimestamp,
		m.ScrapeDurationSeconds,
		m.ScrapeErrorsTotal,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// ResetDynamic clears only low-cardinality vector metrics that need resetting.
func (m *Metrics) ResetDynamic() {
	m.InstanceInfo.Reset()
	m.RetrievalStatus.Reset()
	m.DiskTotal.Reset()
	m.DiskFree.Reset()
	m.DiskUsed.Reset()
	m.DiskUsedBySnapshots.Reset()
	m.DiskUsedByClones.Reset()
	m.DiskDataSize.Reset()
	m.DiskCompressRatio.Reset()
	m.PoolStatus.Reset()
	m.ClonesByStatus.Reset()
	m.SnapshotsByPool.Reset()
	m.DatasetsTotal.Reset()
	m.DatasetsAvailable.Reset()
}
