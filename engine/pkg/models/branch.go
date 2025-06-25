package models

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
	ID          string   `json:"id"`
	Parent      string   `json:"parent"`
	Child       []string `json:"child"`
	Branch      []string `json:"branch"`
	Root        []string `json:"root"`
	DataStateAt string   `json:"dataStateAt"`
	Message     string   `json:"message"`
	Dataset     string   `json:"dataset"`
	Clones      []string `json:"clones"`
}

// BranchView describes branch view.
type BranchView struct {
	Name         string `json:"name"`
	Parent       string `json:"parent"`
	DataStateAt  string `json:"dataStateAt"`
	SnapshotID   string `json:"snapshotID"`
	Dataset      string `json:"dataset"`
	NumSnapshots int    `json:"numSnapshots"`
}

// BranchEntity defines a branch-snapshot pair.
type BranchEntity struct {
	Name       string
	SnapshotID string
}
