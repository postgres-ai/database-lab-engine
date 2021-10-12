/*
2021 © Postgres.ai
*/

package retrieval

import (
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

// State contains state of retrieval service.
type State struct {
	Mode        models.RetrievalMode
	Status      models.RefreshStatus
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

func (s *State) addAlert(alertType models.AlertType, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, ok := s.alerts[alertType]
	if ok {
		alert.Count++
		alert.LastSeen = time.Now()
		alert.Message = message
		s.alerts[alertType] = alert

		return
	}

	alert = models.Alert{
		Level:    models.AlertLevelByType(alertType),
		Message:  message,
		LastSeen: time.Now(),
		Count:    1,
	}

	s.alerts[alertType] = alert
}

func (s *State) cleanAlerts() {
	s.mu.Lock()
	s.alerts = make(map[models.AlertType]models.Alert)
	s.mu.Unlock()
}
