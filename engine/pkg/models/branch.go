package models

import "time"

// Branch defines a branch entity.
type Branch struct {
	Name string `json:"name"`
}

// Repo describes data repository with details about snapshots and branches.
type Repo struct {
	Snapshots map[string]SnapshotDetails `json:"snapshots"`
	Branches  map[string]string          `json:"branches"`
}

// NewRepo creates a new Repo.
func NewRepo() *Repo {
	return &Repo{
		Snapshots: make(map[string]SnapshotDetails),
		Branches:  make(map[string]string),
	}
}

// SnapshotDetails describes snapshot.
type SnapshotDetails struct {
	ID            string     `json:"id"`
	Parent        string     `json:"parent"`
	Child         []string   `json:"child"`
	Branch        []string   `json:"branch"`
	Root          []string   `json:"root"`
	DataStateAt   string     `json:"dataStateAt"`
	Message       string     `json:"message"`
	Dataset       string     `json:"dataset"`
	Clones        []string   `json:"clones"`
	Protected     bool       `json:"protected"`
	ProtectedTill *LocalTime `json:"protectedTill,omitempty"`
	DeleteAt      *LocalTime `json:"deleteAt,omitempty"`
}

// BranchView describes branch view.
type BranchView struct {
	Name          string     `json:"name"`
	BaseDataset   string     `json:"baseDataset"`
	Parent        string     `json:"parent"`
	DataStateAt   string     `json:"dataStateAt"`
	SnapshotID    string     `json:"snapshotID"`
	Dataset       string     `json:"dataset"`
	NumSnapshots  int        `json:"numSnapshots"`
	Protected     bool       `json:"protected"`
	ProtectedTill *LocalTime `json:"protectedTill,omitempty"`
	DeleteAt      *LocalTime `json:"deleteAt,omitempty"`
}

// IsProtected returns true if the branch is currently protected.
func (b *BranchView) IsProtected() bool {
	return isProtected(b.Protected, b.ProtectedTill)
}

// ProtectionExpiresIn returns the duration until protection expires.
// Returns 0 if not protected, protection has no expiry, or protection has already expired.
func (b *BranchView) ProtectionExpiresIn() time.Duration {
	return protectionExpiresIn(b.Protected, b.ProtectedTill)
}

// BranchEntity defines a branch-snapshot pair.
type BranchEntity struct {
	Name       string
	Dataset    string
	SnapshotID string
}
