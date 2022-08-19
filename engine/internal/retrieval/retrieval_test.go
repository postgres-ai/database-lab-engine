package retrieval

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

func TestJobGroup(t *testing.T) {
	testCases := []struct {
		jobName string
		group   jobGroup
	}{
		{
			jobName: "logicalDump",
			group:   refreshJobs,
		},
		{
			jobName: "logicalRestore",
			group:   refreshJobs,
		},
		{
			jobName: "physicalRestore",
			group:   refreshJobs,
		},
		{
			jobName: "logicalSnapshot",
			group:   snapshotJobs,
		},
		{
			jobName: "physicalSnapshot",
			group:   snapshotJobs,
		},
		{
			jobName: "unknownDump",
			group:   "",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.group, getJobGroup(tc.jobName))
	}
}

func TestPendingMarker(t *testing.T) {
	t.Run("check if the marker file affects the retrieval state", func(t *testing.T) {
		pendingFilepath, err := util.GetMetaPath(pendingFilename)
		require.Nil(t, err)

		tmpDir := path.Dir(pendingFilepath)

		err = os.MkdirAll(tmpDir, 0755)
		require.Nil(t, err)

		defer func() {
			err := os.RemoveAll(tmpDir)
			require.Nil(t, err)
		}()

		_, err = os.Create(pendingFilepath)
		require.Nil(t, err)

		defer func() {
			err := os.Remove(pendingFilepath)
			require.Nil(t, err)
		}()

		r := &Retrieval{}

		err = checkPendingMarker(r)
		require.Nil(t, err)
		assert.Equal(t, models.Pending, r.State.Status)
	})

	t.Run("check the deletion of the pending marker", func(t *testing.T) {
		pendingFilepath, err := util.GetMetaPath(pendingFilename)
		require.Nil(t, err)

		tmpDir := path.Dir(pendingFilepath)

		err = os.MkdirAll(tmpDir, 0755)
		require.Nil(t, err)

		defer func() {
			err := os.RemoveAll(tmpDir)
			require.Nil(t, err)
		}()

		_, err = os.Create(pendingFilepath)
		require.Nil(t, err)

		defer func() {
			err := os.Remove(pendingFilepath)
			require.ErrorIs(t, err, os.ErrNotExist)
		}()

		r := &Retrieval{
			State: State{
				Status: models.Pending,
			},
		}

		err = r.RemovePendingMarker()
		require.Nil(t, err)
		assert.Equal(t, models.Inactive, r.State.Status)

		r.State.Status = models.Finished

		err = r.RemovePendingMarker()
		require.Nil(t, err)
		assert.Equal(t, models.Finished, r.State.Status)
	})
}
