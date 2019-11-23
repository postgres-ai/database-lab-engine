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

var clones = []m.Clone{
	{
		Id:          "xxx",
		Name:        "demo-clone-1",
		Project:     "demo",
		Snapshot:    "timestamp-latest",
		CloneSize:   1000,
		CloningTime: 10,
		Protected:   true,
		CreatedAt:   "timestamp",
		Status: m.Status{
			Code:    "OK",
			Message: "Clone is ready",
		},
		Db: m.Database{
			ConnStr:  "connstr",
			Host:     "host",
			Port:     "port",
			Username: "username",
		},
	},
	{
		Id:          "yyy",
		Name:        "demo-clone-2",
		Project:     "demo",
		Snapshot:    "timestamp-latest",
		CloneSize:   1000,
		CloningTime: 10,
		Protected:   true,
		CreatedAt:   "timestamp",
		Status: m.Status{
			Code:    "OK",
			Message: "Clone is ready",
		},
		Db: m.Database{
			ConnStr:  "connstr",
			Host:     "host",
			Port:     "port",
			Username: "username",
		},
	},
}

func startClone(w http.ResponseWriter, r *http.Request) {
	var newClone m.Clone
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// TODO(anatoly): Proper error with loging and status.
		fmt.Fprintf(w, "Kindly enter data with the event title and description only in order to update")
	}

	// TODO(anatoly): Create clone.

	json.Unmarshal(reqBody, &newClone)

	newClone.Id = xid.New().String()
	clones = append(clones, newClone)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newClone)
}

func getClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	for _, clone := range clones {
		if clone.Id == cloneId {
			json.NewEncoder(w).Encode(clone)
		}
	}

	// TODO(anatoly): Error: not found.
}

func resetClone(w http.ResponseWriter, r *http.Request) {
	// Exists?
}

func stopClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	for i, clone := range clones {
		if clone.Id == cloneId {
			clones = append(clones[:i], clones[i+1:]...)
			fmt.Fprintf(w, "The event with ID %v has been deleted successfully", cloneId)
		}
	}

	// TODO(anatoly): Error etc.
}

func getInstanceStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(clones)
}

func RunServer() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/status", getInstanceStatus).Methods("GET")
	router.HandleFunc("/clone", startClone).Methods("POST")
	router.HandleFunc("/clone/{id}/reset", resetClone).Methods("POST")
	router.HandleFunc("/clone/{id}", getClone).Methods("GET")
	router.HandleFunc("/clone/{id}", stopClone).Methods("DELETE")
	log.Fatal(http.ListenAndServe(":8080", router))

	port := 3000
	log.Msg(fmt.Sprintf("Server start listening on localhost:%d", port))
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.Err("HTTP server error:", err)
}
