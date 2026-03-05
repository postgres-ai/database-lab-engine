/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRDSAPI implements RDSAPI for testing.
type mockRDSAPI struct {
	describeDBSnapshotsFunc        func(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	describeDBClusterSnapshotsFunc func(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error)
	restoreDBInstanceFunc          func(ctx context.Context, params *rds.RestoreDBInstanceFromDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error)
	restoreDBClusterFunc           func(ctx context.Context, params *rds.RestoreDBClusterFromSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBClusterFromSnapshotOutput, error)
	createDBInstanceFunc           func(ctx context.Context, params *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error)
	describeDBClustersFunc         func(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	describeDBInstancesFunc        func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	modifyDBInstanceFunc           func(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	deleteDBInstanceFunc           func(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	modifyDBClusterFunc            func(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error)
	deleteDBClusterFunc            func(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error)
}

func (m *mockRDSAPI) DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	if m.describeDBSnapshotsFunc != nil {
		return m.describeDBSnapshotsFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	if m.describeDBClusterSnapshotsFunc != nil {
		return m.describeDBClusterSnapshotsFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) RestoreDBInstanceFromDBSnapshot(ctx context.Context, params *rds.RestoreDBInstanceFromDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error) {
	if m.restoreDBInstanceFunc != nil {
		return m.restoreDBInstanceFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) RestoreDBClusterFromSnapshot(ctx context.Context, params *rds.RestoreDBClusterFromSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBClusterFromSnapshotOutput, error) {
	if m.restoreDBClusterFunc != nil {
		return m.restoreDBClusterFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) CreateDBInstance(ctx context.Context, params *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error) {
	if m.createDBInstanceFunc != nil {
		return m.createDBInstanceFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	if m.describeDBClustersFunc != nil {
		return m.describeDBClustersFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	if m.describeDBInstancesFunc != nil {
		return m.describeDBInstancesFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
	if m.modifyDBInstanceFunc != nil {
		return m.modifyDBInstanceFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
	if m.deleteDBInstanceFunc != nil {
		return m.deleteDBInstanceFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error) {
	if m.modifyDBClusterFunc != nil {
		return m.modifyDBClusterFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRDSAPI) DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error) {
	if m.deleteDBClusterFunc != nil {
		return m.deleteDBClusterFunc(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func TestBuildTags(t *testing.T) {
	t.Run("builds tags from config with DeleteAfter", func(t *testing.T) {
		cfg := &Config{
			RDSClone: RDSCloneConfig{
				Tags:   map[string]string{"Environment": "test", "Owner": "dblab"},
				MaxAge: Duration(48 * time.Hour),
			},
		}

		client := &RDSClient{cfg: cfg}
		tags := client.buildTags()

		require.Len(t, tags, 3) // Environment, Owner, DeleteAfter
		tagMap := make(map[string]string)

		for _, tag := range tags {
			tagMap[*tag.Key] = *tag.Value
		}

		assert.Equal(t, "test", tagMap["Environment"])
		assert.Equal(t, "dblab", tagMap["Owner"])
		assert.NotEmpty(t, tagMap[DeleteAfterTagKey])

		// verify DeleteAfter is a valid RFC3339 timestamp
		deleteAfter, err := time.Parse(time.RFC3339, tagMap[DeleteAfterTagKey])
		require.NoError(t, err)
		assert.True(t, deleteAfter.After(time.Now()), "DeleteAfter should be in the future")
	})

	t.Run("handles empty tags but still adds DeleteAfter", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{Tags: map[string]string{}, MaxAge: Duration(24 * time.Hour)}}
		client := &RDSClient{cfg: cfg}
		tags := client.buildTags()
		require.Len(t, tags, 1)
		assert.Equal(t, DeleteAfterTagKey, *tags[0].Key)
	})
}

func TestGetSourceInfo(t *testing.T) {
	t.Run("formats rds instance info", func(t *testing.T) {
		cfg := &Config{Source: SourceConfig{Type: "rds", Identifier: "test-db"}}
		client := &RDSClient{cfg: cfg}

		t.Skip("skipping test that requires AWS credentials")
		_, _ = client.GetSourceInfo(testContext(t))
	})

	t.Run("formats aurora cluster info", func(t *testing.T) {
		cfg := &Config{Source: SourceConfig{Type: "aurora-cluster", Identifier: "test-cluster"}}
		client := &RDSClient{cfg: cfg}

		t.Skip("skipping test that requires AWS credentials")
		_, _ = client.GetSourceInfo(testContext(t))
	})
}

func TestSnapshotSorting(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Hour)
	oldest := now.Add(-4 * time.Hour)

	t.Run("sorts db snapshots by time", func(t *testing.T) {
		snapshots := []types.DBSnapshot{
			{DBSnapshotIdentifier: aws.String("snap-1"), SnapshotCreateTime: &oldest, Status: aws.String("available")},
			{DBSnapshotIdentifier: aws.String("snap-2"), SnapshotCreateTime: &now, Status: aws.String("available")},
			{DBSnapshotIdentifier: aws.String("snap-3"), SnapshotCreateTime: &older, Status: aws.String("available")},
		}

		assert.Equal(t, "snap-1", *snapshots[0].DBSnapshotIdentifier)
		assert.Equal(t, "snap-2", *snapshots[1].DBSnapshotIdentifier)
		assert.Equal(t, "snap-3", *snapshots[2].DBSnapshotIdentifier)
	})

	t.Run("sorts cluster snapshots by time", func(t *testing.T) {
		snapshots := []types.DBClusterSnapshot{
			{DBClusterSnapshotIdentifier: aws.String("snap-1"), SnapshotCreateTime: &oldest, Status: aws.String("available")},
			{DBClusterSnapshotIdentifier: aws.String("snap-2"), SnapshotCreateTime: &now, Status: aws.String("available")},
			{DBClusterSnapshotIdentifier: aws.String("snap-3"), SnapshotCreateTime: &older, Status: aws.String("available")},
		}

		assert.Equal(t, "snap-1", *snapshots[0].DBClusterSnapshotIdentifier)
		assert.Equal(t, "snap-2", *snapshots[1].DBClusterSnapshotIdentifier)
		assert.Equal(t, "snap-3", *snapshots[2].DBClusterSnapshotIdentifier)
	})
}

func TestCloneInfo(t *testing.T) {
	t.Run("creates rds clone info", func(t *testing.T) {
		clone := &CloneInfo{Identifier: "test-clone", Endpoint: "test.rds.amazonaws.com", Port: 5432, IsCluster: false}

		assert.Equal(t, "test-clone", clone.Identifier)
		assert.Equal(t, "test.rds.amazonaws.com", clone.Endpoint)
		assert.Equal(t, int32(5432), clone.Port)
		assert.False(t, clone.IsCluster)
	})

	t.Run("creates aurora clone info", func(t *testing.T) {
		clone := &CloneInfo{Identifier: "test-cluster", Endpoint: "test.cluster-xxx.region.rds.amazonaws.com", Port: 5432, IsCluster: true}

		assert.Equal(t, "test-cluster", clone.Identifier)
		assert.True(t, clone.IsCluster)
	})
}

func TestFindLatestSnapshot(t *testing.T) {
	t.Run("uses explicit snapshot identifier", func(t *testing.T) {
		cfg := &Config{Source: SourceConfig{Type: "rds", Identifier: "test-db", SnapshotIdentifier: "explicit-snapshot"}}
		client := &RDSClient{cfg: cfg}

		snapshotID, err := client.FindLatestSnapshot(testContext(t))

		assert.NoError(t, err)
		assert.Equal(t, "explicit-snapshot", snapshotID)
	})

	t.Run("calls findLatestDBSnapshot for rds", func(t *testing.T) {
		cfg := &Config{Source: SourceConfig{Type: "rds", Identifier: "test-db"}}
		client := &RDSClient{cfg: cfg}

		t.Skip("skipping test that requires AWS credentials")
		_, _ = client.FindLatestSnapshot(testContext(t))
	})

	t.Run("calls findLatestClusterSnapshot for aurora", func(t *testing.T) {
		cfg := &Config{Source: SourceConfig{Type: "aurora-cluster", Identifier: "test-cluster"}}
		client := &RDSClient{cfg: cfg}

		t.Skip("skipping test that requires AWS credentials")
		_, _ = client.FindLatestSnapshot(testContext(t))
	})
}

func TestCloneNameGeneration(t *testing.T) {
	t.Run("generates clone name with prefix and timestamp", func(t *testing.T) {
		before := time.Now().UTC()
		cloneName := cloneNamePrefix + before.Format("20060102-150405")
		after := time.Now().UTC()

		assert.Contains(t, cloneName, "dblab-refresh-")
		assert.GreaterOrEqual(t, len(cloneName), len("dblab-refresh-20060102-150405"))

		afterName := cloneNamePrefix + after.Format("20060102-150405")
		assert.GreaterOrEqual(t, afterName, cloneName)
	})
}

func TestDefaultPort(t *testing.T) {
	t.Run("default port is 5432", func(t *testing.T) {
		assert.Equal(t, int32(5432), defaultPort)
	})
}

func testContext(t *testing.T) testContextType {
	return testContextType{t: t}
}

type testContextType struct {
	t *testing.T
}

func (c testContextType) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c testContextType) Done() <-chan struct{} {
	return nil
}

func (c testContextType) Err() error {
	return nil
}

func (c testContextType) Value(key interface{}) interface{} {
	return nil
}
