/*
2019 © Postgres.ai
*/

// Package srv contains API routes and handlers.
package srv

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/embeddedui"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	srvCfg "gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/mw"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/ws"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/validator"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

// Server defines an HTTP server of the Database Lab.
type Server struct {
	validator   validator.Service
	Cloning     *cloning.Base
	provisioner *provision.Provisioner
	Config      *srvCfg.Config
	Global      *global.Config
	engProps    global.EngineProps
	Retrieval   *retrieval.Retrieval
	Platform    *platform.Service
	Observer    *observer.Observer
	wsService   WSService
	httpSrv     *http.Server
	docker      *client.Client
	pm          *pool.Manager
	tm          *telemetry.Agent
	startedAt   *models.LocalTime
	filtering   *log.Filtering
	reloadFn    func(server *Server) error
}

// WSService defines a service to manage web-sockets.
type WSService struct {
	upgrader    websocket.Upgrader
	uiManager   *embeddedui.UIManager
	tokenKeeper *ws.TokenKeeper
}

// NewServer initializes a new Server instance with provided configuration.
func NewServer(cfg *srvCfg.Config, globalCfg *global.Config, engineProps global.EngineProps,
	dockerClient *client.Client, cloning *cloning.Base, provisioner *provision.Provisioner,
	retrievalSvc *retrieval.Retrieval, platform *platform.Service, observer *observer.Observer,
	pm *pool.Manager, tm *telemetry.Agent, tokenKeeper *ws.TokenKeeper,
	filtering *log.Filtering, uiManager *embeddedui.UIManager, reloadConfigFn func(server *Server) error) *Server {
	server := &Server{
		Config:      cfg,
		Global:      globalCfg,
		engProps:    engineProps,
		Cloning:     cloning,
		provisioner: provisioner,
		Retrieval:   retrievalSvc,
		Platform:    platform,
		Observer:    observer,
		wsService: WSService{
			upgrader:    websocket.Upgrader{},
			tokenKeeper: tokenKeeper,
			uiManager:   uiManager,
		},
		docker:    dockerClient,
		pm:        pm,
		tm:        tm,
		filtering: filtering,
		startedAt: &models.LocalTime{Time: time.Now().Truncate(time.Second)},
		reloadFn:  reloadConfigFn,
	}

	return server
}

func (s *Server) instanceStatus() *models.InstanceStatus {
	instanceStatus := &models.InstanceStatus{
		Status: &models.Status{
			Code:    models.StatusOK,
			Message: models.InstanceMessageOK,
		},
		Engine: models.Engine{
			Version:                   version.GetVersion(),
			Edition:                   s.engProps.GetEdition(),
			StartedAt:                 s.startedAt,
			Telemetry:                 pointer.ToBool(s.tm.IsEnabled()),
			DisableConfigModification: pointer.ToBool(s.Config.DisableConfigModification),
		},
		Pools:       s.provisioner.GetPoolEntryList(),
		Cloning:     s.Cloning.GetCloningState(),
		Provisioner: s.provisioner.ContainerOptions(),
		Retrieving: models.Retrieving{
			Mode:        s.Retrieval.State.Mode,
			Status:      s.Retrieval.State.Status,
			Alerts:      s.Retrieval.State.Alerts(),
			LastRefresh: s.Retrieval.State.LastRefresh,
		},
	}

	if s.Retrieval.Scheduler.Spec != nil {
		instanceStatus.Retrieving.NextRefresh = models.NewLocalTime(s.Retrieval.Scheduler.Spec.Next(time.Now()))
	}

	s.summarizeStatus(instanceStatus)

	sync, err := s.Retrieval.ReportSyncStatus(context.Background())
	if err != nil {
		log.Warn(err)
	} else {
		instanceStatus.Synchronization = sync
	}

	return instanceStatus
}

func (s *Server) summarizeStatus(instance *models.InstanceStatus) {
	subsystems := []string{}
	if instance.Retrieving.Status == models.Failed {
		subsystems = append(subsystems, "retrieving")
	}

	if len(subsystems) > 0 {
		instance.Status = &models.Status{
			Code:    models.StatusWarning,
			Message: fmt.Sprintf("%s: %s", models.InstanceMessageWarning, strings.Join(subsystems, ", ")),
		}
	}
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
	r.HandleFunc("/observation/start", authMW.Authorized(s.startObservation)).Methods(http.MethodPost)
	r.HandleFunc("/observation/stop", authMW.Authorized(s.stopObservation)).Methods(http.MethodPost)
	r.HandleFunc("/observation/summary/{clone_id}/{session_id}", authMW.Authorized(s.sessionSummaryObservation)).Methods(http.MethodGet)
	r.HandleFunc("/observation/download", authMW.Authorized(s.downloadArtifact)).Methods(http.MethodGet)
	r.HandleFunc("/instance/retrieval", authMW.Authorized(s.retrievalState)).Methods(http.MethodGet)

	// Sub-route /admin
	adminR := r.PathPrefix("/admin").Subrouter()
	adminR.Use(authMW.AdminMW)
	adminR.HandleFunc("/ws-auth", s.websocketAuth).Methods(http.MethodGet)
	adminR.HandleFunc("/config", s.getProjectedAdminConfig).Methods(http.MethodGet)
	adminR.HandleFunc("/config.yaml", s.getAdminConfigYaml).Methods(http.MethodGet)
	adminR.HandleFunc("/config", s.setProjectedAdminConfig).Methods(http.MethodPost)
	adminR.HandleFunc("/test-db-source", s.testDBSource).Methods(http.MethodPost)

	r.HandleFunc("/instance/logs", authMW.WebSocketsMW(s.wsService.tokenKeeper, s.instanceLogs))

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
	reportLaunching(s.Config)
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Msg("Server shutting down...")
	return s.httpSrv.Shutdown(ctx)
}

// Uptime returns the server uptime.
func (s *Server) Uptime() float64 {
	instanceStatus := s.instanceStatus()
	if instanceStatus.Engine.StartedAt != nil && !instanceStatus.Engine.StartedAt.IsZero() {
		return time.Since(instanceStatus.Engine.StartedAt.Time).Truncate(time.Second).Seconds()
	}

	return 0
}

// reportLaunching reports the launch of the HTTP server.
func reportLaunching(cfg *srvCfg.Config) {
	log.Msg(fmt.Sprintf("API server started listening on %s:%d.", cfg.Host, cfg.Port))
}

func (s *Server) initLogRegExp() {
	s.filtering.ReloadLogRegExp([]string{s.Config.VerificationToken, s.Platform.AccessToken()})
}
