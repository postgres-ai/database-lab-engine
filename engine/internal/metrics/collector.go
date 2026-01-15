/*
2025 © Postgres.ai
*/

// Package metrics provides Prometheus metrics collection for Database Lab Engine.
package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	namespace = "dblab"
)

// StatusProvider defines an interface for getting instance status.
type StatusProvider interface {
	InstanceStatus() *models.InstanceStatus
	Uptime() float64
}

// Collector implements prometheus.Collector for DBLab metrics.
type Collector struct {
	statusProvider StatusProvider

	// engine metrics
	engineInfo   *prometheus.Desc
	engineUptime *prometheus.Desc

	// retrieval/sync metrics
	retrievalMode        *prometheus.Desc
	retrievalStatus      *prometheus.Desc
	lastRefreshTimestamp *prometheus.Desc
	nextRefreshTimestamp *prometheus.Desc
	dataFreshness        *prometheus.Desc
	refreshAlerts        *prometheus.Desc

	// synchronization metrics (physical mode)
	replicationLag    *prometheus.Desc
	replicationUptime *prometheus.Desc

	// pool metrics
	poolStatus          *prometheus.Desc
	poolDataStateAt     *prometheus.Desc
	poolSize            *prometheus.Desc
	poolFree            *prometheus.Desc
	poolUsed            *prometheus.Desc
	poolDataSize        *prometheus.Desc
	poolUsedBySnapshots *prometheus.Desc
	poolUsedByClones    *prometheus.Desc
	poolCompressRatio   *prometheus.Desc
	poolCloneCount      *prometheus.Desc

	// clone metrics
	clonesTotal          *prometheus.Desc
	clonesByStatus       *prometheus.Desc
	expectedCloningTime  *prometheus.Desc
	cloneDiffSize        *prometheus.Desc
	cloneLogicalSize     *prometheus.Desc
	cloneCloningTime     *prometheus.Desc
	protectedClonesTotal *prometheus.Desc

	// snapshot metrics
	snapshotsTotal       *prometheus.Desc
	snapshotPhysicalSize *prometheus.Desc
	snapshotLogicalSize  *prometheus.Desc
	snapshotCloneCount   *prometheus.Desc

	// branch metrics
	branchesTotal *prometheus.Desc

	// slots/resources metrics for logical mode
	busySlots *prometheus.Desc
}

// NewCollector creates a new Collector instance.
func NewCollector(statusProvider StatusProvider) *Collector {
	return &Collector{
		statusProvider: statusProvider,

		// engine metrics
		engineInfo: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "engine", "info"),
			"Database Lab Engine information",
			[]string{"version", "edition", "instance_id"},
			nil,
		),
		engineUptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "engine", "uptime_seconds"),
			"Time since Database Lab Engine started in seconds",
			nil,
			nil,
		),

		// retrieval/sync metrics
		retrievalMode: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "mode"),
			"Current retrieval mode (1 for physical, 2 for logical, 0 for unknown)",
			nil,
			nil,
		),
		retrievalStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "status"),
			"Current retrieval status",
			[]string{"status"},
			nil,
		),
		lastRefreshTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "last_refresh_timestamp_seconds"),
			"Unix timestamp of last data refresh",
			nil,
			nil,
		),
		nextRefreshTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "next_refresh_timestamp_seconds"),
			"Unix timestamp of next scheduled data refresh",
			nil,
			nil,
		),
		dataFreshness: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "data_freshness_seconds"),
			"Time since last data refresh in seconds",
			nil,
			nil,
		),
		refreshAlerts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "retrieval", "alerts_total"),
			"Number of retrieval alerts by type and level",
			[]string{"type", "level"},
			nil,
		),

		// synchronization metrics (physical mode)
		replicationLag: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sync", "replication_lag_seconds"),
			"Replication lag in seconds (physical mode)",
			nil,
			nil,
		),
		replicationUptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sync", "replication_uptime_seconds"),
			"Replication uptime in seconds (physical mode)",
			nil,
			nil,
		),

		// pool metrics
		poolStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "status"),
			"Pool status (1=active, 2=refreshing, 3=empty)",
			[]string{"pool", "mode"},
			nil,
		),
		poolDataStateAt: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "data_state_at_timestamp_seconds"),
			"Unix timestamp of the pool data state",
			[]string{"pool"},
			nil,
		),
		poolSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "size_bytes"),
			"Total pool size in bytes",
			[]string{"pool"},
			nil,
		),
		poolFree: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "free_bytes"),
			"Free space in pool in bytes",
			[]string{"pool"},
			nil,
		),
		poolUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "used_bytes"),
			"Used space in pool in bytes",
			[]string{"pool"},
			nil,
		),
		poolDataSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "data_size_bytes"),
			"Logical data size in bytes",
			[]string{"pool"},
			nil,
		),
		poolUsedBySnapshots: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "used_by_snapshots_bytes"),
			"Space used by snapshots in bytes",
			[]string{"pool"},
			nil,
		),
		poolUsedByClones: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "used_by_clones_bytes"),
			"Space used by clones in bytes",
			[]string{"pool"},
			nil,
		),
		poolCompressRatio: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "compress_ratio"),
			"Compression ratio of the pool",
			[]string{"pool"},
			nil,
		),
		poolCloneCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pool", "clones_total"),
			"Number of clones in the pool",
			[]string{"pool"},
			nil,
		),

		// clone metrics
		clonesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clones", "total"),
			"Total number of clones",
			nil,
			nil,
		),
		clonesByStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clones", "by_status"),
			"Number of clones by status",
			[]string{"status"},
			nil,
		),
		expectedCloningTime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clones", "expected_cloning_time_seconds"),
			"Expected time to create a clone in seconds",
			nil,
			nil,
		),
		cloneDiffSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clone", "diff_size_bytes"),
			"Clone diff size in bytes",
			[]string{"clone_id", "branch"},
			nil,
		),
		cloneLogicalSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clone", "logical_size_bytes"),
			"Clone logical size in bytes",
			[]string{"clone_id", "branch"},
			nil,
		),
		cloneCloningTime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clone", "cloning_time_seconds"),
			"Time taken to create clone in seconds",
			[]string{"clone_id", "branch"},
			nil,
		),
		protectedClonesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clones", "protected_total"),
			"Number of protected clones",
			nil,
			nil,
		),

		// snapshot metrics
		snapshotsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "total"),
			"Total number of snapshots",
			[]string{"pool", "branch", "type"},
			nil,
		),
		snapshotPhysicalSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshot", "physical_size_bytes"),
			"Snapshot physical size in bytes",
			[]string{"snapshot_id", "pool", "branch"},
			nil,
		),
		snapshotLogicalSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshot", "logical_size_bytes"),
			"Snapshot logical size in bytes",
			[]string{"snapshot_id", "pool", "branch"},
			nil,
		),
		snapshotCloneCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshot", "clone_count"),
			"Number of clones using this snapshot",
			[]string{"snapshot_id", "pool"},
			nil,
		),

		// branch metrics
		branchesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "branches", "total"),
			"Total number of branches",
			nil,
			nil,
		),

		// slots/resources metrics for logical mode
		busySlots: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slots", "busy_total"),
			"Number of busy slots (clones, custom snapshots, branches) preventing full refresh in logical mode",
			nil,
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.engineInfo
	ch <- c.engineUptime
	ch <- c.retrievalMode
	ch <- c.retrievalStatus
	ch <- c.lastRefreshTimestamp
	ch <- c.nextRefreshTimestamp
	ch <- c.dataFreshness
	ch <- c.refreshAlerts
	ch <- c.replicationLag
	ch <- c.replicationUptime
	ch <- c.poolStatus
	ch <- c.poolDataStateAt
	ch <- c.poolSize
	ch <- c.poolFree
	ch <- c.poolUsed
	ch <- c.poolDataSize
	ch <- c.poolUsedBySnapshots
	ch <- c.poolUsedByClones
	ch <- c.poolCompressRatio
	ch <- c.poolCloneCount
	ch <- c.clonesTotal
	ch <- c.clonesByStatus
	ch <- c.expectedCloningTime
	ch <- c.cloneDiffSize
	ch <- c.cloneLogicalSize
	ch <- c.cloneCloningTime
	ch <- c.protectedClonesTotal
	ch <- c.snapshotsTotal
	ch <- c.snapshotPhysicalSize
	ch <- c.snapshotLogicalSize
	ch <- c.snapshotCloneCount
	ch <- c.branchesTotal
	ch <- c.busySlots
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	status := c.statusProvider.InstanceStatus()
	if status == nil {
		return
	}

	c.collectEngineMetrics(ch, status)
	c.collectRetrievalMetrics(ch, status)
	c.collectSyncMetrics(ch, status)
	c.collectPoolMetrics(ch, status)
	c.collectCloneMetrics(ch, status)
	c.collectBusySlots(ch, status)
}

func (c *Collector) collectEngineMetrics(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	ch <- prometheus.MustNewConstMetric(
		c.engineInfo,
		prometheus.GaugeValue,
		1,
		version.GetVersion(),
		status.Engine.Edition,
		status.Engine.InstanceID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.engineUptime,
		prometheus.GaugeValue,
		c.statusProvider.Uptime(),
	)
}

func (c *Collector) collectRetrievalMetrics(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	// retrieval mode: 1=physical, 2=logical, 0=unknown
	modeValue := float64(0)
	switch status.Retrieving.Mode {
	case models.Physical:
		modeValue = 1
	case models.Logical:
		modeValue = 2
	}

	ch <- prometheus.MustNewConstMetric(
		c.retrievalMode,
		prometheus.GaugeValue,
		modeValue,
	)

	// retrieval status
	statusValues := map[models.RetrievalStatus]float64{
		models.Inactive:     1,
		models.Pending:      2,
		models.Failed:       3,
		models.Refreshing:   4,
		models.Renewed:      5,
		models.Snapshotting: 6,
		models.Finished:     7,
	}

	for s, v := range statusValues {
		val := float64(0)
		if status.Retrieving.Status == s {
			val = v
		}

		ch <- prometheus.MustNewConstMetric(
			c.retrievalStatus,
			prometheus.GaugeValue,
			val,
			string(s),
		)
	}

	// last refresh timestamp
	if status.Retrieving.LastRefresh != nil && !status.Retrieving.LastRefresh.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.lastRefreshTimestamp,
			prometheus.GaugeValue,
			float64(status.Retrieving.LastRefresh.Unix()),
		)

		ch <- prometheus.MustNewConstMetric(
			c.dataFreshness,
			prometheus.GaugeValue,
			time.Since(status.Retrieving.LastRefresh.Time).Seconds(),
		)
	}

	// next refresh timestamp
	if status.Retrieving.NextRefresh != nil && !status.Retrieving.NextRefresh.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.nextRefreshTimestamp,
			prometheus.GaugeValue,
			float64(status.Retrieving.NextRefresh.Unix()),
		)
	}

	// alerts
	for alertType, alert := range status.Retrieving.Alerts {
		ch <- prometheus.MustNewConstMetric(
			c.refreshAlerts,
			prometheus.GaugeValue,
			float64(alert.Count),
			string(alertType),
			string(alert.Level),
		)
	}
}

func (c *Collector) collectSyncMetrics(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	if status.Synchronization == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.replicationLag,
		prometheus.GaugeValue,
		float64(status.Synchronization.ReplicationLag),
	)

	ch <- prometheus.MustNewConstMetric(
		c.replicationUptime,
		prometheus.GaugeValue,
		float64(status.Synchronization.ReplicationUptime),
	)
}

func (c *Collector) collectPoolMetrics(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	for _, pool := range status.Pools {
		// pool status: 1=active, 2=refreshing, 3=empty
		statusValue := float64(0)
		switch pool.Status {
		case "active":
			statusValue = 1
		case "refreshing":
			statusValue = 2
		case "empty":
			statusValue = 3
		}

		ch <- prometheus.MustNewConstMetric(
			c.poolStatus,
			prometheus.GaugeValue,
			statusValue,
			pool.Name,
			pool.Mode,
		)

		if pool.DataStateAt != nil && !pool.DataStateAt.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				c.poolDataStateAt,
				prometheus.GaugeValue,
				float64(pool.DataStateAt.Unix()),
				pool.Name,
			)
		}

		ch <- prometheus.MustNewConstMetric(
			c.poolSize,
			prometheus.GaugeValue,
			float64(pool.FileSystem.Size),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolFree,
			prometheus.GaugeValue,
			float64(pool.FileSystem.Free),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolUsed,
			prometheus.GaugeValue,
			float64(pool.FileSystem.Used),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolDataSize,
			prometheus.GaugeValue,
			float64(pool.FileSystem.DataSize),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolUsedBySnapshots,
			prometheus.GaugeValue,
			float64(pool.FileSystem.UsedBySnapshots),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolUsedByClones,
			prometheus.GaugeValue,
			float64(pool.FileSystem.UsedByClones),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolCompressRatio,
			prometheus.GaugeValue,
			pool.FileSystem.CompressRatio,
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.poolCloneCount,
			prometheus.GaugeValue,
			float64(len(pool.CloneList)),
			pool.Name,
		)
	}
}

func (c *Collector) collectCloneMetrics(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	ch <- prometheus.MustNewConstMetric(
		c.clonesTotal,
		prometheus.GaugeValue,
		float64(status.Cloning.NumClones),
	)

	ch <- prometheus.MustNewConstMetric(
		c.expectedCloningTime,
		prometheus.GaugeValue,
		status.Cloning.ExpectedCloningTime,
	)

	// count clones by status
	statusCounts := make(map[string]int)
	protectedCount := 0
	branches := make(map[string]bool)

	for _, clone := range status.Cloning.Clones {
		if clone == nil {
			continue
		}

		statusCounts[string(clone.Status.Code)]++

		if clone.Protected {
			protectedCount++
		}

		branches[clone.Branch] = true

		ch <- prometheus.MustNewConstMetric(
			c.cloneDiffSize,
			prometheus.GaugeValue,
			float64(clone.Metadata.CloneDiffSize),
			clone.ID,
			clone.Branch,
		)

		ch <- prometheus.MustNewConstMetric(
			c.cloneLogicalSize,
			prometheus.GaugeValue,
			float64(clone.Metadata.LogicalSize),
			clone.ID,
			clone.Branch,
		)

		ch <- prometheus.MustNewConstMetric(
			c.cloneCloningTime,
			prometheus.GaugeValue,
			clone.Metadata.CloningTime,
			clone.ID,
			clone.Branch,
		)
	}

	for statusCode, count := range statusCounts {
		ch <- prometheus.MustNewConstMetric(
			c.clonesByStatus,
			prometheus.GaugeValue,
			float64(count),
			statusCode,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.protectedClonesTotal,
		prometheus.GaugeValue,
		float64(protectedCount),
	)

	ch <- prometheus.MustNewConstMetric(
		c.branchesTotal,
		prometheus.GaugeValue,
		float64(len(branches)),
	)
}

func (c *Collector) collectBusySlots(ch chan<- prometheus.Metric, status *models.InstanceStatus) {
	busyCount := int(status.Cloning.NumClones)

	// count unique branches (excluding default "main" branch)
	branches := make(map[string]bool)
	for _, clone := range status.Cloning.Clones {
		if clone != nil && clone.Branch != "" && clone.Branch != "main" {
			branches[clone.Branch] = true
		}
	}

	busyCount += len(branches)

	// count user snapshots from pools (snapshots with clones or custom snapshots)
	for _, pool := range status.Pools {
		for _, clone := range pool.CloneList {
			if clone != "" {
				busyCount++
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(
		c.busySlots,
		prometheus.GaugeValue,
		float64(busyCount),
	)
}

// snapshotType returns "auto" or "user" based on snapshot ID pattern.
func snapshotType(snapshotID, poolName string) string {
	// automatic snapshots are created directly on the pool dataset
	if len(snapshotID) > len(poolName) && snapshotID[:len(poolName)] == poolName {
		atIndex := len(poolName)
		if atIndex < len(snapshotID) && snapshotID[atIndex] == '@' {
			return "auto"
		}
	}

	return "user"
}

// SnapshotMetrics holds aggregated snapshot counts.
type SnapshotMetrics struct {
	Total     int
	AutoCount int
	UserCount int
	Pool      string
	Branch    string
}

// CollectSnapshotMetrics collects additional snapshot metrics.
// This is a helper method that can be called with snapshot list from the cloning service.
func (c *Collector) CollectSnapshotMetrics(ch chan<- prometheus.Metric, snapshots []*models.Snapshot, poolName string) {
	if len(snapshots) == 0 {
		return
	}

	branchCounts := make(map[string]*SnapshotMetrics)

	for _, snap := range snapshots {
		if snap == nil {
			continue
		}

		branch := snap.Branch
		if branch == "" {
			branch = "main"
		}

		key := snap.Pool + "/" + branch
		if branchCounts[key] == nil {
			branchCounts[key] = &SnapshotMetrics{
				Pool:   snap.Pool,
				Branch: branch,
			}
		}

		branchCounts[key].Total++

		snapType := snapshotType(snap.ID, snap.Pool)
		if snapType == "auto" {
			branchCounts[key].AutoCount++
		} else {
			branchCounts[key].UserCount++
		}

		ch <- prometheus.MustNewConstMetric(
			c.snapshotPhysicalSize,
			prometheus.GaugeValue,
			float64(snap.PhysicalSize),
			snap.ID,
			snap.Pool,
			branch,
		)

		ch <- prometheus.MustNewConstMetric(
			c.snapshotLogicalSize,
			prometheus.GaugeValue,
			float64(snap.LogicalSize),
			snap.ID,
			snap.Pool,
			branch,
		)

		ch <- prometheus.MustNewConstMetric(
			c.snapshotCloneCount,
			prometheus.GaugeValue,
			float64(snap.NumClones),
			snap.ID,
			snap.Pool,
		)
	}

	for _, metrics := range branchCounts {
		ch <- prometheus.MustNewConstMetric(
			c.snapshotsTotal,
			prometheus.GaugeValue,
			float64(metrics.AutoCount),
			metrics.Pool,
			metrics.Branch,
			"auto",
		)

		ch <- prometheus.MustNewConstMetric(
			c.snapshotsTotal,
			prometheus.GaugeValue,
			float64(metrics.UserCount),
			metrics.Pool,
			metrics.Branch,
			"user",
		)
	}
}

// BranchCount returns the number of unique branches used by clones.
func BranchCount(clones []*models.Clone) int {
	branches := make(map[string]bool)

	for _, clone := range clones {
		if clone != nil && clone.Branch != "" {
			branches[clone.Branch] = true
		}
	}

	return len(branches)
}

// ActiveCloneCountByStatus returns clone counts grouped by status.
func ActiveCloneCountByStatus(clones []*models.Clone) map[string]int {
	counts := make(map[string]int)

	for _, clone := range clones {
		if clone != nil {
			counts[string(clone.Status.Code)]++
		}
	}

	return counts
}

// PoolDataFreshness returns time since pool data state in seconds.
func PoolDataFreshness(pool models.PoolEntry) float64 {
	if pool.DataStateAt == nil || pool.DataStateAt.IsZero() {
		return 0
	}

	return time.Since(pool.DataStateAt.Time).Seconds()
}

// DiskUsagePercent returns the disk usage percentage for a pool.
func DiskUsagePercent(fs models.FileSystem) float64 {
	if fs.Size == 0 {
		return 0
	}

	return float64(fs.Used) / float64(fs.Size) * 100
}

// DataStateAtSafe returns data state timestamp for a pool with safe access.
func DataStateAtSafe(pools []models.PoolEntry) int64 {
	for _, pool := range pools {
		if pool.DataStateAt != nil && !pool.DataStateAt.IsZero() {
			return pool.DataStateAt.Unix()
		}
	}

	return 0
}

// NumClonesFromStatus extracts the total number of clones from status.
func NumClonesFromStatus(status *models.InstanceStatus) uint64 {
	if status == nil {
		return 0
	}

	return status.Cloning.NumClones
}

// ReplicationLagFromSync extracts replication lag from sync status.
func ReplicationLagFromSync(sync *models.Sync) int {
	if sync == nil {
		return 0
	}

	return sync.ReplicationLag
}

// BoolToFloat converts a boolean to float64 (1 for true, 0 for false).
func BoolToFloat(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

// StatusCodeToFloat converts status codes to numeric values.
func StatusCodeToFloat(code string) float64 {
	statusMap := map[string]float64{
		"OK":        1,
		"CREATING":  2,
		"DELETING":  3,
		"RESETTING": 4,
		"EXPORTING": 5,
		"FATAL":     6,
	}

	if val, ok := statusMap[code]; ok {
		return val
	}

	return 0
}

// FormatBranch returns branch name, defaulting to "main" if empty.
func FormatBranch(branch string) string {
	if branch == "" {
		return "main"
	}

	return branch
}

// ParseReplicationLag parses replication lag from status.
func ParseReplicationLag(sync *models.Sync) string {
	if sync == nil {
		return "0"
	}

	return strconv.Itoa(sync.ReplicationLag)
}
