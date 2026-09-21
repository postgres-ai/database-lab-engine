/*
2021 © Postgres.ai
*/

package retrieval

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestState(t *testing.T) {
	state := State{
		alerts: make(map[models.AlertType]models.Alert),
	}

	assert.Equal(t, 0, len(state.Alerts()))

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "Test Error Message"})
	state.addAlert(telemetry.Alert{Level: models.RefreshSkipped, Message: "Test Warning Message 1"})

	assert.Equal(t, 2, len(state.Alerts()))
	assert.Equal(t, models.ErrorLevel, state.Alerts()[models.RefreshFailed].Level)

	ts := time.Now()
	state.addAlert(telemetry.Alert{Level: models.RefreshSkipped, Message: "Test Warning Message 2"})

	alerts := state.Alerts()
	assert.Equal(t, 2, len(alerts))
	assert.Equal(t, models.WarningLevel, alerts[models.RefreshSkipped].Level)
	assert.Equal(t, 2, alerts[models.RefreshSkipped].Count)
	assert.GreaterOrEqual(t, alerts[models.RefreshSkipped].LastSeen.String(), ts.String())
	assert.Equal(t, "Test Warning Message 2", alerts[models.RefreshSkipped].Message)

	state.cleanAlerts()

	assert.Equal(t, 0, len(state.alerts))
}

func TestState_AlertsConcurrent(t *testing.T) {
	t.Parallel()

	// addAlert and Alerts both acquire s.mu, so concurrent access is race-safe.
	state := State{alerts: make(map[models.AlertType]models.Alert)}

	const goroutines = 10
	const itersEach = 10

	done := make(chan struct{})

	var readers sync.WaitGroup

	readers.Add(1)

	go func() {
		defer readers.Done()

		for {
			select {
			case <-done:
				return
			default:
				_ = state.Alerts()
			}
		}
	}()

	var writers sync.WaitGroup

	writers.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer writers.Done()

			for j := 0; j < itersEach; j++ {
				state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "concurrent error"})
			}
		}()
	}

	writers.Wait()
	close(done)
	readers.Wait()

	alerts := state.Alerts()
	assert.Equal(t, goroutines*itersEach, alerts[models.RefreshFailed].Count)
}

func TestState_CleanAlertsThenRead(t *testing.T) {
	state := State{alerts: make(map[models.AlertType]models.Alert)}

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "err1"})
	state.addAlert(telemetry.Alert{Level: models.RefreshSkipped, Message: "warn1"})
	assert.Equal(t, 2, len(state.Alerts()))

	state.cleanAlerts()
	assert.Empty(t, state.Alerts())

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "err2"})
	alerts := state.Alerts()
	assert.Equal(t, 1, len(alerts))
	assert.Equal(t, 1, alerts[models.RefreshFailed].Count)
	assert.Equal(t, "err2", alerts[models.RefreshFailed].Message)
}

func TestState_AddAlertUpdatesExisting(t *testing.T) {
	state := State{alerts: make(map[models.AlertType]models.Alert)}

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "first"})
	first := state.Alerts()[models.RefreshFailed]
	assert.Equal(t, 1, first.Count)
	assert.Equal(t, "first", first.Message)

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "second"})
	second := state.Alerts()[models.RefreshFailed]
	assert.Equal(t, 2, second.Count)
	assert.Equal(t, "second", second.Message)
	assert.False(t, second.LastSeen.Before(first.LastSeen), "LastSeen must not go backwards")
}

func TestState_AlertsGetReturnsCopy(t *testing.T) {
	state := State{alerts: make(map[models.AlertType]models.Alert)}

	state.addAlert(telemetry.Alert{Level: models.RefreshFailed, Message: "original"})

	copy := state.Alerts()
	copy[models.RefreshSkipped] = models.Alert{Level: models.WarningLevel, Message: "injected"}

	original := state.Alerts()
	assert.Equal(t, 1, len(original), "modifying copy must not affect original")
	_, hasInjected := original[models.RefreshSkipped]
	assert.False(t, hasInjected, "injected key must not appear in original")
}

func TestState_AccessorsConcurrent(t *testing.T) {
	t.Parallel()

	// Every accessor takes s.mu, so a reload racing a running pipeline is safe.
	state := &State{alerts: make(map[models.AlertType]models.Alert)}

	const goroutines = 8
	const itersEach = 50

	var wg sync.WaitGroup

	wg.Add(2 * goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < itersEach; j++ {
				state.SetMode(models.Logical)
				state.SetStatus(models.Refreshing)
				state.SetLastRefresh(models.NewLocalTime(time.Now()))
				state.SetCurrentJob(nil)
			}
		}()

		go func() {
			defer wg.Done()

			for j := 0; j < itersEach; j++ {
				_ = state.Mode()
				_ = state.Status()
				_ = state.LastRefresh()
				_ = state.CurrentJob()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, models.Logical, state.Mode())
	assert.Equal(t, models.Refreshing, state.Status())
}

func TestState_TryStartRefreshIsExclusive(t *testing.T) {
	t.Parallel()

	state := &State{alerts: make(map[models.AlertType]models.Alert)}

	const goroutines = 16

	var claimed atomic.Int64

	var wg sync.WaitGroup

	start := make(chan struct{})

	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			<-start

			if err := state.TryStartRefresh(); err == nil {
				claimed.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), claimed.Load(), "the refresh slot must be claimed exactly once")
}

func TestState_FinishRefreshReleasesTheSlot(t *testing.T) {
	state := &State{alerts: make(map[models.AlertType]models.Alert)}

	require.NoError(t, state.TryStartRefresh())
	require.ErrorIs(t, state.TryStartRefresh(), ErrRefreshInProgress)
	require.ErrorIs(t, state.CanStartRefresh(), ErrRefreshInProgress)

	state.FinishRefresh()

	require.NoError(t, state.CanStartRefresh())
	assert.NoError(t, state.TryStartRefresh())
}

func TestState_CanStartRefreshDoesNotClaim(t *testing.T) {
	state := &State{alerts: make(map[models.AlertType]models.Alert)}

	require.NoError(t, state.CanStartRefresh())
	require.NoError(t, state.CanStartRefresh())

	assert.NoError(t, state.TryStartRefresh())
}

func TestState_TryStartRefreshRespectsStatus(t *testing.T) {
	testCases := []struct {
		name    string
		status  models.RetrievalStatus
		wantErr error
	}{
		{name: "inactive", status: models.Inactive},
		{name: "finished", status: models.Finished},
		{name: "failed", status: models.Failed},
		{name: "renewed", status: models.Renewed},
		{name: "refreshing", status: models.Refreshing, wantErr: ErrRefreshInProgress},
		{name: "snapshotting", status: models.Snapshotting, wantErr: ErrRefreshInProgress},
		{name: "pending", status: models.Pending, wantErr: ErrRefreshPending},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := &State{status: tc.status, alerts: make(map[models.AlertType]models.Alert)}

			if tc.wantErr != nil {
				assert.ErrorIs(t, state.TryStartRefresh(), tc.wantErr)
				return
			}

			assert.NoError(t, state.TryStartRefresh())
		})
	}
}
