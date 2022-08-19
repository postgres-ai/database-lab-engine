/*
2021 © Postgres.ai
*/

package models

import (
	"time"
)

// RetrievalMode defines mode of retrieval subsystem.
type RetrievalMode string

const (
	// Physical defines physical retrieval mode.
	Physical RetrievalMode = "physical"
	// Logical defines logical retrieval mode.
	Logical RetrievalMode = "logical"
	// Unknown defines the case when retrieval mode is unknown or is not set.
	Unknown RetrievalMode = "unknown"
)

// RetrievalStatus defines status of refreshing data.
type RetrievalStatus string

const (
	// Inactive defines status when data retrieving is disabled.
	Inactive RetrievalStatus = "inactive"
	// Pending defines status when data retrieving is pending configuration verification.
	Pending RetrievalStatus = "pending"
	// Failed defines status when data retrieving is failed.
	Failed RetrievalStatus = "failed"
	// Refreshing defines status when data retrieving is in progress.
	Refreshing RetrievalStatus = "refreshing"
	// Renewed defines status when data retrieving is successfully completed.
	Renewed RetrievalStatus = "renewed"
	// Snapshotting defines status when data snapshotting is in progress.
	Snapshotting RetrievalStatus = "snapshotting"
	// Finished defines status when data retrieving is finished.
	Finished RetrievalStatus = "finished"
)

// AlertLevel defines levels of retrieval alert.
type AlertLevel string

const (
	// ErrorLevel defines error alerts.
	ErrorLevel AlertLevel = "error"
	// WarningLevel defines warning alerts.
	WarningLevel AlertLevel = "warning"
	// UnknownLevel defines unknown alerts.
	UnknownLevel AlertLevel = "unknown"
)

// AlertType defines type of retrieval alert.
type AlertType string

const (
	// RefreshFailed describes alert when data refreshing is failed.
	RefreshFailed AlertType = "refresh_failed"

	// RefreshSkipped describes alert when data refreshing is skipped.
	RefreshSkipped AlertType = "refresh_skipped"
)

// Retrieving represents state of retrieval subsystem.
type Retrieving struct {
	Mode        RetrievalMode       `json:"mode"`
	Status      RetrievalStatus     `json:"status"`
	LastRefresh *LocalTime          `json:"lastRefresh"`
	NextRefresh *LocalTime          `json:"nextRefresh"`
	Alerts      map[AlertType]Alert `json:"alerts"`
	Activity    *Activity           `json:"activity"`
}

// Alert describes retrieval subsystem alert.
type Alert struct {
	Level    AlertLevel `json:"level"`
	Message  string     `json:"message"`
	LastSeen time.Time  `json:"lastSeen"`
	Count    int        `json:"count"`
}

// AlertLevelByType defines relations between alert type and its level.
func AlertLevelByType(alertType AlertType) AlertLevel {
	switch alertType {
	case RefreshFailed:
		return ErrorLevel

	case RefreshSkipped:
		return WarningLevel

	default:
		return UnknownLevel
	}
}

// Activity represents the current job activity.
type Activity struct {
	Source []PGActivityEvent `json:"source"`
	Target []PGActivityEvent `json:"target"`
}

// PGActivityEvent represents pg_stat_activity events.
type PGActivityEvent struct {
	User          string  `json:"user"`
	Query         string  `json:"query"`
	Duration      float64 `json:"duration"`
	WaitEventType string  `json:"waitEventType"`
	WaitEvent     string  `json:"waitEvent"`
}
