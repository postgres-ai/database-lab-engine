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

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

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

	docker, err := client.NewClientWithOpts(client.FromEnv)
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

	networks.Setup(ctx, docker, engProps.InstanceID, id)

	defer func() {
		tools.StopContainer(ctx, docker, "dblab_embedded_ui_"+id, 10*time.Second)
		tools.RemoveContainer(ctx, docker, "dblab_embedded_ui_"+id, 10*time.Second)
	}()

	err = embeddedUI.Run(ctx)
	require.NoError(t, err)

	// wait some time to generate container logs
	time.Sleep(5 * time.Second)

	filterArgs := filters.NewArgs()
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
