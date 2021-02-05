package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/platform"
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

	session := observer.NewSession(observationRequest.Config)
	session.StartedAt = time.Now().Round(time.Millisecond)
	session.Tags = observationRequest.Tags

	s.Observer.AddSession(clone.ID, session)

	// Start session on the Platform.
	platformRequest := platform.StartObservationRequest{
		InstanceID: "", // TODO(akartasov): get InstanceID.
		CloneID:    clone.ID,
		StartedAt:  session.StartedAt.Format("2006-01-02 15:04:05 UTC"),
		Config:     session.Config,
		Tags:       observationRequest.Tags,
	}

	platformResponse, err := s.Platform.Client.StartObservationSession(context.Background(), platformRequest)
	if err != nil {
		sendBadRequestError(w, r, "Failed to start observation session on the Platform")
		return
	}

	session.SessionID = platformResponse.SessionID

	go func() {
		if err := session.Start(clone); err != nil {
			log.Err("failed to observe clone: ", err)
			// TODO(akartasov): Update observation (add a request to Platform) with an error.
			s.Observer.RemoveSession(clone.ID)
		}
	}()

	if err := writeJSON(w, http.StatusOK, session); err != nil {
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

	session, err := s.Observer.GetSession(observationRequest.CloneID)
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

	defer s.Observer.RemoveSession(observationRequest.CloneID)

	defer func() {
		if err := s.Cloning.UpdateCloneStatus(observationRequest.CloneID, models.Status{Code: models.StatusOK}); err != nil {
			log.Err("failed to update clone status", err)
		}
	}()

	session.Stop()

	platformRequest := platform.StopObservationRequest{
		SessionID:  session.SessionID,
		FinishedAt: session.FinishedAt.Format("2006-01-02 15:04:05 UTC"),
		Result:     session.ObservationResult,
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

	logs, err := s.Observer.GetCloneLog(context.TODO(), uint(port), session)
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

	if err := writeJSON(w, http.StatusOK, session.ObservationResult); err != nil {
		sendError(w, r, err)
		return
	}
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
