/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLatestDBSnapshot(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)
	oldest := now.Add(-4 * time.Hour)

	t.Run("returns latest available snapshot", func(t *testing.T) {
		mock := &mockRDSAPI{
			describeDBSnapshotsFunc: func(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
				return &rds.DescribeDBSnapshotsOutput{
					DBSnapshots: []types.DBSnapshot{
						{DBSnapshotIdentifier: aws.String("snap-oldest"), SnapshotCreateTime: &oldest, Status: aws.String("available")},
						{DBSnapshotIdentifier: aws.String("snap-latest"), SnapshotCreateTime: &now, Status: aws.String("available")},
						{DBSnapshotIdentifier: aws.String("snap-older"), SnapshotCreateTime: &older, Status: aws.String("available")},
					},
				}, nil
			},
		}

		cfg := &Config{
			Source: SourceConfig{Type: "rds", Identifier: "test-db"},
		}
		client := NewRDSClientWithAPI(mock, cfg)

		snapshotID, err := client.FindLatestSnapshot(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "snap-latest", snapshotID)
	})

	t.Run("returns error when no snapshots found", func(t *testing.T) {
		mock := &mockRDSAPI{
			describeDBSnapshotsFunc: func(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
				return &rds.DescribeDBSnapshotsOutput{DBSnapshots: []types.DBSnapshot{}}, nil
			},
		}

		cfg := &Config{
			Source: SourceConfig{Type: "rds", Identifier: "test-db"},
		}
		client := NewRDSClientWithAPI(mock, cfg)

		_, err := client.FindLatestSnapshot(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no automated snapshots found")
	})

	t.Run("skips creating snapshots", func(t *testing.T) {
		mock := &mockRDSAPI{
			describeDBSnapshotsFunc: func(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
				return &rds.DescribeDBSnapshotsOutput{
					DBSnapshots: []types.DBSnapshot{
						{DBSnapshotIdentifier: aws.String("snap-available"), SnapshotCreateTime: &now, Status: aws.String("available")},
						{DBSnapshotIdentifier: aws.String("snap-creating"), SnapshotCreateTime: &now, Status: aws.String("creating")},
					},
				}, nil
			},
		}

		cfg := &Config{
			Source: SourceConfig{Type: "rds", Identifier: "test-db"},
		}
		client := NewRDSClientWithAPI(mock, cfg)

		snapshotID, err := client.FindLatestSnapshot(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "snap-available", snapshotID)
	})

	t.Run("uses explicitly configured snapshot identifier", func(t *testing.T) {
		cfg := &Config{
			Source: SourceConfig{
				Type:               "rds",
				Identifier:         "test-db",
				SnapshotIdentifier: "snap-explicit",
			},
		}
		client := NewRDSClientWithAPI(&mockRDSAPI{}, cfg)

		snapshotID, err := client.FindLatestSnapshot(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "snap-explicit", snapshotID)
	})
}

func TestFindLatestClusterSnapshot(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)

	t.Run("returns latest available cluster snapshot", func(t *testing.T) {
		mock := &mockRDSAPI{
			describeDBClusterSnapshotsFunc: func(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error) {
				return &rds.DescribeDBClusterSnapshotsOutput{
					DBClusterSnapshots: []types.DBClusterSnapshot{
						{DBClusterSnapshotIdentifier: aws.String("snap-older"), SnapshotCreateTime: &older, Status: aws.String("available")},
						{DBClusterSnapshotIdentifier: aws.String("snap-latest"), SnapshotCreateTime: &now, Status: aws.String("available")},
					},
				}, nil
			},
		}

		cfg := &Config{
			Source: SourceConfig{Type: "aurora-cluster", Identifier: "test-cluster"},
		}
		client := NewRDSClientWithAPI(mock, cfg)

		snapshotID, err := client.FindLatestSnapshot(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "snap-latest", snapshotID)
	})

	t.Run("returns error when no cluster snapshots found", func(t *testing.T) {
		mock := &mockRDSAPI{
			describeDBClusterSnapshotsFunc: func(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error) {
				return &rds.DescribeDBClusterSnapshotsOutput{DBClusterSnapshots: []types.DBClusterSnapshot{}}, nil
			},
		}

		cfg := &Config{
			Source: SourceConfig{Type: "aurora-cluster", Identifier: "test-cluster"},
		}
		client := NewRDSClientWithAPI(mock, cfg)

		_, err := client.FindLatestSnapshot(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no automated snapshots found")
	})
}

func TestDeleteRDSInstance(t *testing.T) {
	t.Run("successfully deletes instance", func(t *testing.T) {
		modifyCalled := false
		deleteCalled := false

		mock := &mockRDSAPI{
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				modifyCalled = true
				assert.Equal(t, "test-instance", *params.DBInstanceIdentifier)
				assert.False(t, *params.DeletionProtection)
				return &rds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				assert.Equal(t, "test-instance", *params.DBInstanceIdentifier)
				assert.True(t, *params.SkipFinalSnapshot)
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err := client.deleteRDSInstance(context.Background(), "test-instance")

		require.NoError(t, err)
		assert.True(t, modifyCalled, "modify should be called to disable deletion protection")
		assert.True(t, deleteCalled, "delete should be called")
	})

	t.Run("proceeds with deletion even if modify fails", func(t *testing.T) {
		deleteCalled := false

		mock := &mockRDSAPI{
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				return nil, assert.AnError
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err := client.deleteRDSInstance(context.Background(), "test-instance")

		require.NoError(t, err)
		assert.True(t, deleteCalled, "delete should still be attempted")
	})

	t.Run("returns error when deletion fails", func(t *testing.T) {
		mock := &mockRDSAPI{
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				return &rds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				return nil, assert.AnError
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err := client.deleteRDSInstance(context.Background(), "test-instance")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete DB instance")
	})
}

func TestDeleteClone(t *testing.T) {
	t.Run("deletes rds instance clone", func(t *testing.T) {
		deleteCalled := false

		mock := &mockRDSAPI{
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				return &rds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				assert.Equal(t, "clone-id", *params.DBInstanceIdentifier)
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err := client.DeleteClone(context.Background(), &CloneInfo{
			Identifier: "clone-id",
			IsCluster:  false,
		})

		require.NoError(t, err)
		assert.True(t, deleteCalled)
	})

	t.Run("deletes aurora cluster clone", func(t *testing.T) {
		describeInstancesCalled := false
		deleteClusterCalled := false

		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
				describeInstancesCalled = true
				return &rds.DescribeDBInstancesOutput{DBInstances: []types.DBInstance{}}, nil
			},
			modifyDBClusterFunc: func(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error) {
				return &rds.ModifyDBClusterOutput{}, nil
			},
			deleteDBClusterFunc: func(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error) {
				deleteClusterCalled = true
				assert.Equal(t, "cluster-id", *params.DBClusterIdentifier)
				return &rds.DeleteDBClusterOutput{}, nil
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err := client.DeleteClone(context.Background(), &CloneInfo{
			Identifier: "cluster-id",
			IsCluster:  true,
		})

		require.NoError(t, err)
		assert.True(t, describeInstancesCalled, "should check for cluster instances")
		assert.True(t, deleteClusterCalled)
	})
}
