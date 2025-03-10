package srv

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

var branchNameRegexp = regexp.MustCompile(`^[\p{L}\d_-]+$`)

// listBranches returns branch list.
func (s *Server) listBranches(w http.ResponseWriter, r *http.Request) {
	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	branches, err := fsm.ListAllBranches()
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

	// branchRegistry is used to display the "main" branch with only the most recent snapshot.
	branchRegistry := make(map[string]int, 0)

	for _, branchEntity := range branches {
		snapshotDetails, ok := repo.Snapshots[branchEntity.SnapshotID]
		if !ok {
			continue
		}

		numSnapshots, parentSnapshot := findBranchParent(repo.Snapshots, snapshotDetails.ID, branchEntity.Name)

		branchView := models.BranchView{
			Name:         branchEntity.Name,
			Parent:       parentSnapshot,
			DataStateAt:  snapshotDetails.DataStateAt,
			SnapshotID:   snapshotDetails.ID,
			Dataset:      snapshotDetails.Dataset,
			NumSnapshots: numSnapshots,
		}

		if position, ok := branchRegistry[branchEntity.Name]; ok {
			if branchView.DataStateAt > branchDetails[position].DataStateAt {
				branchDetails[position] = branchView
			}

			continue
		}

		branchRegistry[branchView.Name] = len(branchDetails)
		branchDetails = append(branchDetails, branchView)
	}

	if err := api.WriteJSON(w, http.StatusOK, branchDetails); err != nil {
		api.SendError(w, r, err)
		return
	}
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

//nolint:unused
func (s *Server) getFSManagerForBranch(branchName string) (pool.FSManager, error) {
	allBranches, err := s.pm.First().ListAllBranches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch list: %w", err)
	}

	for _, branchEntity := range allBranches {
		if branchEntity.Name == branchName {
			return s.getFSManagerForSnapshot(branchEntity.SnapshotID)
		}
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
		api.SendBadRequestError(w, r, "The branch name must contain only Unicode characters, numbers, underscores, and hyphens. "+
			"Spaces and slashes are not allowed")
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

	snapshotID := createRequest.SnapshotID

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

	snapshotBase := fmt.Sprintf("%s/%s", clone.Snapshot.Pool, clone.ID)
	snapshotName := fmt.Sprintf("%s@%s", snapshotBase, dataStateAt)

	if err := fsm.Snapshot(snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	targetBranchSnap := fmt.Sprintf("%[1]s@%[1]s", dataStateAt)

	targetSnap := fmt.Sprintf("%s/%s", fsm.Pool().BranchName(clone.Snapshot.Pool, clone.Branch), targetBranchSnap)

	if err := fsm.Move(currentSnapshotID, snapshotName, targetSnap); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.AddBranchProp(clone.Branch, targetSnap); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.DeleteBranchProp(clone.Branch, currentSnapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetRelation(currentSnapshotID, targetSnap); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, targetSnap); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetMessage(snapshotRequest.Message, targetSnap); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.DestroySnapshot(snapshotName, thinclones.DestroyOptions{}); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.ResetClone(clone.ID, types.ResetCloneRequest{SnapshotID: targetSnap}); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshot, err := s.Cloning.GetSnapshotByID(targetSnap)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.UpdateCloneSnapshot(clone.ID, snapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.SnapshotCreatedEvent, telemetry.SnapshotCreated{})

	if err := api.WriteJSON(w, http.StatusOK, types.SnapshotResponse{SnapshotID: targetSnap}); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getBranchSnapshots(w http.ResponseWriter, r *http.Request) {
	branchRequest := mux.Vars(r)["branch"]

	if branchRequest == "" {
		api.SendBadRequestError(w, r, "branch must not be empty")
		return
	}

	fsm, err := s.getFSManagerForBranch(branchRequest)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if fsm == nil {
		api.SendBadRequestError(w, r, "no pool manager found")
		return
	}

	snapshots, err := s.Cloning.GetSnapshots()
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	branchSnapshots := filterSnapshotsByBranch(fsm.Pool(), branchRequest, snapshots)

	if err = api.WriteJSON(w, http.StatusOK, branchSnapshots); err != nil {
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

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	var logRequest types.LogRequest
	if err := api.ReadJSON(r, &logRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm, err := s.getFSManagerForBranch(logRequest.BranchName)
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

	snapshotID, ok := repo.Branches[logRequest.BranchName]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+logRequest.BranchName)
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
	var deleteRequest types.BranchDeleteRequest
	if err := api.ReadJSON(r, &deleteRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm, err := s.getFSManagerForBranch(deleteRequest.BranchName)
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

	snapshotID, ok := repo.Branches[deleteRequest.BranchName]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+deleteRequest.BranchName)
		return
	}

	toRemove := snapshotsToRemove(repo, snapshotID, deleteRequest.BranchName)

	if len(toRemove) > 0 {
		// Pre-check.
		preCheckList := make(map[string]int)

		for _, snapshotID := range toRemove {
			if cloneNum := s.Cloning.GetCloneNumber(snapshotID); cloneNum > 0 {
				preCheckList[snapshotID] = cloneNum
			}
		}

		if len(preCheckList) > 0 {
			errMsg := fmt.Sprintf("cannot delete branch %q because", deleteRequest.BranchName)

			for snapID, cloneNum := range preCheckList {
				errMsg += fmt.Sprintf(" snapshot %q contains %d clone(s)", snapID, cloneNum)
			}

			log.Warn(errMsg)
			api.SendBadRequestError(w, r, errMsg)

			return
		}

		brName := fsm.Pool().BranchName(fsm.Pool().Name, deleteRequest.BranchName)

		if err := fsm.DestroyBranch(brName); err != nil {
			log.Warn(fmt.Sprintf("failed to remove snapshot %q:", brName), err)
			api.SendBadRequestError(w, r, fmt.Sprintf("failed to remove snapshot %q:", brName))

			return
		}

		datasetFull := strings.Split(toRemove[0], "@")
		datasetName, _ := strings.CutPrefix(datasetFull[0], fsm.Pool().Name+"/")

		if err := fsm.DestroyClone(datasetName); err != nil {
			errMsg := fmt.Sprintf("cannot destroy the underlying branch dataset: %s", datasetName)
			log.Warn(errMsg, err)
			api.SendBadRequestError(w, r, errMsg)

			return
		}
	}

	// Re-request the repository as the list of snapshots may change significantly.
	repo, err = fsm.GetRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := cleanupSnapshotProperties(repo, fsm, deleteRequest.BranchName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm.RefreshSnapshotList()

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.BranchDeleteEvent,
		EntityID:  deleteRequest.BranchName,
	}

	s.tm.SendEvent(context.Background(), telemetry.BranchDestroyedEvent, telemetry.BranchDestroyed{
		Name: deleteRequest.BranchName,
	})

	if err := api.WriteJSON(w, http.StatusOK, models.Response{
		Status:  models.ResponseOK,
		Message: "Deleted branch",
	}); err != nil {
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

func snapshotsToRemove(repo *models.Repo, snapshotID, branchName string) []string {
	snapshotPointer := repo.Snapshots[snapshotID]

	removingList := []string{}

	for snapshotPointer.Parent != "-" {
		if len(snapshotPointer.Root) > 0 {
			break
		}

		for _, snapshotRoot := range snapshotPointer.Root {
			if snapshotRoot == branchName {
				break
			}
		}

		removingList = append(removingList, snapshotPointer.ID)
		snapshotPointer = repo.Snapshots[snapshotPointer.Parent]
	}

	return removingList
}
