/*
2020 © Postgres.ai
*/

package types

// StartObservationRequest represents a request for the start observation endpoint.
type StartObservationRequest struct {
	CloneID string            `json:"clone_id"`
	Config  Config            `json:"config"`
	Tags    map[string]string `json:"tags"`
	DBName  string            `json:"db_name"`
}

// Config defines configuration options for observer.
type Config struct {
	ObservationInterval uint64 `json:"observation_interval"`
	MaxLockDuration     uint64 `json:"max_lock_duration"`
	MaxDuration         uint64 `json:"max_duration"`
}

// StopObservationRequest represents a request for the stop observation endpoint.
type StopObservationRequest struct {
	CloneID      string `json:"clone_id"`
	OverallError bool   `json:"overall_error"`
}
