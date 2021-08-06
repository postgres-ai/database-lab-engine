/*
2020 © Postgres.ai
*/

// Package resources defines models used for provisioning.
package resources

import (
	"time"
)

// Session defines clone provision information and connection info.
type Session struct {
	ID   string
	Pool string

	// Database.
	Port          uint
	User          string
	SocketHost    string
	EphemeralUser EphemeralUser
	ExtraConfig   map[string]string
}

// Disk defines disk status.
// TODO(anatoly): Merge with disk from models?
type Disk struct {
	Size     uint64 `json:"size"`
	Free     uint64 `json:"free"`
	Used     uint64 `json:"used"`
	DataSize uint64 `json:"dataSize"`
}

// EphemeralUser describes an ephemeral database user defined by Database Lab users.
type EphemeralUser struct {
	// TODO(anatoly): Were private fields. How to keep them private?
	Name        string
	Password    string
	Restricted  bool
	AvailableDB string
}

// Snapshot defines snapshot of the data with related meta-information.
type Snapshot struct {
	ID                string
	CreatedAt         time.Time
	DataStateAt       time.Time
	Used              uint64
	LogicalReferenced uint64
	Pool              string
}

// SessionState defines current state of a Session.
type SessionState struct {
	CloneDiffSize     uint64
	LogicalReferenced uint64
}
