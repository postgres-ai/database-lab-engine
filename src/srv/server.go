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

type Config struct {
	VerificationToken string
	Port              uint
}

type Server struct {
	Config *Config
	Clones []*m.Clone
}

type Route struct {
	Route   string   `json:"route"`
	Methods []string `json:"methods"`
}

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

var instanceStatus = m.InstanceStatus{
	Status: m.Status{
		Code:    "OK",
		Message: "Instance is ready",
	},
	Disk:                m.Disk{},
	ExpectedCloningTime: 5.0,
	NumClones:           2,
	Clones:              clones,
}

var snapshots = []m.Snapshot{
	{
		Id:        "xxx",
		Timestamp: "123",
	},
}

func startClone(w http.ResponseWriter, r *http.Request) {
	var newClone m.Clone
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Dbg(w, "Start clone error:", err)
	}

	// TODO(anatoly): Create clone.

	json.Unmarshal(reqBody, &newClone)

	newClone.Id = xid.New().String()
	clones = append(clones, newClone)

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
		notFound(w, r)
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
		notFound(w, r)
		log.Dbg(fmt.Sprintf("The clone with ID %s was not found", cloneId))
	}

	// TODO(anatoly): Reset clone.
	log.Dbg(fmt.Sprintf("The clone with ID %s has been reset successfully", cloneId))
}

func stopClone(w http.ResponseWriter, r *http.Request) {
	cloneId := mux.Vars(r)["id"]

	_, ind, ok := findClone(cloneId)
	if !ok {
		notFound(w, r)
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

func findClone(cloneId string) (m.Clone, int, bool) {
	for i, clone := range clones {
		if clone.Id == cloneId {
			return clone, i, true
		}
	}

	return m.Clone{}, 0, false
}

func getHelp(routes []Route) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := json.MarshalIndent(routes, "", "  ")
		if err != nil {
			log.Err(err)
		}
		w.Write(b)
	}
}

func getHelpRoutes(router *mux.Router) ([]Route, error) {
	routes := make([]Route, 0)
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		methods, err := route.GetMethods()
		if err != nil {
			return err
		}

		routes = append(routes, Route{
			Route:   pathTemplate,
			Methods: methods,
		})

		return nil
	})

	return routes, err
}

func RunServer() error {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/status", getInstanceStatus).Methods(http.MethodGet)
	router.HandleFunc("/snapshots", getSnapshots).Methods(http.MethodGet)
	router.HandleFunc("/clone", startClone).Methods(http.MethodPost)
	router.HandleFunc("/clone/{id}/reset", resetClone).Methods(http.MethodPost)
	router.HandleFunc("/clone/{id}", getClone).Methods(http.MethodGet)
	router.HandleFunc("/clone/{id}", updateClone).Methods(http.MethodPatch)
	router.HandleFunc("/clone/{id}", stopClone).Methods(http.MethodDelete)

	// Show available routes on index page.
	helpRoutes, err := getHelpRoutes(router)
	if err != nil {
		return err
	}
	router.HandleFunc("/", getHelp(helpRoutes))

	// Show not found error for all other possible routes.
	router.NotFoundHandler = http.HandlerFunc(notFound)

	// Set up global middlewares.
	router.Use(logging)

	// Start server.
	port := 3000
	log.Msg(fmt.Sprintf("Server start listening on localhost:%d", port))
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), router)
	log.Err("HTTP server error:", err)

	return err
}
