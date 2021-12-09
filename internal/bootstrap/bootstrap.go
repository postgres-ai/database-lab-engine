/*
2021 © Postgres.ai
*/

// Package bootstrap manages Database Lab Bootstrap component.
package bootstrap

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/engine"
)

const (
	engineHomeDir      = "/home/dblab"
	dockerSocket       = "/var/run/docker.sock"
	maxRetryStartCount = 3
)

// StartDLE starts Database Lab Engine container.
func StartDLE(ctx context.Context, docker *client.Client, cfg *config.Config) error {
	instanceID, err := config.LoadInstanceID(cfg.PoolManager.MountDir)
	if err != nil {
		return fmt.Errorf("failed to load instance ID: %w", err)
	}

	dleContainer, err := docker.ContainerCreate(ctx,
		buildEngineContainerConfig(instanceID, cfg),
		buildEngineHostConfig(cfg),
		&network.NetworkingConfig{},
		nil,
		cont.DBLabServerName)
	if err != nil {
		return fmt.Errorf("failed to create DLE container: %w", err)
	}

	if err = docker.ContainerStart(ctx, dleContainer.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start DLE container: %w", err)
	}

	return nil
}

func buildEngineContainerConfig(instanceID string, cfg *config.Config) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabServerLabel:     instanceID,
			cont.DBLabInstanceIDLabel: instanceID,
			cont.DBLabEngineNameLabel: cont.DBLabServerName,
		},
		Env:   cfg.Engine.Envs,
		Image: cfg.Engine.DockerImage,
	}
}

func buildEngineHostConfig(cfg *config.Config) *container.HostConfig {
	hostConfig := &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port(strconv.Itoa(cfg.Engine.Port) + "/tcp"): {
				{
					HostIP:   cfg.Engine.Host,
					HostPort: strconv.Itoa(cfg.Engine.Port),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: dockerSocket,
				Target: dockerSocket,
			},
			{
				Type:   mount.TypeBind,
				Source: cfg.PoolManager.MountDir,
				Target: cfg.PoolManager.MountDir,
				BindOptions: &mount.BindOptions{
					Propagation: mount.PropagationRShared,
				},
			},
			{
				Type:     mount.TypeBind,
				Source:   path.Join(cfg.Engine.StateDir, util.ConfigDir),
				Target:   path.Join(engineHomeDir, util.ConfigDir),
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: path.Join(cfg.Engine.StateDir, util.MetaDir),
				Target: path.Join(engineHomeDir, util.MetaDir),
			},
		},
		Privileged: true,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: maxRetryStartCount,
		},
	}

	hostConfig.Binds = append([]string{
		"/sys/kernel/debug:/sys/kernel/debug:rw",
		"/lib/modules:/lib/modules:ro",
		"/proc:/host_proc:ro",
	}, cfg.Engine.Volumes...)

	return hostConfig
}

// ReportLaunching reports the launch of DLE container.
func ReportLaunching(cfg *config.Config) {
	log.Msg(fmt.Sprintf("DLE has started successfully on %s:%d.", getHost(cfg.Engine.Host), cfg.Engine.Port))

	if cfg.LocalUI.Enabled {
		log.Msg(fmt.Sprintf("Local UI has started successfully on %s:%d.", getHost(cfg.LocalUI.Host), cfg.LocalUI.Port))
	}
}

func getHost(cfgHost string) string {
	host := engine.DefaultListenerHost

	if cfgHost != "" {
		host = cfgHost
	}

	return host
}
