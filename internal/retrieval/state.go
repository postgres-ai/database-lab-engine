/*
2021 © Postgres.ai
*/

package retrieval

import (
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// State contains state of retrieval service.
type State struct {
	Mode        models.RetrievalMode
	Status      models.RetrievalStatus
	LastRefresh *time.Time
	mu          sync.Mutex
	alerts      map[models.AlertType]models.Alert
}

// Alerts returns all registered retrieval alerts.
func (s *State) Alerts() map[models.AlertType]models.Alert {
	s.mu.Lock()
	defer s.mu.Unlock()

	alerts := s.alerts

	return alerts
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
