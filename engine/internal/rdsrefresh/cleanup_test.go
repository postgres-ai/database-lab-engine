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

func TestStateFile(t *testing.T) {
	t.Run("write and read state", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		state := &CloneState{
			CloneID:     "test-clone",
			IsCluster:   false,
			AWSRegion:   "us-east-1",
			CreatedAt:   time.Now().UTC().Truncate(time.Second),
			DeleteAfter: time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second),
			SourceID:    "source-db",
		}

		err := stateFile.Write(state)
		require.NoError(t, err)

		readState, err := stateFile.Read()
		require.NoError(t, err)
		require.NotNil(t, readState)

		assert.Equal(t, state.CloneID, readState.CloneID)
		assert.Equal(t, state.IsCluster, readState.IsCluster)
		assert.Equal(t, state.AWSRegion, readState.AWSRegion)
		assert.Equal(t, state.SourceID, readState.SourceID)
	})

	t.Run("read returns nil for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		state, err := stateFile.Read()
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("clear removes state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		state := &CloneState{CloneID: "test-clone"}
		err := stateFile.Write(state)
		require.NoError(t, err)

		assert.True(t, stateFile.Exists())

		err = stateFile.Clear()
		require.NoError(t, err)

		assert.False(t, stateFile.Exists())
	})

	t.Run("exists returns false for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		assert.False(t, stateFile.Exists())
	})

	t.Run("uses meta directory when empty", func(t *testing.T) {
		stateFile := NewStateFile("")
		// should use meta directory or fallback to current directory
		assert.Contains(t, stateFile.path, stateFileName)
	})
}

func TestIsStaleByTags(t *testing.T) {
	cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
	client := &RDSClient{cfg: cfg}

	t.Run("not stale without managed by tag", func(t *testing.T) {
		tags := []types.Tag{{Key: aws.String("Environment"), Value: aws.String("test")}}
		stale, reason := client.isStaleByTags(tags, nil)
		assert.False(t, stale)
		assert.Empty(t, reason)
	})

	t.Run("not stale without auto delete tag", func(t *testing.T) {
		tags := []types.Tag{{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)}}
		stale, reason := client.isStaleByTags(tags, nil)
		assert.False(t, stale)
		assert.Empty(t, reason)
	})

	t.Run("stale when delete after expired", func(t *testing.T) {
		expiredTime := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		tags := []types.Tag{
			{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
			{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
			{Key: aws.String(DeleteAfterTagKey), Value: aws.String(expiredTime)},
		}
		stale, reason := client.isStaleByTags(tags, nil)
		assert.True(t, stale)
		assert.Contains(t, reason, "DeleteAfter tag expired")
	})

	t.Run("not stale when delete after in future", func(t *testing.T) {
		futureTime := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
		tags := []types.Tag{
			{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
			{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
			{Key: aws.String(DeleteAfterTagKey), Value: aws.String(futureTime)},
		}
		stale, reason := client.isStaleByTags(tags, nil)
		assert.False(t, stale)
		assert.Empty(t, reason)
	})

	t.Run("stale when creation time exceeds max age", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		tags := []types.Tag{
			{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
			{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
		}
		stale, reason := client.isStaleByTags(tags, &createdAt)
		assert.True(t, stale)
		assert.Contains(t, reason, "exceeds max age")
	})

	t.Run("not stale when within max age", func(t *testing.T) {
		createdAt := time.Now().Add(-24 * time.Hour)
		tags := []types.Tag{
			{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
			{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
		}
		stale, reason := client.isStaleByTags(tags, &createdAt)
		assert.False(t, stale)
		assert.Empty(t, reason)
	})
}

func TestFindStaleInstances(t *testing.T) {
	t.Run("finds stale rds instances", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
				return &rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("dblab-refresh-20240101-120000"),
							InstanceCreateTime:   &createdAt,
							TagList: []types.Tag{
								{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
								{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
							},
						},
						{
							DBInstanceIdentifier: aws.String("other-instance"),
							InstanceCreateTime:   &createdAt,
						},
					},
				}, nil
			},
		}

		cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
		client := NewRDSClientWithAPI(mock, cfg)

		staleClones, err := client.findStaleInstances(context.Background())
		require.NoError(t, err)
		require.Len(t, staleClones, 1)
		assert.Equal(t, "dblab-refresh-20240101-120000", staleClones[0].Identifier)
		assert.False(t, staleClones[0].IsCluster)
	})

	t.Run("skips cluster member instances", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
				return &rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("dblab-refresh-20240101-120000-instance"),
							DBClusterIdentifier:  aws.String("dblab-refresh-20240101-120000"),
							InstanceCreateTime:   &createdAt,
							TagList: []types.Tag{
								{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
								{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
							},
						},
					},
				}, nil
			},
		}

		cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
		client := NewRDSClientWithAPI(mock, cfg)

		staleClones, err := client.findStaleInstances(context.Background())
		require.NoError(t, err)
		assert.Empty(t, staleClones)
	})
}

func TestFindStaleClusters(t *testing.T) {
	t.Run("finds stale aurora clusters", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		mock := &mockRDSAPI{
			describeDBClustersFunc: func(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
				return &rds.DescribeDBClustersOutput{
					DBClusters: []types.DBCluster{
						{
							DBClusterIdentifier: aws.String("dblab-refresh-20240101-120000"),
							ClusterCreateTime:   &createdAt,
							TagList: []types.Tag{
								{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
								{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
							},
						},
						{
							DBClusterIdentifier: aws.String("other-cluster"),
							ClusterCreateTime:   &createdAt,
						},
					},
				}, nil
			},
		}

		cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
		client := NewRDSClientWithAPI(mock, cfg)

		staleClones, err := client.findStaleClusters(context.Background())
		require.NoError(t, err)
		require.Len(t, staleClones, 1)
		assert.Equal(t, "dblab-refresh-20240101-120000", staleClones[0].Identifier)
		assert.True(t, staleClones[0].IsCluster)
	})
}

func TestCleanupStaleClones(t *testing.T) {
	t.Run("dry run does not delete", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		deleteCalled := false

		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
				return &rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("dblab-refresh-20240101-120000"),
							InstanceCreateTime:   &createdAt,
							TagList: []types.Tag{
								{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
								{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
							},
						},
					},
				}, nil
			},
			describeDBClustersFunc: func(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
				return &rds.DescribeDBClustersOutput{DBClusters: []types.DBCluster{}}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
		client := NewRDSClientWithAPI(mock, cfg)

		result, err := client.CleanupStaleClones(context.Background(), true)
		require.NoError(t, err)
		assert.Equal(t, 1, result.ClonesFound)
		assert.Equal(t, 0, result.ClonesDeleted)
		assert.False(t, deleteCalled)
	})

	t.Run("deletes stale clones", func(t *testing.T) {
		createdAt := time.Now().Add(-72 * time.Hour)
		deleteCalled := false

		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
				return &rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("dblab-refresh-20240101-120000"),
							InstanceCreateTime:   &createdAt,
							TagList: []types.Tag{
								{Key: aws.String(ManagedByTagKey), Value: aws.String(ManagedByTagValue)},
								{Key: aws.String(AutoDeleteTagKey), Value: aws.String("true")},
							},
						},
					},
				}, nil
			},
			describeDBClustersFunc: func(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
				return &rds.DescribeDBClustersOutput{DBClusters: []types.DBCluster{}}, nil
			},
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				return &rds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				assert.Equal(t, "dblab-refresh-20240101-120000", *params.DBInstanceIdentifier)
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{RDSClone: RDSCloneConfig{MaxAge: Duration(48 * time.Hour)}}
		client := NewRDSClientWithAPI(mock, cfg)

		result, err := client.CleanupStaleClones(context.Background(), false)
		require.NoError(t, err)
		assert.Equal(t, 1, result.ClonesFound)
		assert.Equal(t, 1, result.ClonesDeleted)
		assert.True(t, deleteCalled)
	})
}

func TestCleanupFromStateFile(t *testing.T) {
	t.Run("clears state when no clone id", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		err := stateFile.Write(&CloneState{CloneID: ""})
		require.NoError(t, err)

		cfg := &Config{}
		client := NewRDSClientWithAPI(&mockRDSAPI{}, cfg)

		err = client.CleanupFromStateFile(context.Background(), stateFile)
		require.NoError(t, err)
		assert.False(t, stateFile.Exists())
	})

	t.Run("deletes orphaned clone", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		err := stateFile.Write(&CloneState{CloneID: "orphaned-clone", IsCluster: false})
		require.NoError(t, err)

		deleteCalled := false
		mock := &mockRDSAPI{
			modifyDBInstanceFunc: func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
				return &rds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
				deleteCalled = true
				assert.Equal(t, "orphaned-clone", *params.DBInstanceIdentifier)
				return &rds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{}
		client := NewRDSClientWithAPI(mock, cfg)

		err = client.CleanupFromStateFile(context.Background(), stateFile)
		require.NoError(t, err)
		assert.True(t, deleteCalled)
		assert.False(t, stateFile.Exists())
	})

	t.Run("returns nil when no state file", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		cfg := &Config{}
		client := NewRDSClientWithAPI(&mockRDSAPI{}, cfg)

		err := client.CleanupFromStateFile(context.Background(), stateFile)
		require.NoError(t, err)
	})
}

func TestCloneState(t *testing.T) {
	t.Run("json serialization", func(t *testing.T) {
		state := CloneState{
			CloneID:     "test-clone",
			IsCluster:   true,
			AWSRegion:   "us-west-2",
			CreatedAt:   time.Now().UTC().Truncate(time.Second),
			DeleteAfter: time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second),
			SourceID:    "source-snapshot",
		}

		tmpDir := t.TempDir()
		stateFile := NewStateFile(tmpDir)

		err := stateFile.Write(&state)
		require.NoError(t, err)

		readState, err := stateFile.Read()
		require.NoError(t, err)

		assert.Equal(t, state.CloneID, readState.CloneID)
		assert.Equal(t, state.IsCluster, readState.IsCluster)
		assert.Equal(t, state.AWSRegion, readState.AWSRegion)
		assert.WithinDuration(t, state.CreatedAt, readState.CreatedAt, time.Second)
		assert.WithinDuration(t, state.DeleteAfter, readState.DeleteAfter, time.Second)
		assert.Equal(t, state.SourceID, readState.SourceID)
	})
}
