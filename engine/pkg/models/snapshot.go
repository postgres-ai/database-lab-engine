/*
2019 © Postgres.ai
*/

package models

import "time"

// Snapshot defines a snapshot entity.
type Snapshot struct {
	ID            string     `json:"id"`
	CreatedAt     *LocalTime `json:"createdAt"`
	DataStateAt   *LocalTime `json:"dataStateAt"`
	PhysicalSize  uint64     `json:"physicalSize"`
	LogicalSize   uint64     `json:"logicalSize"`
	Pool          string     `json:"pool"`
	NumClones     int        `json:"numClones"`
	Clones        []string   `json:"clones"`
	Branch        string     `json:"branch"`
	Message       string     `json:"message"`
	Protected     bool       `json:"protected"`
	ProtectedTill *LocalTime `json:"protectedTill,omitempty"`
	DeleteAt      *LocalTime `json:"deleteAt,omitempty"`
}

// IsProtected returns true if the snapshot is currently protected.
func (s *Snapshot) IsProtected() bool {
	return isProtected(s.Protected, s.ProtectedTill)
}

// ProtectionExpiresIn returns the duration until protection expires.
// Returns 0 if not protected, protection has no expiry, or protection has already expired.
func (s *Snapshot) ProtectionExpiresIn() time.Duration {
	return protectionExpiresIn(s.Protected, s.ProtectedTill)
}

// SnapshotView represents a view of snapshot.
type SnapshotView struct {
	*Snapshot
	PhysicalSize Size `json:"physicalSize"`
	LogicalSize  Size `json:"logicalSize"`
}
