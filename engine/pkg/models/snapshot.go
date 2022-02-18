/*
2019 © Postgres.ai
*/

package models

// Snapshot defines a snapshot entity.
type Snapshot struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"createdAt"`
	DataStateAt  string `json:"dataStateAt"`
	PhysicalSize uint64 `json:"physicalSize"`
	LogicalSize  uint64 `json:"logicalSize"`
	Pool         string `json:"pool"`
	NumClones    int    `json:"numClones"`
}

// SnapshotView represents a view of snapshot.
type SnapshotView struct {
	*Snapshot
	PhysicalSize Size `json:"physicalSize"`
	LogicalSize  Size `json:"logicalSize"`
}
