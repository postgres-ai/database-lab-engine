package srv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestBranchNames(t *testing.T) {
	t.Run("valid branches", func(t *testing.T) {
		testCases := []struct {
			branchName string
		}{
			{branchName: "001-branch"},
			{branchName: "001_branch"},
			{branchName: "001_"},
			{branchName: "_branch"},
			{branchName: "branch"},
			{branchName: "001"},
			{branchName: "a-branch"},
			{branchName: "branch-001"},
		}

		for _, tc := range testCases {
			require.True(t, isValidBranchName(tc.branchName))
		}
	})

	t.Run("invalid branches", func(t *testing.T) {
		testCases := []struct {
			branchName string
		}{
			{branchName: "001 branch"},
			{branchName: ""},
			{branchName: "branch 001"},
			{branchName: "branch/001"},
			{branchName: "-branch"},
			{branchName: "tři"},
		}

		for _, tc := range testCases {
			require.False(t, isValidBranchName(tc.branchName))
		}
	})

}

func TestSnapshotIDValidation(t *testing.T) {
	t.Run("valid snapshot ids", func(t *testing.T) {
		testCases := []string{
			"pool@snap1",
			"pool@snap.1",
			"dblab_pool@snapshot_20210127123000_pre",
			"pool1/pg14@snapshot_20240912082141",
			"pool1/pg14/branch/dev001@snapshot_20240912082141",
			"pool1/pg14/branch/dev001/20240912082141@20240912082141",
			"pool/branch/main/myclone/r0@snap1",
		}

		for _, tc := range testCases {
			require.True(t, isValidSnapshotID(tc), tc)
		}
	})

	t.Run("invalid or injection-bearing snapshot ids", func(t *testing.T) {
		testCases := []string{
			"",
			"pool",
			"pool@",
			"@snap1",
			"pool@snap1@snap2",
			"pool@snap; rm -rf /",
			"pool@snap`id`",
			"pool@snap$(id)",
			"pool@snap|id",
			"pool@snap&whoami",
			"pool@snap snap",
			"pool@snap\nid",
			"pool@snap%3Bid",
			"pool@snap'id",
			"pool@snap>file",
			"-pool@snap1",
		}

		for _, tc := range testCases {
			require.False(t, isValidSnapshotID(tc), tc)
		}
	})
}

func TestChildForkInBranch(t *testing.T) {
	repo := &models.Repo{
		Snapshots: map[string]models.SnapshotDetails{
			"s1": {ID: "s1", Child: []string{"s2"}},
			"s2": {ID: "s2", Parent: "s1", Root: []string{"feature"}},
			"s3": {ID: "s3", Parent: "s2"},
		},
	}

	t.Run("returns the fork-point snapshot and its child branches", func(t *testing.T) {
		id, children := childForkInBranch(repo, []string{"s1", "s2", "s3"})
		assert.Equal(t, "s2", id)
		assert.Equal(t, []string{"feature"}, children)
	})

	t.Run("returns empty when no snapshot is a fork point", func(t *testing.T) {
		id, children := childForkInBranch(repo, []string{"s1", "s3"})
		assert.Empty(t, id)
		assert.Nil(t, children)
	})

	t.Run("ignores snapshot ids absent from the repo", func(t *testing.T) {
		id, _ := childForkInBranch(repo, []string{"missing"})
		assert.Empty(t, id)
	})
}

func TestSnapshotFiltering(t *testing.T) {
	t.Run("filter snapshots", func(t *testing.T) {
		pool := &resources.Pool{Name: "pool1/pg14"}
		input := []models.Snapshot{
			{ID: "pool1/pg14@snapshot_20240912082141", Pool: "pool1/pg14"},
			{ID: "pool1/pg14@snapshot_20240912082987", Pool: "pool1/pg14"},
			{ID: "pool5/pg14@snapshot_20240912082987", Pool: "pool5/pg14"},
			{ID: "pool1/pg14/branch/main@snapshot_20240912082333", Pool: "pool1/pg14"},
			{ID: "pool1/pg14/branch/dev001@snapshot_20240912082141", Pool: "pool1/pg14"},
			{ID: "pool1/pg14/branch/dev001/20240912082141@20240912082141", Pool: "pool1/pg14"},
			{ID: "pool5/pg14/branch/dev001@snapshot_20240912082141", Pool: "pool5/pg14"},
			{ID: "pool1/pg14/branch/dev002/20240912082141@20240912082141", Pool: "pool1/pg14"},
		}

		outputDev001 := []models.Snapshot{
			{ID: "pool1/pg14/branch/dev001@snapshot_20240912082141", Pool: "pool1/pg14"},
			{ID: "pool1/pg14/branch/dev001/20240912082141@20240912082141", Pool: "pool1/pg14"},
		}

		outputMain := []models.Snapshot{
			{ID: "pool1/pg14@snapshot_20240912082141", Pool: "pool1/pg14"},
			{ID: "pool1/pg14@snapshot_20240912082987", Pool: "pool1/pg14"},
			{ID: "pool1/pg14/branch/main@snapshot_20240912082333", Pool: "pool1/pg14"},
		}

		require.Equal(t, outputDev001, filterSnapshotsByBranch(pool, "dev001", input))
		require.Equal(t, outputMain, filterSnapshotsByBranch(pool, "main", input))
	})
}

func TestContainsString(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{name: "found in middle", slice: []string{"a", "b", "c"}, s: "b", expected: true},
		{name: "found at start", slice: []string{"x", "y"}, s: "x", expected: true},
		{name: "found at end", slice: []string{"x", "y"}, s: "y", expected: true},
		{name: "not found", slice: []string{"a", "b"}, s: "c", expected: false},
		{name: "empty slice", slice: []string{}, s: "a", expected: false},
		{name: "nil slice", slice: nil, s: "a", expected: false},
		{name: "empty search string", slice: []string{"a", ""}, s: "", expected: true},
		{name: "single element match", slice: []string{"only"}, s: "only", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, containsString(tc.slice, tc.s))
		})
	}
}

func TestFindBranchParent(t *testing.T) {
	t.Run("single snapshot with no parent", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{"main"}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap1", "feature")
		assert.Equal(t, 1, count)
		assert.Equal(t, "-", parent)
	})

	t.Run("finds parent branch via root", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{"main"}, Root: []string{"feature"}},
			"snap2": {ID: "snap2", Parent: "snap1", Branch: []string{}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap2", "feature")
		assert.Equal(t, 2, count)
		assert.Equal(t, "main", parent)
	})

	t.Run("root match but no branch returns dash", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{}, Root: []string{"feature"}},
			"snap2": {ID: "snap2", Parent: "snap1", Branch: []string{}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap2", "feature")
		assert.Equal(t, 2, count)
		assert.Equal(t, "-", parent)
	})

	t.Run("traverses chain of three snapshots", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{"main"}, Root: []string{"dev"}},
			"snap2": {ID: "snap2", Parent: "snap1", Branch: []string{}, Root: []string{}},
			"snap3": {ID: "snap3", Parent: "snap2", Branch: []string{}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap3", "dev")
		assert.Equal(t, 3, count)
		assert.Equal(t, "main", parent)
	})

	t.Run("stops at dash parent", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap1", "feature")
		assert.Equal(t, 1, count)
		assert.Equal(t, "-", parent)
	})

	t.Run("empty snapshots map", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{}
		count, parent := findBranchParent(snapshots, "snap1", "feature")
		assert.Equal(t, 0, count)
		assert.Equal(t, "-", parent)
	})

	t.Run("returns first branch when root has multiple branches", func(t *testing.T) {
		snapshots := map[string]models.SnapshotDetails{
			"snap1": {ID: "snap1", Parent: "-", Branch: []string{"main", "backup"}, Root: []string{"feature"}},
			"snap2": {ID: "snap2", Parent: "snap1", Branch: []string{}, Root: []string{}},
		}
		count, parent := findBranchParent(snapshots, "snap2", "feature")
		assert.Equal(t, 2, count)
		assert.Equal(t, "main", parent)
	})
}

func TestSnapshotsToRemove(t *testing.T) {
	t.Run("snapshot collected before root boundary stops traversal", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{"feature"}, Child: []string{}},
				"snap2": {ID: "snap2", Parent: "snap1", Root: []string{}, Child: []string{}},
			},
			Branches: map[string]string{"feature": "snap2"},
		}
		result := snapshotsToRemove(repo, "snap2", "feature")
		assert.Equal(t, []string{"snap2"}, result)
	})

	t.Run("removes snapshots up to root boundary", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{}, Child: []string{"snap2"}},
				"snap2": {ID: "snap2", Parent: "snap1", Root: []string{"feature"}, Child: []string{"snap3"}},
				"snap3": {ID: "snap3", Parent: "snap2", Root: []string{}, Child: []string{}},
			},
			Branches: map[string]string{"feature": "snap3"},
		}
		result := snapshotsToRemove(repo, "snap3", "feature")
		assert.Contains(t, result, "snap3")
	})

	t.Run("includes children in removal list", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{"dev"}, Child: []string{}},
				"snap2": {ID: "snap2", Parent: "snap1", Root: []string{}, Child: []string{"snap3"}},
				"snap3": {ID: "snap3", Parent: "snap2", Root: []string{}, Child: []string{}},
			},
			Branches: map[string]string{"dev": "snap2"},
		}
		result := snapshotsToRemove(repo, "snap2", "dev")
		assert.Contains(t, result, "snap3")
	})

	t.Run("empty repo returns empty list", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{},
			Branches:  map[string]string{},
		}
		result := snapshotsToRemove(repo, "snap1", "feature")
		assert.Empty(t, result)
	})
}

func TestTraverseDown(t *testing.T) {
	t.Run("no children returns empty", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Child: []string{}},
			},
		}
		result := traverseDown(repo, "snap1")
		assert.Empty(t, result)
	})

	t.Run("single child", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Child: []string{"snap2"}},
				"snap2": {ID: "snap2", Parent: "snap1", Child: []string{}},
			},
		}
		result := traverseDown(repo, "snap1")
		assert.Equal(t, []string{"snap2"}, result)
	})

	t.Run("nested children", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Child: []string{"snap2"}},
				"snap2": {ID: "snap2", Parent: "snap1", Child: []string{"snap3"}},
				"snap3": {ID: "snap3", Parent: "snap2", Child: []string{}},
			},
		}
		result := traverseDown(repo, "snap1")
		assert.Equal(t, []string{"snap2", "snap3"}, result)
	})

	t.Run("multiple children at same level", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Child: []string{"snap2", "snap3"}},
				"snap2": {ID: "snap2", Parent: "snap1", Child: []string{}},
				"snap3": {ID: "snap3", Parent: "snap1", Child: []string{}},
			},
		}
		result := traverseDown(repo, "snap1")
		assert.Equal(t, []string{"snap2", "snap3"}, result)
	})
}

func TestTraverseUp(t *testing.T) {
	t.Run("root parent stops immediately", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{}},
			},
		}
		result := traverseUp(repo, "snap1", "feature")
		assert.Empty(t, result)
	})

	t.Run("collects snapshot then stops at root boundary", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{}},
				"snap2": {ID: "snap2", Parent: "snap1", Root: []string{"feature"}},
				"snap3": {ID: "snap3", Parent: "snap2", Root: []string{}},
			},
		}
		result := traverseUp(repo, "snap3", "feature")
		assert.Equal(t, []string{"snap3"}, result)
	})

	t.Run("collects snapshots without matching root", func(t *testing.T) {
		repo := &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"snap1": {ID: "snap1", Parent: "-", Root: []string{}},
				"snap2": {ID: "snap2", Parent: "snap1", Root: []string{}},
				"snap3": {ID: "snap3", Parent: "snap2", Root: []string{}},
			},
		}
		result := traverseUp(repo, "snap3", "feature")
		assert.Equal(t, []string{"snap3", "snap2"}, result)
	})
}

func TestFilterSnapshotsByDataset(t *testing.T) {
	t.Run("filters by matching pool", func(t *testing.T) {
		snapshots := []models.Snapshot{
			{ID: "snap1", Pool: "pool1/pg14"},
			{ID: "snap2", Pool: "pool2/pg14"},
			{ID: "snap3", Pool: "pool1/pg14"},
		}
		result := filterSnapshotsByDataset("pool1/pg14", snapshots)
		require.Len(t, result, 2)
		assert.Equal(t, "snap1", result[0].ID)
		assert.Equal(t, "snap3", result[1].ID)
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		snapshots := []models.Snapshot{
			{ID: "snap1", Pool: "pool1/pg14"},
		}
		result := filterSnapshotsByDataset("pool999", snapshots)
		assert.Empty(t, result)
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		result := filterSnapshotsByDataset("pool1", []models.Snapshot{})
		assert.Empty(t, result)
	})
}

func TestFilterSnapshotsByBranch_EdgeCases(t *testing.T) {
	t.Run("no matching snapshots returns empty", func(t *testing.T) {
		pool := &resources.Pool{Name: "pool1/pg14"}
		input := []models.Snapshot{
			{ID: "pool1/pg14/branch/other@snap1", Pool: "pool1/pg14"},
		}
		result := filterSnapshotsByBranch(pool, "nonexistent", input)
		assert.Empty(t, result)
	})

	t.Run("snapshot without @ separator is skipped", func(t *testing.T) {
		pool := &resources.Pool{Name: "pool1/pg14"}
		input := []models.Snapshot{
			{ID: "pool1/pg14/branch/dev001-no-separator", Pool: "pool1/pg14"},
		}
		result := filterSnapshotsByBranch(pool, "dev001", input)
		assert.Empty(t, result)
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		pool := &resources.Pool{Name: "pool1/pg14"}
		result := filterSnapshotsByBranch(pool, "main", []models.Snapshot{})
		assert.Empty(t, result)
	})
}
