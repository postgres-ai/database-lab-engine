/*
2019 © Postgres.ai
*/

// Package srv contains API routes and handlers.
package srv

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/estimator"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/validator"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/srv/api"
	srvCfg "gitlab.com/postgres-ai/database-lab/v2/pkg/srv/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/srv/mw"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/telemetry"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

// Server defines an HTTP server of the Database Lab.
type Server struct {
	validator validator.Service
	Cloning   *cloning.Base
	Config    *srvCfg.Config
	Global    *global.Config
	engProps  global.EngineProps
	Retrieval *retrieval.Retrieval
	Platform  *platform.Service
	Observer  *observer.Observer
	Estimator *estimator.Estimator
	upgrader  websocket.Upgrader
	httpSrv   *http.Server
	docker    *client.Client
	pm        *pool.Manager
	tm        *telemetry.Agent
	startedAt time.Time
}

// NewServer initializes a new Server instance with provided configuration.
func NewServer(cfg *srvCfg.Config, globalCfg *global.Config, engineProps global.EngineProps, cloning *cloning.Base,
	retrievalSvc *retrieval.Retrieval, platform *platform.Service, dockerClient *client.Client, observer *observer.Observer,
	estimator *estimator.Estimator, pm *pool.Manager, tm *telemetry.Agent) *Server {
	server := &Server{
		Config:    cfg,
		Global:    globalCfg,
		engProps:  engineProps,
		Cloning:   cloning,
		Retrieval: retrievalSvc,
		Platform:  platform,
		Observer:  observer,
		Estimator: estimator,
		upgrader:  websocket.Upgrader{},
		docker:    dockerClient,
		pm:        pm,
		tm:        tm,
		startedAt: time.Now().Truncate(time.Second),
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
func (s *Server) Reload(cfg srvCfg.Config) {
	*s.Config = cfg
}

// InitHandlers initializes handler functions of the HTTP server.
func (s *Server) InitHandlers() {
	r := mux.NewRouter().StrictSlash(true)

	authMW := mw.NewAuth(s.Config.VerificationToken, s.Platform)

	r.HandleFunc("/status", authMW.Authorized(s.getInstanceStatus)).Methods(http.MethodGet)
	r.HandleFunc("/snapshots", authMW.Authorized(s.getSnapshots)).Methods(http.MethodGet)
	r.HandleFunc("/clone", authMW.Authorized(s.createClone)).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}", authMW.Authorized(s.destroyClone)).Methods(http.MethodDelete)
	r.HandleFunc("/clone/{id}", authMW.Authorized(s.patchClone)).Methods(http.MethodPatch)
	r.HandleFunc("/clone/{id}", authMW.Authorized(s.getClone)).Methods(http.MethodGet)
	r.HandleFunc("/clone/{id}/reset", authMW.Authorized(s.resetClone)).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}", authMW.Authorized(s.getClone)).Methods(http.MethodGet)
	r.HandleFunc("/observation/start", authMW.Authorized(s.startObservation)).Methods(http.MethodPost)
	r.HandleFunc("/observation/stop", authMW.Authorized(s.stopObservation)).Methods(http.MethodPost)
	r.HandleFunc("/observation/summary/{clone_id}/{session_id}", authMW.Authorized(s.sessionSummaryObservation)).Methods(http.MethodGet)
	r.HandleFunc("/observation/download", authMW.Authorized(s.downloadArtifact)).Methods(http.MethodGet)
	r.HandleFunc("/estimate", s.startEstimator).Methods(http.MethodGet)

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
	r.NotFoundHandler = http.HandlerFunc(api.SendNotFoundError)

	s.httpSrv = &http.Server{Addr: fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port), Handler: mw.Logging(r)}
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

// Uptime returns the server uptime.
func (s *Server) Uptime() float64 {
	return time.Since(s.startedAt).Truncate(time.Second).Seconds()
}
