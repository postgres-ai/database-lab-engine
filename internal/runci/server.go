/*
2021 © Postgres.ai
*/

package runci

import (
	"context"
	"fmt"
	"net/http"

	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"gitlab.com/postgres-ai/database-lab/v3/internal/runci/source"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/mw"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// Server defines an HTTP server of the CI Migration Checker.
type Server struct {
	config       *Config
	dle          *dblabapi.Client
	codeProvider source.Provider
	platform     *platform.Service
	upgrader     websocket.Upgrader
	httpServer   *http.Server
	docker       *client.Client
	networkID    string
}

// NewServer initializes a new runner Server instance.
func NewServer(cfg *Config, dle *dblabapi.Client, platform *platform.Service, code source.Provider, docker *client.Client,
	networkID string) *Server {
	server := &Server{
		config:       cfg,
		dle:          dle,
		platform:     platform,
		codeProvider: code,
		upgrader:     websocket.Upgrader{},
		docker:       docker,
		networkID:    networkID,
	}

	return server
}

// Run starts HTTP server on specified port in configuration.
func (s *Server) Run() error {
	r := mux.NewRouter().StrictSlash(true)

	authMW := mw.NewAuth(s.config.App.VerificationToken, s.platform)

	r.HandleFunc("/migration/run", authMW.Authorized(s.runMigration)).Methods(http.MethodPost)
	r.HandleFunc("/artifact/download", authMW.Authorized(s.downloadArtifact)).Methods(http.MethodGet)
	r.HandleFunc("/artifact/stop", authMW.Authorized(s.destroyClone)).Methods(http.MethodGet)
	r.HandleFunc("/healthz", s.healthCheck).Methods(http.MethodGet)

	addr := fmt.Sprintf("%s:%d", s.config.App.Host, s.config.App.Port)

	s.httpServer = &http.Server{Addr: addr, Handler: mw.Logging(r)}

	log.Msg(fmt.Sprintf("Server started listening on %s...", addr))

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Msg("Server shutting down...")
	return s.httpServer.Shutdown(ctx)
}
