/*
2025 © Postgres.ai
*/

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// containerCPUState stores previous CPU stats for delta calculation.
type containerCPUState struct {
	totalUsage  uint64
	systemUsage uint64
	timestamp   time.Time
}

// Collector collects metrics from DBLab components.
type Collector struct {
	mu           sync.Mutex
	metrics      *Metrics
	cloning      *cloning.Base
	retrieval    *retrieval.Retrieval
	pm           *pool.Manager
	engProps     *global.EngineProps
	dockerClient *client.Client
	startedAt    time.Time
	prevCPUStats map[string]containerCPUState
}

// NewCollector creates a new metrics collector.
func NewCollector(
	m *Metrics,
	cloning *cloning.Base,
	retrieval *retrieval.Retrieval,
	pm *pool.Manager,
	engProps *global.EngineProps,
	dockerClient *client.Client,
	startedAt time.Time,
) *Collector {
	return &Collector{
		metrics:      m,
		cloning:      cloning,
		retrieval:    retrieval,
		pm:           pm,
		engProps:     engProps,
		dockerClient: dockerClient,
		startedAt:    startedAt,
		prevCPUStats: make(map[string]containerCPUState),
	}
}

// Collect gathers all metrics.
func (c *Collector) Collect(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.Reset()

	c.collectInstanceMetrics()
	c.collectPoolMetrics()
	c.collectCloneMetrics(ctx)
	c.collectSnapshotMetrics()
	c.collectBranchMetrics()
}

func (c *Collector) collectInstanceMetrics() {
	c.metrics.InstanceInfo.WithLabelValues(
		c.engProps.InstanceID,
		c.engProps.Version,
		c.engProps.GetEdition(),
	).Set(1)

	c.metrics.InstanceUptime.Set(time.Since(c.startedAt).Seconds())

	statusCode := models.StatusOK
	c.metrics.InstanceStatus.WithLabelValues(statusCode).Set(1)

	c.metrics.RetrievalStatus.WithLabelValues(
		string(c.retrieval.State.Mode),
		string(c.retrieval.State.Status),
	).Set(1)
}

func (c *Collector) collectPoolMetrics() {
	fsmList := c.pm.GetFSManagerList()

	for _, fsm := range fsmList {
		p := fsm.Pool()
		if p == nil {
			continue
		}

		poolName := p.Name
		poolMode := p.Mode
		poolStatus := string(p.Status())

		c.metrics.PoolStatus.WithLabelValues(poolName, poolMode, poolStatus).Set(1)

		fsState, err := fsm.GetFilesystemState()
		if err != nil {
			log.Err("failed to get filesystem state for pool", poolName, err)
			continue
		}

		c.metrics.DiskTotal.WithLabelValues(poolName).Set(float64(fsState.Size))
		c.metrics.DiskFree.WithLabelValues(poolName).Set(float64(fsState.Free))
		c.metrics.DiskUsed.WithLabelValues(poolName).Set(float64(fsState.Used))
		c.metrics.DiskUsedBySnapshots.WithLabelValues(poolName).Set(float64(fsState.UsedBySnapshots))
		c.metrics.DiskUsedByClones.WithLabelValues(poolName).Set(float64(fsState.UsedByClones))
		c.metrics.DiskDataSize.WithLabelValues(poolName).Set(float64(fsState.DataSize))
		c.metrics.DiskCompressRatio.WithLabelValues(poolName).Set(fsState.CompressRatio)

		c.collectDatasetMetrics(fsm, poolName)
	}
}

func (c *Collector) collectDatasetMetrics(fsm pool.FSManager, poolName string) {
	cloneNames, err := fsm.ListClonesNames()
	if err != nil {
		log.Err("failed to list clone names for pool", poolName, err)
		return
	}

	snapshotList := fsm.SnapshotList()

	// total datasets = snapshots (each snapshot is a dataset slot)
	// available = snapshots without active clones (can be reused after full refresh)
	totalDatasets := len(snapshotList)
	busyDatasets := len(cloneNames)
	availableDatasets := totalDatasets - busyDatasets

	if availableDatasets < 0 {
		availableDatasets = 0
	}

	c.metrics.DatasetsTotal.WithLabelValues(poolName).Set(float64(totalDatasets))
	c.metrics.DatasetsAvailable.WithLabelValues(poolName).Set(float64(availableDatasets))
}

func (c *Collector) collectCloneMetrics(ctx context.Context) {
	clones := c.cloning.GetClones()
	c.metrics.ClonesTotal.Set(float64(len(clones)))

	var maxAge float64

	containerStats := c.getContainerStats(ctx, clones)

	for _, clone := range clones {
		if clone == nil {
			continue
		}

		snapshotID := ""
		poolName := ""

		if clone.Snapshot != nil {
			snapshotID = clone.Snapshot.ID
			poolName = clone.Snapshot.Pool
		}

		protectedStr := strconv.FormatBool(clone.Protected)

		c.metrics.CloneInfo.WithLabelValues(
			clone.ID,
			clone.Branch,
			snapshotID,
			poolName,
			clone.Status.Code,
			protectedStr,
		).Set(1)

		if clone.CreatedAt != nil {
			age := time.Since(clone.CreatedAt.Time).Seconds()
			c.metrics.CloneAgeSeconds.WithLabelValues(clone.ID).Set(age)

			if age > maxAge {
				maxAge = age
			}
		}

		c.metrics.CloneDiffSize.WithLabelValues(clone.ID).Set(float64(clone.Metadata.CloneDiffSize))
		c.metrics.CloneLogicalSize.WithLabelValues(clone.ID).Set(float64(clone.Metadata.LogicalSize))

		if stats, ok := containerStats[clone.ID]; ok {
			c.metrics.CloneCPUUsage.WithLabelValues(clone.ID).Set(stats.cpuPercent)
			c.metrics.CloneMemoryUsage.WithLabelValues(clone.ID).Set(float64(stats.memoryUsage))
			c.metrics.CloneMemoryLimit.WithLabelValues(clone.ID).Set(float64(stats.memoryLimit))
		}
	}

	c.metrics.CloneMaxAgeSeconds.Set(maxAge)
}

type containerStatData struct {
	cpuPercent  float64
	memoryUsage uint64
	memoryLimit uint64
}

func (c *Collector) getContainerStats(ctx context.Context, clones []*models.Clone) map[string]containerStatData {
	result := make(map[string]containerStatData)

	if c.dockerClient == nil {
		return result
	}

	activeCloneIDs := make(map[string]struct{})

	for _, clone := range clones {
		if clone == nil || clone.Status.Code != models.StatusOK {
			continue
		}

		activeCloneIDs[clone.ID] = struct{}{}

		stats, err := c.dockerClient.ContainerStatsOneShot(ctx, clone.ID)
		if err != nil {
			log.Dbg(fmt.Sprintf("failed to get container stats for clone %s: %v", clone.ID, err))
			continue
		}

		var statsJSON container.StatsResponse
		decodeErr := json.NewDecoder(stats.Body).Decode(&statsJSON)
		stats.Body.Close()

		if decodeErr != nil {
			log.Dbg(fmt.Sprintf("failed to decode container stats for clone %s: %v", clone.ID, decodeErr))
			continue
		}

		cpuPercent := c.calculateCPUPercent(clone.ID, &statsJSON)
		memoryUsage := statsJSON.MemoryStats.Usage
		memoryLimit := statsJSON.MemoryStats.Limit

		result[clone.ID] = containerStatData{
			cpuPercent:  cpuPercent,
			memoryUsage: memoryUsage,
			memoryLimit: memoryLimit,
		}
	}

	c.cleanupStaleCPUStats(activeCloneIDs)

	return result
}

// calculateCPUPercent calculates CPU percentage using stored previous stats.
// ContainerStatsOneShot returns empty PreCPUStats, so we maintain our own history.
func (c *Collector) calculateCPUPercent(cloneID string, stats *container.StatsResponse) float64 {
	currentTotalUsage := stats.CPUStats.CPUUsage.TotalUsage
	currentSystemUsage := stats.CPUStats.SystemUsage
	now := time.Now()

	prevStats, hasPrev := c.prevCPUStats[cloneID]

	c.prevCPUStats[cloneID] = containerCPUState{
		totalUsage:  currentTotalUsage,
		systemUsage: currentSystemUsage,
		timestamp:   now,
	}

	if !hasPrev {
		return 0
	}

	timeDelta := now.Sub(prevStats.timestamp)
	if timeDelta < time.Second {
		return 0
	}

	if currentTotalUsage < prevStats.totalUsage || currentSystemUsage < prevStats.systemUsage {
		return 0
	}

	cpuDelta := float64(currentTotalUsage - prevStats.totalUsage)
	systemDelta := float64(currentSystemUsage - prevStats.systemUsage)

	if systemDelta <= 0 {
		return 0
	}

	cpuCount := float64(stats.CPUStats.OnlineCPUs)
	if cpuCount == 0 {
		cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}

	if cpuCount <= 0 {
		cpuCount = 1
	}

	return (cpuDelta / systemDelta) * cpuCount * 100.0
}

func (c *Collector) cleanupStaleCPUStats(activeCloneIDs map[string]struct{}) {
	for cloneID := range c.prevCPUStats {
		if _, ok := activeCloneIDs[cloneID]; !ok {
			delete(c.prevCPUStats, cloneID)
		}
	}
}

func (c *Collector) collectSnapshotMetrics() {
	snapshots, err := c.cloning.GetSnapshots()
	if err != nil {
		log.Err("failed to get snapshots for metrics", err)
		return
	}

	c.metrics.SnapshotsTotal.Set(float64(len(snapshots)))

	var maxAge float64
	var maxDataLag float64
	now := time.Now()

	for _, snapshot := range snapshots {
		c.metrics.SnapshotInfo.WithLabelValues(
			snapshot.ID,
			snapshot.Pool,
			snapshot.Branch,
		).Set(1)

		if snapshot.CreatedAt != nil {
			age := now.Sub(snapshot.CreatedAt.Time).Seconds()
			c.metrics.SnapshotAgeSeconds.WithLabelValues(snapshot.ID).Set(age)

			if age > maxAge {
				maxAge = age
			}
		}

		c.metrics.SnapshotPhysicalSize.WithLabelValues(snapshot.ID).Set(float64(snapshot.PhysicalSize))
		c.metrics.SnapshotLogicalSize.WithLabelValues(snapshot.ID).Set(float64(snapshot.LogicalSize))

		if snapshot.DataStateAt != nil {
			dataLag := now.Sub(snapshot.DataStateAt.Time).Seconds()
			c.metrics.SnapshotDataLag.WithLabelValues(snapshot.ID).Set(dataLag)

			if dataLag > maxDataLag {
				maxDataLag = dataLag
			}
		}

		c.metrics.SnapshotNumClones.WithLabelValues(snapshot.ID).Set(float64(snapshot.NumClones))
	}

	c.metrics.SnapshotMaxAgeSeconds.Set(maxAge)
	c.metrics.SnapshotMaxDataLag.Set(maxDataLag)
}

func (c *Collector) collectBranchMetrics() {
	fsmList := c.pm.GetFSManagerList()
	totalBranches := 0

	for _, fsm := range fsmList {
		p := fsm.Pool()
		if p == nil {
			continue
		}

		poolName := p.Name
		branches, err := fsm.ListBranches()

		if err != nil {
			log.Err("failed to list branches for pool", poolName, err)
			continue
		}

		for branchName, snapshotID := range branches {
			c.metrics.BranchInfo.WithLabelValues(branchName, poolName, snapshotID).Set(1)
			totalBranches++
		}
	}

	c.metrics.BranchesTotal.Set(float64(totalBranches))
}
