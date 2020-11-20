/*
2019 © Postgres.ai
*/

package models

type InstanceStatus struct {
	Status              *Status     `json:"status"`
	FileSystem          *FileSystem `json:"fileSystem"`
	DataSize            uint64      `json:"dataSize"`
	ExpectedCloningTime float64     `json:"expectedCloningTime"`
	NumClones           uint64      `json:"numClones"`
	Clones              []*Clone    `json:"clones"`
}

// Health represents a response for heath-check requests.
type Health struct {
	Version string `json:"version"`
}
