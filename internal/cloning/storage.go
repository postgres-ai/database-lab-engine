/*
2021 © Postgres.ai
*/

package cloning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const sessionsFilename = "sessions.json"

// RestoreClonesState restores clones data from disk.
func (c *Base) RestoreClonesState() error {
	sessionsPath, err := util.GetMetaPath(sessionsFilename)
	if err != nil {
		return fmt.Errorf("failed to get path of a sessions file: %w", err)
	}

	return c.loadSessionState(sessionsPath)
}

// loadSessionState loads and decodes sessions data.
func (c *Base) loadSessionState(sessionsPath string) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	c.clones = make(map[string]*CloneWrapper)

	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// no sessions data, ignore
			return nil
		}

		return fmt.Errorf("failed to read sessions data: %w", err)
	}

	return json.Unmarshal(data, &c.clones)
}

func (c *Base) filterRunningClones(ctx context.Context) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	snapshotCache := make(map[string]struct{})

	for cloneID, wrapper := range c.clones {
		if wrapper.Clone == nil || wrapper.Session == nil || wrapper.Clone.Status.Code == models.StatusFatal {
			delete(c.clones, cloneID)
			continue
		}

		if _, ok := snapshotCache[wrapper.Clone.Snapshot.ID]; !ok {
			snapshot, err := c.getSnapshotByID(wrapper.Clone.Snapshot.ID)
			if err != nil {
				if freePortErr := c.provision.FreePort(wrapper.Session.Port); freePortErr != nil {
					log.Err(freePortErr)
				}

				delete(c.clones, cloneID)

				continue
			}

			snapshotCache[snapshot.ID] = struct{}{}
		}

		if !c.provision.IsCloneRunning(ctx, util.GetCloneName(wrapper.Session.Port)) {
			delete(c.clones, cloneID)
		}

		c.incrementCloneNumber(wrapper.Clone.Snapshot.ID)
	}
}

// SaveClonesState writes clones state to disk.
func (c *Base) SaveClonesState() {
	log.Msg("Saving state of running clones")

	sessionsPath, err := util.GetMetaPath(sessionsFilename)
	if err != nil {
		log.Err("failed to get path of a sessions file", err)
	}

	if err := c.saveClonesState(sessionsPath); err != nil {
		log.Err("Failed to save the state of running clones", err)
	}

	log.Msg("The state of running clones has been saved")
}

// saveClonesState tries to write clones state to disk and returns an error on failure.
func (c *Base) saveClonesState(sessionsPath string) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	data, err := json.Marshal(c.clones)
	if err != nil {
		return fmt.Errorf("failed to encode session data: %w", err)
	}

	return os.WriteFile(sessionsPath, data, 0600)
}
