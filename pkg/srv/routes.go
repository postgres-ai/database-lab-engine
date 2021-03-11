package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/estimator"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
	"gitlab.com/postgres-ai/database-lab/v2/version"
)

func (s *Server) getInstanceStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.Cloning.GetInstanceState()
	if err != nil {
		sendError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusOK, status); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) getSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.Cloning.GetSnapshots()
	if err != nil {
		sendError(w, r, err)
		return
	}

	if err = writeJSON(w, http.StatusOK, snapshots); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) createClone(w http.ResponseWriter, r *http.Request) {
	var cloneRequest *types.CloneCreateRequest
	if err := readJSON(r, &cloneRequest); err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.validator.ValidateCloneRequest(cloneRequest); err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	newClone, err := s.Cloning.CreateClone(cloneRequest)
	if err != nil {
		sendError(w, r, errors.Wrap(err, "failed to create clone"))
		return
	}

	if err := writeJSON(w, http.StatusCreated, newClone); err != nil {
		sendError(w, r, err)
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being created", newClone.ID))
}

func (s *Server) destroyClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		sendBadRequestError(w, r, "ID must not be empty")
		return
	}

	if err := s.Cloning.DestroyClone(cloneID); err != nil {
		sendError(w, r, errors.Wrap(err, "failed to destroy clone"))
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneID))
}

func (s *Server) patchClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		sendBadRequestError(w, r, "ID must not be empty")
		return
	}

	var patchClone *types.CloneUpdateRequest
	if err := readJSON(r, &patchClone); err != nil {
		sendBadRequestError(w, r, err.Error())

		return
	}

	updatedClone, err := s.Cloning.UpdateClone(cloneID, patchClone)
	if err != nil {
		sendError(w, r, errors.Wrap(err, "failed to update clone"))
		return
	}

	if err := writeJSON(w, http.StatusOK, updatedClone); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) getClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		sendBadRequestError(w, r, "ID must not be empty")
		return
	}

	clone, err := s.Cloning.GetClone(cloneID)
	if err != nil {
		sendNotFoundError(w, r)
		return
	}

	if err := writeJSON(w, http.StatusOK, clone); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) resetClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		sendBadRequestError(w, r, "ID must not be empty")
		return
	}

	if err := s.Cloning.ResetClone(cloneID); err != nil {
		sendError(w, r, errors.Wrap(err, "failed to reset clone"))
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being reset", cloneID))
}

func (s *Server) startEstimator(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	cloneID := values.Get("clone_id")

	pid, err := strconv.Atoi(values.Get("pid"))
	if err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	if _, err := s.Cloning.GetClone(cloneID); err != nil {
		sendNotFoundError(w, r)
		return
	}

	ctx := context.Background()

	db, err := s.Cloning.CloneConnection(ctx, cloneID)
	if err != nil {
		sendError(w, r, err)
		return
	}

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		sendError(w, r, err)
		return
	}

	defer func() {
		if err := ws.Close(); err != nil {
			log.Err(err)
		}
	}()

	done := make(chan struct{})

	go wsPing(ws, done)

	if err := s.runEstimator(ctx, ws, db, pid, done); err != nil {
		sendError(w, r, err)
		return
	}

	<-done
}

func (s *Server) runEstimator(ctx context.Context, ws *websocket.Conn, db pgxtype.Querier, pid int, done chan struct{}) error {
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

	go func() {
		if err := receiveClientMessages(ctx, ws, profiler); err != nil {
			log.Dbg("receive client messages: ", err)
		}
	}()

	// Wait for profiling results.
	<-profiler.Finish()

	<-profiler.ReadyToEstimate()

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

func receiveClientMessages(ctx context.Context, ws *websocket.Conn, profiler *estimator.Profiler) error {
	for {
		if ctx.Err() != nil {
			log.Msg(ctx.Err())
			break
		}

		_, message, err := ws.ReadMessage()
		if err != nil {
			return err
		}

		event := estimator.Event{}
		if err := json.Unmarshal(message, &event); err != nil {
			return err
		}

		switch event.EventType {
		case estimator.ReadBlocksType:
			readBlocksEvent := estimator.ReadBlocksEvent{}
			if err := json.Unmarshal(message, &readBlocksEvent); err != nil {
				log.Dbg("failed to read blocks event: ", err)
				break
			}

			profiler.SetReadBlocks(readBlocksEvent.ReadBlocks)
		}

		log.Dbg("received unknown message: ", event.EventType, string(message))
	}

	return nil
}

func (s *Server) startObservation(w http.ResponseWriter, r *http.Request) {
	if s.Platform.Client == nil {
		sendBadRequestError(w, r, "cannot start the session observation because a Platform client is not configured")
		return
	}

	var observationRequest *types.StartObservationRequest
	if err := readJSON(r, &observationRequest); err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	clone, err := s.Cloning.GetClone(observationRequest.CloneID)
	if err != nil {
		sendNotFoundError(w, r)
		return
	}

	observingClone := observer.NewObservingClone(observationRequest.Config)
	startedAt := time.Now().Round(time.Millisecond)

	port, err := strconv.Atoi(clone.DB.Port)
	if err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	s.Observer.AddObservingClone(clone.ID, uint(port), observingClone)

	// Start session on the Platform.
	platformRequest := platform.StartObservationRequest{
		InstanceID: "", // TODO(akartasov): get InstanceID.
		CloneID:    clone.ID,
		StartedAt:  startedAt.Format("2006-01-02 15:04:05 UTC"),
		Config:     observingClone.Config(),
		Tags:       observationRequest.Tags,
	}

	platformResponse, err := s.Platform.Client.StartObservationSession(context.Background(), platformRequest)
	if err != nil {
		sendBadRequestError(w, r, "Failed to start observation session on the Platform")
		return
	}

	if observationRequest.DBName != "" {
		clone.DB.DBName = observationRequest.DBName
	}

	if err := observingClone.Init(clone, platformResponse.SessionID, startedAt, observationRequest.Tags); err != nil {
		sendError(w, r, errors.Wrap(err, "failed to init observing session"))
		return
	}

	go func() {
		if err := observingClone.RunSession(); err != nil {
			// TODO(akartasov): Update observation (add a request to Platform) with an error.
			log.Err("failed to observe clone: ", err)
		}
	}()

	if err := writeJSON(w, http.StatusOK, observingClone.Session()); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) stopObservation(w http.ResponseWriter, r *http.Request) {
	if s.Platform.Client == nil {
		sendBadRequestError(w, r, "cannot stop the session observation because a Platform client is not configured")
		return
	}

	var observationRequest *types.StopObservationRequest

	if err := readJSON(r, &observationRequest); err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	observingClone, err := s.Observer.GetObservingClone(observationRequest.CloneID)
	if err != nil {
		sendNotFoundError(w, r)
		return
	}

	clone, err := s.Cloning.GetClone(observationRequest.CloneID)
	if err != nil {
		sendNotFoundError(w, r)
		return
	}

	if err := s.Cloning.UpdateCloneStatus(observationRequest.CloneID, models.Status{Code: models.StatusExporting}); err != nil {
		sendNotFoundError(w, r)
		return
	}

	defer func() {
		if err := s.Cloning.UpdateCloneStatus(observationRequest.CloneID, models.Status{Code: models.StatusOK}); err != nil {
			log.Err("failed to update clone status", err)
		}
	}()

	if err := observingClone.Stop(); err != nil {
		sendBadRequestError(w, r, err.Error())
		return
	}

	session := observingClone.Session()
	if session == nil || session.Result == nil {
		sendBadRequestError(w, r, "observing session has not been initialized")
		return
	}

	platformRequest := platform.StopObservationRequest{
		SessionID:  session.SessionID,
		FinishedAt: session.FinishedAt.Format("2006-01-02 15:04:05 UTC"),
		Result:     *session.Result,
	}

	if _, err := s.Platform.Client.StopObservationSession(context.Background(), platformRequest); err != nil {
		sendBadRequestError(w, r, "Failed to start observation session on the Platform")
		return
	}

	port, err := strconv.Atoi(clone.DB.Port)
	if err != nil {
		sendError(w, r, errors.Wrap(err, "failed to parse clone port"))
		return
	}

	logs, err := s.Observer.GetCloneLog(context.TODO(), uint(port), observingClone)
	if err != nil {
		log.Err("Failed to get observation logs", err)
	}

	if len(logs) > 0 {
		headers := map[string]string{
			"Prefer":            "params=multiple-objects",
			"Content-Type":      "text/csv",
			"X-PGAI-Session-ID": strconv.FormatUint(session.SessionID, 10),
			"X-PGAI-Part":       "1", // TODO (akartasov): Support chunks.
		}

		if err := s.Platform.Client.UploadObservationLogs(context.Background(), logs, headers); err != nil {
			log.Err("Failed to upload observation logs", err)
		}
	}

	if err := writeJSON(w, http.StatusOK, session); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) sessionSummaryObservation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	cloneID := vars["clone_id"]

	sessionID, err := strconv.ParseUint(vars["session_id"], 10, 64)
	if err != nil {
		sendBadRequestError(w, r, fmt.Sprintf("invalid session_id: %v", sessionID))
		return
	}

	observingClone, err := s.Observer.GetObservingClone(cloneID)
	if err != nil || !observingClone.IsExistArtifacts(sessionID) {
		sendNotFoundError(w, r)
		return
	}

	summaryData, err := observingClone.ReadSummary(sessionID)
	if err != nil {
		sendBadRequestError(w, r, fmt.Sprintf("failed to read summary: %v", err))
		return
	}

	if err := writeData(w, http.StatusOK, summaryData); err != nil {
		sendError(w, r, err)
		return
	}
}

func (s *Server) downloadArtifact(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	artifactType := values.Get("artifact_type")

	if !observer.IsAvailableArtifactType(artifactType) {
		sendBadRequestError(w, r, fmt.Sprintf("artifact %q is not available to download", artifactType))
		return
	}

	sessionID, err := strconv.ParseUint(values.Get("session_id"), 10, 64)
	if err != nil {
		sendBadRequestError(w, r, fmt.Sprintf("invalid session_id: %v", sessionID))
		return
	}

	cloneID := values.Get("clone_id")

	observingClone, err := s.Observer.GetObservingClone(cloneID)
	if err != nil || !observingClone.IsExistArtifacts(sessionID) {
		sendNotFoundError(w, r)
		return
	}

	filePath := observingClone.BuildArtifactPath(sessionID, artifactType)
	http.ServeFile(w, r, filePath)
}

// healthCheck provides a health check handler.
func (s *Server) healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	healthResponse := models.Health{
		Version: version.GetVersion(),
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}
}
