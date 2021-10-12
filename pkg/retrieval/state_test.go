/*
2021 © Postgres.ai
*/

package retrieval

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

func TestState(t *testing.T) {
	state := State{
		mu:     sync.Mutex{},
		alerts: make(map[models.AlertType]models.Alert),
	}

	assert.Equal(t, 0, len(state.alerts))

	state.addAlert(models.RefreshFailed, "Test Error Message")
	state.addAlert(models.RefreshSkipped, "Test Warning Message 1")

	assert.Equal(t, 2, len(state.alerts))
	assert.Equal(t, models.ErrorLevel, state.alerts[models.RefreshFailed].Level)

	ts := time.Now()
	state.addAlert(models.RefreshSkipped, "Test Warning Message 2")

	assert.Equal(t, 2, len(state.alerts))
	assert.Equal(t, models.WarningLevel, state.alerts[models.RefreshSkipped].Level)
	assert.Equal(t, 2, state.alerts[models.RefreshSkipped].Count)
	assert.GreaterOrEqual(t, state.alerts[models.RefreshSkipped].LastSeen.String(), ts.String())
	assert.Equal(t, "Test Warning Message 2", state.alerts[models.RefreshSkipped].Message)

	assert.Equal(t, state.alerts, state.Alerts())

	state.cleanAlerts()

	assert.Equal(t, 0, len(state.alerts))
}
