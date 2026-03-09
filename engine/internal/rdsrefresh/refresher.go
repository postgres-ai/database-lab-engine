/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// Refresher orchestrates the RDS/Aurora refresh workflow.
type Refresher struct {
	cfg       *Config
	rds       *RDSClient
	dblab     *DBLabClient
	stateFile *StateFile
}

// RefreshResult contains the result of a refresh operation.
type RefreshResult struct {
	Success       bool
	SnapshotID    string
	CloneID       string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Error         error
	CloneEndpoint string
}

// NewRefresher creates a new Refresher instance.
func NewRefresher(ctx context.Context, cfg *Config) (*Refresher, error) {
	return NewRefresherWithStateFile(ctx, cfg, nil)
}

// NewRefresherWithStateFile creates a new Refresher instance with state file tracking.
func NewRefresherWithStateFile(ctx context.Context, cfg *Config, stateFile *StateFile) (*Refresher, error) {
	rdsClient, err := NewRDSClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	dblabClient, err := NewDBLabClient(&cfg.DBLab)
	if err != nil {
		return nil, fmt.Errorf("failed to create DBLab client: %w", err)
	}

	return &Refresher{
		cfg:       cfg,
		rds:       rdsClient,
		dblab:     dblabClient,
		stateFile: stateFile,
	}, nil
}

// Run executes the full refresh workflow:
// 1. Verifies DBLab is healthy and not already refreshing
// 2. Gets source database info
// 3. Finds the latest RDS snapshot
// 4. Creates a temporary RDS clone from the RDS snapshot
// 5. Waits for the RDS clone to be available
// 6. Updates DBLab config with the RDS clone endpoint
// 7. Triggers DBLab full refresh
// 8. Waits for refresh to complete
// 9. Deletes the temporary RDS clone
func (r *Refresher) Run(ctx context.Context) *RefreshResult {
	result := &RefreshResult{
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// step 1: check DBLab health and status
	log.Msg("checking DBLab Engine health...")

	if err := r.dblab.Health(ctx); err != nil {
		result.Error = fmt.Errorf("DBLab health check failed: %w", err)
		return result
	}

	inProgress, err := r.dblab.IsRefreshInProgress(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to check DBLab status: %w", err)
		return result
	}

	if inProgress {
		result.Error = fmt.Errorf("refresh already in progress, skipping")
		return result
	}

	// step 2: get source info
	log.Msg("checking source database...")

	sourceInfo, err := r.rds.GetSourceInfo(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to get source info: %w", err)
		return result
	}

	log.Msg("source:", sourceInfo)

	// step 3: find latest RDS snapshot
	log.Msg("finding latest RDS snapshot...")

	snapshotID, err := r.rds.FindLatestSnapshot(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to find RDS snapshot: %w", err)
		return result
	}

	result.SnapshotID = snapshotID
	log.Msg("using RDS snapshot:", snapshotID)

	// step 4: create temporary RDS clone
	log.Msg("creating RDS clone from RDS snapshot...")

	// write state file before clone creation for crash recovery
	if r.stateFile != nil {
		if err := r.writeStateFile("", false, snapshotID); err != nil {
			log.Warn("failed to write initial state file:", err)
		}
	}

	clone, err := r.rds.CreateClone(ctx, snapshotID)
	if err != nil {
		r.clearStateFile()

		result.Error = fmt.Errorf("failed to create RDS clone: %w", err)

		return result
	}

	result.CloneID = clone.Identifier
	log.Msg("created RDS clone:", clone.Identifier)

	// update state file with actual clone ID
	if r.stateFile != nil {
		if err := r.writeStateFile(clone.Identifier, clone.IsCluster, snapshotID); err != nil {
			log.Warn("failed to update state file:", err)
		}
	}

	// ensure cleanup on any exit - use a separate context with timeout for cleanup
	// to ensure cleanup happens even if the main context is cancelled
	defer func() {
		const cleanupTimeout = 10 * time.Minute

		cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()

		log.Msg("deleting temporary RDS clone", clone.Identifier, "...")

		if deleteErr := r.rds.DeleteClone(cleanupCtx, clone); deleteErr != nil {
			log.Err("failed to delete RDS clone", clone.Identifier, ":", deleteErr, "(manual cleanup required)")
		} else {
			log.Msg("deleted RDS clone", clone.Identifier)
			r.clearStateFile()
		}
	}()

	// step 5: wait for RDS clone to be available
	log.Msg("waiting for RDS clone (10-30 min)...")

	if err := r.rds.WaitForCloneAvailable(ctx, clone); err != nil {
		result.Error = fmt.Errorf("RDS clone did not become available: %w", err)
		return result
	}

	result.CloneEndpoint = clone.Endpoint
	log.Msg("RDS clone ready:", fmt.Sprintf("%s:%d", clone.Endpoint, clone.Port))

	// step 6: update DBLab config with RDS clone endpoint
	log.Msg("updating DBLab config...")

	if err := r.dblab.UpdateSourceConfig(ctx, SourceConfigUpdate{
		Host:             clone.Endpoint,
		Port:             int(clone.Port),
		DBName:           r.cfg.Source.DBName,
		Username:         r.cfg.Source.Username,
		Password:         r.cfg.Source.Password,
		RDSIAMDBInstance: clone.Identifier,
	}); err != nil {
		result.Error = fmt.Errorf("failed to update DBLab config: %w", err)
		return result
	}

	log.Msg("DBLab config updated successfully")

	// step 7: trigger DBLab full refresh
	log.Msg("triggering DBLab full refresh...")

	if err := r.dblab.TriggerFullRefresh(ctx); err != nil {
		result.Error = fmt.Errorf("failed to trigger refresh: %w", err)
		return result
	}

	log.Msg("full refresh triggered, waiting for completion...")

	// step 8: wait for refresh to complete
	pollInterval := r.cfg.DBLab.PollInterval.Duration()

	timeout := r.cfg.DBLab.Timeout.Duration()

	if err := r.dblab.WaitForRefreshComplete(ctx, pollInterval, timeout); err != nil {
		result.Error = fmt.Errorf("refresh did not complete: %w", err)
		return result
	}

	log.Msg("DBLab refresh completed successfully!")

	result.Success = true

	return result
}

// DryRun performs all validation steps without actually creating resources.
func (r *Refresher) DryRun(ctx context.Context) error {
	log.Msg("=== DRY RUN MODE ===")

	// check DBLab
	log.Msg("checking DBLab Engine health...")

	if err := r.dblab.Health(ctx); err != nil {
		return fmt.Errorf("DBLab health check failed: %w", err)
	}

	log.Msg("DBLab Engine is healthy")

	// check current status
	status, err := r.dblab.GetStatus(ctx)

	if err != nil {
		return fmt.Errorf("failed to get DBLab status: %w", err)
	}

	log.Msg("DBLab retrieval status:", status.Retrieving.Status)

	// check source
	log.Msg("checking source database...")

	sourceInfo, err := r.rds.GetSourceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get source info: %w", err)
	}

	log.Msg("source:", sourceInfo)

	// check RDS snapshot
	log.Msg("finding latest RDS snapshot...")

	snapshotID, err := r.rds.FindLatestSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find RDS snapshot: %w", err)
	}

	log.Msg("would use RDS snapshot:", snapshotID)
	log.Msg("would create RDS clone with instance class:", r.cfg.RDSClone.InstanceClass)

	log.Msg("=== DRY RUN COMPLETE - all checks passed ===")

	return nil
}

func (r *Refresher) writeStateFile(cloneID string, isCluster bool, sourceID string) error {
	if r.stateFile == nil {
		return nil
	}

	state := &CloneState{
		CloneID:     cloneID,
		IsCluster:   isCluster,
		AWSRegion:   r.cfg.AWS.Region,
		CreatedAt:   time.Now().UTC(),
		DeleteAfter: r.cfg.GetDeleteAfterTime(),
		SourceID:    sourceID,
	}

	return r.stateFile.Write(state)
}

func (r *Refresher) clearStateFile() {
	if r.stateFile == nil {
		return
	}

	if err := r.stateFile.Clear(); err != nil {
		log.Warn("failed to clear state file:", err)
	}
}
