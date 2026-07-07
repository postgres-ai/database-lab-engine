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

// ProbeSourceRequest is the JSON body for POST /admin/probe-source.
// The password is supplied as a separate field so it is never embedded in
// the URL (the engine rejects such inputs).
type ProbeSourceRequest struct {
	URL      string `json:"url"`
	Password string `json:"password"`
}

// SourceConnection is the source endpoint described in a ProposedConfig.
// Password is intentionally absent — proposals never carry the credential.
type SourceConnection struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	DBName   string `json:"dbname"`
}

// ProposedConfig is the response of POST /admin/probe-source. It mirrors
// probe.ProposedConfig as a JSON-tagged DTO so the engine can change the
// internal type without breaking wire compatibility.
//
// DockerImage carries the provider key (e.g. "rds", "supabase") for backward
// compatibility; the UI resolves the concrete docker-image id from its catalog
// when resolvedImage is empty. DockerTag is the registry tag the engine selected
// (empty when no registry tag matched). ResolvedImage is the full `<repo>:<tag>`
// reference the engine resolved from the live registries (with offline
// fallback); when set, clients should prefer it over re-deriving from the
// provider key. CollationVersion is the source's recorded collation version
// (PG15+), surfaced for the glibc-mismatch warning.
//
// There is intentionally no Warnings field — the UI generates copy from
// structured signals (detectedProvider, memoryProbed, collationVersion).
type ProposedConfig struct {
	Source                 SourceConnection  `json:"source"`
	DetectedProvider       string            `json:"detectedProvider"`
	DockerImage            string            `json:"dockerImage"`
	DockerTag              string            `json:"dockerTag"`
	ResolvedImage          string            `json:"resolvedImage"`
	PgMajorVersion         int               `json:"pgMajorVersion"`
	CollationVersion       string            `json:"collationVersion"`
	Databases              []string          `json:"databases"`
	SharedBuffers          string            `json:"sharedBuffers"`
	MemoryProbed           bool              `json:"memoryProbed"`
	SharedPreloadLibraries string            `json:"sharedPreloadLibraries"`
	QueryTuning            map[string]string `json:"queryTuning"`
}
