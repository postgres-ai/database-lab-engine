/*
2021 © Postgres.ai
*/

// Package embeddedui manages embedded UI container.
package embeddedui

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/engine"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

const (
	// EnvEngineName defines the environment variable name to pass a DLE hostname to container.
	EnvEngineName = "DLE_HOST"

	// EnvEnginePort defines the environment variable name to pass a DLE port to container.
	EnvEnginePort = "DLE_PORT"

	// Health check timeout parameters.
	healthCheckInterval = 5 * time.Second
	healthCheckTimeout  = 10 * time.Second
	healthCheckRetries  = 5
)

// Config defines configs for a embedded UI container.
type Config struct {
	Enabled     bool   `yaml:"enabled"`
	DockerImage string `yaml:"dockerImage"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
}

// UIManager manages embedded UI container.
type UIManager struct {
	runner   runners.Runner
	docker   *client.Client
	cfgMu    sync.RWMutex
	cfg      Config
	engProps global.EngineProps
}

// New creates a new UI Manager.
func New(cfg Config, engProps global.EngineProps, runner runners.Runner, docker *client.Client) *UIManager {
	return &UIManager{runner: runner, docker: docker, cfg: cfg, engProps: engProps}
}

// Reload reloads configuration of UI manager and adjusts a UI container according to it.
func (ui *UIManager) Reload(ctx context.Context, cfg Config) error {
	ui.cfgMu.Lock()
	originalConfig := ui.cfg
	ui.cfg = cfg
	ui.cfgMu.Unlock()

	if !isConfigChanged(originalConfig, cfg) {
		return nil
	}

	if !cfg.Enabled {
		ui.Stop(ctx)
		return nil
	}

	if !originalConfig.Enabled {
		return ui.Run(ctx)
	}

	return ui.Restart(ctx)
}

// Config returns a snapshot of the UI container configuration. Reload swaps the whole value, so
// the copy stays consistent after the lock is released.
func (ui *UIManager) Config() Config {
	ui.cfgMu.RLock()
	defer ui.cfgMu.RUnlock()

	return ui.cfg
}

func isConfigChanged(oldCfg, newCfg Config) bool {
	return oldCfg.Enabled != newCfg.Enabled ||
		oldCfg.DockerImage != newCfg.DockerImage ||
		oldCfg.Host != newCfg.Host ||
		oldCfg.Port != newCfg.Port
}

// Run creates a new embedded UI container.
func (ui *UIManager) Run(ctx context.Context) error {
	cfg := ui.Config()

	if err := docker.PrepareImage(ctx, ui.docker, cfg.DockerImage); err != nil {
		return fmt.Errorf("failed to prepare Docker image: %w", err)
	}

	var hostIP netip.Addr

	if cfg.Host != "" {
		parsedIP, err := netip.ParseAddr(cfg.Host)
		if err != nil {
			return fmt.Errorf("invalid embedded UI host %q: %w", cfg.Host, err)
		}

		hostIP = parsedIP
	}

	containerID, err := tools.CreateContainerIfMissing(ctx, ui.docker, getEmbeddedUIName(ui.engProps.InstanceID),
		&container.Config{
			Labels: map[string]string{
				cont.DBLabSatelliteLabel:  cont.DBLabEmbeddedUILabel,
				cont.DBLabInstanceIDLabel: ui.engProps.InstanceID,
				cont.DBLabEngineNameLabel: ui.engProps.ContainerName,
			},
			Image: cfg.DockerImage,
			Env: []string{
				EnvEngineName + "=" + ui.engProps.ContainerName,
				EnvEnginePort + "=" + strconv.FormatUint(uint64(ui.engProps.EnginePort), 10),
			},
			Healthcheck: &container.HealthConfig{
				Interval: healthCheckInterval,
				Timeout:  healthCheckTimeout,
				Retries:  healthCheckRetries,
			},
		},
		&container.HostConfig{
			PortBindings: network.PortMap{
				network.MustParsePort("80/tcp"): {
					{
						HostIP:   hostIP,
						HostPort: strconv.Itoa(cfg.Port),
					},
				},
			},
		})

	if err != nil {
		return fmt.Errorf("failed to prepare Docker image for embedded UI: %w", err)
	}

	if err := networks.Connect(ctx, ui.docker, ui.engProps.InstanceID, containerID); err != nil {
		return fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	if _, err := ui.docker.ContainerStart(ctx, containerID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %q: %w", containerID, err)
	}

	reportLaunching(cfg)

	return nil
}

// Restart destroys and creates a new embedded UI container.
func (ui *UIManager) Restart(ctx context.Context) error {
	ui.Stop(ctx)

	if err := ui.Run(ctx); err != nil {
		return fmt.Errorf("failed to start UI container: %w", err)
	}

	return nil
}

// Stop removes an embedded UI container.
func (ui *UIManager) Stop(ctx context.Context) {
	tools.RemoveContainer(ctx, ui.docker, getEmbeddedUIName(ui.engProps.InstanceID), cont.StopTimeout)
}

func getEmbeddedUIName(instanceID string) string {
	return cont.DBLabEmbeddedUILabel + "_" + instanceID
}

// reportLaunching reports the launch of the embedded UI container.
func reportLaunching(cfg Config) {
	host := engine.DefaultListenerHost

	if cfg.Host != "" {
		host = cfg.Host
	}

	log.Msg(fmt.Sprintf("Embedded UI has started successfully on %s:%d.", host, cfg.Port))
}

// IsEnabled reports if the embedded UI container is enabled.
func (ui *UIManager) IsEnabled() bool {
	return ui.Config().Enabled
}

// GetHost provides a host name from the UI container configuration.
func (ui *UIManager) GetHost() string {
	return ui.Config().Host
}

// OriginURL reports the URL of the embedded UI container.
func (ui *UIManager) OriginURL() string {
	cfg := ui.Config()

	uiHost := cfg.Host
	if uiHost == "" || uiHost == engine.DefaultListenerHost {
		uiHost = engine.Localhost
	}

	localURL := url.URL{Scheme: engine.HTTPScheme, Host: net.JoinHostPort(uiHost, strconv.Itoa(cfg.Port))}

	return localURL.String()
}
