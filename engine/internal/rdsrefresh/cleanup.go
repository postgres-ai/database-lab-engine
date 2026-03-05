/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	stateFileName = "rds-refresh.state"
	stateFileMode = 0600
	stateDirMode  = 0700
)

// CloneState represents the state of a clone being managed.
type CloneState struct {
	CloneID     string    `json:"clone_id"`
	IsCluster   bool      `json:"is_cluster"`
	AWSRegion   string    `json:"aws_region"`
	CreatedAt   time.Time `json:"created_at"`
	DeleteAfter time.Time `json:"delete_after"`
	SourceID    string    `json:"source_id"`
}

// StateFile manages the state file for tracking active clones.
type StateFile struct {
	path string
}

// StaleClone represents a clone that should be deleted.
type StaleClone struct {
	Identifier  string
	IsCluster   bool
	CreatedAt   time.Time
	DeleteAfter time.Time
	Reason      string
}

// CleanupResult contains the result of a cleanup operation.
type CleanupResult struct {
	ClonesFound   int
	ClonesDeleted int
	ClonesFailed  int
	Errors        []error
}

// NewStateFile creates a new StateFile manager.
// If stateDir is empty, uses the DBLab meta directory (./meta/).
func NewStateFile(stateDir string) *StateFile {
	if stateDir != "" {
		return &StateFile{path: filepath.Join(stateDir, stateFileName)}
	}

	// use DBLab meta directory by default
	metaPath, err := util.GetMetaPath(stateFileName)
	if err != nil {
		// fallback to current directory if meta path cannot be determined
		return &StateFile{path: stateFileName}
	}

	return &StateFile{path: metaPath}
}

// Write saves the clone state to the state file.
func (s *StateFile) Write(state *CloneState) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, stateDirMode); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.path, data, stateFileMode); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Read loads the clone state from the state file.
func (s *StateFile) Read() (*CloneState, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state CloneState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// Clear removes the state file.
func (s *StateFile) Clear() error {
	err := os.Remove(s.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}

// Exists checks if the state file exists.
func (s *StateFile) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// FindStaleClones searches for clones that should be deleted based on tags.
func (r *RDSClient) FindStaleClones(ctx context.Context) ([]StaleClone, error) {
	var staleClones []StaleClone

	// find stale RDS instances
	instances, err := r.findStaleInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale instances: %w", err)
	}

	staleClones = append(staleClones, instances...)

	// find stale Aurora clusters
	clusters, err := r.findStaleClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale clusters: %w", err)
	}

	staleClones = append(staleClones, clusters...)

	return staleClones, nil
}

func (r *RDSClient) findStaleInstances(ctx context.Context) ([]StaleClone, error) {
	input := &rds.DescribeDBInstancesInput{}

	result, err := r.client.DescribeDBInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB instances: %w", err)
	}

	staleClones := make([]StaleClone, 0)

	for _, instance := range result.DBInstances {
		if instance.DBInstanceIdentifier == nil {
			continue
		}

		identifier := aws.ToString(instance.DBInstanceIdentifier)

		// skip if not a dblab-refresh clone (by name prefix)
		if !strings.HasPrefix(identifier, cloneNamePrefix) {
			continue
		}

		// skip if part of a cluster (will be handled by cluster cleanup)
		if instance.DBClusterIdentifier != nil && aws.ToString(instance.DBClusterIdentifier) != "" {
			continue
		}

		stale, reason := r.isStaleByTags(instance.TagList, instance.InstanceCreateTime)
		if !stale {
			continue
		}

		clone := StaleClone{
			Identifier: identifier,
			IsCluster:  false,
			Reason:     reason,
		}

		if instance.InstanceCreateTime != nil {
			clone.CreatedAt = *instance.InstanceCreateTime
		}

		staleClones = append(staleClones, clone)
	}

	return staleClones, nil
}

func (r *RDSClient) findStaleClusters(ctx context.Context) ([]StaleClone, error) {
	input := &rds.DescribeDBClustersInput{}

	result, err := r.client.DescribeDBClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB clusters: %w", err)
	}

	staleClones := make([]StaleClone, 0)

	for _, cluster := range result.DBClusters {
		if cluster.DBClusterIdentifier == nil {
			continue
		}

		identifier := aws.ToString(cluster.DBClusterIdentifier)

		// skip if not a dblab-refresh clone (by name prefix)
		if !strings.HasPrefix(identifier, cloneNamePrefix) {
			continue
		}

		stale, reason := r.isStaleByTags(cluster.TagList, cluster.ClusterCreateTime)
		if !stale {
			continue
		}

		clone := StaleClone{
			Identifier: identifier,
			IsCluster:  true,
			Reason:     reason,
		}

		if cluster.ClusterCreateTime != nil {
			clone.CreatedAt = *cluster.ClusterCreateTime
		}

		staleClones = append(staleClones, clone)
	}

	return staleClones, nil
}

func (r *RDSClient) isStaleByTags(tags []types.Tag, createTime *time.Time) (bool, string) {
	var managedBy, autoDelete, deleteAfterStr string

	for _, tag := range tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)

		switch key {
		case ManagedByTagKey:
			managedBy = value
		case AutoDeleteTagKey:
			autoDelete = value
		case DeleteAfterTagKey:
			deleteAfterStr = value
		}
	}

	// must be managed by this tool
	if managedBy != ManagedByTagValue {
		return false, ""
	}

	// must be marked for auto-delete
	if autoDelete != "true" {
		return false, ""
	}

	// check DeleteAfter tag first (preferred method)
	if deleteAfterStr != "" {
		deleteAfter, err := time.Parse(time.RFC3339, deleteAfterStr)
		if err == nil && time.Now().UTC().After(deleteAfter) {
			return true, fmt.Sprintf("DeleteAfter tag expired at %s", deleteAfter.Format(time.RFC3339))
		}
	}

	// fallback: check creation time against max age
	maxAge := r.cfg.RDSClone.MaxAge.Duration()
	if createTime != nil && time.Since(*createTime) > maxAge {
		return true, fmt.Sprintf("clone age %s exceeds max age %s", time.Since(*createTime).Round(time.Minute), maxAge)
	}

	return false, ""
}

// DeleteStaleClone deletes a stale clone.
func (r *RDSClient) DeleteStaleClone(ctx context.Context, clone StaleClone) error {
	cloneInfo := &CloneInfo{
		Identifier: clone.Identifier,
		IsCluster:  clone.IsCluster,
	}

	return r.DeleteClone(ctx, cloneInfo)
}

// CleanupStaleClones finds and deletes all stale clones.
func (r *RDSClient) CleanupStaleClones(ctx context.Context, dryRun bool) (*CleanupResult, error) {
	result := &CleanupResult{}

	staleClones, err := r.FindStaleClones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale clones: %w", err)
	}

	result.ClonesFound = len(staleClones)

	for _, clone := range staleClones {
		cloneType := "instance"
		if clone.IsCluster {
			cloneType = "cluster"
		}

		if dryRun {
			log.Msg("would delete stale", cloneType, clone.Identifier, "-", clone.Reason)
			continue
		}

		log.Msg("deleting stale", cloneType, clone.Identifier, "-", clone.Reason)

		if err := r.DeleteStaleClone(ctx, clone); err != nil {
			log.Err("failed to delete", cloneType, clone.Identifier, ":", err)

			result.ClonesFailed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to delete %s %s: %w", cloneType, clone.Identifier, err))

			continue
		}

		log.Msg("deleted stale", cloneType, clone.Identifier)

		result.ClonesDeleted++
	}

	return result, nil
}

// CleanupFromStateFile attempts to clean up a clone recorded in the state file.
func (r *RDSClient) CleanupFromStateFile(ctx context.Context, stateFile *StateFile) error {
	state, err := stateFile.Read()
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if state == nil {
		return nil
	}

	// check if clone still exists and should be deleted
	if state.CloneID == "" {
		log.Msg("state file found but no clone ID recorded, clearing state")
		return stateFile.Clear()
	}

	log.Msg("found orphaned clone in state file:", state.CloneID)

	cloneInfo := &CloneInfo{
		Identifier: state.CloneID,
		IsCluster:  state.IsCluster,
	}

	// try to delete the clone
	if err := r.DeleteClone(ctx, cloneInfo); err != nil {
		// check if clone doesn't exist (already deleted)
		if strings.Contains(err.Error(), "DBInstanceNotFound") ||
			strings.Contains(err.Error(), "DBClusterNotFound") {
			log.Msg("clone", state.CloneID, "already deleted, clearing state")
			return stateFile.Clear()
		}

		return fmt.Errorf("failed to delete orphaned clone %s: %w", state.CloneID, err)
	}

	log.Msg("deleted orphaned clone:", state.CloneID)

	return stateFile.Clear()
}
