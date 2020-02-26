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
	Name string

	// Database.
	Host       string
	Port       uint
	User       string
	Password   string
	SocketHost string

	// TODO(anatoly): Were private fields. How to keep them private?
	// For user-defined username and password.
	EphemeralUser     string
	EphemeralPassword string
}

// Disk defines disk status.
// TODO(anatoly): Merge with disk from models?
type Disk struct {
	Size     uint64
	Free     uint64
	DataSize uint64
}

// Snapshot defines snapshot of the data with related meta-information.
type Snapshot struct {
	ID          string
	CreatedAt   time.Time
	DataStateAt time.Time
}

// SessionState defines current state of a Session.
type SessionState struct {
	CloneSize uint64
}
