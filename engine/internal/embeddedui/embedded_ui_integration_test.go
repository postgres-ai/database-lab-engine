//go:build integration
// +build integration

/*
2021 © Postgres.ai
*/

package embeddedui

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

func TestStartExistingContainer(t *testing.T) {
	t.Parallel()
	docker, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)

	engProps := global.EngineProps{
		InstanceID: "testuistart",
	}

	embeddedUI := New(
		Config{
			// "mock" UI image
			DockerImage: "gcr.io/google_containers/pause-amd64:3.0",
		},
		engProps,
		runners.NewLocalRunner(false),
		docker,
	)

	ctx := context.TODO()

	networks.Setup(ctx, docker, engProps.InstanceID, getEmbeddedUIName(engProps.InstanceID))

	// clean test UI container
	defer tools.RemoveContainer(ctx, docker, getEmbeddedUIName(engProps.InstanceID), 30*time.Second)

	// start UI container
	err = embeddedUI.Run(ctx)
	require.NoError(t, err)

	// explicitly stop container
	tools.StopContainer(ctx, docker, getEmbeddedUIName(engProps.InstanceID), 30*time.Second)

	// start UI container back
	err = embeddedUI.Run(ctx)
	require.NoError(t, err)

	// list containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", getEmbeddedUIName(engProps.InstanceID))

	list, err := docker.ContainerList(
		ctx,
		types.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		},
	)

	require.NoError(t, err)
	assert.NotEmpty(t, list)
	// initial container
	assert.Equal(t, "running", list[0].State)
}
