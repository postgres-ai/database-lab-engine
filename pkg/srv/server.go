/*
2019 © Postgres.ai
*/

// Package srv contains API routes and handlers.
package srv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/observer"
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
	Observer  *observer.Observer
	httpSrv   *http.Server
}

// NewServer initializes a new Server instance with provided configuration.
func NewServer(cfg *Config, obsCfg *observer.Observer, cloning cloning.Cloning, platform *platform.Service,
	dockerClient *client.Client) *Server {
	// TODO(anatoly): Stop using mock data.
	server := &Server{
		Config:   cfg,
		Cloning:  cloning,
		Platform: platform,
		Observer: obsCfg,
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

// Reload reloads server configuration.
func (s *Server) Reload(cfg Config) {
	*s.Config = cfg
}

// InitHandlers initializes handler functions of the HTTP server.
func (s *Server) InitHandlers() {
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
	r.HandleFunc("/clone/{id}", authMW.authorized(s.getClone)).Methods(http.MethodGet)
	r.HandleFunc("/observation/start", authMW.authorized(s.startObservation)).Methods(http.MethodPost)
	r.HandleFunc("/observation/stop", authMW.authorized(s.stopObservation)).Methods(http.MethodPost)

	// Health check.
	r.HandleFunc("/healthz", s.healthCheck).Methods(http.MethodGet)

	// Show Swagger UI on index page.
	if err := attachAPI(r); err != nil {
		log.Err("Cannot load API description.")
	}

	// Show Swagger UI on index page.
	if err := attachSwaggerUI(r); err != nil {
		log.Err("Cannot start Swagger UI.")
	}

	// Show not found error for all other possible routes.
	r.NotFoundHandler = http.HandlerFunc(sendNotFoundError)

	s.httpSrv = &http.Server{Addr: fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port), Handler: logging(r)}
}

// Run starts HTTP server on specified port in configuration.
func (s *Server) Run() error {
	log.Msg(fmt.Sprintf("Server started listening on %s:%d.", s.Config.Host, s.Config.Port))
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Msg("Server shutting down...")
	return s.httpSrv.Shutdown(ctx)
}
