//go:build integration
// +build integration

/*
2021 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dockerutils "gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
)

func TestStartExisingDumpContainer(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	docker, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)

	// create dump job

	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	engProps := global.EngineProps{
		InstanceID: fmt.Sprintf("dumpjob-%d", random.Intn(10000)),
	}

	job, err := NewDumpJob(
		config.JobConfig{
			Spec: config.JobSpec{Name: "test"},
			FSPool: &resources.Pool{
				DataSubDir: t.TempDir(),
			},
			Docker: docker,
		},
		&global.Config{},
		engProps,
	)
	assert.NoError(t, err)
	job.DockerImage = "postgresai/extended-postgres:14"
	job.DumpOptions.DumpLocation = t.TempDir()

	err = dockerutils.PrepareImage(ctx, docker, job.DockerImage)
	assert.NoError(t, err)

	// create dump container and stop it
	container, err := docker.ContainerCreate(ctx, job.buildContainerConfig(""), nil, &network.NetworkingConfig{},
		nil, job.dumpContainerName(),
	)
	assert.NoError(t, err)

	// clean container in case of any error
	defer tools.RemoveContainer(ctx, docker, container.ID, 10*time.Second)

	job.Run(ctx)

	// list containers and check that container job container got processed
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", job.dumpContainerName())

	list, err := docker.ContainerList(
		ctx,
		types.ContainerListOptions{
			All:     false,
			Filters: filterArgs,
		},
	)

	require.NoError(t, err)
	assert.Empty(t, list)

}
