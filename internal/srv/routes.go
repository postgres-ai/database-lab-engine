package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/estimator"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

func (s *Server) getInstanceStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.Cloning.GetInstanceState()
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	refresh := models.Retrieving{
		Mode:        s.Retrieval.State.Mode,
		Status:      s.Retrieval.State.Status,
		LastRefresh: s.Retrieval.State.LastRefresh,
	}

	if s.Retrieval.Scheduler.Spec != nil {
		refresh.NextRefresh = pointer.ToTimeOrNil(s.Retrieval.Scheduler.Spec.Next(time.Now()))
	}

	status.Retrieving = refresh

	if err = api.WriteJSON(w, http.StatusOK, status); err != nil {
		api.SendError(w, r, err)
		return
	}
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
		DSADiff:     util.GetDataFreshness(newClone.Snapshot.DataStateAt),
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

func (s *Server) startEstimator(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	cloneID := values.Get("clone_id")

	pid, err := strconv.Atoi(values.Get("pid"))
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	clone, err := s.Cloning.GetClone(cloneID)
	if err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	ctx := context.Background()

	cloneContainer, err := s.docker.ContainerInspect(ctx, util.GetCloneNameStr(clone.DB.Port))
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	db, err := s.Cloning.ConnectToClone(ctx, cloneID)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	defer func() {
		if err := ws.Close(); err != nil {
			log.Err(err)
		}
	}()

	done := make(chan struct{})

	go wsPing(ws, done)

	if err := s.runEstimator(ctx, ws, db, pid, cloneContainer.ID, done); err != nil {
		api.SendError(w, r, err)
		return
	}

	<-done
}

func (s *Server) runEstimator(ctx context.Context, ws *websocket.Conn, db pgxtype.Querier, pid int, containerID string,
	done chan struct{}) error {
	defer close(done)

	estCfg := s.Estimator.Config()

	profiler := estimator.NewProfiler(db, estimator.TraceOptions{
		Pid:             pid,
		Interval:        estCfg.ProfilingInterval,
		SampleThreshold: estCfg.SampleThreshold,
		ReadRatio:       estCfg.ReadRatio,
		WriteRatio:      estCfg.WriteRatio,
	})

	// Start profiling.
	s.Estimator.Run(ctx, profiler)

	readyEventData, err := json.Marshal(estimator.Event{EventType: estimator.ReadyEventType})
	if err != nil {
		return err
	}

	if err := ws.WriteMessage(websocket.TextMessage, readyEventData); err != nil {
		return errors.Wrap(err, "failed to write message with the ready event")
	}

	monitor := estimator.NewMonitor(pid, containerID, profiler)

	go func() {
		if err := monitor.InspectIOBlocks(ctx); err != nil {
			log.Err(err)
		}
	}()

	// Wait for profiling results.
	<-profiler.Finish()

	estTime, err := profiler.EstimateTime(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to estimate time")
	}

	resultEvent := estimator.ResultEvent{
		EventType: estimator.ResultEventType,
		Payload: estimator.Result{
			IsEnoughStat:    profiler.IsEnoughSamples(),
			SampleCounter:   profiler.CountSamples(),
			TotalTime:       profiler.TotalTime(),
			EstTime:         estTime,
			RenderedStat:    profiler.RenderStat(),
			WaitEventsRatio: profiler.WaitEventsRatio(),
		},
	}

	resultEventData, err := json.Marshal(resultEvent)
	if err != nil {
		return err
	}

	if err := ws.WriteMessage(websocket.TextMessage, resultEventData); err != nil {
		return errors.Wrap(err, "failed to write message with the ready event")
	}

	return nil
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	healthResponse := models.Engine{
		Version: version.GetVersion(),
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}
}
