package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (s *Server) getInstanceStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := s.Cloning.GetInstanceState()
		if err != nil {
			failInternalServer(w, r, err)
			return
		}

		if err = writeJSON(w, http.StatusOK, status); err != nil {
			failInternalServer(w, r, err)
			return
		}
	}
}

func (s *Server) getSnapshots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := s.Cloning.GetSnapshots()
		if err != nil {
			failInternalServer(w, r, err)
			return
		}

		if err = writeJSON(w, http.StatusOK, snapshots); err != nil {
			failInternalServer(w, r, err)
			return
		}
	}
}

func (s *Server) createClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newClone models.Clone
		if err := readJSON(r, &newClone); err != nil {
			log.Err(err)
			failBadRequest(w, r)

			return
		}

		if err := s.Cloning.CreateClone(&newClone); err != nil {
			failInternalServer(w, r, errors.Wrap(err, "failed to create clone"))
			return
		}

		if err := writeJSON(w, http.StatusCreated, newClone); err != nil {
			failInternalServer(w, r, err)
			return
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being created", newClone.ID))
	}
}

func (s *Server) destroyClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		if err := s.Cloning.DestroyClone(cloneID); err != nil {
			// TODO(anatoly): Not found case.
			failInternalServer(w, r, errors.Wrap(err, "failed to destroy clone"))
			return
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneID))
	}
}

func (s *Server) patchClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		var patchClone *models.Clone
		if err := readJSON(r, &patchClone); err != nil {
			log.Err(err)
			failBadRequest(w, r)

			return
		}

		if err := s.Cloning.UpdateClone(cloneID, patchClone); err != nil {
			failInternalServer(w, r, errors.Wrap(err, "failed to update clone"))
			return
		}
	}
}

func (s *Server) getClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		clone, err := s.Cloning.GetClone(cloneID)
		if err != nil {
			log.Errf("failed to get clone: %+v", err)
			failNotFound(w, r)

			return
		}

		if err := writeJSON(w, http.StatusOK, clone); err != nil {
			failInternalServer(w, r, err)
			return
		}
	}
}

func (s *Server) resetClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		if err := s.Cloning.ResetClone(cloneID); err != nil {
			failInternalServer(w, r, errors.Wrap(err, "failed to reset clone"))
			return
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being reset", cloneID))
	}
}

func getHelp(routes []Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := json.MarshalIndent(routes, "", "  ")
		if err != nil {
			log.Err(err)
			return
		}

		if _, err = w.Write(b); err != nil {
			log.Err(err)
			return
		}
	}
}
