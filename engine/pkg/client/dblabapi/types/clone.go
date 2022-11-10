/*
2020 © Postgres.ai
*/

// Package types provides request structures for Database Lab HTTP API.
package types

// CloneCreateRequest represents clone params of a create request.
type CloneCreateRequest struct {
	ID        string                     `json:"id"`
	Protected bool                       `json:"protected"`
	DB        *DatabaseRequest           `json:"db"`
	Snapshot  *SnapshotCloneFieldRequest `json:"snapshot"`
	ExtraConf map[string]string          `json:"extra_conf"`
}

// CloneUpdateRequest represents params of an update request.
type CloneUpdateRequest struct {
	Protected bool `json:"protected"`
}

// DatabaseRequest represents database params of a clone request.
type DatabaseRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Restricted bool   `json:"restricted"`
	DBName     string `json:"db_name"`
}

// SnapshotCloneFieldRequest represents snapshot params of a create request.
type SnapshotCloneFieldRequest struct {
	ID string `json:"id"`
}

// ResetCloneRequest represents snapshot params of a reset request.
type ResetCloneRequest struct {
	SnapshotID string `json:"snapshotID"`
	Latest     bool   `json:"latest"`
}

// SnapshotCreateRequest describes params for a creating snapshot request.
type SnapshotCreateRequest struct {
	PoolName string `json:"poolName"`
}

// SnapshotDestroyRequest describes params for a destroying snapshot request.
type SnapshotDestroyRequest struct {
	SnapshotID string `json:"snapshotID"`
}
