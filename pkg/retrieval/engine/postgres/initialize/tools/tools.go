/*
2020 © Postgres.ai
*/

// Package tools provides helpers to initialize data.
package tools

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

const (
	maxValuesToReturn     = 1
	essentialLogsInterval = "10s"
)

// IsEmptyDirectory checks whether a directory is empty.
func IsEmptyDirectory(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}

	names, err := f.Readdirnames(maxValuesToReturn)
	if err != nil && err != io.EOF {
		return false, err
	}

	return len(names) == 0, nil
}

// InspectCommandResponse inspects success of command execution.
func InspectCommandResponse(ctx context.Context, dockerClient *client.Client, containerID, commandID string) error {
	inspect, err := dockerClient.ContainerExecInspect(ctx, commandID)
	if err != nil {
		return errors.Wrap(err, "failed to create command")
	}

	if inspect.ExitCode == 0 {
		return nil
	}

	logs, err := dockerClient.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		Since:      essentialLogsInterval,
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get container logs")
	}

	errorDetails, err := ioutil.ReadAll(logs)
	if err != nil {
		return errors.Wrap(err, "failed to get error logs")
	}

	defer func() { _ = logs.Close() }()

	return errors.Errorf("exit code: %d.\nContainer logs:\n%s", inspect.ExitCode, string(errorDetails))
}

// CheckContainerReadiness checks health and reports if container is ready.
func CheckContainerReadiness(ctx context.Context, dockerClient *client.Client, containerID string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		resp, err := dockerClient.ContainerInspect(ctx, containerID)
		if err != nil {
			return errors.Wrap(err, "failed to create container")
		}

		if resp.State != nil && resp.State.Health != nil {
			switch resp.State.Health.Status {
			case types.Healthy:
				return nil

			case types.Unhealthy:
				return errors.New("container health check failed")
			}

			log.Msg(fmt.Sprintf("Container is not ready yet. The current state is %v.", resp.State.Health.Status))
		}

		time.Sleep(time.Second)
	}
}
