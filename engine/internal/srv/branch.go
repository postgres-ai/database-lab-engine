package srv

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

var (
	branchNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]*$`)
	// snapshotIDRegexp matches a legal ZFS snapshot id (dataset@snapshot). Both parts allow only
	// ZFS-legal characters (alphanumerics and _ . : / -), which excludes every shell metacharacter.
	// Snapshot ids reach the runner interpolated into shell commands, so this gate blocks command
	// injection through the id before it touches any runner.
	snapshotIDRegexp = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.:/-]*@[a-zA-Z0-9_][a-zA-Z0-9_.:-]*$`)
)

// listBranches returns branch list.
func (s *Server) listBranches(w http.ResponseWriter, r *http.Request) {
	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	branches, err := s.getAllAvailableBranches(fsm)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	repo, err := fsm.GetAllRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	branchDetails := make([]models.BranchView, 0, len(branches))

	for _, branchEntity := range branches {
		snapshotDetails, ok := repo.Snapshots[branchEntity.SnapshotID]
		if !ok {
			continue
		}

		numSnapshots, parentSnapshot := findBranchParent(repo.Snapshots, snapshotDetails.ID, branchEntity.Name)

		branchView := models.BranchView{
			Name:         branchEntity.Name,
			BaseDataset:  branchEntity.Dataset,
			Parent:       parentSnapshot,
			DataStateAt:  snapshotDetails.DataStateAt,
			SnapshotID:   snapshotDetails.ID,
			Dataset:      snapshotDetails.Dataset,
			NumSnapshots: numSnapshots,
		}

		if bfsm, err := s.getFSManagerForSnapshot(branchEntity.SnapshotID); err == nil {
			branchDataset := bfsm.Pool().BranchName(bfsm.Pool().Name, branchEntity.Name)
			branchView.Protected, branchView.ProtectedTill, branchView.DeleteAt = readProtection(bfsm, branchDataset)
		}

		branchDetails = append(branchDetails, branchView)
	}

	if err := api.WriteJSON(w, http.StatusOK, branchDetails); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getAllAvailableBranches(fsm pool.FSManager) ([]models.BranchEntity, error) {
	if fsm == nil {
		return nil, fmt.Errorf("no available pools")
	}

	// Filter by available pools in case if two or more DLE is running on the same pool and use the selectedPool feature.
	poolNames := []string{}

	for _, fsManager := range s.pm.GetFSManagerList() {
		poolNames = append(poolNames, fsManager.Pool().Name)
	}

	return fsm.ListAllBranches(poolNames)
}

func findBranchParent(snapshots map[string]models.SnapshotDetails, parentID, branch string) (int, string) {
	snapshotCounter := 0

	for i := len(snapshots); i > 0; i-- {
		snapshotPointer := snapshots[parentID]
		snapshotCounter++

		if containsString(snapshotPointer.Root, branch) {
			if len(snapshotPointer.Branch) > 0 {
				return snapshotCounter, snapshotPointer.Branch[0]
			}

			break
		}

		if snapshotPointer.Parent == "-" {
			break
		}

		parentID = snapshotPointer.Parent
	}

	return snapshotCounter, "-"
}

func containsString(slice []string, s string) bool {
	for _, str := range slice {
		if str == s {
			return true
		}
	}

	return false
}

func (s *Server) getFSManagerForBranch(branchName string) (pool.FSManager, error) {
	return s.getFSManagerForBranchAndDataset(branchName, "")
}

func (s *Server) getFSManagerForBranchAndDataset(branchName, dataset string) (pool.FSManager, error) {
	allBranches, err := s.getAllAvailableBranches(s.pm.First())
	if err != nil {
		return nil, fmt.Errorf("failed to get branch list: %w", err)
	}

	for _, branchEntity := range allBranches {
		if branchEntity.Name != branchName {
			continue
		}

		if dataset == "" {
			return s.getFSManagerForSnapshot(branchEntity.SnapshotID)
		}

		fsm, err := s.getFSManagerForSnapshot(branchEntity.SnapshotID)
		if err != nil {
			continue
		}

		if fsm.Pool().Name == dataset {
			return fsm, nil
		}
	}

	if dataset != "" {
		return nil, fmt.Errorf("failed to find dataset %s of the branch: %s", dataset, branchName)
	}

	return nil, fmt.Errorf("failed to found dataset of the branch: %s", branchName)
}

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request) {
	var createRequest types.BranchCreateRequest
	if err := api.ReadJSON(r, &createRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if createRequest.BranchName == "" {
		api.SendBadRequestError(w, r, "The branch name must not be empty")
		return
	}

	if createRequest.BranchName == createRequest.BaseBranch {
		api.SendBadRequestError(w, r, "new and base branches must have different names")
		return
	}

	if !isValidBranchName(createRequest.BranchName) {
		api.SendBadRequestError(w, r, "The branch name must start with a letter, number, or underscore, "+
			"and contain only letters, numbers, underscores, and hyphens. Spaces and slashes are not allowed")
		return
	}

	var err error

	fsm := s.pm.First()

	if createRequest.BaseBranch != "" {
		fsm, err = s.getFSManagerForBranch(createRequest.BaseBranch)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}
	}

	snapshotID := createRequest.SnapshotID

	if snapshotID != "" {
		fsm, err = s.getFSManagerForSnapshot(snapshotID)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}
	}

	if fsm == nil {
		api.SendBadRequestError(w, r, "no pool manager found")
		return
	}

	branches, err := fsm.ListBranches()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if _, ok := branches[createRequest.BranchName]; ok {
		api.SendBadRequestError(w, r, fmt.Sprintf("branch '%s' already exists", createRequest.BranchName))
		return
	}

	if snapshotID == "" {
		if createRequest.BaseBranch == "" {
			api.SendBadRequestError(w, r, "either base branch name or base snapshot ID must be specified")
			return
		}

		branchPointer, ok := branches[createRequest.BaseBranch]
		if !ok {
			api.SendBadRequestError(w, r, "base branch not found")
			return
		}

		snapshotID = branchPointer
	}

	poolName, err := s.detectPoolName(snapshotID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	brName := fsm.Pool().BranchName(poolName, createRequest.BranchName)
	dataStateAt := time.Now().Format(util.DataStateAtFormat)

	if err := fsm.CreateBranch(brName, snapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	branchSnapshot := fmt.Sprintf("%s@%s", brName, dataStateAt)

	if err := fsm.Snapshot(branchSnapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.AddBranchProp(createRequest.BranchName, branchSnapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetRoot(createRequest.BranchName, snapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetRelation(snapshotID, branchSnapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, branchSnapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm.RefreshSnapshotList()

	branch := models.Branch{Name: createRequest.BranchName}

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.BranchCreateEvent,
		EntityID:  branch.Name,
	}

	s.tm.SendEvent(context.Background(), telemetry.BranchCreatedEvent, telemetry.BranchCreated{
		Name: branch.Name,
	})

	if err := api.WriteJSON(w, http.StatusOK, branch); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func isValidBranchName(branchName string) bool {
	return branchNameRegexp.MatchString(branchName)
}

// isValidSnapshotID reports whether the id is a syntactically valid, injection-safe ZFS snapshot id.
func isValidSnapshotID(snapshotID string) bool {
	return snapshotIDRegexp.MatchString(snapshotID)
}

func (s *Server) getSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := mux.Vars(r)["id"]

	if snapshotID == "" {
		api.SendBadRequestError(w, r, "snapshotID must not be empty")
		return
	}

	snapshot, err := s.Cloning.GetSnapshotByID(snapshotID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, snapshot); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getCommit(w http.ResponseWriter, r *http.Request) {
	snapshotID := mux.Vars(r)["id"]

	if snapshotID == "" {
		api.SendBadRequestError(w, r, "snapshotID must not be empty")
		return
	}

	fsm, err := s.getFSManagerForSnapshot(snapshotID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	repo, err := fsm.GetRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshotPointer, ok := repo.Snapshots[snapshotID]

	if !ok {
		api.SendNotFoundError(w, r)
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, snapshotPointer); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getFSManagerForSnapshot(snapshotID string) (pool.FSManager, error) {
	poolName, err := s.detectPoolName(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to detect pool name for the snapshot %s: %w", snapshotID, err)
	}

	fsm, err := s.pm.GetFSManager(poolName)
	if err != nil {
		return nil, fmt.Errorf("pool manager not available %s: %w", poolName, err)
	}

	return fsm, nil
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	var snapshotRequest types.SnapshotCloneCreateRequest
	if err := api.ReadJSON(r, &snapshotRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	clone, err := s.Cloning.GetClone(snapshotRequest.CloneID)
	if err != nil {
		api.SendBadRequestError(w, r, "clone not found")
		return
	}

	if clone.Branch == "" {
		api.SendBadRequestError(w, r, "clone was not created on branch")
		return
	}

	fsm, err := s.pm.GetFSManager(clone.Snapshot.Pool)

	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("pool %q not found", clone.Snapshot.Pool))
		return
	}

	branches, err := fsm.ListBranches()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	currentSnapshotID, ok := branches[clone.Branch]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+clone.Branch)
		return
	}

	log.Dbg("Current snapshot ID", currentSnapshotID)

	dataStateAt := time.Now().Format(util.DataStateAtFormat)
	snapshotBase := fsm.Pool().CloneName(clone.Branch, clone.ID, clone.Revision)
	snapshotName := fmt.Sprintf("%s@%s", snapshotBase, dataStateAt)

	if err := fsm.Snapshot(snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.AddBranchProp(clone.Branch, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.DeleteBranchProp(clone.Branch, currentSnapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetRelation(currentSnapshotID, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetMessage(snapshotRequest.Message, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.SnapshotCreatedEvent, telemetry.SnapshotCreated{})

	if err := api.WriteJSON(w, http.StatusOK, types.SnapshotResponse{SnapshotID: snapshotName}); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func filterSnapshotsByBranch(pool *resources.Pool, branch string, snapshots []models.Snapshot) []models.Snapshot {
	filtered := make([]models.Snapshot, 0)

	branchName := pool.BranchName(pool.Name, branch)

	for _, sn := range snapshots {
		dataset, _, found := strings.Cut(sn.ID, "@")
		if !found {
			continue
		}

		if strings.HasPrefix(dataset, branchName) || (branch == branching.DefaultBranch && pool.Name == dataset) {
			filtered = append(filtered, sn)
		}
	}

	return filtered
}

func filterSnapshotsByDataset(dataset string, snapshots []models.Snapshot) []models.Snapshot {
	filtered := make([]models.Snapshot, 0)

	for _, sn := range snapshots {
		if sn.Pool == dataset {
			filtered = append(filtered, sn)
		}
	}

	return filtered
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	branchName := mux.Vars(r)["branchName"]

	if !isValidBranchName(branchName) {
		api.SendBadRequestError(w, r, "invalid branch name")
		return
	}

	fsm, err := s.getFSManagerForBranch(branchName)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if fsm == nil {
		api.SendBadRequestError(w, r, "no pool manager found")
		return
	}

	repo, err := fsm.GetRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshotID, ok := repo.Branches[branchName]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+branchName)
		return
	}

	snapshotPointer := repo.Snapshots[snapshotID]

	logList := []models.SnapshotDetails{snapshotPointer}

	// Limit the number of iterations to the number of snapshots.
	for i := len(repo.Snapshots); i > 1; i-- {
		if snapshotPointer.Parent == "-" || snapshotPointer.Parent == "" {
			break
		}

		snapshotPointer = repo.Snapshots[snapshotPointer.Parent]
		logList = append(logList, snapshotPointer)
	}

	if err := api.WriteJSON(w, http.StatusOK, logList); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) deleteBranch(w http.ResponseWriter, r *http.Request) {
	branchName := mux.Vars(r)["branchName"]

	if !isValidBranchName(branchName) {
		api.SendBadRequestError(w, r, "invalid branch name")
		return
	}

	if err := s.destroyBranchByName(branchName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, models.Response{
		Status:  models.ResponseOK,
		Message: "Deleted branch",
	}); err != nil {
		api.SendError(w, r, err)
		return
	}

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.BranchDeleteEvent,
		EntityID:  branchName,
	}

	s.tm.SendEvent(context.Background(), telemetry.BranchDestroyedEvent, telemetry.BranchDestroyed{
		Name: branchName,
	})

	log.Dbg(fmt.Sprintf("Branch %s has been deleted", branchName))
}

// destroyBranchByName resolves the branch's filesystem manager from its head snapshot and
// deletes the branch. It is the entry point for the HTTP delete handler.
func (s *Server) destroyBranchByName(branchName string) error {
	fsm, err := s.getFSManagerForBranch(branchName)
	if err != nil {
		return err
	}

	if fsm == nil {
		return errors.New("no pool manager found")
	}

	return s.destroyBranchOnPool(fsm, branchName)
}

// destroyBranchOnPool removes a branch from the dataset of the given pool's filesystem manager.
// The caller supplies the fsm so the auto-deletion sweeper destroys the same pool's dataset it
// evaluated for protection, instead of re-resolving an arbitrary pool via s.pm.First(); the HTTP
// delete handler passes the fsm resolved from the branch's head snapshot. It deletes a branch
// atomically with respect to clone creation: a protected branch is refused, a branch whose
// snapshots are the fork point of another branch is refused (the recursive destroy would cascade
// into that sibling branch dataset), and the recursive destroy (DestroyBranchDataset) runs under
// the clone-deletion lock because the branch path has no ZFS TOCTOU backstop of its own; the lock
// guarantees no registered clone depends on the branch's snapshots before the sweep.
func (s *Server) destroyBranchOnPool(fsm pool.FSManager, branchName string) error {
	if branchName == branching.DefaultBranch {
		return fmt.Errorf("cannot delete default branch: %s", branching.DefaultBranch)
	}

	repo, err := fsm.GetRepo()
	if err != nil {
		return err
	}

	snapshotID, ok := repo.Branches[branchName]
	if !ok {
		return errors.New("branch not found: " + branchName)
	}

	branchDataset := fsm.Pool().BranchName(fsm.Pool().Name, branchName)

	if err := ensureNotProtected(fsm, branchDataset, "branch", branchName); err != nil {
		return err
	}

	toRemove := snapshotsToRemove(repo, snapshotID, branchName)
	log.Dbg("Snapshots to remove", toRemove)

	if snapshotID, children := childForkInBranch(repo, toRemove); snapshotID != "" {
		return fmt.Errorf("cannot delete branch %q: snapshot %s is the fork point of branch(es) %s; delete them first",
			branchName, snapshotID, strings.Join(children, ", "))
	}

	destroyErr := s.Cloning.WithBranchDeletionLock(toRemove, func() error {
		return fsm.DestroyBranchDataset(branchDataset)
	})
	if destroyErr != nil {
		return destroyErr
	}

	return s.cleanupAfterBranchDeletion(fsm, branchName)
}

// cleanupAfterBranchDeletion clears branch properties left on remaining snapshots and
// refreshes the snapshot list after the branch dataset has been removed.
func (s *Server) cleanupAfterBranchDeletion(fsm pool.FSManager, branchName string) error {
	repo, err := fsm.GetRepo()
	if err != nil {
		return err
	}

	if err := cleanupSnapshotProperties(repo, fsm, branchName); err != nil {
		return err
	}

	fsm.RefreshSnapshotList()

	return nil
}

// branchDatasetRef binds a pool's filesystem manager to that pool's branch dataset.
type branchDatasetRef struct {
	fsm     pool.FSManager
	dataset string
}

// branchDatasets returns the <pool>/branch/<name> dataset on every pool that has the branch.
// Protection is anchored on the per-pool branch dataset, so a write must fan out to all of
// them rather than only the s.pm.First()-resolved pool.
func (s *Server) branchDatasets(branchName string) []branchDatasetRef {
	refs := make([]branchDatasetRef, 0)

	for _, fsm := range s.pm.GetFSManagerList() {
		branches, err := fsm.ListBranches()
		if err != nil {
			log.Dbg(fmt.Sprintf("failed to list branches for pool %s: %v", fsm.Pool().Name, err))
			continue
		}

		if _, ok := branches[branchName]; !ok {
			continue
		}

		refs = append(refs, branchDatasetRef{
			fsm:     fsm,
			dataset: fsm.Pool().BranchName(fsm.Pool().Name, branchName),
		})
	}

	return refs
}

// readProtection reads protection of a snapshot or branch-dataset target into model fields.
// A read failure is logged and reported as unprotected so it cannot block listing/display.
func readProtection(fsm pool.FSManager, target string) (bool, *models.LocalTime, *models.LocalTime) {
	protection, err := fsm.GetProtection(target)
	if err != nil {
		log.Dbg(fmt.Sprintf("failed to read protection for %s: %v", target, err))
		return false, nil, nil
	}

	protected, till, err := models.ParseProtectedTill(protection.ProtectedTill)
	if err != nil {
		log.Warn(err)
	}

	deleteAt, err := models.ParseDeleteAt(protection.DeleteAt)
	if err != nil {
		log.Warn(err)
	}

	return protected, till, deleteAt
}

// branchProtectionWrite applies write to every pool's branch dataset, attempting all pools even
// when one fails, and returns the combined per-pool error. Not stopping at the first failure
// avoids leaving the branch protected on the pools written before the error and unprotected on
// the rest, so a retry can converge the remaining pools.
func branchProtectionWrite(datasets []branchDatasetRef, write func(branchDatasetRef) error) error {
	errs := make([]error, 0, len(datasets))

	for _, d := range datasets {
		if err := write(d); err != nil {
			errs = append(errs, fmt.Errorf("pool %s: %w", d.fsm.Pool().Name, err))
		}
	}

	return errors.Join(errs...)
}

// readBranchProtection reports a branch as protected when any pool's dataset is protected
// (fail-safe), so an inconsistency across pools is resolved towards protected rather than
// deletable. It returns the times of the first protected pool, falling back to the first pool's
// state when none is protected, and reports unprotected for an empty dataset set.
func readBranchProtection(datasets []branchDatasetRef) (bool, *models.LocalTime, *models.LocalTime) {
	if len(datasets) == 0 {
		return false, nil, nil
	}

	for _, d := range datasets {
		if protected, till, deleteAt := readProtection(d.fsm, d.dataset); protected {
			return true, till, deleteAt
		}
	}

	return readProtection(datasets[0].fsm, datasets[0].dataset)
}

// patchBranch updates a branch's deletion protection or scheduled deletion. The write fans out
// to the branch dataset on every pool, attempting all pools even when one errors so a mid-fan-out
// failure does not leave the branch protected on some pools and unprotected on others; the
// combined per-pool error is returned. The response is read fail-safe across pools (protected if
// any pool's dataset is protected). Single-pool deployments are unaffected.
func (s *Server) patchBranch(w http.ResponseWriter, r *http.Request) {
	branchName := mux.Vars(r)["branchName"]
	if !isValidBranchName(branchName) {
		api.SendBadRequestError(w, r, "invalid branch name")
		return
	}

	var req types.BranchUpdateRequest
	if err := api.ReadJSON(r, &req); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	datasets := s.branchDatasets(branchName)
	if len(datasets) == 0 {
		api.SendBadRequestError(w, r, "branch not found: "+branchName)
		return
	}

	setTill := func(v string) error {
		return branchProtectionWrite(datasets, func(d branchDatasetRef) error {
			return d.fsm.SetProtectedTill(v, d.dataset)
		})
	}

	setDeleteAt := func(v string) error {
		return branchProtectionWrite(datasets, func(d branchDatasetRef) error {
			return d.fsm.SetDeleteAt(v, d.dataset)
		})
	}

	if err := applyProtectionUpdate(s.Retention().ProtectionMaxDurationMinutes,
		req.Protected, req.ProtectionDurationMinutes, req.DeleteAt, setTill, setDeleteAt); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	protected, till, deleteAt := readBranchProtection(datasets)
	view := models.BranchView{Name: branchName, Protected: protected, ProtectedTill: till, DeleteAt: deleteAt}

	s.tm.SendEvent(context.Background(), telemetry.BranchUpdatedEvent, telemetry.BranchUpdated{
		Name:      branchName,
		Protected: protected,
	})

	if err := api.WriteJSON(w, http.StatusOK, view); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func cleanupSnapshotProperties(repo *models.Repo, fsm pool.FSManager, branchName string) error {
	for _, snap := range repo.Snapshots {
		for _, rootBranch := range snap.Root {
			if rootBranch == branchName {
				if err := fsm.DeleteRootProp(branchName, snap.ID); err != nil {
					return err
				}

				if err := fsm.DeleteBranchProp(branchName, snap.ID); err != nil {
					return err
				}

				for _, child := range snap.Child {
					if _, ok := repo.Snapshots[child]; !ok {
						if err := fsm.DeleteChildProp(child, snap.ID); err != nil {
							return err
						}
					}
				}

				break
			}
		}
	}

	return nil
}

// childForkInBranch returns the first of the branch's own snapshots that is the fork point
// (dle:root) of another branch, together with the forked branch names. Branches are sibling
// datasets cloned from such a snapshot (<pool>/branch/<child>), so the recursive destroy of the
// branch dataset would cascade-destroy them; the clone-deletion lock guards only user clones,
// not branch forks, so deletion must be refused while a fork exists. The branch's own fork point
// is excluded from snapshotsToRemove, and residual committed-snapshot datasets that belong to the
// branch carry no dle:root, so neither falsely blocks deletion.
func childForkInBranch(repo *models.Repo, snapshotIDs []string) (string, []string) {
	for _, id := range snapshotIDs {
		if details, ok := repo.Snapshots[id]; ok && len(details.Root) > 0 {
			return id, details.Root
		}
	}

	return "", nil
}

func snapshotsToRemove(repo *models.Repo, snapshotID, branchName string) []string {
	removingList := make([]string, 0, len(repo.Snapshots))

	// Traverse up the snapshot tree
	removingList = append(removingList, traverseUp(repo, snapshotID, branchName)...)

	// Traverse down the snapshot tree
	removingList = append(removingList, traverseDown(repo, snapshotID)...)

	return removingList
}

func traverseUp(repo *models.Repo, snapshotID, branchName string) []string {
	removingList := []string{}
	visited := make(map[string]struct{})
	snapshotPointer := repo.Snapshots[snapshotID]

	for snapshotPointer.Parent != "-" && snapshotPointer.Parent != "" {
		if _, found := visited[snapshotPointer.ID]; found {
			break
		}

		visited[snapshotPointer.ID] = struct{}{}

		for _, snapshotRoot := range snapshotPointer.Root {
			if snapshotRoot == branchName {
				return removingList
			}
		}

		removingList = append(removingList, snapshotPointer.ID)

		nextSnapshot, exists := repo.Snapshots[snapshotPointer.Parent]
		if !exists {
			break
		}

		snapshotPointer = nextSnapshot
	}

	return removingList
}

func traverseDown(repo *models.Repo, snapshotID string) []string {
	snapshotPointer := repo.Snapshots[snapshotID]

	removingList := make([]string, 0, len(snapshotPointer.Child))

	for _, snapshotChild := range snapshotPointer.Child {
		removingList = append(removingList, snapshotChild)
		removingList = append(removingList, traverseDown(repo, snapshotChild)...)
	}

	return removingList
}
