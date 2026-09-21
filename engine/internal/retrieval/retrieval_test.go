package retrieval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
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
		assert.Equal(t, models.Pending, r.State.Status())
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
				status: models.Pending,
			},
		}

		err = r.RemovePendingMarker()
		require.Nil(t, err)
		assert.Equal(t, models.Inactive, r.State.Status())

		r.State.SetStatus(models.Finished)

		err = r.RemovePendingMarker()
		require.Nil(t, err)
		assert.Equal(t, models.Finished, r.State.Status())
	})
}

func TestSyncStatusNotReportedForLogicalMode(t *testing.T) {
	var r = Retrieval{
		State: State{
			mode: models.Logical,
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
			r := &Retrieval{State: State{mode: tc.mode}}
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
			r := &Retrieval{State: State{status: tc.status}}
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
		assert.Equal(t, models.Physical, r.State.Mode())
	})

	t.Run("logical mode when logical jobs present", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}, "logicalRestore": {}},
		}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Logical, r.State.Mode())
	})

	t.Run("unknown mode when no recognized jobs", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{JobsSpec: map[string]config.JobSpec{}}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Unknown, r.State.Mode())
	})

	t.Run("physical takes precedence over logical", func(t *testing.T) {
		r := &Retrieval{cfg: &config.Config{
			JobsSpec: map[string]config.JobSpec{"physicalRestore": {}, "logicalDump": {}},
		}}
		r.defineRetrievalMode()
		assert.Equal(t, models.Physical, r.State.Mode())
	})
}

func TestCanStartRefresh(t *testing.T) {
	t.Run("allows when inactive", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Inactive}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("allows when finished", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Finished}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("allows when failed", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Failed}}
		assert.NoError(t, r.CanStartRefresh())
	})

	t.Run("blocks when refreshing", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Refreshing}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshInProgress)
	})

	t.Run("blocks when snapshotting", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Snapshotting}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshInProgress)
	})

	t.Run("blocks when pending", func(t *testing.T) {
		r := &Retrieval{State: State{status: models.Pending}}
		assert.ErrorIs(t, r.CanStartRefresh(), ErrRefreshPending)
	})
}

func TestReportState(t *testing.T) {
	t.Run("with refresh timetable", func(t *testing.T) {
		r := &Retrieval{
			State: State{mode: models.Physical},
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
			State: State{mode: models.Logical},
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

func newRefreshTestRetrieval() *Retrieval {
	return &Retrieval{
		State:       State{alerts: make(map[models.AlertType]models.Alert)},
		tm:          telemetry.New(&platform.Service{}, "instanceID"),
		poolManager: pool.NewPoolManager(&pool.Config{}, nil),
	}
}

func TestFullRefreshRejectsASecondRefresh(t *testing.T) {
	r := newRefreshTestRetrieval()

	// Hold the slot as a running refresh would.
	require.NoError(t, r.State.TryStartRefresh())

	const callers = 8

	errs := make([]error, callers)

	var wg sync.WaitGroup

	wg.Add(callers)

	for i := 0; i < callers; i++ {
		go func(i int) {
			defer wg.Done()

			errs[i] = r.FullRefresh(context.Background())
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		assert.ErrorIs(t, err, ErrRefreshInProgress, "caller %d must not start a concurrent refresh", i)
	}

	r.State.FinishRefresh()

	// The released slot lets the next refresh run: it gets as far as the pool check.
	require.NoError(t, r.FullRefresh(context.Background()))
	assert.Equal(t, ErrNoAvailablePool.Error(), r.State.Alerts()[models.RefreshSkipped].Message)
}

func TestCanStartRefreshDoesNotConsumeTheSlot(t *testing.T) {
	r := newRefreshTestRetrieval()

	// The handler precheck runs before FullRefresh and must leave the slot free for it.
	require.NoError(t, r.CanStartRefresh())
	require.NoError(t, r.CanStartRefresh())

	require.NoError(t, r.FullRefresh(context.Background()))
	assert.Equal(t, ErrNoAvailablePool.Error(), r.State.Alerts()[models.RefreshSkipped].Message)
}

func TestFullRefreshReleasesTheSlotOnEveryReturn(t *testing.T) {
	r := newRefreshTestRetrieval()

	for i := 0; i < 3; i++ {
		require.NoError(t, r.FullRefresh(context.Background()))
		require.NoError(t, r.State.CanStartRefresh(), "the slot must be free after run %d", i)
	}
}

func TestFullRefreshReportsPendingState(t *testing.T) {
	r := newRefreshTestRetrieval()
	r.State.SetStatus(models.Pending)

	assert.ErrorIs(t, r.FullRefresh(context.Background()), ErrRefreshPending)
}

func TestIsRefreshSkipped(t *testing.T) {
	assert.True(t, IsRefreshSkipped(ErrRefreshInProgress))
	assert.True(t, IsRefreshSkipped(fmt.Errorf("wrapped: %w", ErrRefreshPending)))
	assert.False(t, IsRefreshSkipped(ErrNoAvailablePool))
}

func TestReloadRacesWithConfigReaders(t *testing.T) {
	t.Parallel()

	r := newRefreshTestRetrieval()
	r.setup(&config.Config{Jobs: []string{logical.DumpJobType}, JobsSpec: map[string]config.JobSpec{logical.DumpJobType: {}}},
		global.Config{Database: global.Database{Username: "postgres", DBName: "postgres"}})

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			r.Reload(context.Background(), &config.Config{
				Jobs:     []string{logical.DumpJobType},
				JobsSpec: map[string]config.JobSpec{logical.DumpJobType: {}},
				Refresh:  &config.Refresh{Timetable: "0 3 * * *"},
			}, global.Config{Database: global.Database{Username: "postgres", DBName: fmt.Sprintf("db-%d", i)}})
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			_ = r.GetRetrievalMode()
			_ = r.GetRetrievalStatus()
			_ = r.ReportState()
			_ = r.GlobalConfig()
			_, _ = r.GetStageSpec(logical.DumpJobType)
		}
	}()

	wg.Wait()

	r.Stop()
}

func TestReloadUpdatesTheGlobalConfig(t *testing.T) {
	r := newRefreshTestRetrieval()
	r.setup(&config.Config{}, global.Config{Database: global.Database{DBName: "before"}})
	require.Equal(t, "before", r.GlobalConfig().Database.DBName)

	r.Reload(context.Background(), &config.Config{}, global.Config{Database: global.Database{DBName: "after"}})
	assert.Equal(t, "after", r.GlobalConfig().Database.DBName)

	r.Stop()
}

func TestScheduleSpecRacesWithSchedulerReload(t *testing.T) {
	t.Parallel()

	r := newRefreshTestRetrieval()
	r.setup(&config.Config{Refresh: &config.Refresh{Timetable: "0 3 * * *"}}, global.Config{})

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			r.setupScheduler(context.Background())
			r.stopScheduler()
		}
	}()

	go func() {
		defer wg.Done()

		// The status handlers must never observe a schedule that a reload has already dropped.
		for i := 0; i < iterations; i++ {
			if spec := r.ScheduleSpec(); spec != nil {
				_ = spec.Next(time.Now())
			}
		}
	}()

	wg.Wait()

	r.Stop()
	assert.Nil(t, r.ScheduleSpec())
}

func TestStatefulJobsRaceWithReload(t *testing.T) {
	t.Parallel()

	r := newRefreshTestRetrieval()
	r.setup(&config.Config{JobsSpec: map[string]config.JobSpec{}}, global.Config{})

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		// The snapshot pipeline replaces the stateful jobs while a reload walks them.
		for i := 0; i < iterations; i++ {
			r.setStatefulJobs(nil)
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			r.reloadStatefulJobs()
		}
	}()

	wg.Wait()
}

func TestRestartRunContextCancelsThePreviousRun(t *testing.T) {
	t.Parallel()

	r := newRefreshTestRetrieval()

	previous := r.restartRunContext(context.Background())
	require.NoError(t, previous.Err())

	current := r.restartRunContext(context.Background())
	assert.ErrorIs(t, previous.Err(), context.Canceled)
	assert.NoError(t, current.Err())

	const iterations = 100

	var wg sync.WaitGroup

	wg.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				r.restartRunContext(context.Background())
			}
		}()
	}

	wg.Wait()
}
