/*
2024 © Postgres.ai
*/

package srv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

const autoDeleteCheckInterval = 5 * time.Minute

// RunAutoDeleteCheck starts a background job that periodically checks for expired branches and snapshots.
func (s *Server) RunAutoDeleteCheck(ctx context.Context) {
	timer := time.NewTimer(autoDeleteCheckInterval)

	for {
		select {
		case <-timer.C:
			s.processExpiredBranches(ctx)
			s.processExpiredSnapshots(ctx)
			timer.Reset(autoDeleteCheckInterval)

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *Server) processExpiredBranches(ctx context.Context) {
	expiredBranches := s.Cloning.GetExpiredBranches()
	if len(expiredBranches) == 0 {
		return
	}

	for _, meta := range expiredBranches {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if meta.Protected {
			log.Dbg(fmt.Sprintf("Skipping scheduled deletion of protected branch %q", meta.Name))
			continue
		}

		if meta.Name == branching.DefaultBranch {
			log.Dbg(fmt.Sprintf("Skipping scheduled deletion of default branch %q", meta.Name))
			continue
		}

		fsm, err := s.getFSManagerForBranch(meta.Name)
		if err != nil {
			log.Errf("failed to get FSManager for branch %q: %v", meta.Name, err)
			continue
		}

		if fsm == nil {
			log.Errf("no pool manager found for branch %q", meta.Name)
			continue
		}

		canDelete, reason := s.canDeleteBranch(fsm, meta)
		if !canDelete {
			switch meta.AutoDeleteMode {
			case models.AutoDeleteSoft:
				log.Dbg(fmt.Sprintf("Skipping scheduled deletion of branch %q: %s (soft mode)", meta.Name, reason))
				continue
			case models.AutoDeleteForce:
				log.Msg(fmt.Sprintf("Force deleting branch %q: %s", meta.Name, reason))
				if err := s.forceDeleteBranch(ctx, fsm, meta.Name); err != nil {
					log.Errf("failed to force delete branch %q: %v", meta.Name, err)
					continue
				}

				s.Cloning.DeleteBranchMeta(meta.Name)

				continue
			}
		}

		log.Msg(fmt.Sprintf("Scheduled branch %q is going to be removed (deleteAt: %s)", meta.Name, meta.DeleteAt.Time.Format(time.RFC3339)))

		if err := s.destroyBranchDataset(fsm, meta.Name); err != nil {
			log.Errf("failed to destroy scheduled branch %q: %v", meta.Name, err)
			continue
		}

		s.Cloning.DeleteBranchMeta(meta.Name)
	}
}

func (s *Server) canDeleteBranch(fsm pool.FSManager, meta *cloning.BranchMeta) (bool, string) {
	repo, err := fsm.GetRepo()
	if err != nil {
		return false, fmt.Sprintf("failed to get repo: %v", err)
	}

	snapshotID, ok := repo.Branches[meta.Name]
	if !ok {
		return false, "branch not found"
	}

	toRemove := snapshotsToRemove(repo, snapshotID, meta.Name)

	for _, snapID := range toRemove {
		if cloneNum := s.Cloning.GetCloneNumber(snapID); cloneNum > 0 {
			return false, fmt.Sprintf("snapshot %q has %d dependent clone(s)", snapID, cloneNum)
		}
	}

	return true, ""
}

func (s *Server) forceDeleteBranch(ctx context.Context, fsm pool.FSManager, branchName string) error {
	repo, err := fsm.GetRepo()
	if err != nil {
		return fmt.Errorf("failed to get repo: %w", err)
	}

	snapshotID, ok := repo.Branches[branchName]
	if !ok {
		return fmt.Errorf("branch not found: %s", branchName)
	}

	toRemove := snapshotsToRemove(repo, snapshotID, branchName)

	for _, snapID := range toRemove {
		clones := s.getClonesForSnapshot(snapID)
		for _, cloneID := range clones {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := s.Cloning.DestroyCloneSync(cloneID); err != nil {
				log.Errf("failed to destroy clone %q during branch force delete: %v", cloneID, err)
			}
		}
	}

	return s.destroyBranchDataset(fsm, branchName)
}

func (s *Server) getClonesForSnapshot(snapshotID string) []string {
	state := s.Cloning.GetCloningState()
	var cloneIDs []string

	for _, clone := range state.Clones {
		if clone.Snapshot != nil && clone.Snapshot.ID == snapshotID {
			cloneIDs = append(cloneIDs, clone.ID)
		}
	}

	return cloneIDs
}

func (s *Server) processExpiredSnapshots(ctx context.Context) {
	expiredSnapshots := s.Cloning.GetExpiredSnapshots()
	if len(expiredSnapshots) == 0 {
		return
	}

	for _, meta := range expiredSnapshots {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if meta.Protected {
			log.Dbg(fmt.Sprintf("Skipping scheduled deletion of protected snapshot %q", meta.ID))
			continue
		}

		poolName, err := s.detectPoolName(meta.ID)
		if err != nil {
			log.Errf("failed to detect pool for snapshot %q: %v", meta.ID, err)
			continue
		}

		if poolName == "" {
			log.Errf("pool for snapshot %q not found", meta.ID)
			continue
		}

		fsm, err := s.pm.GetFSManager(poolName)
		if err != nil {
			log.Errf("failed to get FSManager for pool %q: %v", poolName, err)
			continue
		}

		canDelete, reason, cloneIDs := s.canDeleteSnapshot(fsm, meta, poolName)
		if !canDelete {
			switch meta.AutoDeleteMode {
			case models.AutoDeleteSoft:
				log.Dbg(fmt.Sprintf("Skipping scheduled deletion of snapshot %q: %s (soft mode)", meta.ID, reason))
				continue
			case models.AutoDeleteForce:
				log.Msg(fmt.Sprintf("Force deleting snapshot %q: %s", meta.ID, reason))
				if err := s.forceDeleteSnapshot(ctx, fsm, meta.ID, cloneIDs); err != nil {
					log.Errf("failed to force delete snapshot %q: %v", meta.ID, err)
					continue
				}

				s.Cloning.GetEntityStorage().DeleteSnapshotMeta(meta.ID)

				continue
			}
		}

		log.Msg(fmt.Sprintf("Scheduled snapshot %q is going to be removed (deleteAt: %s)", meta.ID, meta.DeleteAt.Time.Format(time.RFC3339)))

		if err := fsm.DestroySnapshot(meta.ID); err != nil {
			log.Errf("failed to destroy scheduled snapshot %q: %v", meta.ID, err)
			continue
		}

		s.Cloning.GetEntityStorage().DeleteSnapshotMeta(meta.ID)

		fsm.RefreshSnapshotList()

		if err := s.Cloning.ReloadSnapshots(); err != nil {
			log.Dbg("Failed to reload snapshots:", err.Error())
		}

		s.webhookCh <- webhooks.BasicEvent{
			EventType: webhooks.SnapshotDeleteEvent,
			EntityID:  meta.ID,
		}
	}
}

func (s *Server) canDeleteSnapshot(fsm pool.FSManager, meta *cloning.SnapshotMeta, poolName string) (bool, string, []string) {
	dependentCloneDatasets, err := fsm.HasDependentEntity(meta.ID)
	if err != nil {
		return false, fmt.Sprintf("failed to check dependencies: %v", err), nil
	}

	var cloneIDs []string
	var protectedClones []string

	for _, cloneDataset := range dependentCloneDatasets {
		cloneID, ok := branching.ParseCloneName(cloneDataset, poolName)
		if !ok {
			continue
		}

		clone, err := s.Cloning.GetClone(cloneID)
		if err != nil {
			continue
		}

		cloneIDs = append(cloneIDs, clone.ID)

		if clone.Protected {
			protectedClones = append(protectedClones, clone.ID)
		}
	}

	if len(protectedClones) > 0 {
		return false, fmt.Sprintf("has protected clones: %s", strings.Join(protectedClones, ",")), cloneIDs
	}

	if len(cloneIDs) > 0 {
		return false, fmt.Sprintf("has dependent clones: %s", strings.Join(cloneIDs, ",")), cloneIDs
	}

	return true, "", nil
}

func (s *Server) forceDeleteSnapshot(ctx context.Context, fsm pool.FSManager, snapshotID string, cloneIDs []string) error {
	for _, cloneID := range cloneIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.Cloning.DestroyCloneSync(cloneID); err != nil {
			log.Errf("failed to destroy clone %q during snapshot force delete: %v", cloneID, err)
		}
	}

	if err := fsm.DestroySnapshot(snapshotID); err != nil {
		return fmt.Errorf("failed to destroy snapshot: %w", err)
	}

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		log.Dbg("Failed to reload snapshots:", err.Error())
	}

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.SnapshotDeleteEvent,
		EntityID:  snapshotID,
	}

	return nil
}
