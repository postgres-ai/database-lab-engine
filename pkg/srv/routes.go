package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/version"
)

// HealthResponse represents a response for heath-check requests.
type HealthResponse struct {
	Version string `json:"version"`
}

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

// healthCheck provides a health check handler.
func (s *Server) healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	healthResponse := HealthResponse{
		Version: version.GetVersion(),
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}
}
