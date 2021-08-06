/*
2019 © Postgres.ai
*/

package models

import (
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

// InstanceStatus represents status of a Database Lab Engine instance.
type InstanceStatus struct {
	Status              *Status     `json:"status"`
	FileSystem          *FileSystem `json:"fileSystem"`
	DataSize            uint64      `json:"dataSize"`
	DataSizeHR          string      `json:"dataSizeHR"`
	ExpectedCloningTime float64     `json:"expectedCloningTime"`
	NumClones           uint64      `json:"numClones"`
	Clones              []*Clone    `json:"clones"`
	Pools               []PoolEntry `json:"pools"`
}

// PoolEntry represents a pool entry.
type PoolEntry struct {
	Name        string               `json:"name"`
	Mode        string               `json:"mode"`
	DataStateAt string               `json:"dataStateAt"`
	Status      resources.PoolStatus `json:"status"`
	CloneList   []string             `json:"cloneList"`
	Disk        *resources.Disk      `json:"disk"`
}

// Health represents a response for heath-check requests.
type Health struct {
	Version string `json:"engine_version"`
}
