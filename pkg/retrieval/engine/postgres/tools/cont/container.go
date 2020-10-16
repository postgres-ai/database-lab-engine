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
)

const (
	labelFilter = "label"

	// StopTimeout defines a container stop timeout.
	StopTimeout = time.Minute

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
	// DBLabDumpLabel defines a label value for dump containers.
	DBLabDumpLabel = "dblab_dump"
	// DBLabRestoreLabel defines a label value for restore containers.
	DBLabRestoreLabel = "dblab_restore"
)

// CleanUpServiceContainers removes service containers run by Database Lab Engine.
func CleanUpServiceContainers(ctx context.Context, dockerClient *client.Client, instanceID string) error {
	log.Msg("Cleanup service containers")

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

	list, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filterPairs...),
	})
	if err != nil {
		return err
	}

	for _, serviceCont := range list {
		log.Msg("Removing service container:", strings.Join(serviceCont.Names, ", "))

		if err := dockerClient.ContainerRemove(ctx, serviceCont.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			return err
		}
	}

	return nil
}
