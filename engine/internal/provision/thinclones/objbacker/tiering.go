/*
2025 © Postgres.ai
*/

package objbacker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// TieringPolicy defines when snapshots should be archived to object storage.
type TieringPolicy struct {
	// ArchiveAfter is the age threshold after which snapshots are archived.
	// Snapshots older than this will be moved to object storage.
	ArchiveAfter time.Duration `yaml:"archiveAfter"`

	// KeepLocalCount is the minimum number of recent snapshots to keep locally.
	// Even if they're older than ArchiveAfter, these will stay local.
	KeepLocalCount int `yaml:"keepLocalCount"`

	// DeleteArchivedAfter is how long to keep archived snapshots.
	// Set to 0 to keep forever.
	DeleteArchivedAfter time.Duration `yaml:"deleteArchivedAfter"`

	// ExcludeBranches lists branches that should not be archived.
	ExcludeBranches []string `yaml:"excludeBranches"`

	// ScheduleInterval is how often to run the tiering check.
	ScheduleInterval time.Duration `yaml:"scheduleInterval"`
}

// DefaultTieringPolicy returns a sensible default tiering policy.
func DefaultTieringPolicy() TieringPolicy {
	return TieringPolicy{
		ArchiveAfter:        7 * 24 * time.Hour, // 7 days
		KeepLocalCount:      3,
		DeleteArchivedAfter: 90 * 24 * time.Hour, // 90 days
		ExcludeBranches:     []string{},
		ScheduleInterval:    6 * time.Hour,
	}
}

// TieringService manages automatic snapshot archival to object storage.
type TieringService struct {
	poolManager *PoolManager
	policy      TieringPolicy
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}

	snapshotProvider SnapshotProvider
}

// SnapshotProvider is an interface for getting snapshots.
// This allows integration with the existing ZFS manager.
type SnapshotProvider interface {
	ListSnapshots() []resources.Snapshot
	GetSnapshotByID(id string) (*resources.Snapshot, error)
}

// NewTieringService creates a new TieringService.
func NewTieringService(poolManager *PoolManager, policy TieringPolicy, provider SnapshotProvider) *TieringService {
	return &TieringService{
		poolManager:      poolManager,
		policy:           policy,
		snapshotProvider: provider,
		stopCh:           make(chan struct{}),
	}
}

// Start begins the automatic tiering background service.
func (ts *TieringService) Start(ctx context.Context) error {
	ts.mu.Lock()
	if ts.running {
		ts.mu.Unlock()
		return errors.New("tiering service already running")
	}

	ts.running = true
	ts.stopCh = make(chan struct{})
	ts.mu.Unlock()

	log.Msg("objbacker: starting tiering service, interval:", ts.policy.ScheduleInterval.String())

	go ts.runLoop(ctx)

	return nil
}

// Stop stops the tiering service.
func (ts *TieringService) Stop() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return
	}

	close(ts.stopCh)
	ts.running = false

	log.Msg("objbacker: tiering service stopped")
}

// IsRunning returns whether the tiering service is running.
func (ts *TieringService) IsRunning() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.running
}

// UpdatePolicy updates the tiering policy.
func (ts *TieringService) UpdatePolicy(policy TieringPolicy) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.policy = policy
}

// GetPolicy returns the current tiering policy.
func (ts *TieringService) GetPolicy() TieringPolicy {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.policy
}

// RunNow triggers an immediate tiering check.
func (ts *TieringService) RunNow(ctx context.Context) (*TieringResult, error) {
	return ts.performTiering(ctx)
}

// TieringResult contains the results of a tiering operation.
type TieringResult struct {
	Timestamp         time.Time       `json:"timestamp"`
	SnapshotsChecked  int             `json:"snapshotsChecked"`
	SnapshotsArchived int             `json:"snapshotsArchived"`
	SnapshotsDeleted  int             `json:"snapshotsDeleted"`
	BytesArchived     uint64          `json:"bytesArchived"`
	Errors            []string        `json:"errors,omitempty"`
	Details           []TieringAction `json:"details,omitempty"`
}

// TieringAction describes a single tiering action taken.
type TieringAction struct {
	SnapshotID string `json:"snapshotId"`
	Action     string `json:"action"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func (ts *TieringService) runLoop(ctx context.Context) {
	ticker := time.NewTicker(ts.policy.ScheduleInterval)
	defer ticker.Stop()

	if _, err := ts.performTiering(ctx); err != nil {
		log.Err("objbacker: initial tiering check failed", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ts.stopCh:
			return
		case <-ticker.C:
			if _, err := ts.performTiering(ctx); err != nil {
				log.Err("objbacker: tiering check failed", err)
			}
		}
	}
}

func (ts *TieringService) performTiering(ctx context.Context) (*TieringResult, error) {
	result := &TieringResult{
		Timestamp: time.Now(),
		Details:   make([]TieringAction, 0),
	}

	snapshots := ts.snapshotProvider.ListSnapshots()
	result.SnapshotsChecked = len(snapshots)

	log.Msg("objbacker: checking snapshots for tiering, count:", len(snapshots))

	candidates := ts.findArchiveCandidates(snapshots)

	for _, snapshot := range candidates {
		action := TieringAction{
			SnapshotID: snapshot.ID,
			Action:     "archive",
		}

		if err := ts.archiveSnapshot(ctx, snapshot); err != nil {
			action.Success = false
			action.Error = err.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("failed to archive %s: %v", snapshot.ID, err))
		} else {
			action.Success = true
			result.SnapshotsArchived++
		}

		result.Details = append(result.Details, action)
	}

	if ts.policy.DeleteArchivedAfter > 0 {
		deleted, deleteErrors := ts.cleanupOldArchives(ctx)
		result.SnapshotsDeleted = deleted
		result.Errors = append(result.Errors, deleteErrors...)
	}

	log.Msg("objbacker: tiering complete, archived:", result.SnapshotsArchived, "deleted:", result.SnapshotsDeleted, "errors:", len(result.Errors))

	return result, nil
}

func (ts *TieringService) findArchiveCandidates(snapshots []resources.Snapshot) []resources.Snapshot {
	ts.mu.RLock()
	policy := ts.policy
	ts.mu.RUnlock()

	now := time.Now()
	candidates := make([]resources.Snapshot, 0)

	branchExcluded := make(map[string]bool)
	for _, branch := range policy.ExcludeBranches {
		branchExcluded[branch] = true
	}

	branchSnapshots := make(map[string][]resources.Snapshot)
	for _, s := range snapshots {
		branchSnapshots[s.Branch] = append(branchSnapshots[s.Branch], s)
	}

	for branch, branchSnaps := range branchSnapshots {
		if branchExcluded[branch] {
			continue
		}

		for i, s := range branchSnaps {
			if i < policy.KeepLocalCount {
				continue
			}

			age := now.Sub(s.CreatedAt)
			if age > policy.ArchiveAfter {
				candidates = append(candidates, s)
			}
		}
	}

	return candidates
}

func (ts *TieringService) archiveSnapshot(ctx context.Context, snapshot resources.Snapshot) error {
	log.Msg("objbacker: archiving snapshot", snapshot.ID, "branch:", snapshot.Branch, "createdAt:", snapshot.CreatedAt)

	return ts.poolManager.MigrateSnapshotToObject(ctx, snapshot.ID)
}

func (ts *TieringService) cleanupOldArchives(ctx context.Context) (int, []string) {
	var deleted int
	var errs []string

	archived, err := ts.poolManager.ListArchivedSnapshots(ctx)
	if err != nil {
		return 0, []string{fmt.Sprintf("failed to list archived snapshots: %v", err)}
	}

	cutoff := time.Now().Add(-ts.policy.DeleteArchivedAfter)

	for _, snap := range archived {
		if snap.LastModified.Before(cutoff) {
			log.Msg("objbacker: deleting old archived snapshot", snap.Name, "lastModified:", snap.LastModified)
			deleted++
		}
	}

	return deleted, errs
}

// GetTieringStats returns statistics about tiered storage usage.
func (ts *TieringService) GetTieringStats(ctx context.Context) (*TieringStats, error) {
	archived, err := ts.poolManager.ListArchivedSnapshots(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list archived snapshots")
	}

	localSnapshots := ts.snapshotProvider.ListSnapshots()

	var archivedSize uint64
	for _, snap := range archived {
		archivedSize += snap.Size
	}

	stats := &TieringStats{
		LocalSnapshotCount:    len(localSnapshots),
		ArchivedSnapshotCount: len(archived),
		ArchivedTotalSize:     archivedSize,
		Policy:                ts.GetPolicy(),
	}

	if ts.poolManager.manager.IsEnabled() {
		estimate := ts.poolManager.manager.EstimateCost(archivedSize)
		stats.EstimatedMonthlyCost = estimate.StorageCostMonth
		stats.EstimatedSavings = estimate.SavingsPercent
	}

	return stats, nil
}

// TieringStats contains statistics about tiered storage.
type TieringStats struct {
	LocalSnapshotCount    int           `json:"localSnapshotCount"`
	ArchivedSnapshotCount int           `json:"archivedSnapshotCount"`
	ArchivedTotalSize     uint64        `json:"archivedTotalSize"`
	EstimatedMonthlyCost  float64       `json:"estimatedMonthlyCost"`
	EstimatedSavings      float64       `json:"estimatedSavings"`
	Policy                TieringPolicy `json:"policy"`
}
