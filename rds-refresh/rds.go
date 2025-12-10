/*
2024 © Postgres.ai
*/

package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

const (
	cloneNamePrefix        = "dblab-refresh-"
	waitPollInterval       = 30 * time.Second
	maxWaitTime            = 2 * time.Hour
	defaultPort      int32 = 5432
)

// RDSClient wraps the AWS RDS client with convenience methods.
type RDSClient struct {
	client *rds.Client
	cfg    *Config
}

// CloneInfo holds information about a created clone.
type CloneInfo struct {
	Identifier string
	Endpoint   string
	Port       int32
	IsCluster  bool
}

// NewRDSClient creates a new RDS client.
func NewRDSClient(ctx context.Context, cfg *Config) (*RDSClient, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*rds.Options)
	if cfg.AWS.Endpoint != "" {
		opts = append(opts, func(o *rds.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	}

	return &RDSClient{
		client: rds.NewFromConfig(awsCfg, opts...),
		cfg:    cfg,
	}, nil
}

// FindLatestSnapshot finds the latest available snapshot for the source.
func (r *RDSClient) FindLatestSnapshot(ctx context.Context) (string, error) {
	if r.cfg.Source.SnapshotIdentifier != "" {
		return r.cfg.Source.SnapshotIdentifier, nil
	}

	if r.cfg.Source.Type == "aurora-cluster" {
		return r.findLatestClusterSnapshot(ctx)
	}

	return r.findLatestDBSnapshot(ctx)
}

func (r *RDSClient) findLatestDBSnapshot(ctx context.Context) (string, error) {
	input := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(r.cfg.Source.Identifier),
		SnapshotType:         aws.String("automated"),
	}

	result, err := r.client.DescribeDBSnapshots(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe DB snapshots: %w", err)
	}

	if len(result.DBSnapshots) == 0 {
		return "", fmt.Errorf("no automated snapshots found for RDS instance %q", r.cfg.Source.Identifier)
	}

	// Sort by creation time (newest first)
	sort.Slice(result.DBSnapshots, func(i, j int) bool {
		ti := result.DBSnapshots[i].SnapshotCreateTime
		tj := result.DBSnapshots[j].SnapshotCreateTime

		if ti == nil || tj == nil {
			return ti != nil
		}

		return ti.After(*tj)
	})

	// Find the first available snapshot
	for _, snap := range result.DBSnapshots {
		if snap.Status != nil && *snap.Status == "available" {
			return *snap.DBSnapshotIdentifier, nil
		}
	}

	return "", fmt.Errorf("no available snapshots found for RDS instance %q", r.cfg.Source.Identifier)
}

func (r *RDSClient) findLatestClusterSnapshot(ctx context.Context) (string, error) {
	input := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier: aws.String(r.cfg.Source.Identifier),
		SnapshotType:        aws.String("automated"),
	}

	result, err := r.client.DescribeDBClusterSnapshots(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe DB cluster snapshots: %w", err)
	}

	if len(result.DBClusterSnapshots) == 0 {
		return "", fmt.Errorf("no automated snapshots found for Aurora cluster %q", r.cfg.Source.Identifier)
	}

	// Sort by creation time (newest first)
	sort.Slice(result.DBClusterSnapshots, func(i, j int) bool {
		ti := result.DBClusterSnapshots[i].SnapshotCreateTime
		tj := result.DBClusterSnapshots[j].SnapshotCreateTime

		if ti == nil || tj == nil {
			return ti != nil
		}

		return ti.After(*tj)
	})

	// Find the first available snapshot
	for _, snap := range result.DBClusterSnapshots {
		if snap.Status != nil && *snap.Status == "available" {
			return *snap.DBClusterSnapshotIdentifier, nil
		}
	}

	return "", fmt.Errorf("no available snapshots found for Aurora cluster %q", r.cfg.Source.Identifier)
}

// CreateClone creates a temporary clone from a snapshot.
func (r *RDSClient) CreateClone(ctx context.Context, snapshotID string) (*CloneInfo, error) {
	cloneName := fmt.Sprintf("%s%s", cloneNamePrefix, time.Now().UTC().Format("20060102-150405"))

	if r.cfg.Source.Type == "aurora-cluster" {
		return r.createAuroraClone(ctx, snapshotID, cloneName)
	}

	return r.createRDSClone(ctx, snapshotID, cloneName)
}

func (r *RDSClient) createRDSClone(ctx context.Context, snapshotID, cloneName string) (*CloneInfo, error) {
	tags := r.buildTags()

	input := &rds.RestoreDBInstanceFromDBSnapshotInput{
		DBInstanceIdentifier: aws.String(cloneName),
		DBSnapshotIdentifier: aws.String(snapshotID),
		DBInstanceClass:      aws.String(r.cfg.Clone.InstanceClass),
		PubliclyAccessible:   aws.Bool(r.cfg.Clone.PubliclyAccessible),
		Tags:                 tags,
		DeletionProtection:   aws.Bool(r.cfg.Clone.DeletionProtection),
	}

	if r.cfg.Clone.DBSubnetGroupName != "" {
		input.DBSubnetGroupName = aws.String(r.cfg.Clone.DBSubnetGroupName)
	}

	if len(r.cfg.Clone.VPCSecurityGroupIDs) > 0 {
		input.VpcSecurityGroupIds = r.cfg.Clone.VPCSecurityGroupIDs
	}

	if r.cfg.Clone.ParameterGroupName != "" {
		input.DBParameterGroupName = aws.String(r.cfg.Clone.ParameterGroupName)
	}

	if r.cfg.Clone.OptionGroupName != "" {
		input.OptionGroupName = aws.String(r.cfg.Clone.OptionGroupName)
	}

	if r.cfg.Clone.Port > 0 {
		input.Port = aws.Int32(r.cfg.Clone.Port)
	}

	if r.cfg.Clone.EnableIAMAuth {
		input.EnableIAMDatabaseAuthentication = aws.Bool(true)
	}

	if r.cfg.Clone.StorageType != "" {
		input.StorageType = aws.String(r.cfg.Clone.StorageType)
	}

	_, err := r.client.RestoreDBInstanceFromDBSnapshot(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to restore DB instance from snapshot: %w", err)
	}

	return &CloneInfo{
		Identifier: cloneName,
		IsCluster:  false,
	}, nil
}

func (r *RDSClient) createAuroraClone(ctx context.Context, snapshotID, cloneName string) (*CloneInfo, error) {
	tags := r.buildTags()

	// Get the engine from the snapshot first
	snapshotResp, err := r.client.DescribeDBClusterSnapshots(ctx, &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: aws.String(snapshotID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster snapshot: %w", err)
	}

	if len(snapshotResp.DBClusterSnapshots) == 0 {
		return nil, fmt.Errorf("snapshot %q not found", snapshotID)
	}

	snapshot := snapshotResp.DBClusterSnapshots[0]

	// Restore the Aurora cluster
	clusterInput := &rds.RestoreDBClusterFromSnapshotInput{
		DBClusterIdentifier: aws.String(cloneName),
		SnapshotIdentifier:  aws.String(snapshotID),
		Engine:              snapshot.Engine,
		Tags:                tags,
		DeletionProtection:  aws.Bool(r.cfg.Clone.DeletionProtection),
	}

	if r.cfg.Clone.DBSubnetGroupName != "" {
		clusterInput.DBSubnetGroupName = aws.String(r.cfg.Clone.DBSubnetGroupName)
	}

	if len(r.cfg.Clone.VPCSecurityGroupIDs) > 0 {
		clusterInput.VpcSecurityGroupIds = r.cfg.Clone.VPCSecurityGroupIDs
	}

	if r.cfg.Clone.DBClusterParameterGroupName != "" {
		clusterInput.DBClusterParameterGroupName = aws.String(r.cfg.Clone.DBClusterParameterGroupName)
	}

	if r.cfg.Clone.Port > 0 {
		clusterInput.Port = aws.Int32(r.cfg.Clone.Port)
	}

	if r.cfg.Clone.EnableIAMAuth {
		clusterInput.EnableIAMDatabaseAuthentication = aws.Bool(true)
	}

	_, err = r.client.RestoreDBClusterFromSnapshot(ctx, clusterInput)
	if err != nil {
		return nil, fmt.Errorf("failed to restore DB cluster from snapshot: %w", err)
	}

	// Wait for cluster to be available before creating instance
	if err := r.waitForClusterAvailable(ctx, cloneName); err != nil {
		// Try to clean up the cluster
		_ = r.deleteAuroraCluster(ctx, cloneName)
		return nil, fmt.Errorf("cluster did not become available: %w", err)
	}

	// Create a DB instance in the cluster
	instanceName := cloneName + "-instance"
	instanceInput := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceName),
		DBInstanceClass:      aws.String(r.cfg.Clone.InstanceClass),
		DBClusterIdentifier:  aws.String(cloneName),
		Engine:               snapshot.Engine,
		Tags:                 tags,
	}

	if r.cfg.Clone.ParameterGroupName != "" {
		instanceInput.DBParameterGroupName = aws.String(r.cfg.Clone.ParameterGroupName)
	}

	_, err = r.client.CreateDBInstance(ctx, instanceInput)
	if err != nil {
		// Try to clean up the cluster
		_ = r.deleteAuroraCluster(ctx, cloneName)
		return nil, fmt.Errorf("failed to create DB instance in cluster: %w", err)
	}

	return &CloneInfo{
		Identifier: cloneName,
		IsCluster:  true,
	}, nil
}

func (r *RDSClient) buildTags() []types.Tag {
	tags := make([]types.Tag, 0, len(r.cfg.Clone.Tags))

	for k, v := range r.cfg.Clone.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	return tags
}

// WaitForCloneAvailable waits for the clone to become available and returns connection info.
func (r *RDSClient) WaitForCloneAvailable(ctx context.Context, clone *CloneInfo) error {
	if clone.IsCluster {
		instanceName := clone.Identifier + "-instance"

		if err := r.waitForInstanceAvailable(ctx, instanceName); err != nil {
			return err
		}

		// Get the cluster endpoint
		clusterResp, err := r.client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(clone.Identifier),
		})
		if err != nil {
			return fmt.Errorf("failed to describe cluster: %w", err)
		}

		if len(clusterResp.DBClusters) == 0 {
			return fmt.Errorf("cluster %q not found", clone.Identifier)
		}

		cluster := clusterResp.DBClusters[0]
		clone.Endpoint = aws.ToString(cluster.Endpoint)
		clone.Port = aws.ToInt32(cluster.Port)

		if clone.Port == 0 {
			clone.Port = defaultPort
		}

		return nil
	}

	if err := r.waitForInstanceAvailable(ctx, clone.Identifier); err != nil {
		return err
	}

	// Get the instance endpoint
	instanceResp, err := r.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(clone.Identifier),
	})
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(instanceResp.DBInstances) == 0 {
		return fmt.Errorf("instance %q not found", clone.Identifier)
	}

	instance := instanceResp.DBInstances[0]

	if instance.Endpoint != nil {
		clone.Endpoint = aws.ToString(instance.Endpoint.Address)
		clone.Port = aws.ToInt32(instance.Endpoint.Port)
	}

	if clone.Port == 0 {
		clone.Port = defaultPort
	}

	return nil
}

func (r *RDSClient) waitForInstanceAvailable(ctx context.Context, identifier string) error {
	waiter := rds.NewDBInstanceAvailableWaiter(r.client)

	return waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(identifier),
	}, maxWaitTime)
}

func (r *RDSClient) waitForClusterAvailable(ctx context.Context, identifier string) error {
	waiter := rds.NewDBClusterAvailableWaiter(r.client)

	return waiter.Wait(ctx, &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(identifier),
	}, maxWaitTime)
}

// DeleteClone deletes the temporary clone.
func (r *RDSClient) DeleteClone(ctx context.Context, clone *CloneInfo) error {
	if clone.IsCluster {
		return r.deleteAuroraCluster(ctx, clone.Identifier)
	}

	return r.deleteRDSInstance(ctx, clone.Identifier)
}

func (r *RDSClient) deleteRDSInstance(ctx context.Context, identifier string) error {
	// First, disable deletion protection if enabled
	_, _ = r.client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(identifier),
		DeletionProtection:   aws.Bool(false),
		ApplyImmediately:     aws.Bool(true),
	})

	_, err := r.client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   aws.String(identifier),
		SkipFinalSnapshot:      aws.Bool(true),
		DeleteAutomatedBackups: aws.Bool(true),
	})

	if err != nil {
		return fmt.Errorf("failed to delete DB instance: %w", err)
	}

	return nil
}

func (r *RDSClient) deleteAuroraCluster(ctx context.Context, clusterIdentifier string) error {
	// First, delete all instances in the cluster
	instancesResp, err := r.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("db-cluster-id"),
				Values: []string{clusterIdentifier},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list cluster instances: %w", err)
	}

	for _, instance := range instancesResp.DBInstances {
		if err := r.deleteRDSInstance(ctx, aws.ToString(instance.DBInstanceIdentifier)); err != nil {
			return fmt.Errorf("failed to delete cluster instance: %w", err)
		}
	}

	// Wait for all instances to be deleted
	for _, instance := range instancesResp.DBInstances {
		waiter := rds.NewDBInstanceDeletedWaiter(r.client)

		if err := waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
		}, maxWaitTime); err != nil {
			return fmt.Errorf("failed waiting for instance deletion: %w", err)
		}
	}

	// Disable deletion protection on cluster
	_, _ = r.client.ModifyDBCluster(ctx, &rds.ModifyDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		DeletionProtection:  aws.Bool(false),
		ApplyImmediately:    aws.Bool(true),
	})

	// Delete the cluster
	_, err = r.client.DeleteDBCluster(ctx, &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		SkipFinalSnapshot:   aws.Bool(true),
	})

	if err != nil {
		return fmt.Errorf("failed to delete DB cluster: %w", err)
	}

	return nil
}

// GetSourceInfo returns information about the source database.
func (r *RDSClient) GetSourceInfo(ctx context.Context) (string, error) {
	if r.cfg.Source.Type == "aurora-cluster" {
		resp, err := r.client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(r.cfg.Source.Identifier),
		})
		if err != nil {
			return "", fmt.Errorf("failed to describe source cluster: %w", err)
		}

		if len(resp.DBClusters) == 0 {
			return "", fmt.Errorf("source cluster %q not found", r.cfg.Source.Identifier)
		}

		cluster := resp.DBClusters[0]

		return fmt.Sprintf("Aurora cluster %s (engine: %s, version: %s)",
			r.cfg.Source.Identifier,
			aws.ToString(cluster.Engine),
			aws.ToString(cluster.EngineVersion)), nil
	}

	resp, err := r.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(r.cfg.Source.Identifier),
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe source instance: %w", err)
	}

	if len(resp.DBInstances) == 0 {
		return "", fmt.Errorf("source instance %q not found", r.cfg.Source.Identifier)
	}

	instance := resp.DBInstances[0]

	return fmt.Sprintf("RDS instance %s (engine: %s, version: %s)",
		r.cfg.Source.Identifier,
		aws.ToString(instance.Engine),
		aws.ToString(instance.EngineVersion)), nil
}
