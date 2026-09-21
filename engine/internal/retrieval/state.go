/*
2021 © Postgres.ai
*/

package retrieval

import (
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// State contains state of retrieval service. Every field is guarded by mu because the HTTP
// handlers report the state while the retrieval pipeline advances it.
type State struct {
	mu          sync.RWMutex
	mode        models.RetrievalMode
	status      models.RetrievalStatus
	lastRefresh *models.LocalTime
	currentJob  components.JobRunner
	refreshing  bool
	alerts      map[models.AlertType]models.Alert
}

// Mode returns the current retrieval mode.
func (s *State) Mode() models.RetrievalMode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.mode
}

// SetMode sets the retrieval mode.
func (s *State) SetMode(mode models.RetrievalMode) {
	s.mu.Lock()
	s.mode = mode
	s.mu.Unlock()
}

// Status returns the current retrieval status.
func (s *State) Status() models.RetrievalStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status
}

// SetStatus sets the retrieval status.
func (s *State) SetStatus(status models.RetrievalStatus) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
}

// LastRefresh returns the start time of the most recent data refresh.
func (s *State) LastRefresh() *models.LocalTime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastRefresh
}

// SetLastRefresh sets the start time of the most recent data refresh.
func (s *State) SetLastRefresh(lastRefresh *models.LocalTime) {
	s.mu.Lock()
	s.lastRefresh = lastRefresh
	s.mu.Unlock()
}

// CurrentJob returns the retrieval job being run, or nil when the pipeline is idle.
func (s *State) CurrentJob() components.JobRunner {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentJob
}

// SetCurrentJob sets the retrieval job being run.
func (s *State) SetCurrentJob(job components.JobRunner) {
	s.mu.Lock()
	s.currentJob = job
	s.mu.Unlock()
}

// CanStartRefresh reports whether a full refresh may start. It is a read-only precondition check
// meant for request validation; it claims nothing, so a caller that goes on to run the refresh
// must still take the slot with TryStartRefresh.
func (s *State) CanStartRefresh() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.canStartRefresh()
}

// TryStartRefresh claims the single full-refresh slot. It returns ErrRefreshInProgress when
// another refresh already holds the slot or the pipeline is busy, and ErrRefreshPending when
// retrieval is suspended. On success the caller must release the slot with FinishRefresh.
func (s *State) TryStartRefresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.canStartRefresh(); err != nil {
		return err
	}

	s.refreshing = true

	return nil
}

// FinishRefresh releases the full-refresh slot claimed by TryStartRefresh.
func (s *State) FinishRefresh() {
	s.mu.Lock()
	s.refreshing = false
	s.mu.Unlock()
}

func (s *State) canStartRefresh() error {
	if s.refreshing || s.status == models.Refreshing || s.status == models.Snapshotting {
		return ErrRefreshInProgress
	}

	if s.status == models.Pending {
		return ErrRefreshPending
	}

	return nil
}

// Alerts returns a snapshot copy of all registered retrieval alerts.
func (s *State) Alerts() map[models.AlertType]models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[models.AlertType]models.Alert, len(s.alerts))

	for k, v := range s.alerts {
		result[k] = v
	}

	return result
}

func (s *State) addAlert(telemetryAlert telemetry.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, ok := s.alerts[telemetryAlert.Level]
	if ok {
		alert.Count++
		alert.LastSeen = time.Now()
		alert.Message = telemetryAlert.Message
		s.alerts[telemetryAlert.Level] = alert

		return
	}

	alert = models.Alert{
		Level:    models.AlertLevelByType(telemetryAlert.Level),
		Message:  telemetryAlert.Message,
		LastSeen: time.Now(),
		Count:    1,
	}

	s.alerts[telemetryAlert.Level] = alert
}

func (s *State) cleanAlerts() {
	s.mu.Lock()
	s.alerts = make(map[models.AlertType]models.Alert)
	s.mu.Unlock()
}
