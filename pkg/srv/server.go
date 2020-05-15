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
	"gitlab.com/postgres-ai/database-lab/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/pkg/services/validator"
	"gitlab.com/postgres-ai/database-lab/pkg/util"

	"github.com/gorilla/mux"
)

// Config provides configuration for an HTTP server of the Database Lab.
type Config struct {
	VerificationToken string `yaml:"verificationToken"`
	Host              string `yaml:"host"`
	Port              uint   `yaml:"port"`
}

// Server defines an HTTP server of the Database Lab.
type Server struct {
	validator validator.Service
	Cloning   cloning.Cloning
	Config    *Config
	Platform  *platform.Service
}

// NewServer initializes a new Server instance with provided configuration.
func NewServer(cfg *Config, cloning cloning.Cloning, platform *platform.Service) *Server {
	// TODO(anatoly): Stop using mock data.
	server := &Server{
		Config:   cfg,
		Cloning:  cloning,
		Platform: platform,
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

	authMW := authMW{
		verificationToken:     s.Config.VerificationToken,
		personalTokenVerifier: s.Platform,
	}

	r.HandleFunc("/status", authMW.authorized(s.getInstanceStatus)).Methods(http.MethodGet)
	r.HandleFunc("/snapshots", authMW.authorized(s.getSnapshots)).Methods(http.MethodGet)
	r.HandleFunc("/clone", authMW.authorized(s.createClone)).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}", authMW.authorized(s.destroyClone)).Methods(http.MethodDelete)
	r.HandleFunc("/clone/{id}", authMW.authorized(s.patchClone)).Methods(http.MethodPatch)
	r.HandleFunc("/clone/{id}", authMW.authorized(s.getClone)).Methods(http.MethodGet)
	r.HandleFunc("/clone/{id}/reset", authMW.authorized(s.resetClone)).Methods(http.MethodPost)

	// Health check.
	r.HandleFunc("/healthz", s.healthCheck).Methods(http.MethodGet)

	// Show Swagger UI on index page.
	if err := attachAPI(r); err != nil {
		log.Err(fmt.Sprintf("Cannot load API description."))
	}

	// Show Swagger UI on index page.
	if err := attachSwaggerUI(r); err != nil {
		log.Err(fmt.Sprintf("Cannot start Swagger UI."))
	}

	// Show not found error for all other possible routes.
	r.NotFoundHandler = http.HandlerFunc(sendNotFoundError)

	// Start server.
	log.Msg(fmt.Sprintf("Server started listening on %s:%d.", s.Config.Host, s.Config.Port))
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port), logging(r))

	return errors.WithMessage(err, "HTTP server error")
}
