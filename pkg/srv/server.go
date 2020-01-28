/*
2019 © Postgres.ai
*/

// Package srv contains API routes and handlers.
package srv

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/util"

	"github.com/gorilla/mux"
)

type Config struct {
	VerificationToken string `yaml:"verificationToken"`
	Port              uint   `yaml:"port"`
}

type Server struct {
	Config  *Config
	Cloning cloning.Cloning
}

type Route struct {
	Route   string   `json:"route"`
	Methods []string `json:"methods"`
}

// Initializes Server instance with provided configuration.
func NewServer(cfg *Config, cloning cloning.Cloning) *Server {
	// TODO(anatoly): Stop using mock data.
	server := &Server{
		Config:  cfg,
		Cloning: cloning,
	}

	return server
}

func attachSwaggerUI(r *mux.Router) error {
	swaggerUIPath, err := util.GetSwaggerUIPath()
	if err != nil {
		return errors.Wrap(err, "cannot find Swagger UI directory")
	}

	swaggerHandler := http.StripPrefix("/", http.FileServer(http.Dir(swaggerUIPath)))
	r.PathPrefix("/").Handler(swaggerHandler).Methods(http.MethodGet)

	return nil
}

func attachAPI(r *mux.Router) error {
	APIPath, err := util.GetAPIPath()
	if err != nil {
		return errors.Wrap(err, "cannot find API directory")
	}

	apiHandler := http.StripPrefix("/api/", http.FileServer(http.Dir(APIPath)))
	r.PathPrefix("/api/").Handler(apiHandler).Methods(http.MethodGet)

	return nil
}

// Run starts HTTP server on specified port in configuration.
func (s *Server) Run() error {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/status",
		s.authorized(s.getInstanceStatus())).Methods(http.MethodGet)
	r.HandleFunc("/snapshots",
		s.authorized(s.getSnapshots())).Methods(http.MethodGet)
	r.HandleFunc("/clone",
		s.authorized(s.createClone())).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.destroyClone())).Methods(http.MethodDelete)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.patchClone())).Methods(http.MethodPatch)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.getClone())).Methods(http.MethodGet)
	r.HandleFunc("/clone/{id}/reset",
		s.authorized(s.resetClone())).Methods(http.MethodPost)

	// Show Swagger UI on index page.
	if err := attachAPI(r); err != nil {
		log.Err(fmt.Sprintf("Cannot load API description."))
	}

	// Show Swagger UI on index page.
	if err := attachSwaggerUI(r); err != nil {
		log.Err(fmt.Sprintf("Cannot start Swagger UI."))
	}

	// Show not found error for all other possible routes.
	r.NotFoundHandler = http.HandlerFunc(failNotFound)

	// Start server.
	port := s.Config.Port
	log.Msg(fmt.Sprintf("Server started listening on localhost:%d.", port))
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), logging(r))

	return errors.WithMessage(err, "HTTP server error")
}
