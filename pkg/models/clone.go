/*
2019 © Postgres.ai
*/

package models

// Clone defines a clone model.
type Clone struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Snapshot  *Snapshot      `json:"snapshot"`
	Protected bool           `json:"protected"`
	DeleteAt  string         `json:"deleteAt"`
	CreatedAt string         `json:"createdAt"`
	Status    *Status        `json:"status"`
	DB        *Database      `json:"db"`
	Metadata  *CloneMetadata `json:"metadata"`

	// TODO(anatoly): Remove?
	Project string `json:"project"`
}

// CloneMetadata contains fields describing a clone model.
type CloneMetadata struct {
	CloneSize      uint64  `json:"cloneSize"`
	CloningTime    float64 `json:"cloningTime"`
	MaxIdleMinutes uint    `json:"maxIdleMinutes"`
}

// PatchCloneRequest defines a struct for clone updating.
type PatchCloneRequest struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}
