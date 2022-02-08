/*
2019 © Postgres.ai
*/

package models

// Clone defines a clone model.
type Clone struct {
	ID        string        `json:"id"`
	Snapshot  *Snapshot     `json:"snapshot"`
	Protected bool          `json:"protected"`
	DeleteAt  string        `json:"deleteAt"`
	CreatedAt string        `json:"createdAt"`
	Status    Status        `json:"status"`
	DB        Database      `json:"db"`
	Metadata  CloneMetadata `json:"metadata"`
}

// CloneMetadata contains fields describing a clone model.
type CloneMetadata struct {
	CloneDiffSize  uint64  `json:"cloneDiffSize"`
	LogicalSize    uint64  `json:"logicalSize"`
	CloningTime    float64 `json:"cloningTime"`
	MaxIdleMinutes uint    `json:"maxIdleMinutes"`
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
