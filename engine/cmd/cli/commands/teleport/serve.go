/*
2026 © Postgres.ai
*/

package teleport

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	reconcileInterval   = 5 * time.Minute
	shutdownTimeout     = 10 * time.Second
	httpReadTimeout     = 10 * time.Second
	httpWriteTimeout    = 10 * time.Second
	httpIdleTimeout     = 60 * time.Second
	editionCheckTimeout = 10 * time.Second
	maxRequestBodySize  = 1 << 20 // 1 MiB
)

// Config holds configuration for the teleport sidecar.
type Config struct {
	ListenAddr       string
	EnvironmentID    string
	TeleportProxy    string
	TeleportIdentity string
	TctlPath         string
	DblabURL         string
	DblabToken       string
	WebhookSecret    string
	Labels           map[string]string
}

// service holds runtime state for the teleport sidecar.
type service struct {
	cfg      *Config
	dbClient *dblabapi.Client
	// mu serialises calls to createDB/removeDB so that concurrent webhook
	// requests and the periodic reconcile loop do not race on tctl operations.
	mu sync.Mutex
}

// WebhookPayload defines the incoming webhook body from DBLab Engine.
type WebhookPayload struct {
	EventType     string `json:"event_type"`
	EntityID      string `json:"entity_id"`
	Host          string `json:"host,omitempty"`
	Port          uint   `json:"port,omitempty"`
	Username      string `json:"username,omitempty"`
	DBName        string `json:"dbname,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
}

// CommandList returns available commands for the teleport sidecar.
func CommandList() []*cli.Command {
	return []*cli.Command{{
		Name:  "teleport",
		Usage: "manage Teleport integration",
		Subcommands: []*cli.Command{
			{
				Name:   "serve",
				Usage:  "run webhook sidecar to sync DBLab clones with Teleport",
				Action: serveAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "listen-addr",
						Usage: "HTTP listen address",
						Value: "localhost:9876",
					},
					&cli.StringFlag{
						Name:     "environment-id",
						Usage:    "environment ID for Teleport naming",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "teleport-proxy",
						Usage:    "Teleport proxy address",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "teleport-identity",
						Usage:    "path to tctl identity file",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "tctl-path",
						Usage: "path to tctl binary",
						Value: "tctl",
					},
					&cli.StringFlag{
						Name:  "dblab-url",
						Usage: "DBLab API URL",
						Value: "http://localhost:2345",
					},
					&cli.StringFlag{
						Name:     "dblab-token",
						Usage:    "DBLab verification token",
						Required: true,
						EnvVars:  []string{"DBLAB_TOKEN"},
					},
					&cli.StringFlag{
						Name:     "webhook-secret",
						Usage:    "shared secret that DBLab Engine sends in the DBLab-Webhook-Token header",
						Required: true,
						EnvVars:  []string{"WEBHOOK_SECRET"},
					},
					&cli.StringSliceFlag{
						Name:    "label",
						Usage:   "additional Teleport resource label in key=value form (repeatable)",
						EnvVars: []string{"TELEPORT_LABELS"},
					},
				},
			},
		},
	}}
}

// parseLabels converts repeated key=value flag entries into a label map,
// rejecting malformed entries and keys reserved by the sidecar.
func parseLabels(entries []string) (map[string]string, error) {
	labels := make(map[string]string, len(entries))

	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == "" || value == "" {
			return nil, fmt.Errorf("invalid label %q: expected non-empty key=value", entry)
		}

		if _, dup := labels[key]; dup {
			return nil, fmt.Errorf("duplicate label %q", key)
		}

		if reservedLabels[key] {
			return nil, fmt.Errorf("label %q is reserved and managed by the sidecar", key)
		}

		if _, err := sanitizeLabelKey(key); err != nil {
			return nil, err
		}

		if _, err := sanitizeYAMLValue(value, key); err != nil {
			return nil, err
		}

		labels[key] = value
	}

	return labels, nil
}

func serveAction(c *cli.Context) error {
	labels, err := parseLabels(c.StringSlice("label"))
	if err != nil {
		return err
	}

	cfg := &Config{
		ListenAddr:       c.String("listen-addr"),
		EnvironmentID:    c.String("environment-id"),
		TeleportProxy:    c.String("teleport-proxy"),
		TeleportIdentity: c.String("teleport-identity"),
		TctlPath:         c.String("tctl-path"),
		DblabURL:         c.String("dblab-url"),
		DblabToken:       c.String("dblab-token"),
		WebhookSecret:    c.String("webhook-secret"),
		Labels:           labels,
	}

	if cfg.WebhookSecret == "" {
		return fmt.Errorf("webhook secret must not be empty")
	}

	dbClient, err := newDblabClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create DBLab client: %w", err)
	}

	if err := checkEdition(dbClient); err != nil {
		return err
	}

	svc := &service{cfg: cfg, dbClient: dbClient}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/teleport-sync", svc.makeWebhookHandler())

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}

	go svc.runReconcileLoop(ctx)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Errf("server shutdown error: %v", err)
		}
	}()

	log.Msg(fmt.Sprintf("teleport sidecar listening on %s", cfg.ListenAddr))

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *service) runReconcileLoop(ctx context.Context) {
	s.reconcile(ctx)

	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

func (s *service) makeWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if subtle.ConstantTimeCompare([]byte(r.Header.Get("DBLab-Webhook-Token")), []byte(s.cfg.WebhookSecret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var payload WebhookPayload
		if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&payload); err != nil {
			log.Errf("webhook: failed to decode payload: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		var handleErr error

		// Use a detached context so that tctl operations are not cancelled
		// if the HTTP client disconnects before the command completes.
		// runTctl applies its own per-command timeout.
		ctx := context.Background()

		switch payload.EventType {
		case "clone_create":
			handleErr = s.handleCloneCreate(ctx, &payload)
		case "clone_delete":
			handleErr = s.handleCloneDelete(ctx, &payload)
		default:
			log.Dbg(fmt.Sprintf("webhook: unknown event_type %q, ignoring", payload.EventType))
		}

		if handleErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *service) handleCloneCreate(ctx context.Context, p *WebhookPayload) error {
	if p.EntityID == "" || p.Port == 0 {
		log.Errf("webhook clone_create: missing entity_id or port in payload")
		return fmt.Errorf("missing entity_id or port")
	}

	name := CloneServiceName(s.cfg.EnvironmentID, p.EntityID, int(p.Port))

	res := dbResource{Name: name, Port: int(p.Port), EnvID: s.cfg.EnvironmentID, CloneID: p.EntityID, Username: p.Username}

	s.mu.Lock()
	err := createDB(ctx, s.cfg, res)
	s.mu.Unlock()

	if err != nil {
		log.Errf("webhook clone_create: failed to register db %s: %v", name, err)
		return err
	}

	log.Msg(fmt.Sprintf("webhook clone_create: registered db %s", name))

	return nil
}

func (s *service) handleCloneDelete(ctx context.Context, p *WebhookPayload) error {
	if p.EntityID == "" || p.Port == 0 {
		log.Errf("webhook clone_delete: missing entity_id or port in payload")
		return fmt.Errorf("missing entity_id or port")
	}

	name := CloneServiceName(s.cfg.EnvironmentID, p.EntityID, int(p.Port))

	s.mu.Lock()
	err := removeDB(ctx, s.cfg, name)
	s.mu.Unlock()

	if err != nil {
		log.Errf("webhook clone_delete: failed to remove db %s: %v", name, err)
		return err
	}

	log.Msg(fmt.Sprintf("webhook clone_delete: removed db %s", name))

	return nil
}

// checkEdition verifies that the connected DBLab Engine instance is running
// Standard or Enterprise edition. Teleport integration is not available
// for Community edition.
func checkEdition(client *dblabapi.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), editionCheckTimeout)
	defer cancel()

	status, err := client.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to check DBLab instance status: %w", err)
	}

	if status.Engine.Edition == global.CommunityEdition {
		return fmt.Errorf("teleport integration requires Standard or Enterprise edition (current: %s)", status.Engine.Edition)
	}

	log.Msg(fmt.Sprintf("edition check passed: %s", status.Engine.Edition))

	return nil
}
