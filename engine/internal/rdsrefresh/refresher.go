/*
2024 © Postgres.ai
*/

package rdsrefresh

import (
	"context"
	"fmt"
	"time"
)

// Logger defines the logging interface.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// DefaultLogger is a simple stdout logger.
type DefaultLogger struct{}

// Info logs an info message.
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

// Error logs an error message.
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}

// Debug logs a debug message.
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+msg+"\n", args...)
}

// Refresher orchestrates the RDS/Aurora clone and DBLab refresh workflow.
type Refresher struct {
	cfg    *Config
	rds    *RDSClient
	dblab  *DBLabClient
	logger Logger
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
func NewRefresher(ctx context.Context, cfg *Config, logger Logger) (*Refresher, error) {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	rdsClient, err := NewRDSClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	dblabClient := NewDBLabClient(&cfg.DBLab)

	return &Refresher{
		cfg:    cfg,
		rds:    rdsClient,
		dblab:  dblabClient,
		logger: logger,
	}, nil
}

// Run executes the full refresh workflow:
// 1. Verifies DBLab is healthy and not already refreshing
// 2. Finds the latest snapshot
// 3. Creates a temporary clone from the snapshot
// 4. Waits for the clone to be available
// 5. Triggers DBLab full refresh
// 6. Waits for refresh to complete
// 7. Deletes the temporary clone
func (r *Refresher) Run(ctx context.Context) *RefreshResult {
	result := &RefreshResult{
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// Step 1: Check DBLab health and status
	r.logger.Info("Checking DBLab Engine health...")

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

	// Step 2: Get source info
	r.logger.Info("Checking source database...")

	sourceInfo, err := r.rds.GetSourceInfo(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to get source info: %w", err)
		return result
	}

	r.logger.Info("Source: %s", sourceInfo)

	// Step 3: Find latest snapshot
	r.logger.Info("Finding latest snapshot...")

	snapshotID, err := r.rds.FindLatestSnapshot(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to find snapshot: %w", err)
		return result
	}

	result.SnapshotID = snapshotID
	r.logger.Info("Using snapshot: %s", snapshotID)

	// Step 4: Create temporary clone
	r.logger.Info("Creating temporary RDS clone from snapshot...")

	clone, err := r.rds.CreateClone(ctx, snapshotID)
	if err != nil {
		result.Error = fmt.Errorf("failed to create clone: %w", err)
		return result
	}

	result.CloneID = clone.Identifier
	r.logger.Info("Created clone: %s", clone.Identifier)

	// Ensure cleanup on any exit
	defer func() {
		r.logger.Info("Cleaning up temporary clone %s...", clone.Identifier)

		if deleteErr := r.rds.DeleteClone(context.Background(), clone); deleteErr != nil {
			r.logger.Error("Failed to delete clone %s: %v (manual cleanup may be required)", clone.Identifier, deleteErr)
		} else {
			r.logger.Info("Successfully deleted temporary clone %s", clone.Identifier)
		}
	}()

	// Step 5: Wait for clone to be available
	r.logger.Info("Waiting for clone to become available (this may take 10-30 minutes)...")

	if err := r.rds.WaitForCloneAvailable(ctx, clone); err != nil {
		result.Error = fmt.Errorf("clone did not become available: %w", err)
		return result
	}

	result.CloneEndpoint = clone.Endpoint
	r.logger.Info("Clone available at: %s:%d", clone.Endpoint, clone.Port)

	// Step 6: Trigger DBLab full refresh
	r.logger.Info("Triggering DBLab full refresh...")

	if err := r.dblab.TriggerFullRefresh(ctx); err != nil {
		result.Error = fmt.Errorf("failed to trigger refresh: %w", err)
		return result
	}

	r.logger.Info("Full refresh triggered, waiting for completion...")

	// Step 7: Wait for refresh to complete
	pollInterval := r.cfg.DBLab.PollInterval.Duration()
	timeout := r.cfg.DBLab.Timeout.Duration()

	if err := r.dblab.WaitForRefreshComplete(ctx, pollInterval, timeout); err != nil {
		result.Error = fmt.Errorf("refresh did not complete: %w", err)
		return result
	}

	r.logger.Info("DBLab refresh completed successfully!")
	result.Success = true

	return result
}

// DryRun performs all validation steps without actually creating resources.
func (r *Refresher) DryRun(ctx context.Context) error {
	r.logger.Info("=== DRY RUN MODE ===")

	// Check DBLab
	r.logger.Info("Checking DBLab Engine health...")

	if err := r.dblab.Health(ctx); err != nil {
		return fmt.Errorf("DBLab health check failed: %w", err)
	}

	r.logger.Info("DBLab Engine is healthy")

	// Check current status
	status, err := r.dblab.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DBLab status: %w", err)
	}

	r.logger.Info("DBLab retrieval status: %s", status.Retrieving.Status)

	// Check source
	r.logger.Info("Checking source database...")

	sourceInfo, err := r.rds.GetSourceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get source info: %w", err)
	}

	r.logger.Info("Source: %s", sourceInfo)

	// Check snapshot
	r.logger.Info("Finding latest snapshot...")

	snapshotID, err := r.rds.FindLatestSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find snapshot: %w", err)
	}

	r.logger.Info("Would use snapshot: %s", snapshotID)
	r.logger.Info("Would create clone with instance class: %s", r.cfg.Clone.InstanceClass)

	r.logger.Info("=== DRY RUN COMPLETE - All checks passed ===")

	return nil
}
