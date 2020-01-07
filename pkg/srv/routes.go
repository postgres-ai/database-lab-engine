package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/gorilla/mux"
)

func (s *Server) getInstanceStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := s.Cloning.GetInstanceState()
		if err != nil {
			failInternalServer(w, r, err.Error())
		}

		err = writeJSON(w, status)
		if err != nil {
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) getSnapshots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := s.Cloning.GetSnapshots()
		if err != nil {
			failInternalServer(w, r, err.Error())
		}

		err = writeJSON(w, snapshots)
		if err != nil {
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) createClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newClone models.Clone
		err := readJSON(r, &newClone)
		if err != nil {
			failBadRequest(w, r)
			return
		}

		err = s.Cloning.CreateClone(&newClone)
		if err != nil {
			log.Err("Create clone", err)
			// TODO(anatoly): Improve error processing.
			failInternalServer(w, r, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		err = writeJSON(w, newClone)
		if err != nil {
			log.Err(err)
			failInternalServer(w, r, err.Error())
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being created", newClone.ID))
	}
}

func (s *Server) destroyClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		err := s.Cloning.DestroyClone(cloneID)
		if err != nil {
			// TODO(anatoly): Not found case.
			failInternalServer(w, r, err.Error())
			return
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneID))
	}
}

func (s *Server) patchClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		var patchClone *models.Clone
		err := readJSON(r, &patchClone)
		if err != nil {
			failBadRequest(w, r)
			return
		}

		err = s.Cloning.UpdateClone(cloneID, patchClone)
		if err != nil {
			failInternalServer(w, r, err.Error())
			return
		}
	}
}

func (s *Server) getClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		clone, ok := s.Cloning.GetClone(cloneID)
		if !ok {
			failNotFound(w, r)
			return
		}

		err := writeJSON(w, clone)
		if err != nil {
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) resetClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneID := mux.Vars(r)["id"]

		err := s.Cloning.ResetClone(cloneID)
		if err != nil {
			failInternalServer(w, r, err.Error())
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
		}

		if _, err = w.Write(b); err != nil {
			log.Err(err)
		}
	}
}
