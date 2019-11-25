package srv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"../log"
	m "../models"

	"github.com/gorilla/mux"
	"github.com/rs/xid"
)

func startClone(w http.ResponseWriter, r *http.Request) {
	var newClone m.Clone
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Dbg(w, "Start clone error:", err)
	}

	// TODO(anatoly): Create clone.

	json.Unmarshal(reqBody, &newClone)

	newClone.Id = xid.New().String()
	clones = append(clones, &newClone)

	w.WriteHeader(http.StatusCreated)
	b, err := json.MarshalIndent(newClone, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}

func updateClone(w http.ResponseWriter, r *http.Request) {
	// TODO(anatoly): Update fields:
	// - Protected
}

func getClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	clone, _, ok := findClone(cloneId)
	if !ok {
		failNotFound(w, r)
		log.Dbg(fmt.Sprintf("The clone with ID %s was not found", cloneId))
		return
	}

	b, err := json.MarshalIndent(clone, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}

func resetClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	_, _, ok := findClone(cloneId)
	if !ok {
		failNotFound(w, r)
		log.Dbg(fmt.Sprintf("The clone with ID %s was not found", cloneId))
	}

	// TODO(anatoly): Reset clone.
	log.Dbg(fmt.Sprintf("The clone with ID %s has been reset successfully", cloneId))
}

func stopClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	_, ind, ok := findClone(cloneId)
	if !ok {
		failNotFound(w, r)
		log.Dbg(fmt.Sprintf("The clone with ID %s was not found", cloneId))
		return
	}

	//TODO(anatoly): Stop clone.
	clones = append(clones[:ind], clones[ind+1:]...)
	log.Dbg(fmt.Sprintf("The clone with ID %s has been deleted successfully", cloneId))
}

func getInstanceStatus(w http.ResponseWriter, r *http.Request) {
	instanceStatus.Clones = clones
	b, err := json.MarshalIndent(instanceStatus, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}

func getSnapshots(w http.ResponseWriter, r *http.Request) {
	b, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		log.Err(err)
	}
	w.Write(b)
}
