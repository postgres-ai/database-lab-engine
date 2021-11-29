/*
2020 © Postgres.ai
*/

// Package observer provides clone monitoring.
package observer

import (
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// Session describes a session of service monitoring.
type Session struct {
	SessionID  uint64                    `json:"session_id"`
	StartedAt  time.Time                 `json:"started_at"`
	FinishedAt time.Time                 `json:"finished_at"`
	Config     types.Config              `json:"config"`
	Tags       map[string]string         `json:"tags"`
	Artifacts  []string                  `json:"artifacts,omitempty"`
	Result     *models.ObservationResult `json:"result,omitempty"`
	state      State
}

// State contains database state of the session.
type State struct {
	InitialDBSize    int64
	CurrentDBSize    int64
	MaxDBQueryTimeMS float64
	ObjectStat       ObjectsStat
	LogErrors        LogErrors
	OverallError     bool
}

// NewSession creates a new observing session.
func NewSession(sessionID uint64, startedAt time.Time, config types.Config, tags map[string]string) *Session {
	return &Session{
		SessionID: sessionID,
		StartedAt: startedAt,
		Config:    config,
		Tags:      tags,
	}
}

// IsFinished checks if the value FinishedAt is zero.
func (s Session) IsFinished() bool {
	return !s.FinishedAt.IsZero()
}
