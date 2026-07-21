package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/mw"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	logsSinceInterval = "5m"

	// activityTimeout defines the timeout for retrieving activity data.
	activityTimeout = 15 * time.Second
)

func (s *Server) getInstanceStatus(w http.ResponseWriter, r *http.Request) {
	if err := api.WriteJSON(w, http.StatusOK, s.instanceStatus()); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) retrievalState(w http.ResponseWriter, r *http.Request) {
	retrieving := models.Retrieving{
		Mode:        s.Retrieval.State.Mode,
		Status:      s.Retrieval.State.Status,
		Alerts:      s.Retrieval.State.Alerts(),
		LastRefresh: s.Retrieval.State.LastRefresh,
	}

	if spec := s.Retrieval.Scheduler.Spec; spec != nil {
		retrieving.NextRefresh = models.NewLocalTime(spec.Next(time.Now()))
	}

	activityCtx, cancel := context.WithTimeout(context.Background(), activityTimeout)
	defer cancel()

	retrieving.Activity = s.jobActivity(activityCtx)

	if err := api.WriteJSON(w, http.StatusOK, retrieving); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) jobActivity(ctx context.Context) *models.Activity {
	currentJob := s.Retrieval.State.CurrentJob
	if currentJob == nil {
		return nil
	}

	if s.Retrieval.State.Status != models.Refreshing {
		return nil
	}

	jobActivity, err := currentJob.ReportActivity(ctx)
	if err != nil {
		log.Dbg("Failed to get job activity", err)
		return nil
	}

	return &models.Activity{
		Source: toPGActivityEvent(jobActivity.Source),
		Target: toPGActivityEvent(jobActivity.Target),
	}
}

func toPGActivityEvent(pgEvents []activity.PGEvent) []models.PGActivityEvent {
	pgActivityEvents := make([]models.PGActivityEvent, 0, len(pgEvents))

	for _, event := range pgEvents {
		pgActivityEvents = append(pgActivityEvents, models.PGActivityEvent{
			User:          event.User,
			Query:         event.Query,
			Duration:      event.Duration,
			WaitEventType: event.WaitEventType,
			WaitEvent:     event.WaitEvent,
		})
	}

	return pgActivityEvents
}

func (s *Server) getSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.Cloning.GetSnapshots()
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	branchRequest := r.URL.Query().Get("branch")
	datasetRequest := r.URL.Query().Get("dataset")

	if branchRequest != "" {
		fsm, err := s.getFSManagerForBranchAndDataset(branchRequest, datasetRequest)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}

		if fsm == nil {
			api.SendBadRequestError(w, r, "no pool manager found")
			return
		}

		snapshots = filterSnapshotsByBranch(fsm.Pool(), branchRequest, snapshots)
	}

	if branchRequest == "" && datasetRequest != "" {
		snapshots = filterSnapshotsByDataset(datasetRequest, snapshots)
	}

	if err = api.WriteJSON(w, http.StatusOK, snapshots); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) createSnapshot(w http.ResponseWriter, r *http.Request) {
	var poolName string

	if r.Body != http.NoBody {
		var createRequest types.SnapshotCreateRequest
		if err := api.ReadJSON(r, &createRequest); err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}

		poolName = createRequest.PoolName
	}

	if poolName == "" {
		firstFSM := s.pm.First()

		if firstFSM == nil || firstFSM.Pool() == nil {
			api.SendBadRequestError(w, r, pool.ErrNoPools.Error())
			return
		}

		poolName = firstFSM.Pool().Name
	}

	if err := s.Retrieval.SnapshotData(context.Background(), poolName); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsManager, err := s.pm.GetFSManager(poolName)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsManager.RefreshSnapshotList()

	snapshotList := fsManager.SnapshotList()

	if len(snapshotList) == 0 {
		api.SendBadRequestError(w, r, "No snapshots at pool: "+poolName)
		return
	}

	sort.SliceStable(snapshotList, func(i, j int) bool {
		return snapshotList[i].CreatedAt.After(snapshotList[j].CreatedAt)
	})

	if err := fsManager.InitBranching(); err != nil {
		api.SendBadRequestError(w, r, "Cannot init branch metadata: "+err.Error())
		return
	}

	if err := fsManager.VerifyBranchMetadata(); err != nil {
		log.Warn(fmt.Sprintf("failed to verify branch metadata: %v", err))
	}

	latestSnapshot := snapshotList[0]

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.SnapshotCreateEvent,
		EntityID:  latestSnapshot.ID,
	}

	if err := api.WriteJSON(w, http.StatusOK, latestSnapshot); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) deleteSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := mux.Vars(r)["id"]
	if snapshotID == "" {
		api.SendBadRequestError(w, r, "snapshot ID must not be empty")
		return
	}

	forceParam := r.URL.Query().Get("force")
	force := false

	if forceParam != "" {
		var err error
		force, err = strconv.ParseBool(forceParam)

		if err != nil {
			api.SendBadRequestError(w, r, "invalid value for `force`, must be boolean")
			return
		}
	}

	if err := s.destroySnapshotByID(snapshotID, force); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	log.Dbg(fmt.Sprintf("Snapshot %s has been deleted", snapshotID))

	if err := api.WriteJSON(w, http.StatusOK, models.Response{
		Status:  models.ResponseOK,
		Message: "Deleted snapshot",
	}); err != nil {
		api.SendError(w, r, err)
		return
	}

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		log.Dbg("Failed to reload snapshots", err.Error())
	}

	s.webhookCh <- webhooks.BasicEvent{
		EventType: webhooks.SnapshotDeleteEvent,
		EntityID:  snapshotID,
	}

	s.tm.SendEvent(context.Background(), telemetry.SnapshotDestroyedEvent, telemetry.SnapshotDestroyed{
		ID: snapshotID,
	})
}

// ensureNotProtected returns an error when the target snapshot or branch dataset is
// currently protected, reading the local (non-inherited) protection. Shared by the
// snapshot and branch delete paths so the protection rule lives in one place.
func ensureNotProtected(fsm pool.FSManager, target, kind, name string) error {
	protection, err := fsm.GetProtection(target)
	if err != nil {
		return err
	}

	if models.ProtectedTillActive(protection.ProtectedTill) {
		return fmt.Errorf("%s %s is protected", kind, name)
	}

	return nil
}

// destroySnapshotByID deletes a snapshot and its now-orphaned datasets. It is the
// HTTP-independent core shared by the delete handler and the auto-deletion sweeper:
// protected snapshots are refused, dependent clones block a non-force delete, and the
// underlying zfs destroy is non-force unless force is set.
func (s *Server) destroySnapshotByID(snapshotID string, force bool) error {
	poolName, err := s.detectPoolName(snapshotID)
	if err != nil {
		return err
	}

	if poolName == "" {
		return fmt.Errorf("pool for requested snapshot (%s) not found", snapshotID)
	}

	fsm, err := s.pm.GetFSManager(poolName)
	if err != nil {
		return err
	}

	// Prevent deletion of automatic snapshots in the pool.
	if fullDataset, _, found := strings.Cut(snapshotID, "@"); found && fullDataset == poolName {
		return errors.New("cannot destroy automatic snapshot in the pool")
	}

	// Check if snapshot exists.
	if _, err := fsm.GetSnapshotProperties(snapshotID); err != nil {
		if runnerError, ok := err.(runners.RunnerError); ok {
			return errors.New(runnerError.Stderr)
		}

		return err
	}

	// protected snapshots cannot be deleted manually, by force, or by the sweeper.
	if err := ensureNotProtected(fsm, snapshotID, "snapshot", snapshotID); err != nil {
		return err
	}

	cloneIDs, protectedClones, err := s.dependentClones(fsm, snapshotID, poolName)
	if err != nil {
		return err
	}

	if len(protectedClones) != 0 {
		return fmt.Errorf("cannot delete snapshot %s because it has dependent protected clones: %s",
			snapshotID, strings.Join(protectedClones, ","))
	}

	if len(cloneIDs) != 0 && !force {
		return fmt.Errorf("cannot delete snapshot %s because it has dependent clones: %s",
			snapshotID, strings.Join(cloneIDs, ","))
	}

	snapshotProperties, err := fsm.GetSnapshotProperties(snapshotID)
	if err != nil {
		return err
	}

	if snapshotProperties.Clones != "" && !force {
		return fmt.Errorf("cannot delete snapshot %s because it has dependent datasets: %s",
			snapshotID, snapshotProperties.Clones)
	}

	for _, cloneID := range cloneIDs {
		if err = s.Cloning.DestroyCloneSync(cloneID); err != nil {
			return err
		}
	}

	if !force {
		if err := fsm.KeepRelation(snapshotID); err != nil {
			return err
		}
	}

	if force && snapshotProperties.Parent != "" {
		s.relabelParentBranch(fsm, snapshotID, poolName, snapshotProperties)
	}

	if err = fsm.DestroySnapshot(snapshotID, thinclones.DestroyOptions{Force: force}); err != nil {
		return err
	}

	snapshot, err := s.Cloning.GetSnapshotByID(snapshotID)
	if err != nil {
		return err
	}

	if snapshotProperties.Clones == "" && snapshot.NumClones == 0 && snapshotProperties.Child == "" {
		if err := s.cleanupSnapshotDataset(fsm, snapshotID, poolName, snapshotProperties); err != nil {
			return err
		}
	}

	fsm.RefreshSnapshotList()

	return nil
}

// dependentClones returns the IDs of clones that depend on the snapshot and, separately,
// those among them that are protected.
func (s *Server) dependentClones(fsm pool.FSManager, snapshotID, poolName string) (
	cloneIDs, protectedClones []string, err error,
) {
	dependentCloneDatasets, err := fsm.HasDependentEntity(snapshotID)
	if err != nil {
		return nil, nil, err
	}

	for _, cloneDataset := range dependentCloneDatasets {
		cloneID, ok := branching.ParseCloneName(cloneDataset, poolName)
		if !ok {
			log.Dbg(fmt.Sprintf("cannot parse clone ID from %q", cloneDataset))
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

	return cloneIDs, protectedClones, nil
}

// relabelParentBranch moves a branch label to the parent snapshot when a snapshot is
// force-deleted, so the branch keeps pointing at a live snapshot. Failures are logged, not fatal.
func (s *Server) relabelParentBranch(fsm pool.FSManager, snapshotID, poolName string, props thinclones.SnapshotProperties) {
	parentProps, err := fsm.GetSnapshotProperties(props.Parent)
	if err != nil {
		log.Err(err.Error())
	}

	branchName := props.Branch
	fullDataset, _, found := strings.Cut(snapshotID, "@")

	if branchName == "" && found {
		branchName, _ = branching.ParseBranchName(fullDataset, poolName)
	}

	if branchName != "" && !isRoot(parentProps.Root, branchName) {
		if err := fsm.AddBranchProp(branchName, props.Parent); err != nil {
			log.Err(err.Error())
		}
	}
}

// cleanupSnapshotDataset removes the snapshot's dataset and base revision dataset once no
// active datasets or clones remain, and clears stale branch labels from the parent snapshot.
func (s *Server) cleanupSnapshotDataset(
	fsm pool.FSManager, snapshotID, poolName string, props thinclones.SnapshotProperties,
) error {
	fullDataset, _, found := strings.Cut(snapshotID, "@")
	if !found || fullDataset == poolName {
		return nil
	}

	activeDatasets, err := fsm.GetActiveDatasets(fullDataset)
	if err != nil {
		return err
	}

	if len(activeDatasets) == 0 && !s.hasActiveClone(fullDataset, poolName) {
		if err = fsm.DestroyDataset(fullDataset); err != nil {
			return err
		}
	}

	if props.Parent != "" {
		s.cleanupParentBranchLabels(fsm, fullDataset, poolName, props)
	}

	// check if the dataset ends with a revision (for example, /r0).
	if branching.RevisionPattern.MatchString(fullDataset) {
		baseDataset := branching.RevisionPattern.ReplaceAllString(fullDataset, "")
		origins := fsm.GetDatasetOrigins(baseDataset)

		// if this is the last revision, remove the base dataset.
		if len(origins) < branching.MinDatasetNumber {
			if err = fsm.DestroyDataset(baseDataset); err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanupParentBranchLabels removes dle:branch and dle:root for a user branch from the
// parent snapshot, leaving the default branch intact. Failures are logged, not fatal.
func (s *Server) cleanupParentBranchLabels(
	fsm pool.FSManager, fullDataset, poolName string, props thinclones.SnapshotProperties,
) {
	parentProps, err := fsm.GetSnapshotProperties(props.Parent)
	if err != nil {
		log.Err(err.Error())
	}

	branchName := props.Branch
	if branchName == "" {
		branchName, _ = branching.ParseBranchName(fullDataset, poolName)
	}

	if branchName != "" && branchName != branching.DefaultBranch && isRoot(parentProps.Root, branchName) {
		if err := fsm.DeleteBranchProp(branchName, props.Parent); err != nil {
			log.Err(err.Error())
		}

		if err := fsm.DeleteRootProp(branchName, props.Parent); err != nil {
			log.Err(err.Error())
		}
	}
}

func (s *Server) hasActiveClone(fullDataset, poolName string) bool {
	cloneID, ok := branching.ParseCloneName(fullDataset, poolName)
	if !ok {
		return false
	}

	_, errClone := s.Cloning.GetClone(cloneID)
	if errClone != nil && errClone.Error() == "clone not found" {
		return false
	}

	return true
}

func isRoot(root, branch string) bool {
	if root == "" || branch == "" {
		return false
	}

	rootBranches := strings.Split(root, ",")

	return containsString(rootBranches, branch)
}

func (s *Server) detectPoolName(snapshotID string) (string, error) {
	if !isValidSnapshotID(snapshotID) {
		return "", fmt.Errorf("invalid snapshot ID given: %q", snapshotID)
	}

	const snapshotParts = 2

	parts := strings.Split(snapshotID, "@")
	if len(parts) != snapshotParts {
		return "", fmt.Errorf("invalid snapshot name given: %s. Should contain `dataset@snapname`", snapshotID)
	}

	poolName := ""

	for _, fsm := range s.pm.GetFSManagerList() {
		if strings.HasPrefix(parts[0], fsm.Pool().Name) {
			poolName = fsm.Pool().Name
			break
		}
	}

	return poolName, nil
}

func (s *Server) createSnapshotClone(w http.ResponseWriter, r *http.Request) {
	if r.Body == http.NoBody {
		api.SendBadRequestError(w, r, "request body cannot be empty")
		return
	}

	var createRequest types.SnapshotCloneCreateRequest
	if err := api.ReadJSON(r, &createRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if createRequest.CloneID == "" {
		api.SendBadRequestError(w, r, "cloneID cannot be empty")
		return
	}

	clone, err := s.Cloning.GetClone(createRequest.CloneID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	fsm, err := s.pm.GetFSManager(clone.Snapshot.Pool)
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("failed to find filesystem manager: %s", err.Error()))
		return
	}

	fullClonePath := path.Join(branching.BranchDir, clone.Branch, clone.ID, branching.RevisionSegment(branching.DefaultRevision))

	snapshotID, err := fsm.CreateSnapshot(fullClonePath, time.Now().Format(util.DataStateAtFormat))
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("failed to create a snapshot: %s", err.Error()))
		return
	}

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		log.Dbg("Failed to reload snapshots", err.Error())
	}

	snapshot, err := s.Cloning.GetSnapshotByID(snapshotID)
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("failed to find a new snapshot: %s", err.Error()))
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, snapshot); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) clones(w http.ResponseWriter, r *http.Request) {
	cloningState := s.Cloning.GetCloningState()

	if err := api.WriteJSON(w, http.StatusOK, cloningState.Clones); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) createClone(w http.ResponseWriter, r *http.Request) {
	if s.engProps.GetEdition() == global.StandardEdition {
		if err := s.engProps.CheckBilling(); err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}
	}

	var cloneRequest *types.CloneCreateRequest
	if err := api.ReadJSON(r, &cloneRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.validator.ValidateCloneRequest(cloneRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if s.Platform != nil && s.Platform.BindClonesToUser() {
		cloneRequest.DB.OwnerUser = ownerFromContext(r.Context())
	}

	if cloneRequest.Snapshot != nil && cloneRequest.Snapshot.ID != "" {
		fsm, err := s.getFSManagerForSnapshot(cloneRequest.Snapshot.ID)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}

		if fsm == nil {
			api.SendBadRequestError(w, r, "no pool manager found")
			return
		}

		branch := branching.ParseBranchNameFromSnapshot(cloneRequest.Snapshot.ID, fsm.Pool().Name)
		if branch == "" {
			branch = branching.DefaultBranch
		}

		// Snapshot ID takes precedence over the branch name.
		cloneRequest.Branch = branch
	} else {
		if cloneRequest.Branch == "" {
			cloneRequest.Branch = branching.DefaultBranch
		}

		fsm, err := s.getFSManagerForBranch(cloneRequest.Branch)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
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

		snapshotID, ok := branches[cloneRequest.Branch]
		if !ok {
			api.SendBadRequestError(w, r, "branch not found")
			return
		}

		cloneRequest.Snapshot = &types.SnapshotCloneFieldRequest{ID: snapshotID}
	}

	if cloneRequest.ID != "" {
		fsm, err := s.getFSManagerForBranch(cloneRequest.Branch)
		if err != nil {
			api.SendBadRequestError(w, r, err.Error())
			return
		}

		// Check if there is any clone revision under the dataset.
		cloneRequest.Revision = findMaxCloneRevision(fsm.Pool().CloneRevisionLocation(cloneRequest.Branch, cloneRequest.ID))
	}

	newClone, err := s.Cloning.CreateClone(cloneRequest)
	if err != nil {
		var reqErr *models.Error
		if errors.As(err, &reqErr) {
			api.SendBadRequestError(w, r, reqErr.Error())
			return
		}

		api.SendError(w, r, errors.Wrap(err, "failed to create clone"))

		return
	}

	if err := api.WriteJSON(w, http.StatusCreated, newClone); err != nil {
		api.SendError(w, r, err)
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.CloneCreatedEvent, telemetry.CloneCreated{
		ID:          util.HashID(newClone.ID),
		CloningTime: newClone.Metadata.CloningTime,
		DSADiff:     util.GetDataFreshness(newClone.Snapshot.DataStateAt.Time),
	})

	log.Dbg(fmt.Sprintf("Clone ID=%s is being created", newClone.ID))
}

// maxOwnerLabelLength caps the derived owner label length; RFC 5321 limits an
// email address to 254 characters.
const maxOwnerLabelLength = 254

// safeOwnerLabel restricts the derived owner label to characters valid as a
// Teleport label value (a subset of the sidecar's safeYAMLValue), so the engine
// label always matches Teleport's external.email value.
var safeOwnerLabel = regexp.MustCompile(`^[a-zA-Z0-9._@-]+$`)

// ownerFromContext resolves the clone owner label from the authenticated user
// identity on the context. It returns an empty string when no identity is
// present, when the identity has no email, or when the email cannot be
// represented as a valid owner label — such requests create unlabeled clones
// instead of failing; the fallback is logged as a warning so a misconfiguration
// (the Platform not returning the email) or an unlabelable address (e.g. a
// plus-tagged local part) stays visible without blocking clone creation.
func ownerFromContext(ctx context.Context) string {
	identity, ok := mw.UserIdentityFromContext(ctx)
	if !ok {
		return ""
	}

	if identity.Email == "" {
		log.Warn("clone-to-user binding is enabled but the authenticated identity has no email; " +
			"creating an unlabeled clone (check that the Platform returns the user email)")

		return ""
	}

	owner, err := ownerFromEmail(identity.Email)
	if err != nil {
		log.Warn(fmt.Sprintf("clone-to-user binding is enabled but no owner label can be derived: %v; "+
			"creating an unlabeled clone", err))

		return ""
	}

	return owner
}

// ownerFromEmail normalizes the email exactly as Teleport's external.email trait
// carries it — mail.ParseAddress strips any display name and surrounding
// whitespace — and validates that the full address is usable as a Teleport label
// value, so the engine label always matches Teleport's external.email value. The
// full address is used (not just the local part) so users whose local parts
// collide across domains still map to distinct labels.
func ownerFromEmail(email string) (string, error) {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("cannot parse email %q: %w", email, err)
	}

	if !isValidOwnerLabel(addr.Address) {
		return "", fmt.Errorf("cannot derive a valid owner label from email %q", email)
	}

	return addr.Address, nil
}

// isValidOwnerLabel reports whether name is usable as a Teleport dblab_user
// label value: a non-empty identifier of at most maxOwnerLabelLength characters
// from the safeOwnerLabel set.
func isValidOwnerLabel(name string) bool {
	return name != "" && len(name) <= maxOwnerLabelLength && safeOwnerLabel.MatchString(name)
}

func findMaxCloneRevision(path string) int {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Err(err)
		return 0
	}

	maxIndex := -1

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		revisionIndex, ok := strings.CutPrefix(file.Name(), "r")
		if !ok {
			continue
		}

		index, err := strconv.Atoi(revisionIndex)
		if err != nil {
			log.Err(err)
			continue
		}

		if index > maxIndex {
			maxIndex = index
		}
	}

	return maxIndex + 1
}

func (s *Server) destroyClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		api.SendBadRequestError(w, r, "ID must not be empty")
		return
	}

	if err := s.Cloning.DestroyClone(cloneID); err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to destroy clone"))
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.CloneDestroyedEvent, telemetry.CloneDestroyed{
		ID: util.HashID(cloneID),
	})

	log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneID))
}

// applyProtectionUpdate enforces the mutual-exclusivity rules of a protection update and
// writes the resulting values via the given setters (a snapshot or one-or-more branch
// datasets). Enabling protection clears the scheduled deletion and vice versa.
func applyProtectionUpdate(maxMin uint, protected *bool, durationMinutes *uint, deleteAt *models.LocalTime,
	setProtectedTill, setDeleteAt func(string) error) error {
	if protected == nil && deleteAt == nil {
		return errors.New("nothing to update: specify protected or deleteAt")
	}

	protect := protected != nil && *protected

	if protect && deleteAt != nil {
		return errors.New("cannot enable protection and schedule deletion at the same time")
	}

	if protect {
		value := models.ProtectionForever
		if till := models.CalculateProtectionTime(durationMinutes, 0, maxMin); till != nil {
			value = till.UTC().Format(time.RFC3339)
		}

		// clear the scheduled deletion before enabling protection so a mid-write failure
		// leaves the entity unprotected rather than both protected and scheduled to delete.
		if err := setDeleteAt(""); err != nil {
			return err
		}

		return setProtectedTill(value)
	}

	if deleteAt != nil {
		if err := setProtectedTill(""); err != nil {
			return err
		}

		return setDeleteAt(deleteAt.UTC().Format(time.RFC3339))
	}

	// protected == false with no deleteAt: clear protection only.
	return setProtectedTill("")
}

func (s *Server) patchSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := mux.Vars(r)["id"]
	if snapshotID == "" {
		api.SendBadRequestError(w, r, "snapshot ID must not be empty")
		return
	}

	var req types.SnapshotUpdateRequest
	if err := api.ReadJSON(r, &req); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	poolName, err := s.detectPoolName(snapshotID)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if poolName == "" {
		api.SendBadRequestError(w, r, fmt.Sprintf("pool for requested snapshot (%s) not found", snapshotID))
		return
	}

	fsm, err := s.pm.GetFSManager(poolName)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if _, err := fsm.GetSnapshotProperties(snapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	setTill := func(v string) error { return fsm.SetProtectedTill(v, snapshotID) }
	setDeleteAt := func(v string) error { return fsm.SetDeleteAt(v, snapshotID) }

	if err := applyProtectionUpdate(s.Retention().ProtectionMaxDurationMinutes,
		req.Protected, req.ProtectionDurationMinutes, req.DeleteAt, setTill, setDeleteAt); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		log.Dbg("failed to reload snapshots", err.Error())
	}

	// build the response from the authoritative live protection. Enrich from the in-memory
	// snapshot when it is present, but never fail after a successful write: a snapshot that
	// exists in ZFS yet is absent from the cloning box (e.g. a branch commit or _pre snapshot)
	// must still return the change that was just applied.
	snapshot := &models.Snapshot{ID: snapshotID}

	if existing, err := s.Cloning.GetSnapshotByID(snapshotID); err == nil {
		snapshotCopy := *existing
		snapshot = &snapshotCopy
	}

	snapshot.Protected, snapshot.ProtectedTill, snapshot.DeleteAt = readProtection(fsm, snapshotID)

	s.tm.SendEvent(context.Background(), telemetry.SnapshotUpdatedEvent, telemetry.SnapshotUpdated{
		ID:        snapshotID,
		Protected: snapshot.Protected,
	})

	if err := api.WriteJSON(w, http.StatusOK, snapshot); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) patchClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		api.SendBadRequestError(w, r, "ID must not be empty")
		return
	}

	var patchClone types.CloneUpdateRequest
	if err := api.ReadJSON(r, &patchClone); err != nil {
		api.SendBadRequestError(w, r, err.Error())

		return
	}

	updatedClone, err := s.Cloning.UpdateClone(cloneID, patchClone)
	if err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to update clone"))
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.CloneUpdatedEvent, telemetry.CloneUpdated{
		ID:        util.HashID(cloneID),
		Protected: patchClone.Protected,
	})

	if err := api.WriteJSON(w, http.StatusOK, updatedClone); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		api.SendBadRequestError(w, r, "ID must not be empty")
		return
	}

	clone, err := s.Cloning.GetClone(cloneID)
	if err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, clone); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) resetClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		api.SendBadRequestError(w, r, "ID must not be empty")
		return
	}

	var resetOptions types.ResetCloneRequest

	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&resetOptions); err != nil {
			api.SendError(w, r, errors.Wrap(err, "failed to parse request parameters"))
			return
		}
	}

	if resetOptions.Latest && resetOptions.SnapshotID != "" {
		api.SendBadRequestError(w, r, "parameters `latest` and `snapshot ID` must not be specified together")
		return
	}

	if err := s.Cloning.ResetClone(cloneID, resetOptions); err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to reset clone"))
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being reset", cloneID))
}

func (s *Server) startObservation(w http.ResponseWriter, r *http.Request) {
	if s.Platform.Client == nil {
		api.SendBadRequestError(w, r, "cannot start the session observation because a Platform client is not configured")
		return
	}

	var observationRequest *types.StartObservationRequest
	if err := api.ReadJSON(r, &observationRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	clone, err := s.Cloning.GetClone(observationRequest.CloneID)
	if err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	clone.DB.Username = s.Global.Database.User()

	db, err := observer.InitConnection(clone, s.pm.First().Pool().SocketDir())
	if err != nil {
		api.SendError(w, r, errors.Wrap(err, "cannot connect to database"))
		return
	}

	observingClone := observer.NewObservingClone(observationRequest.Config, db)
	startedAt := time.Now().Round(time.Millisecond)

	port, err := strconv.Atoi(clone.DB.Port)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	s.Observer.AddObservingClone(clone.ID, clone.Branch, clone.Revision, uint(port), observingClone)

	// Start session on the Platform.
	platformRequest := platform.StartObservationRequest{
		InstanceID: s.engProps.InstanceID,
		CloneID:    clone.ID,
		StartedAt:  startedAt.Format("2006-01-02 15:04:05 UTC"),
		Config:     observingClone.Config(),
		Tags:       observationRequest.Tags,
	}

	platformResponse, err := s.Platform.Client.StartObservationSession(context.Background(), platformRequest)
	if err != nil {
		api.SendBadRequestError(w, r, "Failed to start observation session on the Platform")
		return
	}

	if observationRequest.DBName != "" {
		clone.DB.DBName = observationRequest.DBName
	}

	if err := observingClone.Init(clone, platformResponse.SessionID, startedAt, observationRequest.Tags); err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to init observing session"))
		return
	}

	go func() {
		if err := observingClone.RunSession(); err != nil {
			// TODO(akartasov): Update observation (add a request to Platform) with an error.
			log.Err("failed to observe clone: ", err)
		}
	}()

	if err := api.WriteJSON(w, http.StatusOK, observingClone.Session()); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) stopObservation(w http.ResponseWriter, r *http.Request) {
	if s.Platform.Client == nil {
		api.SendBadRequestError(w, r, "cannot stop the session observation because a Platform client is not configured")
		return
	}

	var observationRequest *types.StopObservationRequest

	if err := api.ReadJSON(r, &observationRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	observingClone, err := s.Observer.GetObservingClone(observationRequest.CloneID)
	if err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	if _, err := s.Cloning.GetClone(observationRequest.CloneID); err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	if err := s.Cloning.UpdateCloneStatus(observationRequest.CloneID, models.Status{Code: models.StatusExporting}); err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	defer func() {
		if err := s.Cloning.UpdateCloneStatus(observationRequest.CloneID, models.Status{Code: models.StatusOK}); err != nil {
			log.Err("failed to update clone status", err)
		}
	}()

	if observationRequest.OverallError {
		// This is the single way to determine that an external migration command fails.
		observingClone.SetOverallError(true)
	}

	if err := observingClone.Stop(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	session := observingClone.Session()
	if session == nil || session.Result == nil {
		api.SendBadRequestError(w, r, "observing session has not been initialized")
		return
	}

	platformRequest := platform.StopObservationRequest{
		SessionID:  session.SessionID,
		FinishedAt: session.FinishedAt.Format("2006-01-02 15:04:05 UTC"),
		Result:     *session.Result,
	}

	if _, err := s.Platform.Client.StopObservationSession(context.Background(), platformRequest); err != nil {
		api.SendBadRequestError(w, r, "Failed to start observation session on the Platform")
		return
	}

	sessionID := strconv.FormatUint(session.SessionID, 10)

	logs, err := s.Observer.GetCloneLog(context.TODO(), observingClone)
	if err != nil {
		log.Err("failed to get observation logs", err)
	}

	if len(logs) > 0 {
		if err := s.Platform.Client.UploadObservationLogs(context.Background(), logs, sessionID); err != nil {
			log.Err("failed to upload observation logs", err)
		}
	}

	for _, artifactType := range session.Artifacts {
		artPath := observingClone.BuildArtifactPath(session.SessionID, artifactType)

		data, err := os.ReadFile(artPath)
		if err != nil {
			log.Errf("failed to read artifact %s: %s", artPath, err)
			continue
		}

		if err := s.Platform.Client.UploadObservationArtifact(context.Background(), data, sessionID, artifactType); err != nil {
			log.Err("failed to upload observation artifact", err)
		}
	}

	if err := api.WriteJSON(w, http.StatusOK, session); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) sessionSummaryObservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	cloneID := vars["clone_id"]

	sessionID, err := strconv.ParseUint(vars["session_id"], 10, 64)
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("invalid session_id: %v", sessionID))
		return
	}

	observingClone, err := s.Observer.GetObservingClone(cloneID)
	if err != nil || !observingClone.IsExistArtifacts(sessionID) {
		api.SendNotFoundError(w, r)
		return
	}

	summaryData, err := observingClone.ReadSummary(sessionID)
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("failed to read summary: %v", err))
		return
	}

	if err := api.WriteData(w, http.StatusOK, summaryData); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) downloadArtifact(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	artifactType := values.Get("artifact_type")

	if !observer.IsAvailableArtifactType(artifactType) {
		api.SendBadRequestError(w, r, fmt.Sprintf("artifact %q is not available to download", artifactType))
		return
	}

	sessionID, err := strconv.ParseUint(values.Get("session_id"), 10, 64)
	if err != nil {
		api.SendBadRequestError(w, r, fmt.Sprintf("invalid session_id: %v", sessionID))
		return
	}

	cloneID := values.Get("clone_id")

	observingClone, err := s.Observer.GetObservingClone(cloneID)
	if err != nil || !observingClone.IsExistArtifacts(sessionID) {
		api.SendNotFoundError(w, r)
		return
	}

	filePath := observingClone.BuildArtifactPath(sessionID, artifactType)
	http.ServeFile(w, r, filePath)
}

// healthCheck provides a health check handler.
func (s *Server) healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", api.JSONContentType)

	healthResponse := models.Engine{
		Version:    version.GetVersion(),
		Edition:    s.engProps.GetEdition(),
		InstanceID: s.engProps.InstanceID,
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if err := s.Retrieval.CanStartRefresh(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Retrieval.HasAvailablePool(); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	go func() {
		if err := s.Retrieval.FullRefresh(context.Background()); err != nil {
			log.Err("failed to initiate full refresh", err)
		}
	}()

	if err := api.WriteJSON(w, http.StatusOK, models.Response{
		Status:  models.ResponseOK,
		Message: "Full refresh started",
	}); err != nil {
		api.SendError(w, r, err)
	}
}
