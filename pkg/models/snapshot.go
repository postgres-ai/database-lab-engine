/*
2019 © Postgres.ai
*/

package models

// Snapshot defines a snapshot entity.
type Snapshot struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"createdAt"`
	DataStateAt  string `json:"dataStateAt"`
	PhysicalSize string `json:"physicalSize"`
	LogicalSize  string `json:"logicalSize"`
	Pool         string `json:"pool"`
}
