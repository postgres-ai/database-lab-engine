/*
2021 © Postgres.ai
*/

// Package embeddedui manages embedded UI container.
package embeddedui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

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
	cfg      Config
	engProps global.EngineProps
}

// New creates a new UI Manager.
func New(cfg Config, engProps global.EngineProps, runner runners.Runner, docker *client.Client) *UIManager {
	return &UIManager{runner: runner, docker: docker, cfg: cfg, engProps: engProps}
}

// Reload reloads configuration of UI manager and adjusts a UI container according to it.
func (ui *UIManager) Reload(ctx context.Context, cfg Config) error {
	originalConfig := ui.cfg
	ui.cfg = cfg

	if !ui.isConfigChanged(originalConfig) {
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

func (ui *UIManager) isConfigChanged(cfg Config) bool {
	return ui.cfg.Enabled != cfg.Enabled ||
		ui.cfg.DockerImage != cfg.DockerImage ||
		ui.cfg.Host != cfg.Host ||
		ui.cfg.Port != cfg.Port
}

// Run creates a new embedded UI container.
func (ui *UIManager) Run(ctx context.Context) error {
	if err := docker.PrepareImage(ui.runner, ui.cfg.DockerImage); err != nil {
		return fmt.Errorf("failed to prepare Docker image: %w", err)
	}

	embeddedUI, err := ui.docker.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{
				cont.DBLabSatelliteLabel:  cont.DBLabEmbeddedUILabel,
				cont.DBLabInstanceIDLabel: ui.engProps.InstanceID,
				cont.DBLabEngineNameLabel: ui.engProps.ContainerName,
			},
			Image: ui.cfg.DockerImage,
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
			PortBindings: map[nat.Port][]nat.PortBinding{
				"80/tcp": {
					{
						HostIP:   ui.cfg.Host,
						HostPort: strconv.Itoa(ui.cfg.Port),
					},
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		getEmbeddedUIName(ui.engProps.InstanceID),
	)

	if err != nil {
		return fmt.Errorf("failed to prepare Docker image for embedded UI: %w", err)
	}

	if err := networks.Connect(ctx, ui.docker, ui.engProps.InstanceID, embeddedUI.ID); err != nil {
		return fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	if err := ui.docker.ContainerStart(ctx, embeddedUI.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %q: %w", embeddedUI.ID, err)
	}

	reportLaunching(ui.cfg)

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

// Stop removes a embedded UI container.
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
