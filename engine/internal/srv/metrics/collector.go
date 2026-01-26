/*
2025 © Postgres.ai
*/

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	cpuPercentMultiplier = 100.0
	dockerStatsTimeout   = 5 * time.Second
	dockerStatsWorkers   = 10
	cpuNoData            = -1.0
)

// CloningService defines the interface for clone and snapshot operations needed by metrics.
type CloningService interface {
	GetClones() []*models.Clone
	GetSnapshots() ([]models.Snapshot, error)
}

// RetrievalService defines the interface for retrieval state needed by metrics.
type RetrievalService interface {
	GetRetrievalMode() models.RetrievalMode
	GetRetrievalStatus() models.RetrievalStatus
}

// PoolService defines the interface for pool operations needed by metrics.
type PoolService interface {
	GetFSManagerList() []pool.FSManager
}

// containerCPUState stores previous CPU stats for delta calculation.
type containerCPUState struct {
	totalUsage  uint64
	systemUsage uint64
	timestamp   time.Time
}

// Collector collects metrics from DBLab components.
type Collector struct {
	mu           sync.RWMutex
	metrics      *Metrics
	cloning      CloningService
	retrieval    RetrievalService
	pm           PoolService
	engProps     *global.EngineProps
	dockerClient *client.Client
	startedAt    time.Time
	cpuStatsMu   sync.Mutex
	prevCPUStats map[string]containerCPUState
}

// NewCollector creates a new metrics collector.
func NewCollector(
	m *Metrics,
	cloning CloningService,
	retrieval RetrievalService,
	pm PoolService,
	engProps *global.EngineProps,
	dockerClient *client.Client,
	startedAt time.Time,
) (*Collector, error) {
	if m == nil {
		return nil, fmt.Errorf("metrics is required")
	}

	if cloning == nil {
		return nil, fmt.Errorf("cloning is required")
	}

	if retrieval == nil {
		return nil, fmt.Errorf("retrieval is required")
	}

	if pm == nil {
		return nil, fmt.Errorf("pool manager is required")
	}

	if engProps == nil {
		return nil, fmt.Errorf("engine props is required")
	}

	return &Collector{
		metrics:      m,
		cloning:      cloning,
		retrieval:    retrieval,
		pm:           pm,
		engProps:     engProps,
		dockerClient: dockerClient,
		startedAt:    startedAt,
		prevCPUStats: make(map[string]containerCPUState),
	}, nil
}

// Collect gathers all metrics (uses write lock).
func (c *Collector) Collect(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.collectAll(ctx)
}

// CollectAndServe serves metrics using read lock only.
func (c *Collector) CollectAndServe(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler.ServeHTTP(w, r)
}

// StartBackgroundCollection starts periodic metrics collection.
func (c *Collector) StartBackgroundCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.Collect(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.Collect(ctx)
		}
	}
}

func (c *Collector) collectAll(ctx context.Context) {
	start := time.Now()

	c.metrics.ResetDynamic()

	c.collectInstanceMetrics()
	c.collectPoolMetrics()
	c.collectCloneMetrics(ctx)
	c.collectSnapshotMetrics()
	c.collectBranchMetrics()

	c.metrics.ScrapeDurationSeconds.Set(time.Since(start).Seconds())
	c.metrics.ScrapeSuccessTimestamp.Set(float64(time.Now().Unix()))
}

func (c *Collector) collectInstanceMetrics() {
	c.metrics.InstanceInfo.WithLabelValues(
		c.engProps.InstanceID,
		version.GetVersion(),
		c.engProps.GetEdition(),
	).Set(1)

	c.metrics.InstanceUptime.Set(time.Since(c.startedAt).Seconds())

	c.metrics.InstanceStatusCode.Set(StatusCodeOK)

	c.metrics.RetrievalStatus.WithLabelValues(
		string(c.retrieval.GetRetrievalMode()),
		string(c.retrieval.GetRetrievalStatus()),
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
			c.metrics.ScrapeErrorsTotal.Inc()

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
		c.metrics.ScrapeErrorsTotal.Inc()

		return
	}

	snapshotList := fsm.SnapshotList()

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

	var (
		maxAge           float64
		totalDiffSize    uint64
		totalLogicalSize uint64
		totalCPU         float64
		totalMemoryUsage uint64
		totalMemoryLimit uint64
		protectedCount   int
		cpuValidCount    int
		statusCounts     = make(map[string]int)
	)

	containerStats := c.getContainerStats(ctx, clones)

	for _, clone := range clones {
		if clone == nil {
			continue
		}

		statusCounts[string(clone.Status.Code)]++

		if clone.Protected {
			protectedCount++
		}

		if clone.CreatedAt != nil {
			age := time.Since(clone.CreatedAt.Time).Seconds()
			if age > maxAge {
				maxAge = age
			}
		}

		totalDiffSize += clone.Metadata.CloneDiffSize
		totalLogicalSize += clone.Metadata.LogicalSize

		if stats, ok := containerStats[clone.ID]; ok {
			if stats.cpuPercent >= 0 {
				totalCPU += stats.cpuPercent
				cpuValidCount++
			}

			totalMemoryUsage += stats.memoryUsage
			totalMemoryLimit += stats.memoryLimit
		}
	}

	c.metrics.CloneMaxAgeSeconds.Set(maxAge)
	c.metrics.CloneTotalDiffSize.Set(float64(totalDiffSize))
	c.metrics.CloneTotalLogicalSize.Set(float64(totalLogicalSize))
	c.metrics.CloneTotalCPUUsage.Set(totalCPU)
	c.metrics.CloneTotalMemoryUsage.Set(float64(totalMemoryUsage))
	c.metrics.CloneTotalMemoryLimit.Set(float64(totalMemoryLimit))
	c.metrics.CloneProtectedCount.Set(float64(protectedCount))

	if cpuValidCount > 0 {
		c.metrics.CloneAvgCPUUsage.Set(totalCPU / float64(cpuValidCount))
	} else {
		c.metrics.CloneAvgCPUUsage.Set(0)
	}

	for status, count := range statusCounts {
		c.metrics.ClonesByStatus.WithLabelValues(status).Set(float64(count))
	}
}

type containerStatData struct {
	cpuPercent  float64
	memoryUsage uint64
	memoryLimit uint64
}

type statsJob struct {
	clone *models.Clone
}

type statsResult struct {
	cloneID string
	data    containerStatData
	ok      bool
}

func (c *Collector) getContainerStats(ctx context.Context, clones []*models.Clone) map[string]containerStatData {
	result := make(map[string]containerStatData)

	if c.dockerClient == nil {
		return result
	}

	activeClones := filterActiveClones(clones)
	if len(activeClones) == 0 {
		c.cleanupStaleCPUStats(make(map[string]struct{}))

		return result
	}

	jobs := make(chan statsJob, len(activeClones))
	results := make(chan statsResult, len(activeClones))

	workerCount := min(dockerStatsWorkers, len(activeClones))

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range jobs {
				data, ok := c.fetchSingleContainerStats(ctx, job.clone)
				results <- statsResult{cloneID: job.clone.ID, data: data, ok: ok}
			}
		}()
	}

	for _, clone := range activeClones {
		jobs <- statsJob{clone: clone}
	}

	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	activeCloneIDs := make(map[string]struct{})

	for r := range results {
		activeCloneIDs[r.cloneID] = struct{}{}

		if r.ok {
			result[r.cloneID] = r.data
		}
	}

	c.cleanupStaleCPUStats(activeCloneIDs)

	return result
}

func filterActiveClones(clones []*models.Clone) []*models.Clone {
	result := make([]*models.Clone, 0, len(clones))

	for _, clone := range clones {
		if clone != nil && clone.Status.Code == models.StatusOK {
			result = append(result, clone)
		}
	}

	return result
}

func (c *Collector) fetchSingleContainerStats(ctx context.Context, clone *models.Clone) (containerStatData, bool) {
	statsCtx, cancel := context.WithTimeout(ctx, dockerStatsTimeout)
	defer cancel()

	stats, err := c.dockerClient.ContainerStatsOneShot(statsCtx, clone.ID)
	if err != nil {
		log.Dbg(fmt.Sprintf("failed to get container stats for clone %s: %v", clone.ID, err))

		return containerStatData{}, false
	}

	var statsJSON container.StatsResponse

	decodeErr := json.NewDecoder(stats.Body).Decode(&statsJSON)
	_ = stats.Body.Close()

	if decodeErr != nil {
		log.Dbg(fmt.Sprintf("failed to decode container stats for clone %s: %v", clone.ID, decodeErr))

		return containerStatData{}, false
	}

	cpuPercent := c.calculateCPUPercent(clone.ID, &statsJSON)

	return containerStatData{
		cpuPercent:  cpuPercent,
		memoryUsage: statsJSON.MemoryStats.Usage,
		memoryLimit: statsJSON.MemoryStats.Limit,
	}, true
}

func (c *Collector) calculateCPUPercent(cloneID string, stats *container.StatsResponse) float64 {
	currentTotalUsage := stats.CPUStats.CPUUsage.TotalUsage
	currentSystemUsage := stats.CPUStats.SystemUsage
	now := time.Now()

	c.cpuStatsMu.Lock()
	prevStats, hasPrev := c.prevCPUStats[cloneID]
	c.prevCPUStats[cloneID] = containerCPUState{
		totalUsage:  currentTotalUsage,
		systemUsage: currentSystemUsage,
		timestamp:   now,
	}
	c.cpuStatsMu.Unlock()

	if !hasPrev {
		return cpuNoData
	}

	timeDelta := now.Sub(prevStats.timestamp)
	if timeDelta < time.Second {
		return cpuNoData
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

	return (cpuDelta / systemDelta) * cpuCount * cpuPercentMultiplier
}

func (c *Collector) cleanupStaleCPUStats(activeCloneIDs map[string]struct{}) {
	c.cpuStatsMu.Lock()
	defer c.cpuStatsMu.Unlock()

	keysToDelete := make([]string, 0)

	for cloneID := range c.prevCPUStats {
		if _, ok := activeCloneIDs[cloneID]; !ok {
			keysToDelete = append(keysToDelete, cloneID)
		}
	}

	for _, cloneID := range keysToDelete {
		delete(c.prevCPUStats, cloneID)
	}
}

func (c *Collector) collectSnapshotMetrics() {
	snapshots, err := c.cloning.GetSnapshots()
	if err != nil {
		log.Err("failed to get snapshots for metrics", err)
		c.metrics.ScrapeErrorsTotal.Inc()

		return
	}

	c.metrics.SnapshotsTotal.Set(float64(len(snapshots)))

	var (
		maxAge            float64
		maxDataLag        float64
		totalPhysicalSize uint64
		totalLogicalSize  uint64
		totalNumClones    int
		poolCounts        = make(map[string]int)
	)

	now := time.Now()

	for _, snapshot := range snapshots {
		poolCounts[snapshot.Pool]++

		if snapshot.CreatedAt != nil {
			age := now.Sub(snapshot.CreatedAt.Time).Seconds()
			if age > maxAge {
				maxAge = age
			}
		}

		totalPhysicalSize += snapshot.PhysicalSize
		totalLogicalSize += snapshot.LogicalSize

		if snapshot.DataStateAt != nil {
			dataLag := now.Sub(snapshot.DataStateAt.Time).Seconds()
			if dataLag > maxDataLag {
				maxDataLag = dataLag
			}
		}

		totalNumClones += snapshot.NumClones
	}

	c.metrics.SnapshotMaxAgeSeconds.Set(maxAge)
	c.metrics.SnapshotTotalPhysicalSize.Set(float64(totalPhysicalSize))
	c.metrics.SnapshotTotalLogicalSize.Set(float64(totalLogicalSize))
	c.metrics.SnapshotMaxDataLag.Set(maxDataLag)
	c.metrics.SnapshotTotalNumClones.Set(float64(totalNumClones))

	for poolName, count := range poolCounts {
		c.metrics.SnapshotsByPool.WithLabelValues(poolName).Set(float64(count))
	}
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
			c.metrics.ScrapeErrorsTotal.Inc()

			continue
		}

		totalBranches += len(branches)
	}

	c.metrics.BranchesTotal.Set(float64(totalBranches))
}
