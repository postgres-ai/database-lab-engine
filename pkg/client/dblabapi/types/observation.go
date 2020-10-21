/*
2020 © Postgres.ai
*/

package types

// StartObservationRequest represents a request for a start observation endpoint.
type StartObservationRequest struct {
	CloneID string            `json:"clone_id"`
	Config  Config            `json:"config"`
	Tags    map[string]string `json:"tags"`
}

// Config defines configuration options for observer.
type Config struct {
	ObservationInterval uint64 `json:"observation_interval"`
	MaxLockDuration     uint64 `json:"max_lock_duration"`
	MaxDuration         uint64 `json:"max_duration"`
}

// StopObservationRequest represents a request for a stop observation endpoint.
type StopObservationRequest struct {
	CloneID string `json:"clone_id"`
}
