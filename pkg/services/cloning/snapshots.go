/*
2021 © Postgres.ai
*/

package cloning

import (
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

func (c *baseCloning) incrementCloneNumber(snapshotID string) {
	for i := range c.snapshots {
		if c.snapshots[i].ID == snapshotID {
			c.snapshotMutex.Lock()
			c.snapshots[i].NumClones++
			c.snapshotMutex.Unlock()

			break
		}
	}
}

func (c *baseCloning) decrementCloneNumber(snapshotID string) {
	for i := range c.snapshots {
		if c.snapshots[i].ID == snapshotID {
			c.snapshotMutex.Lock()
			if c.snapshots[i].NumClones == 0 {
				log.Err("The number of clones for the snapshot is negative. Snapshot ID:", snapshotID)
			}
			c.snapshots[i].NumClones--
			c.snapshotMutex.Unlock()

			break
		}
	}
}
