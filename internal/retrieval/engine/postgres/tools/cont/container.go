/*
2020 © Postgres.ai
*/

// Package cont provides tools to manage service containers started by Database Lab Engine.
package cont

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	labelFilter = "label"

	// StopTimeout defines a container stop timeout.
	StopTimeout = 30 * time.Second

	// StopPhysicalTimeout defines stop timeout for a physical container.
	StopPhysicalTimeout = 5 * time.Second

	// SyncInstanceContainerPrefix defines a sync container name.
	SyncInstanceContainerPrefix = "dblab_sync_"

	// DBLabControlLabel defines a label to mark service containers.
	DBLabControlLabel = "dblab_control"
	// DBLabSatelliteLabel defines a label to mark satellite containers.
	DBLabSatelliteLabel = "dblab_satellite"
	// DBLabInstanceIDLabel defines a label to mark service containers related to the current Database Lab instance.
	DBLabInstanceIDLabel = "dblab_instance_id"
	// DBLabEngineNameLabel defines the label value providing the container name of the current Database Lab instance.
	DBLabEngineNameLabel = "dblab_engine_name"

	// DBLabSyncLabel defines a label value for sync containers.
	DBLabSyncLabel = "dblab_sync"
	// DBLabPromoteLabel defines a label value for promote containers.
	DBLabPromoteLabel = "dblab_promote"
	// DBLabPatchLabel defines a label value for patch containers.
	DBLabPatchLabel = "dblab_patch"
	// DBLabDumpLabel defines a label value for dump containers.
	DBLabDumpLabel = "dblab_dump"
	// DBLabRestoreLabel defines a label value for restore containers.
	DBLabRestoreLabel = "dblab_restore"
	// DBLabEmbedUILabel defines a label value for embed UI containers.
	DBLabEmbedUILabel = "dblab_embed_ui"

	// DBLabRunner defines a label to mark runner containers.
	DBLabRunner = "dblab_runner"
)

// TODO(akartasov): Control container manager.

// StopControlContainers stops control containers run by Database Lab Engine.
func StopControlContainers(ctx context.Context, dockerClient *client.Client, instanceID, dataDir string) error {
	log.Msg("Stop control containers")

	list, err := getContainerList(ctx, dockerClient, instanceID, getControlContainerFilters())
	if err != nil {
		return err
	}

	for _, controlCont := range list {
		containerName := getContainerName(controlCont)

		controlLabel, ok := controlCont.Labels[DBLabControlLabel]
		if !ok {
			log.Msg("Control label not found for container: ", containerName)
			continue
		}

		if shouldStopInternalProcess(controlLabel) {
			log.Msg("Stopping control container: ", containerName)

			if err := tools.StopPostgres(ctx, dockerClient, controlCont.ID, dataDir, tools.DefaultStopTimeout); err != nil {
				log.Msg("Failed to stop Postgres", err)
				tools.PrintContainerLogs(ctx, dockerClient, controlCont.ID)

				continue
			}
		}

		log.Msg("Removing control container:", containerName)

		if err := dockerClient.ContainerRemove(ctx, controlCont.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return err
		}
	}

	return nil
}

// CleanUpControlContainers removes control containers run by Database Lab Engine.
func CleanUpControlContainers(ctx context.Context, dockerClient *client.Client, instanceID string) error {
	log.Msg("Clean up control containers")
	return cleanUpContainers(ctx, dockerClient, instanceID, getControlContainerFilters())
}

// CleanUpSatelliteContainers removes satellite containers run by Database Lab Engine.
func CleanUpSatelliteContainers(ctx context.Context, dockerClient *client.Client, instanceID string) error {
	log.Msg("Clean up satellite containers")
	return cleanUpContainers(ctx, dockerClient, instanceID, getSatelliteContainerFilters())
}

// cleanUpContainers removes containers run by Database Lab Engine.
func cleanUpContainers(ctx context.Context, dockerCli *client.Client, instanceID string, filter []filters.KeyValuePair) error {
	list, err := getContainerList(ctx, dockerCli, instanceID, filter)
	if err != nil {
		return err
	}

	for _, controlCont := range list {
		log.Msg("Removing container:", getContainerName(controlCont))

		if err := dockerCli.ContainerRemove(ctx, controlCont.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return err
		}
	}

	return nil
}

func getContainerList(ctx context.Context, d *client.Client, instanceID string, pairs []filters.KeyValuePair) ([]types.Container, error) {
	filterPairs := append([]filters.KeyValuePair{
		{
			Key:   labelFilter,
			Value: DBLabInstanceIDLabel + "=" + instanceID,
		},
	}, pairs...)

	return d.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filterPairs...),
	})
}

func getControlContainerFilters() []filters.KeyValuePair {
	return []filters.KeyValuePair{{
		Key:   labelFilter,
		Value: DBLabControlLabel,
	}}
}

func getSatelliteContainerFilters() []filters.KeyValuePair {
	return []filters.KeyValuePair{{
		Key:   labelFilter,
		Value: DBLabSatelliteLabel,
	}}
}

func shouldStopInternalProcess(controlLabel string) bool {
	return controlLabel == DBLabSyncLabel
}

func getContainerName(controlCont types.Container) string {
	return strings.Join(controlCont.Names, ", ")
}

// BuildHostConfig builds host config.
func BuildHostConfig(ctx context.Context, docker *client.Client, dataDir string,
	contConf map[string]interface{}) (*container.HostConfig, error) {
	hostOptions, err := ResourceOptions(contConf)
	if err != nil {
		return nil, err
	}

	hostConfig := &container.HostConfig{
		Resources: hostOptions.Resources,
		ShmSize:   hostOptions.ShmSize,
	}

	if err := tools.AddVolumesToHostConfig(ctx, docker, hostConfig, dataDir); err != nil {
		return nil, err
	}

	return hostConfig, nil
}

// ResourceOptions parses host config options.
func ResourceOptions(containerConfigs map[string]interface{}) (*container.HostConfig, error) {
	normalizedConfig := make(map[string]interface{}, len(containerConfigs))

	for configKey, configValue := range containerConfigs {
		normalizedKey := strings.ToLower(strings.ReplaceAll(configKey, "-", ""))

		// Convert human-readable string representing an amount of memory.
		if valueString, ok := configValue.(string); ok {
			ramInBytes, err := units.RAMInBytes(valueString)
			if err == nil {
				normalizedConfig[normalizedKey] = ramInBytes
				continue
			}
		}

		normalizedConfig[normalizedKey] = configValue
	}

	// Unmarshal twice because composite types do not unmarshal correctly: https://github.com/go-yaml/yaml/issues/63
	hostConfig := &container.HostConfig{}
	if err := options.Unmarshal(normalizedConfig, &hostConfig); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal container configuration options")
	}

	resources := container.Resources{}
	if err := options.Unmarshal(normalizedConfig, &resources); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal container configuration options")
	}

	hostConfig.Resources = resources

	return hostConfig, nil
}
