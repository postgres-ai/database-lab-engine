/*
2021 © Postgres.ai
*/

package cloning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	sessionsFilename = "sessions.json"
	sessionsFileMode = 0600
)

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

	if err := json.Unmarshal(data, &c.clones); err != nil {
		return fmt.Errorf("failed to decode sessions data: %w", err)
	}

	return nil
}

func (c *Base) restartCloneContainers(ctx context.Context) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	for _, wrapper := range c.clones {
		if wrapper.Clone == nil || wrapper.Session == nil {
			continue
		}

		cloneName := wrapper.Clone.ID
		if c.provision.IsCloneRunning(ctx, cloneName) {
			continue
		}

		if err := c.provision.ReconnectClone(ctx, cloneName); err != nil {
			log.Err(fmt.Sprintf("clone container %s cannot be reconnected to internal network: %s", cloneName, err))
			continue
		}

		if err := c.provision.StartCloneContainer(ctx, cloneName); err != nil {
			log.Err(fmt.Sprintf("clone container %s cannot start: %s", cloneName, err))
			continue
		}

		log.Dbg(fmt.Sprintf("Clone container %s is running", cloneName))
	}
}

func (c *Base) filterRunningClones(ctx context.Context) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	snapshotCache := make(map[string]struct{})

	for cloneID, wrapper := range c.clones {
		if wrapper.Clone == nil || wrapper.Clone.Snapshot == nil || wrapper.Session == nil ||
			wrapper.Clone.Status.Code == models.StatusFatal {
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

		// A clone whose upgrade has not settled keeps its place in the registry even without a
		// container. RecoverInterruptedUpgrades runs just before this and normally brings such a
		// clone back up, but it can fail - an unreachable registry when the target image is not
		// local is enough - and dropping the clone here would put it outside keepClones, so
		// cleanupInvalidClones would destroy a dataset holding a finished or still-recoverable
		// upgrade. The state on disk outlives any number of failed recovery attempts; the clone
		// must outlive them too.
		//
		// Only the removal is skipped, never the clone-count bookkeeping below: the origin
		// snapshot is what a reprovision falls back on, so it has to keep counting this clone.
		pendingUpgrade := c.provision.HasPendingUpgrade(wrapper.Session, wrapper.Clone)

		if !pendingUpgrade && !c.provision.IsCloneRunning(ctx, wrapper.Clone.ID) {
			delete(c.clones, cloneID)
		}

		c.IncrementCloneNumber(wrapper.Clone.Snapshot.ID)
	}
}

// SaveClonesState writes clones state to disk.
func (c *Base) SaveClonesState() {
	sessionsPath, err := util.GetMetaPath(sessionsFilename)
	if err != nil {
		log.Err("failed to get path of sessions file", err)
	}

	if err := c.saveClonesState(sessionsPath); err != nil {
		log.Err("failed to save state of running clones", err)
	}
}

// saveClonesState tries to write clones state to disk and returns an error on failure.
func (c *Base) saveClonesState(sessionsPath string) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	data, err := json.Marshal(c.clones)
	if err != nil {
		return fmt.Errorf("failed to encode session data: %w", err)
	}

	return writeFileAtomic(sessionsPath, data, sessionsFileMode)
}

// writeFileAtomic writes data to a temporary file in the target directory, syncs it, and renames it
// over the target path, so a crash at any point leaves either the old or the new file, never a
// torn one. The directory is synced afterwards to make the rename durable.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpPath := tmpFile.Name()

	if err := writeAndSync(tmpFile, data, perm); err != nil {
		_ = os.Remove(tmpPath)

		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return syncDir(dir)
}

func writeAndSync(f *os.File, data []byte, perm os.FileMode) error {
	if _, err := f.Write(data); err != nil {
		_ = f.Close()

		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := f.Chmod(perm); err != nil {
		_ = f.Close()

		return fmt.Errorf("failed to set temporary file permissions: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()

		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	return nil
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open directory %s: %w", dir, err)
	}

	defer func() { _ = d.Close() }()

	if err := d.Sync(); err != nil {
		return fmt.Errorf("failed to sync directory %s: %w", dir, err)
	}

	return nil
}
