/*
2019 © Postgres.ai
*/

package models

// Snapshot defines a snapshot entity.
type Snapshot struct {
	ID           string     `json:"id"`
	CreatedAt    *LocalTime `json:"createdAt"`
	DataStateAt  *LocalTime `json:"dataStateAt"`
	PhysicalSize uint64     `json:"physicalSize"`
	LogicalSize  uint64     `json:"logicalSize"`
	Pool         string     `json:"pool"`
	NumClones    int        `json:"numClones"`
	Clones       []string   `json:"clones"`
	Branch       string     `json:"branch"`
	Message      string     `json:"message"`
}

// SnapshotView represents a view of snapshot.
type SnapshotView struct {
	*Snapshot
	PhysicalSize Size `json:"physicalSize"`
	LogicalSize  Size `json:"logicalSize"`
}
