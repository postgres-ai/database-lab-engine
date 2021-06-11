package observer

import (
	"time"
)

// SummaryArtifact represents session summary.
type SummaryArtifact struct {
	SessionID     uint64    `json:"session_id"`
	CloneID       string    `json:"clone_id"`
	Duration      Duration  `json:"duration"`
	DBSize        DBSize    `json:"db_size"`
	Locks         Locks     `json:"locks"`
	LogErrors     LogErrors `json:"log_errors"`
	ArtifactTypes []string  `json:"artifact_types"`
}

// Duration represents summary statistics about session duration.
type Duration struct {
	Total            string    `json:"total"`
	StartedAt        time.Time `json:"started_at"`
	FinishedAt       time.Time `json:"finished_at"`
	MaxQueryDuration string    `json:"query_duration_longest"`
}

// DBSize represents summary statistics about database size.
type DBSize struct {
	Total       string      `json:"total"`
	Diff        string      `json:"diff"`
	ObjectsStat ObjectsStat `json:"objects_stat"`
}

// ObjectsStat represents summary statistics about objects size.
type ObjectsStat struct {
	Count               int   `json:"count"`
	RowEstimateSum      int64 `json:"row_estimate_sum"`
	TotalSizeBytesSum   int64 `json:"total_size_bytes_sum"`
	TableSizeBytesSum   int64 `json:"table_size_bytes_sum"`
	IndexesSizeBytesSum int64 `json:"indexes_size_bytes_sum"`
	ToastSizeBytesSum   int64 `json:"toast_size_bytes_sum"`
}

// Locks represents summary statistics about locks.
type Locks struct {
	TotalInterval   int `json:"total_interval"`
	WarningInterval int `json:"warning_interval"`
}

// LogErrors contains details about log errors statistics.
type LogErrors struct {
	Count   int    `json:"count"`
	Message string `json:"message"`
}
