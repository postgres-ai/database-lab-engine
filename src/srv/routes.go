package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"../log"
	m "../models"

	"github.com/gorilla/mux"
)

func (s *Server) getInstanceStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := s.Cloning.GetInstanceState()

		_, err = writeJson(w, status)
		if err != nil {
			failInternalServer(w, r)
		}
	}
}

func (s *Server) getSnapshots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots := s.Cloning.GetSnapshots()
		_, err := writeJson(w, snapshots)
		if err != nil {
			failInternalServer(w, r)
		}
	}
}

func (s *Server) startClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newClone m.Clone
		err := readJson(r, &newClone)
		if err != nil {
			failBadRequest(w, r)
			return
		}

		err = s.Cloning.CreateClone(&newClone)
		if err != nil {
			log.Err("Create clone", err)
			// TODO(anatoly): Improve error processing.
			failInternalServer(w, r)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, err = writeJson(w, newClone)
		if err != nil {
			log.Err(err)
			failInternalServer(w, r)
		}
	}
}

func (s *Server) resetClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		_, ok := s.Cloning.GetClone(cloneId)
		if !ok {
			failNotFound(w, r)
			return
		}

		// TODO(anatoly): Reset clone.

		log.Dbg(fmt.Sprintf("The clone with ID %s has been reset successfully", cloneId))
	}
}

func (s *Server) getClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		clone, ok := s.Cloning.GetClone(cloneId)
		if !ok {
			failNotFound(w, r)
			return
		}

		_, err := writeJson(w, clone)
		if err != nil {
			failInternalServer(w, r)
		}
	}
}

func (s *Server) patchClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO(anatoly): Update fields:
		// - Protected
	}
}

func (s *Server) stopClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		err := s.Cloning.DestroyClone(cloneId)
		if err != nil {
			// TODO(anatoly): Not found case.
			failInternalServer(w, r)
			return
		}

		log.Dbg(fmt.Sprintf("The clone with ID %s has been deleted successfully", cloneId))
	}
}

func getHelp(routes []Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := json.MarshalIndent(routes, "", "  ")
		if err != nil {
			log.Err(err)
		}
		w.Write(b)
	}
}
