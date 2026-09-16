package retrieval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
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

func TestSyncStatusNotReportedForLogicalMode(t *testing.T) {
	var r = Retrieval{
		State: State{
			Mode: models.Logical,
		},
	}
	status, err := r.ReportSyncStatus(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, models.SyncStatusNotAvailable, status.Status.Code)
}

func TestGetRetrievalMode(t *testing.T) {
	testCases := []struct {
		name string
		mode models.RetrievalMode
	}{
		{name: "physical mode", mode: models.Physical},
		{name: "logical mode", mode: models.Logical},
		{name: "unknown mode", mode: models.Unknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Retrieval{State: State{Mode: tc.mode}}
			assert.Equal(t, tc.mode, r.GetRetrievalMode())
		})
	}
}

func TestGetRetrievalStatus(t *testing.T) {
	testCases := []struct {
		name   string
		status models.RetrievalStatus
	}{
		{name: "inactive", status: models.Inactive},
		{name: "pending", status: models.Pending},
		{name: "refreshing", status: models.Refreshing},
		{name: "snapshotting", status: models.Snapshotting},
		{name: "finished", status: models.Finished},
		{name: "failed", status: models.Failed},
		{name: "renewed", status: models.Renewed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Retrieval{State: State{Status: tc.status}}
			assert.Equal(t, tc.status, r.GetRetrievalStatus())
		})
	}
}

func TestDefineRetrievalMode(t *testing.T) {
	t.Run("physical mode when physical jobs present", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{
			JobsSpec: map[string]config.JobSpec{"physicalRestore": {}, "physicalSnapshot": {}},
		}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Physical, r.State.Mode)
	})

	t.Run("logical mode when logical jobs present", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}, "logicalRestore": {}},
		}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Logical, r.State.Mode)
	})

	t.Run("unknown mode when no recognized jobs", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{JobsSpec: map[string]config.JobSpec{}}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Unknown, r.State.Mode)
	})

	t.Run("physical takes precedence over logical", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{
			JobsSpec: map[string]config.JobSpec{"physicalRestore": {}, "logicalDump": {}},
		}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Physical, r.State.Mode)
	})
}

func TestCanStartRefresh(t *testing.T) {
	t.Run("allows when inactive", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Inactive}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("allows when finished", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Finished}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("allows when failed", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Failed}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("blocks when refreshing", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Refreshing}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshInProgress)
	})

	t.Run("blocks when snapshotting", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Snapshotting}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshInProgress)
	})

	t.Run("blocks when pending", func(t *testing.T) {
		r := &Retrieval{State: State{Status: models.Pending}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshPending)
	})
}

func TestReportState(t *testing.T) {
	t.Run("with refresh timetable", func(t *testing.T) {
		r := &Retrieval{
			State: State{Mode: models.Physical},
			cfg: &config.Config{
				Refresh: &config.Refresh{Timetable: "0 3 * * *"},
				Jobs:    []string{"physicalRestore", "physicalSnapshot"},
			},
		}

		report := r.ReportState()
		assert.Equal(t, models.Physical, report.Mode)
		assert.Equal(t, "0 3 * * *", report.Refreshing)
		assert.Equal(t, []string{"physicalRestore", "physicalSnapshot"}, report.Jobs)
	})

	t.Run("without refresh config", func(t *testing.T) {
		r := &Retrieval{
			State: State{Mode: models.Logical},
			cfg:   &config.Config{Jobs: []string{"logicalDump"}},
		}

		report := r.ReportState()
		assert.Equal(t, models.Logical, report.Mode)
		assert.Empty(t, report.Refreshing)
	})
}

func TestGetStageSpec(t *testing.T) {
	spec := config.JobSpec{Name: "logicalDump", Options: map[string]interface{}{"key": "value"}}
	r := &Retrieval{cfg: &config.Config{JobsSpec: map[string]config.JobSpec{"logicalDump": spec}}}

	t.Run("returns spec when found", func(t *testing.T) {
		result, err := r.GetStageSpec("logicalDump")
		require.NoError(t, err)
		assert.Equal(t, spec, result)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := r.GetStageSpec("nonexistent")
		assert.ErrorIs(t, err, ErrStageNotFound)
	})
}

func TestCollectDBList(t *testing.T) {
	t.Run("returns database names from definitions", func(t *testing.T) {
		defs := map[string]logical.DumpDefinition{"db1": {}, "db2": {}}
		result := collectDBList(defs)
		assert.Len(t, result, 2)
		assert.ElementsMatch(t, []string{"db1", "db2"}, result)
	})

	t.Run("returns empty list for empty definitions", func(t *testing.T) {
		result := collectDBList(map[string]logical.DumpDefinition{})
		assert.Empty(t, result)
	})
}

func TestSkipRefreshingError(t *testing.T) {
	t.Run("returns provided message", func(t *testing.T) {
		err := NewSkipRefreshingError("test message")
		assert.Equal(t, "test message", err.Error())
	})

	t.Run("implements error interface", func(t *testing.T) {
		var err error = NewSkipRefreshingError("some error")
		assert.EqualError(t, err, "some error")
	})
}

func TestIsSnapshotExempt(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		exempt bool
	}{
		{name: "no jobs", err: errNoJobs, exempt: true},
		{name: "wrapped no jobs", err: fmt.Errorf("snapshot: %w", errNoJobs), exempt: true},
		{name: "snapshot exists", err: thinclones.NewSnapshotExistsError("snap"), exempt: true},
		{name: "wrapped snapshot exists", err: fmt.Errorf("snapshot: %w", thinclones.NewSnapshotExistsError("snap")), exempt: true},
		{name: "other error", err: errors.New("zfs failed"), exempt: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.exempt, isSnapshotExempt(tc.err))
		})
	}
}
