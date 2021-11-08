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

// RefreshStatus defines status of refreshing data.
type RefreshStatus string

const (
	// Inactive defines status when data refreshing is disabled.
	Inactive RefreshStatus = "inactive"
	// Refreshing defines status when data refreshing is in progress.
	Refreshing RefreshStatus = "refreshing"
	// Finished defines status when data refreshing is finished.
	Finished RefreshStatus = "finished"
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
	Mode        RetrievalMode `json:"mode"`
	Status      RefreshStatus `json:"status"`
	LastRefresh *time.Time    `json:"lastRefresh"`
	NextRefresh *time.Time    `json:"nextRefresh"`
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
