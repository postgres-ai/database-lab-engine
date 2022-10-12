/*
2019 © Postgres.ai
*/

package models

import (
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
)

// InstanceStatus represents status of a Database Lab Engine instance.
type InstanceStatus struct {
	Status          *Status          `json:"status"`
	Engine          Engine           `json:"engine"`
	Pools           []PoolEntry      `json:"pools"`
	Cloning         Cloning          `json:"cloning"`
	Retrieving      Retrieving       `json:"retrieving"`
	Provisioner     ContainerOptions `json:"provisioner"`
	Synchronization *Sync            `json:"synchronization"`
}

// PoolEntry represents a pool entry.
type PoolEntry struct {
	Name        string               `json:"name"`
	Mode        string               `json:"mode"`
	DataStateAt *LocalTime           `json:"dataStateAt"`
	Status      resources.PoolStatus `json:"status"`
	CloneList   []string             `json:"cloneList"`
	FileSystem  FileSystem           `json:"fileSystem"`
}

// ContainerOptions describes options for running containers.
type ContainerOptions struct {
	DockerImage     string            `json:"dockerImage"`
	ContainerConfig map[string]string `json:"containerConfig"`
}

// Cloning represents info about the cloning process.
type Cloning struct {
	ExpectedCloningTime float64  `json:"expectedCloningTime"`
	NumClones           uint64   `json:"numClones"`
	Clones              []*Clone `json:"clones"`
}

// Engine represents info about Database Lab Engine instance.
type Engine struct {
	Version              string     `json:"version"`
	Edition              string     `json:"edition"`
	StartedAt            *LocalTime `json:"startedAt,omitempty"`
	Telemetry            *bool      `json:"telemetry,omitempty"`
	AllowModifyingConfig *bool      `json:"allowModifyingConfig,omitempty"`
}

// CloneList represents a list of clones.
type CloneList struct {
	Cloning Cloning `json:"cloning"`
}

// CloneListView represents cloning process views.
type CloneListView struct {
	Cloning CloningView `json:"cloning"`
}

// CloningView represents a list of clone views.
type CloningView struct {
	Clones []*CloneView `json:"clones"`
}

// InstanceStatusView represents view of a Database Lab Engine instance status.
type InstanceStatusView struct {
	*InstanceStatus
	Pools []PoolEntryView `json:"pools"`
}

// PoolEntryView represents a pool entry view.
type PoolEntryView struct {
	*PoolEntry
	FileSystem FileSystemView `json:"fileSystem"`
}

// WSToken represents web-socket token response.
type WSToken struct {
	Token string `json:"token"`
}
