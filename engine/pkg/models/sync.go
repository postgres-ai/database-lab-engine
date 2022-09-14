package models

// Sync defines the status of synchronization containers
type Sync struct {
	Status            Status `json:"status"`
	StartedAt         string `json:"startedAt,omitempty"`
	LastReplayedLsn   string `json:"lastReplayedLsn"`
	LastReplayedLsnAt string `json:"lastReplayedLsnAt"`
	ReplicationLag    int    `json:"replicationLag"`
	ReplicationUptime int    `json:"replicationUptime"`
}
