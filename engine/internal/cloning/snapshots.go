/*
2021 © Postgres.ai
*/

package cloning

import (
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

// SnapshotBox contains instance snapshots.
type SnapshotBox struct {
	snapshotMutex  sync.RWMutex
	items          map[string]*models.Snapshot
	latestSnapshot *models.Snapshot
}

func (c *Base) fetchSnapshots() error {
	entries, err := c.provision.GetSnapshots()
	if err != nil {
		return errors.Wrap(err, "failed to get snapshots")
	}

	var latestSnapshot *models.Snapshot

	snapshots := make(map[string]*models.Snapshot, len(entries))
	cloneCounters := c.counterClones()

	for _, entry := range entries {
		cloneList := []string{}

		if foundList, ok := cloneCounters[entry.ID]; ok {
			cloneList = foundList
		}

		currentSnapshot := &models.Snapshot{
			ID:           entry.ID,
			CreatedAt:    models.NewLocalTime(entry.CreatedAt),
			DataStateAt:  models.NewLocalTime(entry.DataStateAt),
			PhysicalSize: entry.Used,
			LogicalSize:  entry.LogicalReferenced,
			Pool:         entry.Pool,
			Branch:       entry.Branch,
			NumClones:    len(cloneList),
			Clones:       cloneList,
			Message:      entry.Message,
		}

		snapshots[entry.ID] = currentSnapshot
		latestSnapshot = defineLatestSnapshot(latestSnapshot, currentSnapshot)

		log.Dbg("snapshot:", *currentSnapshot)
	}

	c.resetSnapshots(snapshots, latestSnapshot)

	return nil
}

func (c *Base) counterClones() map[string][]string {
	clones := make(map[string][]string, 0)

	c.cloneMutex.RLock()

	for cloneName := range c.clones {
		if c.clones[cloneName] != nil && c.clones[cloneName].Clone.Snapshot != nil {
			snapshotID := c.clones[cloneName].Clone.Snapshot.ID
			clones[snapshotID] = append(clones[snapshotID], cloneName)
		}
	}

	c.cloneMutex.RUnlock()

	return clones
}

func (c *Base) resetSnapshots(snapshotMap map[string]*models.Snapshot, latestSnapshot *models.Snapshot) {
	c.snapshotBox.snapshotMutex.Lock()

	c.snapshotBox.latestSnapshot = latestSnapshot
	c.snapshotBox.items = snapshotMap

	c.snapshotBox.snapshotMutex.Unlock()
}

func (c *Base) addSnapshot(snapshot *models.Snapshot) {
	c.snapshotBox.snapshotMutex.Lock()

	c.snapshotBox.items[snapshot.ID] = snapshot
	c.snapshotBox.latestSnapshot = defineLatestSnapshot(c.snapshotBox.latestSnapshot, snapshot)

	c.snapshotBox.snapshotMutex.Unlock()
}

// defineLatestSnapshot compares two snapshots and defines the latest one.
func defineLatestSnapshot(latest, challenger *models.Snapshot) *models.Snapshot {
	if latest == nil || latest.DataStateAt == nil || latest.DataStateAt.IsZero() || latest.DataStateAt.Before(challenger.DataStateAt.Time) {
		return challenger
	}

	return latest
}

// getLatestSnapshot returns the latest snapshot.
func (c *Base) getLatestSnapshot() (*models.Snapshot, error) {
	c.snapshotBox.snapshotMutex.RLock()
	defer c.snapshotBox.snapshotMutex.RUnlock()

	if c.snapshotBox.latestSnapshot != nil {
		return c.snapshotBox.latestSnapshot, nil
	}

	return nil, errors.New("no snapshot found")
}

// getSnapshotByID returns the snapshot by ID.
func (c *Base) getSnapshotByID(snapshotID string) (*models.Snapshot, error) {
	c.snapshotBox.snapshotMutex.RLock()
	defer c.snapshotBox.snapshotMutex.RUnlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		return nil, errors.New("no snapshot found")
	}

	return snapshot, nil
}

// IncrementCloneNumber increases clone counter by 1.
func (c *Base) IncrementCloneNumber(snapshotID string) {
	c.snapshotBox.snapshotMutex.Lock()
	defer c.snapshotBox.snapshotMutex.Unlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		log.Err("snapshot not found:", snapshotID)
		return
	}

	snapshot.NumClones++
}

func (c *Base) decrementCloneNumber(snapshotID string) {
	c.snapshotBox.snapshotMutex.Lock()
	defer c.snapshotBox.snapshotMutex.Unlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		log.Err("snapshot not found:", snapshotID)
		return
	}

	if snapshot.NumClones == 0 {
		log.Err("number of clones for snapshot is negative. Snapshot ID:", snapshotID)
		return
	}

	snapshot.NumClones--
}

// GetCloneNumber counts snapshot clones.
func (c *Base) GetCloneNumber(snapshotID string) int {
	c.snapshotBox.snapshotMutex.Lock()
	defer c.snapshotBox.snapshotMutex.Unlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		log.Err("snapshot not found:", snapshotID)
		return 0
	}

	return snapshot.NumClones
}

func (c *Base) getSnapshotList() []models.Snapshot {
	c.snapshotBox.snapshotMutex.RLock()
	defer c.snapshotBox.snapshotMutex.RUnlock()

	snapshots := make([]models.Snapshot, 0, len(c.snapshotBox.items))

	if len(c.snapshotBox.items) == 0 {
		return snapshots
	}

	for _, sn := range c.snapshotBox.items {
		if sn != nil {
			snapshots = append(snapshots, *sn)
		}
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt.Time)
	})

	return snapshots
}

func (c *Base) hasDependentSnapshots(w *CloneWrapper) bool {
	c.snapshotBox.snapshotMutex.RLock()
	defer c.snapshotBox.snapshotMutex.RUnlock()

	poolName := branching.CloneName(w.Clone.Snapshot.Pool, w.Clone.Branch, w.Clone.ID, w.Clone.Revision)

	for name := range c.snapshotBox.items {
		if strings.HasPrefix(name, poolName) {
			return true
		}
	}

	return false
}
