/*
2021 © Postgres.ai
*/

// Package localui manages local UI container.
package localui

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

	"gitlab.com/postgres-ai/database-lab/v2/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v2/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util/networks"
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

// Config defines configs for a local UI container.
type Config struct {
	Enabled     bool   `yaml:"enabled"`
	DockerImage string `yaml:"dockerImage"`
	Port        int    `yaml:"port"`
}

// UIManager manages local UI container.
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
		ui.cfg.Port != cfg.Port
}

// Run creates a new local UI container.
func (ui *UIManager) Run(ctx context.Context) error {
	if err := docker.PrepareImage(ui.runner, ui.cfg.DockerImage); err != nil {
		return fmt.Errorf("failed to prepare Docker image: %w", err)
	}

	localUI, err := ui.docker.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{
				cont.DBLabSatelliteLabel:  cont.DBLabLocalUILabel,
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
						HostIP:   "127.0.0.1",
						HostPort: strconv.Itoa(ui.cfg.Port),
					},
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		getLocalUIName(ui.engProps.InstanceID),
	)

	if err != nil {
		return fmt.Errorf("failed to prepare Docker image for LocalUI: %w", err)
	}

	if err := networks.Connect(ctx, ui.docker, ui.engProps.InstanceID, localUI.ID); err != nil {
		return fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	if err := ui.docker.ContainerStart(ctx, localUI.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %q: %w", localUI.ID, err)
	}

	return nil
}

// Restart destroys and creates a new local UI container.
func (ui *UIManager) Restart(ctx context.Context) error {
	ui.Stop(ctx)

	if err := ui.Run(ctx); err != nil {
		return fmt.Errorf("failed to start UI container: %w", err)
	}

	return nil
}

// Stop removes a local UI container.
func (ui *UIManager) Stop(ctx context.Context) {
	tools.RemoveContainer(ctx, ui.docker, getLocalUIName(ui.engProps.InstanceID), cont.StopTimeout)
}

func getLocalUIName(instanceID string) string {
	return cont.DBLabLocalUILabel + "_" + instanceID
}
