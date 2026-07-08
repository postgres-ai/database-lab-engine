/*
2026 © Postgres.ai
*/

package srv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	srvCfg "gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

const (
	// defaultRetentionInterval is the sweep cadence when checkIntervalMinutes is unset.
	defaultRetentionInterval = 5 * time.Minute
	// defaultMaxDeletionsPerTick bounds deletions per sweep when maxDeletionsPerTick is unset.
	defaultMaxDeletionsPerTick = 50
)

// deletionBudget bounds the number of entities deleted in a single sweep so a first tick
// after downtime cannot storm the webhook channel and telemetry endpoint. Once the budget
// is exhausted the remaining candidates wait for the next tick.
type deletionBudget struct {
	remaining int
	capped    bool
}

// available reports whether the budget still permits a deletion, recording that the cap was
// hit when it does not.
func (b *deletionBudget) available() bool {
	if b.remaining <= 0 {
		b.capped = true
		return false
	}

	return true
}

// consume records that one deletion was performed.
func (b *deletionBudget) consume() {
	b.remaining--
}

// sweep carries the state of a single retention sweep so the per-entity reconcilers do not
// thread the same sweep-wide values (current time, deletion budget, dedup set) through every
// signature. changed is set whenever a ZFS property is written or an entity deleted, so the
// cloning protection cache is reloaded once at the end rather than after every write.
type sweep struct {
	s                *Server
	retention        srvCfg.Retention
	now              time.Time
	budget           deletionBudget
	changed          bool
	deletedSnapshots []string
	deletedBranches  map[string]struct{}
}

// retentionEnabled reports whether any time-based auto-deletion window is configured.
func (s *Server) retentionEnabled() bool {
	r := s.Retention()
	return r.UnusedSnapshotMinutes > 0 || r.UnusedBranchMinutes > 0
}

// retentionInterval returns the configured sweep cadence, falling back to the default.
func (s *Server) retentionInterval() time.Duration {
	interval := s.Retention().CheckIntervalMinutes
	if interval == 0 {
		return defaultRetentionInterval
	}

	return time.Duration(interval) * time.Minute
}

// maxDeletionsPerTick returns the configured per-sweep deletion cap, falling back to the default.
func (s *Server) maxDeletionsPerTick() int {
	maxDeletions := s.Retention().MaxDeletionsPerTick
	if maxDeletions == 0 {
		return defaultMaxDeletionsPerTick
	}

	return int(maxDeletions)
}

// runAutoDeletion runs the background sweeper that schedules and performs safe-only
// auto-deletion of unused branches and snapshots. It returns immediately when no retention
// window is configured, so a disabled deployment spawns no work. The enabled/disabled state
// and the sweep interval are read once at startup, so enabling retention or changing the
// interval on a running server (via config reload) takes effect only after a restart. The
// deletion windows and per-tick cap are re-read each sweep, so those are picked up on reload.
func (s *Server) runAutoDeletion(ctx context.Context) {
	if !s.retentionEnabled() {
		return
	}

	interval := s.retentionInterval()
	log.Msg(fmt.Sprintf("auto-deletion sweeper started with interval %s", interval))

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			s.runRetentionSweep(ctx)
			timer.Reset(interval)

		case <-ctx.Done():
			return
		}
	}
}

// runRetentionSweep performs one sweep across all active pools. Non-active pools are skipped
// (a refreshing or empty pool may be mid-retrieval), and branches deleted on one pool are not
// retried on another within the same tick.
func (s *Server) runRetentionSweep(ctx context.Context) {
	sw := &sweep{
		s:               s,
		retention:       s.Retention(),
		now:             time.Now(),
		budget:          deletionBudget{remaining: s.maxDeletionsPerTick()},
		deletedBranches: make(map[string]struct{}),
	}

	for _, fsm := range s.pm.GetFSManagerList() {
		if ctx.Err() != nil {
			return
		}

		p := fsm.Pool()
		if p == nil || p.Status() != resources.ActivePool {
			continue
		}

		sw.pool(fsm)
	}

	if sw.budget.capped {
		log.Msg(fmt.Sprintf("auto-deletion reached the per-tick cap of %d; remaining entities are deferred to the next tick",
			s.maxDeletionsPerTick()))
	}

	if sw.changed {
		if err := s.Cloning.ReloadSnapshots(); err != nil {
			log.Err("auto-deletion: failed to reload snapshots:", err)
		}
	}

	// emit deletion notifications from a single goroutine per sweep (never one per entity):
	// the telemetry POST is synchronous and the webhook channel is buffer-1, so inline or
	// per-entity-goroutine emission of a bulk sweep would stall or flood.
	if len(sw.deletedSnapshots) > 0 || len(sw.deletedBranches) > 0 {
		go s.emitAutoDeleteEvents(sw.deletedSnapshots, mapKeys(sw.deletedBranches))
	}
}

// emitAutoDeleteEvents sends webhook and telemetry notifications for entities removed by a
// sweep. Webhook sends are non-blocking so a slow consumer cannot stall it; telemetry POSTs are
// synchronous but bounded by the per-tick deletion cap.
func (s *Server) emitAutoDeleteEvents(snapshotIDs, branchNames []string) {
	for _, id := range snapshotIDs {
		s.emitWebhook(webhooks.BasicEvent{EventType: webhooks.SnapshotDeleteEvent, EntityID: id})
		s.tm.SendEvent(context.Background(), telemetry.SnapshotDestroyedEvent, telemetry.SnapshotDestroyed{ID: id})
	}

	for _, name := range branchNames {
		s.emitWebhook(webhooks.BasicEvent{EventType: webhooks.BranchDeleteEvent, EntityID: name})
		s.tm.SendEvent(context.Background(), telemetry.BranchDestroyedEvent, telemetry.BranchDestroyed{Name: name})
	}
}

// emitWebhook sends a webhook event without blocking. The webhook channel is buffer-1, so a
// background sweep deleting many entities must never stall on a slow consumer; a dropped event
// is logged.
func (s *Server) emitWebhook(event webhooks.EventTyper) {
	select {
	case s.webhookCh <- event:
	default:
		log.Dbg(fmt.Sprintf("auto-deletion: webhook channel full, dropped %s event", event.GetType()))
	}
}

// mapKeys returns the keys of a set as a slice.
func mapKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	return keys
}

// pool reconciles one active pool. Snapshots and branches share a single repo read.
func (sw *sweep) pool(fsm pool.FSManager) {
	repo, err := fsm.GetRepo()
	if err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to read repo for pool %s: %v", fsm.Pool().Name, err))
		return
	}

	if sw.retention.UnusedSnapshotMinutes > 0 {
		sw.snapshots(fsm, repo)
	}

	if sw.retention.UnusedBranchMinutes > 0 {
		sw.branches(fsm, repo)
	}
}

// snapshots reconciles the scheduled deletion of every user snapshot in one pool.
func (sw *sweep) snapshots(fsm pool.FSManager, repo *models.Repo) {
	poolName := fsm.Pool().Name
	retention := time.Duration(sw.retention.UnusedSnapshotMinutes) * time.Minute

	// authoritative protection must be read with -s local (ListProtection), not from the repo:
	// the repo read is inheritance-aware and would report a snapshot under a protected branch
	// dataset as protected/scheduled when it has no protection of its own.
	protection, err := fsm.ListProtection()
	if err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to list protection for pool %s: %v", poolName, err))
		return
	}

	branchHeads := branchHeadSet(repo)

	for id, details := range repo.Snapshots {
		if skipAutoDelete(id, poolName) {
			continue
		}

		sw.reconcileSnapshot(fsm, details, protection[id], branchHeads, retention)
	}
}

// reconcileSnapshot decides and applies the scheduled-deletion state for one snapshot.
func (sw *sweep) reconcileSnapshot(fsm pool.FSManager, details models.SnapshotDetails,
	prot thinclones.ProtectionProperties, branchHeads map[string]struct{}, retention time.Duration) {
	protected := models.ProtectedTillActive(prot.ProtectedTill)
	current := parseDeleteAt(prot.DeleteAt)
	hasDependents := !snapshotIsLeaf(details, branchHeads)

	next, shouldDelete := nextDeleteState(sw.now, protected, hasDependents, current, retention)

	if shouldDelete {
		sw.autoDeleteSnapshot(fsm, details.ID)
		return
	}

	sw.reconcileDeleteAt(fsm, details.ID, current, next)
}

// autoDeleteSnapshot deletes an expired, unused snapshot through the shared, protection-aware
// destroy helper. It re-reads protection immediately before the destroy because PATCH and the
// sweeper are not serialized on ZFS-property writes.
func (sw *sweep) autoDeleteSnapshot(fsm pool.FSManager, snapshotID string) {
	if !sw.budget.available() {
		return
	}

	if sw.s.isProtectedNow(fsm, snapshotID) {
		return
	}

	if err := sw.s.destroySnapshotByID(snapshotID, false); err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to delete snapshot %s: %v", snapshotID, err))
		return
	}

	sw.budget.consume()
	sw.changed = true
	sw.deletedSnapshots = append(sw.deletedSnapshots, snapshotID)

	log.Msg(fmt.Sprintf("auto-deletion: deleted unused snapshot %s", snapshotID))
}

// branches reconciles the scheduled deletion of every non-default branch in one pool.
func (sw *sweep) branches(fsm pool.FSManager, repo *models.Repo) {
	retention := time.Duration(sw.retention.UnusedBranchMinutes) * time.Minute

	for branchName, headID := range repo.Branches {
		if branchName == branching.DefaultBranch {
			continue
		}

		sw.reconcileBranch(fsm, repo, branchName, headID, retention)
	}
}

// reconcileBranch decides and applies the scheduled-deletion state for one branch. Branch
// protection and delete-at live on the per-pool branch dataset, not on its snapshots.
func (sw *sweep) reconcileBranch(fsm pool.FSManager, repo *models.Repo, branchName, headID string, retention time.Duration) {
	branchDataset := fsm.Pool().BranchName(fsm.Pool().Name, branchName)

	prot, err := fsm.GetProtection(branchDataset)
	if err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to read protection for branch %s: %v", branchName, err))
		return
	}

	protected := models.ProtectedTillActive(prot.ProtectedTill)
	current := parseDeleteAt(prot.DeleteAt)
	hasDependents := branchHasDependents(repo, branchName, headID)

	next, shouldDelete := nextDeleteState(sw.now, protected, hasDependents, current, retention)

	if shouldDelete {
		sw.autoDeleteBranch(fsm, branchName, branchDataset)
		return
	}

	sw.reconcileDeleteAt(fsm, branchDataset, current, next)
}

// autoDeleteBranch deletes an expired, unused branch through the atomic, protection-aware
// destroy helper, deduplicating across pools within a tick.
func (sw *sweep) autoDeleteBranch(fsm pool.FSManager, branchName, branchDataset string) {
	if _, done := sw.deletedBranches[branchName]; done {
		return
	}

	if !sw.budget.available() {
		return
	}

	if sw.s.isProtectedNow(fsm, branchDataset) {
		return
	}

	if err := sw.s.destroyBranchOnPool(fsm, branchName); err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to delete branch %s: %v", branchName, err))
		return
	}

	sw.budget.consume()
	sw.changed = true
	sw.deletedBranches[branchName] = struct{}{}

	log.Msg(fmt.Sprintf("auto-deletion: deleted unused branch %s", branchName))
}

// reconcileDeleteAt writes the delete_at property only when it differs from the stored value,
// avoiding a redundant ZFS write (and cache reload) on every tick.
func (sw *sweep) reconcileDeleteAt(fsm pool.FSManager, target string, current, next *time.Time) {
	if !deleteAtChanged(current, next) {
		return
	}

	value := ""
	if next != nil {
		value = next.UTC().Format(time.RFC3339)
	}

	if err := fsm.SetDeleteAt(value, target); err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to update delete_at for %s: %v", target, err))
		return
	}

	sw.changed = true
}

// isProtectedNow re-reads the local protection of a target. A read failure is treated as
// protected so a transient error never authorizes a deletion.
func (s *Server) isProtectedNow(fsm pool.FSManager, target string) bool {
	prot, err := fsm.GetProtection(target)
	if err != nil {
		log.Err(fmt.Sprintf("auto-deletion: failed to re-read protection for %s: %v", target, err))
		return true
	}

	return models.ProtectedTillActive(prot.ProtectedTill)
}

// nextDeleteState is the pure sweep decision: given the observed state of an entity it returns
// the delete_at it should carry (nil = none) and whether it should be deleted now. It is fully
// unit-testable without ZFS.
//
//   - protected or in-use -> no schedule (any stale schedule is cleared by the caller)
//   - first seen unused    -> schedule now+retention
//   - schedule reached     -> delete
//   - schedule not reached -> keep the existing schedule
func nextDeleteState(now time.Time, protected, hasDependents bool, currentDeleteAt *time.Time,
	retention time.Duration) (newDeleteAt *time.Time, shouldDelete bool) {
	if protected || hasDependents {
		return nil, false
	}

	if currentDeleteAt == nil {
		scheduled := now.Add(retention)
		return &scheduled, false
	}

	if !now.Before(*currentDeleteAt) {
		return currentDeleteAt, true
	}

	return currentDeleteAt, false
}

// snapshotIsLeaf reports whether a snapshot is a true leaf with no dependents: it has no child
// snapshot, is not a fork root, is not a branch head, has no ZFS clones, and is not in the
// branch-head set. Anything else is load-bearing branch lineage and must never be auto-deleted
// (deleting an intermediate commit breaks dblab log and parent traversal).
func snapshotIsLeaf(details models.SnapshotDetails, branchHeads map[string]struct{}) bool {
	if len(details.Child) > 0 || len(details.Root) > 0 || len(details.Branch) > 0 || len(details.Clones) > 0 {
		return false
	}

	_, isHead := branchHeads[details.ID]

	return !isHead
}

// branchHasDependents reports whether any snapshot belonging to the branch has a dependent:
// a ZFS clone (native clones property) or a child branch forked from it (dle:root). The fork
// check matters because deleting the branch destroys its dataset recursively, which would
// cascade into a child branch cloned off one of its snapshots; that cascade is not caught by
// destroyBranchOnPool's lock, which tracks user clones only, so the clock must never start
// while a fork exists. This governs only whether the deletion clock runs.
func branchHasDependents(repo *models.Repo, branchName, headID string) bool {
	for _, id := range snapshotsToRemove(repo, headID, branchName) {
		details, ok := repo.Snapshots[id]
		if !ok {
			continue
		}

		if len(details.Clones) > 0 || len(details.Root) > 0 {
			return true
		}
	}

	return false
}

// branchHeadSet returns the set of snapshot IDs that are branch heads in the repo. Branch
// heads must never be auto-deleted; this complements the per-snapshot dle:branch check as
// defense-in-depth. The clone-origin (fork-point) snapshots upstream of a head carry dle:root
// and are already excluded by snapshotIsLeaf's root check, so they need not be enumerated.
func branchHeadSet(repo *models.Repo) map[string]struct{} {
	heads := make(map[string]struct{}, len(repo.Branches))

	for _, snapshotID := range repo.Branches {
		heads[snapshotID] = struct{}{}
	}

	return heads
}

// skipAutoDelete reports whether a snapshot is exempt from time-based auto-deletion. Automatic
// pool-level snapshots and physical "_pre" snapshots remain governed by the existing
// count-based retention.
func skipAutoDelete(snapshotID, poolName string) bool {
	dataset, _, found := strings.Cut(snapshotID, "@")
	if !found {
		return true
	}

	if dataset == poolName {
		return true
	}

	return strings.HasSuffix(snapshotID, "_pre")
}

// deleteAtChanged reports whether a delete_at value must be rewritten.
func deleteAtChanged(current, next *time.Time) bool {
	if current == nil && next == nil {
		return false
	}

	if current == nil || next == nil {
		return true
	}

	return !current.Equal(*next)
}

// parseDeleteAt converts a stored delete_at property to a plain time pointer, logging and
// dropping a malformed value rather than failing the sweep.
func parseDeleteAt(value string) *time.Time {
	deleteAt, err := models.ParseDeleteAt(value)
	if err != nil {
		log.Warn(err)
		return nil
	}

	if deleteAt == nil {
		return nil
	}

	return &deleteAt.Time
}
