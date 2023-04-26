package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	logsSinceInterval = "5m"
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

	retrieving.Activity = s.jobActivity(r.Context())

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

	if err := fsManager.InitBranching(); err != nil {
		api.SendBadRequestError(w, r, "Cannot verify branch metadata: "+err.Error())
		return
	}

	// TODO: set branching metadata.

	latestSnapshot := snapshotList[0]

	if err := api.WriteJSON(w, http.StatusOK, latestSnapshot); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) deleteSnapshot(w http.ResponseWriter, r *http.Request) {
	var destroyRequest types.SnapshotDestroyRequest
	if err := api.ReadJSON(r, &destroyRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	const snapshotParts = 2

	parts := strings.Split(destroyRequest.SnapshotID, "@")
	if len(parts) != snapshotParts {
		api.SendBadRequestError(w, r, fmt.Sprintf("invalid snapshot name given: %s", destroyRequest.SnapshotID))
		return
	}

	rootParts := strings.Split(parts[0], "/")
	if len(rootParts) < 1 {
		api.SendBadRequestError(w, r, fmt.Sprintf("invalid root part of snapshot name given: %s", destroyRequest.SnapshotID))
		return
	}

	poolName := rootParts[0]

	fsm, err := s.pm.GetFSManager(poolName)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err = fsm.DestroySnapshot(destroyRequest.SnapshotID); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	// TODO: update branching metadata.

	log.Dbg(fmt.Sprintf("Snapshot %s has been deleted", destroyRequest.SnapshotID))

	if err := api.WriteJSON(w, http.StatusOK, ""); err != nil {
		api.SendError(w, r, err)
		return
	}

	fsm.RefreshSnapshotList()

	if err := s.Cloning.ReloadSnapshots(); err != nil {
		log.Dbg("Failed to reload snapshots", err.Error())
	}
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

	cloneName := util.GetCloneNameStr(clone.DB.Port)

	snapshotID, err := fsm.CreateSnapshot(cloneName, time.Now().Format(util.DataStateAtFormat))
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

func (s *Server) createClone(w http.ResponseWriter, r *http.Request) {
	var cloneRequest *types.CloneCreateRequest
	if err := api.ReadJSON(r, &cloneRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.validator.ValidateCloneRequest(cloneRequest); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if cloneRequest.Branch != "" {
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

		snapshotID, ok := branches[cloneRequest.Branch]
		if !ok {
			api.SendBadRequestError(w, r, "branch not found")
			return
		}

		cloneRequest.Snapshot = &types.SnapshotCloneFieldRequest{ID: snapshotID}
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

	s.Observer.AddObservingClone(clone.ID, uint(port), observingClone)

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

	clone, err := s.Cloning.GetClone(observationRequest.CloneID)
	if err != nil {
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

	logs, err := s.Observer.GetCloneLog(context.TODO(), clone.DB.Port, observingClone)
	if err != nil {
		log.Err("Failed to get observation logs", err)
	}

	if len(logs) > 0 {
		if err := s.Platform.Client.UploadObservationLogs(context.Background(), logs, sessionID); err != nil {
			log.Err("Failed to upload observation logs", err)
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
			log.Err("Failed to upload observation artifact", err)
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
