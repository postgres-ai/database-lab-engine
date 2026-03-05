/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	cloneNamePrefix       = "dblab-refresh-"
	maxWaitTime           = 2 * time.Hour
	defaultPort     int32 = 5432
)

// RDSAPI defines the interface for RDS client operations.
// this interface enables mocking for unit tests.
//
//nolint:dupl,lll
type RDSAPI interface {
	DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error)
	RestoreDBInstanceFromDBSnapshot(ctx context.Context, params *rds.RestoreDBInstanceFromDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error)
	RestoreDBClusterFromSnapshot(ctx context.Context, params *rds.RestoreDBClusterFromSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBClusterFromSnapshotOutput, error)
	CreateDBInstance(ctx context.Context, params *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error)
	DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error)
}

// RDSClient wraps the AWS RDS client with convenience methods.
type RDSClient struct {
	client RDSAPI
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

// NewRDSClientWithAPI creates a new RDS client with a custom API client.
// this is primarily used for testing with mock implementations.
func NewRDSClientWithAPI(api RDSAPI, cfg *Config) *RDSClient {
	return &RDSClient{
		client: api,
		cfg:    cfg,
	}
}

// FindLatestSnapshot finds the latest available snapshot for the source.
func (r *RDSClient) FindLatestSnapshot(ctx context.Context) (string, error) {
	if r.cfg.Source.SnapshotIdentifier != "" {
		return r.cfg.Source.SnapshotIdentifier, nil
	}

	if r.cfg.Source.Type == sourceTypeAuroraCluster {
		return r.findLatestClusterSnapshot(ctx)
	}

	return r.findLatestDBSnapshot(ctx)
}

//nolint:dupl
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

	// sort by creation time (newest first)
	sort.Slice(result.DBSnapshots, func(i, j int) bool {
		ti := result.DBSnapshots[i].SnapshotCreateTime
		tj := result.DBSnapshots[j].SnapshotCreateTime

		if ti == nil || tj == nil {
			return ti != nil
		}

		return ti.After(*tj)
	})

	// find the first available snapshot
	for _, snap := range result.DBSnapshots {
		if snap.Status != nil && *snap.Status == "available" && snap.DBSnapshotIdentifier != nil {
			return *snap.DBSnapshotIdentifier, nil
		}
	}

	return "", fmt.Errorf("no available snapshots found for RDS instance %q", r.cfg.Source.Identifier)
}

//nolint:dupl
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

	// sort by creation time (newest first)
	sort.Slice(result.DBClusterSnapshots, func(i, j int) bool {
		ti := result.DBClusterSnapshots[i].SnapshotCreateTime
		tj := result.DBClusterSnapshots[j].SnapshotCreateTime

		if ti == nil || tj == nil {
			return ti != nil
		}

		return ti.After(*tj)
	})

	// find the first available snapshot
	for _, snap := range result.DBClusterSnapshots {
		if snap.Status != nil && *snap.Status == "available" && snap.DBClusterSnapshotIdentifier != nil {
			return *snap.DBClusterSnapshotIdentifier, nil
		}
	}

	return "", fmt.Errorf("no available snapshots found for Aurora cluster %q", r.cfg.Source.Identifier)
}

// CreateClone creates a temporary clone from a snapshot.
func (r *RDSClient) CreateClone(ctx context.Context, snapshotID string) (*CloneInfo, error) {
	cloneName := fmt.Sprintf("%s%s", cloneNamePrefix, time.Now().UTC().Format("20060102-150405"))

	if r.cfg.Source.Type == sourceTypeAuroraCluster {
		return r.createAuroraClone(ctx, snapshotID, cloneName)
	}

	return r.createRDSClone(ctx, snapshotID, cloneName)
}

func (r *RDSClient) createRDSClone(ctx context.Context, snapshotID, cloneName string) (*CloneInfo, error) {
	tags := r.buildTags()

	input := &rds.RestoreDBInstanceFromDBSnapshotInput{
		DBInstanceIdentifier: aws.String(cloneName),
		DBSnapshotIdentifier: aws.String(snapshotID),
		DBInstanceClass:      aws.String(r.cfg.RDSClone.InstanceClass),
		PubliclyAccessible:   aws.Bool(r.cfg.RDSClone.PubliclyAccessible),
		Tags:                 tags,
		DeletionProtection:   aws.Bool(r.cfg.RDSClone.DeletionProtection),
	}

	if r.cfg.RDSClone.DBSubnetGroupName != "" {
		input.DBSubnetGroupName = aws.String(r.cfg.RDSClone.DBSubnetGroupName)
	}

	if len(r.cfg.RDSClone.VPCSecurityGroupIDs) > 0 {
		input.VpcSecurityGroupIds = r.cfg.RDSClone.VPCSecurityGroupIDs
	}

	if r.cfg.RDSClone.ParameterGroupName != "" {
		input.DBParameterGroupName = aws.String(r.cfg.RDSClone.ParameterGroupName)
	}

	if r.cfg.RDSClone.OptionGroupName != "" {
		input.OptionGroupName = aws.String(r.cfg.RDSClone.OptionGroupName)
	}

	if r.cfg.RDSClone.Port > 0 {
		input.Port = aws.Int32(r.cfg.RDSClone.Port)
	}

	if r.cfg.RDSClone.EnableIAMAuth {
		input.EnableIAMDatabaseAuthentication = aws.Bool(true)
	}

	if r.cfg.RDSClone.StorageType != "" {
		input.StorageType = aws.String(r.cfg.RDSClone.StorageType)
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

	// get the engine from the snapshot first
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

	// restore the Aurora cluster
	clusterInput := &rds.RestoreDBClusterFromSnapshotInput{
		DBClusterIdentifier: aws.String(cloneName),
		SnapshotIdentifier:  aws.String(snapshotID),
		Engine:              snapshot.Engine,
		Tags:                tags,
		DeletionProtection:  aws.Bool(r.cfg.RDSClone.DeletionProtection),
	}

	if r.cfg.RDSClone.DBSubnetGroupName != "" {
		clusterInput.DBSubnetGroupName = aws.String(r.cfg.RDSClone.DBSubnetGroupName)
	}

	if len(r.cfg.RDSClone.VPCSecurityGroupIDs) > 0 {
		clusterInput.VpcSecurityGroupIds = r.cfg.RDSClone.VPCSecurityGroupIDs
	}

	if r.cfg.RDSClone.DBClusterParameterGroupName != "" {
		clusterInput.DBClusterParameterGroupName = aws.String(r.cfg.RDSClone.DBClusterParameterGroupName)
	}

	if r.cfg.RDSClone.Port > 0 {
		clusterInput.Port = aws.Int32(r.cfg.RDSClone.Port)
	}

	if r.cfg.RDSClone.EnableIAMAuth {
		clusterInput.EnableIAMDatabaseAuthentication = aws.Bool(true)
	}

	_, err = r.client.RestoreDBClusterFromSnapshot(ctx, clusterInput)
	if err != nil {
		return nil, fmt.Errorf("failed to restore DB cluster from snapshot: %w", err)
	}

	// wait for cluster to be available before creating instance
	if err := r.waitForClusterAvailable(ctx, cloneName); err != nil {
		// try to clean up the cluster
		_ = r.deleteAuroraCluster(ctx, cloneName)
		return nil, fmt.Errorf("cluster did not become available: %w", err)
	}

	// create a DB instance in the cluster
	instanceName := cloneName + "-instance"
	instanceInput := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceName),
		DBInstanceClass:      aws.String(r.cfg.RDSClone.InstanceClass),
		DBClusterIdentifier:  aws.String(cloneName),
		Engine:               snapshot.Engine,
		Tags:                 tags,
	}

	if r.cfg.RDSClone.ParameterGroupName != "" {
		instanceInput.DBParameterGroupName = aws.String(r.cfg.RDSClone.ParameterGroupName)
	}

	_, err = r.client.CreateDBInstance(ctx, instanceInput)
	if err != nil {
		// try to clean up the cluster
		_ = r.deleteAuroraCluster(ctx, cloneName)
		return nil, fmt.Errorf("failed to create DB instance in cluster: %w", err)
	}

	return &CloneInfo{
		Identifier: cloneName,
		IsCluster:  true,
	}, nil
}

func (r *RDSClient) buildTags() []types.Tag {
	tags := make([]types.Tag, 0, len(r.cfg.RDSClone.Tags)+1)

	for k, v := range r.cfg.RDSClone.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// add DeleteAfter tag with timestamp for stale clone cleanup
	deleteAfter := r.cfg.GetDeleteAfterTime()
	tags = append(tags, types.Tag{
		Key:   aws.String(DeleteAfterTagKey),
		Value: aws.String(deleteAfter.Format(time.RFC3339)),
	})

	return tags
}

// WaitForCloneAvailable waits for the clone to become available and returns connection info.
func (r *RDSClient) WaitForCloneAvailable(ctx context.Context, clone *CloneInfo) error {
	if clone.IsCluster {
		instanceName := clone.Identifier + "-instance"

		if err := r.waitForInstanceAvailable(ctx, instanceName); err != nil {
			return err
		}

		// get the cluster endpoint
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

	// get the instance endpoint
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

	if instance.Endpoint == nil || instance.Endpoint.Address == nil {
		return fmt.Errorf("instance %q has no endpoint available", clone.Identifier)
	}

	clone.Endpoint = aws.ToString(instance.Endpoint.Address)
	clone.Port = aws.ToInt32(instance.Endpoint.Port)

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
	// first, disable deletion protection if enabled
	_, err := r.client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(identifier),
		DeletionProtection:   aws.Bool(false),
		ApplyImmediately:     aws.Bool(true),
	})
	if err != nil {
		log.Warn("failed to disable deletion protection on instance", identifier, ":", err)
	}

	_, err = r.client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
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
	// first, delete all instances in the cluster
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

	// wait for all instances to be deleted
	for _, instance := range instancesResp.DBInstances {
		waiter := rds.NewDBInstanceDeletedWaiter(r.client)

		if err := waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
		}, maxWaitTime); err != nil {
			return fmt.Errorf("failed waiting for instance deletion: %w", err)
		}
	}

	// disable deletion protection on cluster
	_, err = r.client.ModifyDBCluster(ctx, &rds.ModifyDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		DeletionProtection:  aws.Bool(false),
		ApplyImmediately:    aws.Bool(true),
	})
	if err != nil {
		log.Warn("failed to disable deletion protection on cluster", clusterIdentifier, ":", err)
	}

	// delete the cluster
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
	if r.cfg.Source.Type == sourceTypeAuroraCluster {
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
