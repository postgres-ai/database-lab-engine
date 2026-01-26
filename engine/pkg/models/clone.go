/*
2019 © Postgres.ai
*/

package models

import "time"

// Clone defines a clone model.
type Clone struct {
	ID                    string        `json:"id"`
	Snapshot              *Snapshot     `json:"snapshot"`
	Branch                string        `json:"branch"`
	Revision              int           `json:"revision"`
	Protected             bool          `json:"protected"`
	ProtectedTill         *LocalTime    `json:"protectedTill,omitempty"`
	ProtectionWarningSent bool          `json:"-"`
	DeleteAt              *LocalTime    `json:"deleteAt"`
	CreatedAt             *LocalTime    `json:"createdAt"`
	Status                Status        `json:"status"`
	DB                    Database      `json:"db"`
	Metadata              CloneMetadata `json:"metadata"`
}

// IsProtected returns true if the clone is currently protected.
func (c *Clone) IsProtected() bool {
	if !c.Protected {
		return false
	}

	if c.ProtectedTill == nil {
		return true
	}

	return c.ProtectedTill.After(time.Now())
}

// ProtectionExpiresIn returns the duration until protection expires.
// Returns 0 if not protected, protection has no expiry, or protection has already expired.
func (c *Clone) ProtectionExpiresIn() time.Duration {
	if !c.Protected || c.ProtectedTill == nil {
		return 0
	}

	duration := time.Until(c.ProtectedTill.Time)
	if duration < 0 {
		return 0
	}

	return duration
}

// CloneMetadata contains fields describing a clone model.
type CloneMetadata struct {
	CloneDiffSize                  uint64  `json:"cloneDiffSize"`
	LogicalSize                    uint64  `json:"logicalSize"`
	CloningTime                    float64 `json:"cloningTime"`
	MaxIdleMinutes                 uint    `json:"maxIdleMinutes"`
	ProtectionLeaseDurationMinutes uint    `json:"protectionLeaseDurationMinutes,omitempty"`
	ProtectionMaxDurationMinutes   uint    `json:"protectionMaxDurationMinutes,omitempty"`
}

// CloneView represents a view of clone model.
type CloneView struct {
	*Clone
	Snapshot *SnapshotView     `json:"snapshot"`
	Metadata CloneMetadataView `json:"metadata"`
}

// CloneMetadataView represents a view of clone metadata.
type CloneMetadataView struct {
	*CloneMetadata
	CloneDiffSize Size `json:"cloneDiffSize"`
	LogicalSize   Size `json:"logicalSize"`
}
