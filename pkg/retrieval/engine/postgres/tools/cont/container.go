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
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools"
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
	// DBLabInstanceIDLabel defines a label to mark service containers related to the current Database Lab instance.
	DBLabInstanceIDLabel = "dblab_instance_id"

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
)

// TODO(akartasov): Control container manager.

// StopControlContainers stops control containers run by Database Lab Engine.
func StopControlContainers(ctx context.Context, dockerClient *client.Client, instanceID, dataDir string) error {
	log.Msg("Stop control containers")

	list, err := getControlContainerList(ctx, dockerClient, instanceID)
	if err != nil {
		return err
	}

	for _, controlCont := range list {
		containerName := getControlContainerName(controlCont)

		controlLabel, ok := controlCont.Labels[DBLabControlLabel]
		if !ok {
			log.Msg("Control label not found for container: ", containerName)
			continue
		}

		if shouldStopInternalProcess(controlLabel) {
			log.Msg("Stopping control container: ", containerName)

			if err := tools.StopPostgres(ctx, dockerClient, controlCont.ID, dataDir); err != nil {
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
	log.Msg("Cleanup control containers")

	list, err := getControlContainerList(ctx, dockerClient, instanceID)
	if err != nil {
		return err
	}

	for _, controlCont := range list {
		log.Msg("Removing control container:", getControlContainerName(controlCont))

		if err := dockerClient.ContainerRemove(ctx, controlCont.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return err
		}
	}

	return nil
}

func getControlContainerList(ctx context.Context, dockerClient *client.Client, instanceID string) ([]types.Container, error) {
	filterPairs := []filters.KeyValuePair{
		{
			Key:   labelFilter,
			Value: DBLabControlLabel,
		},
		{
			Key:   labelFilter,
			Value: DBLabInstanceIDLabel + "=" + instanceID,
		},
	}

	return dockerClient.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filterPairs...),
	})
}

func shouldStopInternalProcess(controlLabel string) bool {
	return controlLabel == DBLabSyncLabel
}

func getControlContainerName(controlCont types.Container) string {
	return strings.Join(controlCont.Names, ", ")
}
