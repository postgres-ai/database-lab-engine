/*
2020 © Postgres.ai
*/

package models

import (
	"time"
)

// ObservationResult represents a result of observation session.
type ObservationResult struct {
	Status    string     `json:"status"`
	Intervals []Interval `json:"intervals"`
	Summary   Summary    `json:"summary"`
}

// Interval represents data of an observation interval.
type Interval struct {
	StartedAt time.Time `json:"started_at"`
	Duration  float64   `json:"duration"`
	Warning   string    `json:"warning"`
}

// Summary represents a summary of observation.
type Summary struct {
	TotalDuration    float64   `json:"total_duration"`
	TotalIntervals   uint      `json:"total_intervals"`
	WarningIntervals uint      `json:"warning_intervals"`
	Checklist        Checklist `json:"checklist"`
}

// Checklist represents a list of observation checks.
type Checklist struct {
	Success  bool `json:"overall_success"`
	Duration bool `json:"session_duration_acceptable"`
	Locks    bool `json:"no_long_dangerous_locks"`
}
