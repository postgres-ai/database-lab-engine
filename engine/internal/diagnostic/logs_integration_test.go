//go:build integration
// +build integration

/*
2022 © Postgres.ai
*/

package diagnostic

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/moby/moby/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/embeddedui"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

func TestContainerOutputCollection(t *testing.T) {
	t.Parallel()

	id := "logs_collection_" + time.Now().Format(timeFormat)

	dir := t.TempDir()

	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)

	engProps := global.EngineProps{
		InstanceID: id,
	}

	// start test container as UI
	embeddedUI := embeddedui.New(
		embeddedui.Config{
			DockerImage: "nginx:1.23.0",
		},
		engProps,
		runners.NewLocalRunner(false),
		docker,
	)

	ctx := context.Background()

	// only the network is needed: embeddedUI.Run connects its container to it. Setup's own connect
	// step fails because no container is named id, and the network outlives that error.
	_, _ = networks.Setup(ctx, docker, engProps.InstanceID, id)

	// the instance ID is unique per run, so a network left behind is never reused and every run
	// would take another subnet from the daemon's address pools. t.Cleanup runs after the deferred
	// container removal below, once nothing is attached to the network.
	t.Cleanup(func() {
		_, err := docker.NetworkRemove(context.Background(), networks.NetworkPrefix+engProps.InstanceID, client.NetworkRemoveOptions{})
		assert.NoError(t, err)
	})

	defer func() {
		tools.StopContainer(ctx, docker, "dblab_embedded_ui_"+id, 10)
		tools.RemoveContainer(ctx, docker, "dblab_embedded_ui_"+id, 10)
	}()

	err = embeddedUI.Run(ctx)
	require.NoError(t, err)

	// wait some time to generate container logs
	time.Sleep(5 * time.Second)

	filterArgs := make(client.Filters)
	filterArgs.Add("label", fmt.Sprintf("%s=%s", cont.DBLabSatelliteLabel, cont.DBLabEmbeddedUILabel))
	err = collectContainersOutput(ctx, docker, dir, filterArgs)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dirList))

	stat, err := os.Stat(path.Join(dir, dirList[0].Name(), containerOutputFile))
	assert.NoError(t, err)
	assert.NotEqual(t, 0, stat.Size())
}
