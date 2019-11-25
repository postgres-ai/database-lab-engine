package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"../log"
	m "../models"

	"github.com/gorilla/mux"
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

func NewServer(cfg Config) *Server {
	return &Server{
		Config: &cfg,
		Clones: clones,
	}
}

func findClone(cloneId string) (*m.Clone, int, bool) {
	for i, clone := range clones {
		if clone.Id == cloneId {
			return clone, i, true
		}
	}

	return &m.Clone{}, 0, false
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

func (s *Server) Run() error {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/status", s.authorized(getInstanceStatus)).Methods(http.MethodGet)
	router.HandleFunc("/snapshots", s.authorized(getSnapshots)).Methods(http.MethodGet)
	router.HandleFunc("/clone", s.authorized(startClone)).Methods(http.MethodPost)
	router.HandleFunc("/clone/{id}/reset", s.authorized(resetClone)).Methods(http.MethodPost)
	router.HandleFunc("/clone/{id}", s.authorized(getClone)).Methods(http.MethodGet)
	router.HandleFunc("/clone/{id}", s.authorized(updateClone)).Methods(http.MethodPatch)
	router.HandleFunc("/clone/{id}", s.authorized(stopClone)).Methods(http.MethodDelete)

	// Show available routes on index page.
	helpRoutes, err := getHelpRoutes(router)
	if err != nil {
		return err
	}
	router.HandleFunc("/", getHelp(helpRoutes))

	// Show not found error for all other possible routes.
	router.NotFoundHandler = http.HandlerFunc(failNotFound)

	// Set up global middlewares.
	router.Use(logging)

	// Start server.
	port := s.Config.Port
	log.Msg(fmt.Sprintf("Server start listening on localhost:%d", port))
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), router)
	log.Err("HTTP server error:", err)

	return err
}
