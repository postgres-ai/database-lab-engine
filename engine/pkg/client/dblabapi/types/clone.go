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
	Branch    string                     `json:"branch"`
	Revision  int                        `json:"-"`
}

// CloneUpdateRequest represents params of an update request.
type CloneUpdateRequest struct {
	Protected  bool `json:"protected"`
	RenewLease bool `json:"renewLease,omitempty"`
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

// SnapshotCreateRequest describes params for creating snapshot request.
type SnapshotCreateRequest struct {
	PoolName string `json:"poolName"`
}

// SnapshotDestroyRequest describes params for destroying snapshot request.
type SnapshotDestroyRequest struct {
	SnapshotID string `json:"snapshotID"`
	Force      bool   `json:"force"`
}

// SnapshotCloneCreateRequest describes params for creating snapshot request from clone.
type SnapshotCloneCreateRequest struct {
	CloneID string `json:"cloneID"`
	Message string `json:"message"`
}

// BranchCreateRequest describes params for creating branch request.
type BranchCreateRequest struct {
	BranchName string `json:"branchName"`
	BaseBranch string `json:"baseBranch"`
	SnapshotID string `json:"snapshotID"`
}

// SnapshotResponse describes commit response.
type SnapshotResponse struct {
	SnapshotID string `json:"snapshotID"`
}

// ResetRequest describes params for reset request.
type ResetRequest struct {
	SnapshotID string `json:"snapshotID"`
}

// LogRequest describes params for log request.
type LogRequest struct {
	BranchName string `json:"branchName"`
}

// BranchDeleteRequest describes params for deleting branch request.
type BranchDeleteRequest struct {
	BranchName string `json:"branchName"`
}
