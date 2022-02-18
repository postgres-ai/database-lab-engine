/*
2021 © Postgres.ai
*/

package cloning

import (
	"sort"
	"sync"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
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

	for _, entry := range entries {
		numClones := 0

		for cloneName := range c.clones {
			if c.clones[cloneName] != nil && c.clones[cloneName].Clone.Snapshot != nil &&
				c.clones[cloneName].Clone.Snapshot.ID == entry.ID {
				numClones++
			}
		}

		currentSnapshot := &models.Snapshot{
			ID:           entry.ID,
			CreatedAt:    util.FormatTime(entry.CreatedAt),
			DataStateAt:  util.FormatTime(entry.DataStateAt),
			PhysicalSize: entry.Used,
			LogicalSize:  entry.LogicalReferenced,
			Pool:         entry.Pool,
			NumClones:    numClones,
		}

		snapshots[entry.ID] = currentSnapshot
		latestSnapshot = defineLatestSnapshot(latestSnapshot, currentSnapshot)

		log.Dbg("snapshot:", *currentSnapshot)
	}

	c.resetSnapshots(snapshots, latestSnapshot)

	return nil
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
	if latest == nil || latest.DataStateAt == "" || latest.DataStateAt < challenger.DataStateAt {
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

func (c *Base) incrementCloneNumber(snapshotID string) {
	c.snapshotBox.snapshotMutex.Lock()
	defer c.snapshotBox.snapshotMutex.Unlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		log.Err("Snapshot not found:", snapshotID)
		return
	}

	snapshot.NumClones++
}

func (c *Base) decrementCloneNumber(snapshotID string) {
	c.snapshotBox.snapshotMutex.Lock()
	defer c.snapshotBox.snapshotMutex.Unlock()

	snapshot, ok := c.snapshotBox.items[snapshotID]
	if !ok {
		log.Err("Snapshot not found:", snapshotID)
		return
	}

	if snapshot.NumClones == 0 {
		log.Err("The number of clones for the snapshot is negative. Snapshot ID:", snapshotID)
		return
	}

	snapshot.NumClones--
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
		return snapshots[i].CreatedAt > snapshots[j].CreatedAt
	})

	return snapshots
}
