/*
2021 © Postgres.ai
*/

package retrieval

import (
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// State contains state of retrieval service.
type State struct {
	Mode        models.RetrievalMode
	Status      models.RefreshStatus
	LastRefresh *time.Time
}
