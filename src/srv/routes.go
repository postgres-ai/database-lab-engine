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
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) getSnapshots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots := s.Cloning.GetSnapshots()
		_, err := writeJson(w, snapshots)
		if err != nil {
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) createClone() http.HandlerFunc {
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
			failInternalServer(w, r, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, err = writeJson(w, newClone)
		if err != nil {
			log.Err(err)
			failInternalServer(w, r, err.Error())
		}
		log.Dbg(fmt.Sprintf("Clone ID=%s is being created", newClone.Id))
	}
}

func (s *Server) destroyClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		err := s.Cloning.DestroyClone(cloneId)
		if err != nil {
			// TODO(anatoly): Not found case.
			failInternalServer(w, r, err.Error())
			return
		}
		log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneId))
	}
}

func (s *Server) patchClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		var patchClone *m.Clone
		err := readJson(r, &patchClone)
		if err != nil {
			failBadRequest(w, r)
			return
		}

		err = s.Cloning.UpdateClone(cloneId, patchClone)
		if err != nil {
			failInternalServer(w, r, err.Error())
			return
		}
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
			failInternalServer(w, r, err.Error())
		}
	}
}

func (s *Server) resetClone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cloneId := mux.Vars(r)["id"]

		err := s.Cloning.ResetClone(cloneId)
		if err != nil {
			failInternalServer(w, r, err.Error())
			return
		}

		log.Dbg(fmt.Sprintf("Clone ID=%s is being reset", cloneId))
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
