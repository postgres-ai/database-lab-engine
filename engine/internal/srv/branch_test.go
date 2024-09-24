package srv

import (
	"testing"

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
			{branchName: "tři"},
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
		}

		for _, tc := range testCases {
			require.False(t, isValidBranchName(tc.branchName))
		}
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
