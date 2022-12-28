package srv

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
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

	branches, err := fsm.ListBranches()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	repo, err := fsm.GetRepo()
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

		branchDetails = append(branchDetails,
			models.BranchView{
				Name:        branchName,
				Parent:      findBranchParent(repo.Snapshots, snapshotDetails.ID, branchName),
				DataStateAt: snapshotDetails.DataStateAt,
				SnapshotID:  snapshotDetails.ID,
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

	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
		return
	}

	snapshotID := createRequest.SnapshotID

	if snapshotID == "" {
		branches, err := fsm.ListBranches()
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}

		branchPointer, ok := branches[createRequest.BaseBranch]
		if !ok {
			api.SendBadRequestError(w, r, "branch not found")
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

	fsm := s.pm.First()

	if fsm == nil {
		api.SendBadRequestError(w, r, "no available pools")
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

	dataStateAt := time.Now().Format(util.DataStateAtFormat)

	snapshotBase := fmt.Sprintf("%s/%s", clone.Snapshot.Pool, util.GetCloneNameStr(clone.DB.Port))
	snapshotName := fmt.Sprintf("%s@%s", snapshotBase, dataStateAt)

	if err := fsm.Snapshot(snapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	newSnapshotName := fmt.Sprintf("%s/%s/%s", fsm.Pool().Name, clone.Branch, dataStateAt)

	if err := fsm.Rename(snapshotBase, newSnapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	snapshotPath := fmt.Sprintf("%s/%s@%s", fsm.Pool().ClonesDir(), clone.Branch, dataStateAt)
	if err := fsm.SetMountpoint(snapshotPath, newSnapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.AddBranchProp(clone.Branch, newSnapshotName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.DeleteBranchProp(clone.Branch, currentSnapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	childID := newSnapshotName + "@" + dataStateAt
	if err := fsm.SetRelation(currentSnapshotID, childID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetDSA(dataStateAt, childID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := fsm.SetMessage(snapshotRequest.Message, childID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	newSnapshot, err := s.Cloning.GetSnapshotByID(childID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.UpdateCloneSnapshot(clone.ID, newSnapshot); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, types.SnapshotResponse{SnapshotID: childID}); err != nil {
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

	snapshotID, ok := repo.Branches[logRequest.BranchName]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+logRequest.BranchName)
		return
	}

	snapshotPointer := repo.Snapshots[snapshotID]

	logList := []models.SnapshotDetails{snapshotPointer}

	// Limit the number of iterations to the number of snapshots.
	for i := len(repo.Snapshots); i > 1; i-- {
		snapshotPointer = repo.Snapshots[snapshotPointer.Parent]
		logList = append(logList, snapshotPointer)

		if snapshotPointer.Parent == "-" || snapshotPointer.Parent == "" {
			break
		}
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

	snapshotID, ok := repo.Branches[deleteRequest.BranchName]
	if !ok {
		api.SendBadRequestError(w, r, "branch not found: "+deleteRequest.BranchName)
		return
	}

	if hasSnapshots(repo, snapshotID, deleteRequest.BranchName) {
		if err := fsm.DeleteBranch(deleteRequest.BranchName); err != nil {
			api.SendBadRequestError(w, r, err.Error())
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

func hasSnapshots(repo *models.Repo, snapshotID, branchName string) bool {
	snapshotPointer := repo.Snapshots[snapshotID]

	for _, rootBranch := range snapshotPointer.Root {
		if rootBranch == branchName {
			return false
		}
	}

	return true
}
