/*
2024 © Postgres.ai
*/

package cloning

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	branchesFilename  = "branches.json"
	snapshotsFilename = "snapshots_meta.json"
)

// BranchMeta stores branch metadata that needs to be persisted.
type BranchMeta struct {
	Name           string                `json:"name"`
	Protected      bool                  `json:"protected"`
	DeleteAt       *models.LocalTime     `json:"deleteAt,omitempty"`
	AutoDeleteMode models.AutoDeleteMode `json:"autoDeleteMode"`
}

// SnapshotMeta stores snapshot metadata that needs to be persisted.
type SnapshotMeta struct {
	ID             string                `json:"id"`
	Protected      bool                  `json:"protected"`
	DeleteAt       *models.LocalTime     `json:"deleteAt,omitempty"`
	AutoDeleteMode models.AutoDeleteMode `json:"autoDeleteMode"`
}

// EntityStorage manages branch and snapshot metadata persistence.
type EntityStorage struct {
	branchMutex   sync.RWMutex
	snapshotMutex sync.RWMutex
	branches      map[string]*BranchMeta
	snapshots     map[string]*SnapshotMeta
}

// NewEntityStorage creates a new entity storage instance.
func NewEntityStorage() *EntityStorage {
	return &EntityStorage{
		branches:  make(map[string]*BranchMeta),
		snapshots: make(map[string]*SnapshotMeta),
	}
}

// RestoreBranchesState restores branch metadata from disk.
func (es *EntityStorage) RestoreBranchesState() error {
	branchesPath, err := util.GetMetaPath(branchesFilename)
	if err != nil {
		return fmt.Errorf("failed to get path of branches file: %w", err)
	}

	es.branchMutex.Lock()
	defer es.branchMutex.Unlock()

	es.branches = make(map[string]*BranchMeta)

	data, err := os.ReadFile(branchesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to read branches data: %w", err)
	}

	return json.Unmarshal(data, &es.branches)
}

// SaveBranchesState writes branch metadata to disk.
func (es *EntityStorage) SaveBranchesState() {
	branchesPath, err := util.GetMetaPath(branchesFilename)
	if err != nil {
		log.Err("failed to get path of branches file", err)
		return
	}

	es.branchMutex.RLock()
	defer es.branchMutex.RUnlock()

	data, err := json.Marshal(es.branches)
	if err != nil {
		log.Err("failed to encode branches data", err)
		return
	}

	if err := os.WriteFile(branchesPath, data, 0600); err != nil {
		log.Err("failed to save branches state", err)
	}
}

// RestoreSnapshotsState restores snapshot metadata from disk.
func (es *EntityStorage) RestoreSnapshotsState() error {
	snapshotsPath, err := util.GetMetaPath(snapshotsFilename)
	if err != nil {
		return fmt.Errorf("failed to get path of snapshots file: %w", err)
	}

	es.snapshotMutex.Lock()
	defer es.snapshotMutex.Unlock()

	es.snapshots = make(map[string]*SnapshotMeta)

	data, err := os.ReadFile(snapshotsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to read snapshots metadata: %w", err)
	}

	return json.Unmarshal(data, &es.snapshots)
}

// SaveSnapshotsState writes snapshot metadata to disk.
func (es *EntityStorage) SaveSnapshotsState() {
	snapshotsPath, err := util.GetMetaPath(snapshotsFilename)
	if err != nil {
		log.Err("failed to get path of snapshots file", err)
		return
	}

	es.snapshotMutex.RLock()
	defer es.snapshotMutex.RUnlock()

	data, err := json.Marshal(es.snapshots)
	if err != nil {
		log.Err("failed to encode snapshots metadata", err)
		return
	}

	if err := os.WriteFile(snapshotsPath, data, 0600); err != nil {
		log.Err("failed to save snapshots metadata state", err)
	}
}

// GetBranchMeta returns metadata for a branch.
func (es *EntityStorage) GetBranchMeta(name string) *BranchMeta {
	es.branchMutex.RLock()
	defer es.branchMutex.RUnlock()

	return es.branches[name]
}

// SetBranchMeta sets metadata for a branch.
func (es *EntityStorage) SetBranchMeta(meta *BranchMeta) {
	es.branchMutex.Lock()
	es.branches[meta.Name] = meta
	es.branchMutex.Unlock()

	es.SaveBranchesState()
}

// UpdateBranchMeta updates existing branch metadata or creates new one.
func (es *EntityStorage) UpdateBranchMeta(name string, protected *bool, deleteAt *models.LocalTime, autoDeleteMode *models.AutoDeleteMode) *BranchMeta {
	es.branchMutex.Lock()
	defer es.branchMutex.Unlock()

	meta, exists := es.branches[name]
	if !exists {
		meta = &BranchMeta{Name: name}
		es.branches[name] = meta
	}

	if protected != nil {
		meta.Protected = *protected
	}

	if deleteAt != nil {
		meta.DeleteAt = deleteAt
	}

	if autoDeleteMode != nil {
		meta.AutoDeleteMode = *autoDeleteMode
	}

	go es.SaveBranchesState()

	return meta
}

// DeleteBranchMeta removes metadata for a branch.
func (es *EntityStorage) DeleteBranchMeta(name string) {
	es.branchMutex.Lock()
	delete(es.branches, name)
	es.branchMutex.Unlock()

	es.SaveBranchesState()
}

// IsBranchProtected checks if a branch is protected.
func (es *EntityStorage) IsBranchProtected(name string) bool {
	es.branchMutex.RLock()
	defer es.branchMutex.RUnlock()

	if meta, ok := es.branches[name]; ok {
		return meta.Protected
	}

	return false
}

// GetSnapshotMeta returns metadata for a snapshot.
func (es *EntityStorage) GetSnapshotMeta(id string) *SnapshotMeta {
	es.snapshotMutex.RLock()
	defer es.snapshotMutex.RUnlock()

	return es.snapshots[id]
}

// SetSnapshotMeta sets metadata for a snapshot.
func (es *EntityStorage) SetSnapshotMeta(meta *SnapshotMeta) {
	es.snapshotMutex.Lock()
	es.snapshots[meta.ID] = meta
	es.snapshotMutex.Unlock()

	es.SaveSnapshotsState()
}

// UpdateSnapshotMeta updates existing snapshot metadata or creates new one.
func (es *EntityStorage) UpdateSnapshotMeta(id string, protected *bool, deleteAt *models.LocalTime, autoDeleteMode *models.AutoDeleteMode) *SnapshotMeta {
	es.snapshotMutex.Lock()
	defer es.snapshotMutex.Unlock()

	meta, exists := es.snapshots[id]
	if !exists {
		meta = &SnapshotMeta{ID: id}
		es.snapshots[id] = meta
	}

	if protected != nil {
		meta.Protected = *protected
	}

	if deleteAt != nil {
		meta.DeleteAt = deleteAt
	}

	if autoDeleteMode != nil {
		meta.AutoDeleteMode = *autoDeleteMode
	}

	go es.SaveSnapshotsState()

	return meta
}

// DeleteSnapshotMeta removes metadata for a snapshot.
func (es *EntityStorage) DeleteSnapshotMeta(id string) {
	es.snapshotMutex.Lock()
	delete(es.snapshots, id)
	es.snapshotMutex.Unlock()

	es.SaveSnapshotsState()
}

// IsSnapshotProtected checks if a snapshot is protected.
func (es *EntityStorage) IsSnapshotProtected(id string) bool {
	es.snapshotMutex.RLock()
	defer es.snapshotMutex.RUnlock()

	if meta, ok := es.snapshots[id]; ok {
		return meta.Protected
	}

	return false
}

// GetExpiredBranches returns branches that have passed their deleteAt time.
func (es *EntityStorage) GetExpiredBranches() []*BranchMeta {
	es.branchMutex.RLock()
	defer es.branchMutex.RUnlock()

	now := time.Now()
	expired := make([]*BranchMeta, 0)

	for _, meta := range es.branches {
		if meta.DeleteAt != nil && meta.AutoDeleteMode != models.AutoDeleteOff {
			if time.Time(*meta.DeleteAt).Before(now) {
				expired = append(expired, meta)
			}
		}
	}

	return expired
}

// GetExpiredSnapshots returns snapshots that have passed their deleteAt time.
func (es *EntityStorage) GetExpiredSnapshots() []*SnapshotMeta {
	es.snapshotMutex.RLock()
	defer es.snapshotMutex.RUnlock()

	now := time.Now()
	expired := make([]*SnapshotMeta, 0)

	for _, meta := range es.snapshots {
		if meta.DeleteAt != nil && meta.AutoDeleteMode != models.AutoDeleteOff {
			if time.Time(*meta.DeleteAt).Before(now) {
				expired = append(expired, meta)
			}
		}
	}

	return expired
}

// ListBranchMetas returns all branch metadata.
func (es *EntityStorage) ListBranchMetas() map[string]*BranchMeta {
	es.branchMutex.RLock()
	defer es.branchMutex.RUnlock()

	result := make(map[string]*BranchMeta, len(es.branches))
	for k, v := range es.branches {
		result[k] = v
	}

	return result
}

// ListSnapshotMetas returns all snapshot metadata.
func (es *EntityStorage) ListSnapshotMetas() map[string]*SnapshotMeta {
	es.snapshotMutex.RLock()
	defer es.snapshotMutex.RUnlock()

	result := make(map[string]*SnapshotMeta, len(es.snapshots))
	for k, v := range es.snapshots {
		result[k] = v
	}

	return result
}
