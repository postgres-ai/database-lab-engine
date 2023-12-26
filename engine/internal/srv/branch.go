package srv

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

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

	for branchName, snapshotID := range branches {
		snapshotDetails, ok := repo.Snapshots[snapshotID]
		if !ok {
			continue
		}

		_, branchNam, _ := strings.Cut(branchName, "_")

		branchDetails = append(branchDetails,
			models.BranchView{
				Name:        branchNam,
				Parent:      findBranchParent(repo.Snapshots, snapshotDetails.ID, branchNam),
				DataStateAt: snapshotDetails.DataStateAt,
				SnapshotID:  snapshotDetails.ID,
				Dataset:     snapshotDetails.Dataset,
			})
	}

	if err := api.WriteJSON(w, http.StatusOK, branchDetails); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func findBranchParent(snapshots map[string]models.SnapshotDetails, parentID, branch string) string {
	for i := len(snapshots); i > 0; i-- {
		snapshotPointer := snapshots[parentID]

		if containsString(snapshotPointer.Root, branch) {
			if len(snapshotPointer.Branch) > 0 {
				return snapshotPointer.Branch[0]
			}

			break
		}

		if snapshotPointer.Parent == "-" {
			break
		}

		parentID = snapshotPointer.Parent
	}

	return "-"
}

func containsString(slice []string, s string) bool {
	for _, str := range slice {
		if str == s {
			return true
		}
	}

	return false
}

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request) {
	var createRequest types.BranchCreateRequest
	if err := api.ReadJSON(r, &createRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if createRequest.BranchName == "" {
		api.SendBadRequestError(w, r, "branchName must not be empty")
		return
	}

	if createRequest.BranchName == createRequest.BaseBranch {
		api.SendBadRequestError(w, r, "new and base branches must have different names")
		return
	}

	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	branches, err := fsm.ListBranches()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if _, ok := branches[models.BranchName(fsm.Pool().Name, createRequest.BranchName)]; ok {
		api.SendBadRequestError(w, r, fmt.Sprintf("branch '%s' already exists", createRequest.BranchName))
		return
	}

	snapshotID := createRequest.SnapshotID

	if snapshotID == "" {
		if createRequest.BaseBranch == "" {
			api.SendBadRequestError(w, r, "either base branch name or base snapshot ID must be specified")
			return
		}

		branchPointer, ok := branches[models.BranchName(fsm.Pool().Name, createRequest.BranchName)]
		if !ok {
			api.SendBadRequestError(w, r, "base branch not found")
			return
		}

		snapshotID = branchPointer
	}

	if err := fsm.AddBranchProp(createRequest.BranchName, snapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetRoot(createRequest.BranchName, snapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	branch := models.Branch{Name: createRequest.BranchName}

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.BranchCreateEvent,
		EntityID:  branch.Name,
	}

	if err := api.WriteJSON(w, http.StatusOK, branch); err != nil {
		api.SendError(w, r, err)
		return
	}
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

	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
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

	currentSnapshotID, ok := branches[models.BranchName(fsm.Pool().Name, clone.Branch)]
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

	// Since the snapshot is created from a clone, it already has one associated clone.
	s.Cloning.IncrementCloneNumber(snapshotName)

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshot, err := s.Cloning.GetSnapshotByID(snapshotName)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.UpdateCloneSnapshot(clone.ID, snapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, types.SnapshotResponse{SnapshotID: snapshotName}); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	var logRequest types.LogRequest
	if err := api.ReadJSON(r, &logRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	repo, err := fsm.GetRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshotID, ok := repo.Branches[models.BranchName(fsm.Pool().Name, logRequest.BranchName)]
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

	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	repo, err := fsm.GetRepo()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshotID, ok := repo.Branches[models.BranchName(fsm.Pool().Name, deleteRequest.BranchName)]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+deleteRequest.BranchName)
		return
	}

	toRemove := snapshotsToRemove(repo, snapshotID, deleteRequest.BranchName)

	// Pre-check.
	for _, snapshotID := range toRemove {
		if cloneNum := s.Cloning.GetCloneNumber(snapshotID); cloneNum > 0 {
			log.Warn(fmt.Sprintf("cannot delete branch %q because snapshot %q contains %d clone(s)",
				deleteRequest.BranchName, snapshotID, cloneNum))
		}
	}

	for _, snapshotID := range toRemove {
		if err := fsm.DestroySnapshot(snapshotID); err != nil {
			log.Warn(fmt.Sprintf("failed to remove snapshot %q:", snapshotID), err.Error())
		}
	}

	if len(toRemove) > 0 {
		datasetFull := strings.Split(toRemove[0], "@")
		datasetName, _ := strings.CutPrefix(datasetFull[0], fsm.Pool().Name+"/")

		if err := fsm.DestroyClone(datasetName); err != nil {
			log.Warn("cannot destroy the underlying branch dataset", err)
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

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.BranchDeleteEvent,
		EntityID:  deleteRequest.BranchName,
	}

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
