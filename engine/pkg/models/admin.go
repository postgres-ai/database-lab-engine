package models

import (
	"fmt"
)

// StatusType defines the DB status.
type StatusType int

const (
	// TCStatusOK defines the status code OK of the test connection request.
	TCStatusOK = iota

	// TCStatusNotice defines the status code "notice" of the test connection request.
	TCStatusNotice

	// TCStatusWarning defines the status code "warning" of the test connection request.
	TCStatusWarning

	// TCStatusError defines the status code "error" of the test connection request.
	TCStatusError

	// TCResultOK defines the result without errors of the test connection request.
	TCResultOK = "ok"

	// TCResultConnectionError defines a connection error of the test connection request.
	TCResultConnectionError = "connection_error"

	// TCResultQueryError defines a query error of the test connection request.
	TCResultQueryError = "query_error"

	// TCResultUnexploredImage defines the notice about unexplored Docker image yet.
	TCResultUnexploredImage = "unexplored_image"

	// TCResultMissingExtension defines the warning about a missing extension.
	TCResultMissingExtension = "missing_extension"

	// TCResultMissingLocale defines the warning about a missing locale.
	TCResultMissingLocale = "missing_locale"

	// TCResultUnverifiedDB defines notification of the presence of unverified databases.
	TCResultUnverifiedDB = "unverified_database"

	// TCMessageOK defines the source database is ready for dump and restore.
	TCMessageOK = "Database ready for dump and restore"
)

// String prints status name.
func (s StatusType) String() string {
	switch s {
	case TCStatusOK:
		return "ok"

	case TCStatusNotice:
		return "notice"

	case TCStatusWarning:
		return "warning"

	case TCStatusError:
		return "error"
	}

	return "unknown"
}

// MarshalJSON marshals the StatusType struct.
func (s StatusType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", s.String())), nil
}

// TestConnection represents the response of the test connection request.
type TestConnection struct {
	Status  StatusType `json:"status"`
	Result  string     `json:"result"`
	Message string     `json:"message"`
}

// DBSource represents the response of the database source checks.
type DBSource struct {
	*TestConnection
	DBVersion    int               `json:"dbVersion,omitempty"`
	TuningParams map[string]string `json:"tuningParams"`
}
