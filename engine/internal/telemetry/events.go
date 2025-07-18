/*
2021 © Postgres.ai
*/

// Package telemetry contains tools to collect Database Lab Engine data.
package telemetry

import (
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// EngineStarted describes the engine start event.
type EngineStarted struct {
	EngineVersion string        `json:"engine_version"`
	DBEngine      string        `json:"db_engine"`
	DBVersion     string        `json:"db_version"`
	Pools         PoolStat      `json:"pools"`
	Restore       Restore       `json:"restore"`
	System        models.System `json:"system"`
}

// PoolStat describes the pool stat data.
type PoolStat struct {
	FSType    string `json:"fs_type"`
	Number    int    `json:"number"`
	TotalSize uint64 `json:"total_size"`
	TotalUsed uint64 `json:"total_used"`
}

// Restore describes the restore data.
type Restore struct {
	Mode       models.RetrievalMode `json:"mode"`
	Refreshing string               `json:"refreshing"`
	Jobs       []string             `json:"jobs"`
}

// EngineStopped describes the engine stop event.
type EngineStopped struct {
	Uptime float64 `json:"uptime"`
}

// SnapshotCreated describes a snapshot creation event.
type SnapshotCreated struct{}

// CloneCreated describes the clone creation and clone reset events.
type CloneCreated struct {
	ID          string   `json:"id"`
	CloningTime float64  `json:"cloning_time"`
	DSADiff     *float64 `json:"dsa_diff,omitempty"`
}

// CloneUpdated describes the clone updates.
type CloneUpdated struct {
	ID        string `json:"id"`
	Protected bool   `json:"protected"`
}

// CloneDestroyed describes a clone destruction event.
type CloneDestroyed struct {
	ID string `json:"id"`
}

// BranchCreated describes a branch creation event.
type BranchCreated struct {
	Name string `json:"name"`
}

// BranchDestroyed describes a branch destruction event.
type BranchDestroyed struct {
	Name string `json:"name"`
}

// ConfigUpdated describes the config updates.
type ConfigUpdated struct{}

// Alert describes alert events.
type Alert struct {
	Level   models.AlertType `json:"level"`
	Message string           `json:"message"`
}
