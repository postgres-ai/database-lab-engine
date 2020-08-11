/*
2020 © Postgres.ai
*/

// Package tools provides helpers to initialize data.
package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

const (
	maxValuesToReturn     = 1
	essentialLogsInterval = "10s"

	// StopTimeout defines a container stop timeout.
	StopTimeout = 10 * time.Second
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

// TouchFile creates an empty file.
func TouchFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to touch file: %s", filename)
	}

	defer func() { _ = file.Close() }()

	return nil
}

// DetectPGVersion defines PostgreSQL version of PGDATA.
func DetectPGVersion(dataDir string) (string, error) {
	version, err := exec.Command("cat", fmt.Sprintf(`%s/PG_VERSION`, dataDir)).CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(version)), nil
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
			return errors.Wrapf(err, "failed to inspect container %s", containerID)
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

// RemoveContainer stops and removes container.
func RemoveContainer(ctx context.Context, dockerClient *client.Client, containerID string, stopTimeout time.Duration) {
	if err := dockerClient.ContainerStop(ctx, containerID, pointer.ToDuration(stopTimeout)); err != nil {
		log.Err("Failed to stop container: ", err)
	}

	if err := dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		log.Err("Failed to remove container: ", err)

		return
	}

	log.Msg(fmt.Sprintf("Stop container ID: %v", containerID))
}

// PullImage pulls a Docker image.
func PullImage(ctx context.Context, dockerClient *client.Client, image string) error {
	pullOutput, err := dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to pull image %s", image)
	}

	defer func() { _ = pullOutput.Close() }()

	if _, err := io.Copy(os.Stdout, pullOutput); err != nil {
		log.Err("Failed to render pull image output: ", err)
	}

	return nil
}

// ExecCommand runs command in Docker container.
func ExecCommand(ctx context.Context, dockerClient *client.Client, containerID string, execCfg types.ExecConfig) error {
	execCfg.AttachStdout = true
	execCfg.AttachStderr = true
	execCfg.Tty = true

	execCommand, err := dockerClient.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return errors.Wrap(err, "failed to create command")
	}

	attachResponse, err := dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return errors.Wrap(err, "failed to attach to exec command")
	}

	defer attachResponse.Close()

	if err := waitForCommandResponse(ctx, attachResponse); err != nil {
		return errors.Wrap(err, "failed to exec command")
	}

	if err := InspectCommandResponse(ctx, dockerClient, containerID, execCommand.ID); err != nil {
		return errors.Wrap(err, "unsuccessful command response")
	}

	return nil
}

func waitForCommandResponse(ctx context.Context, attachResponse types.HijackedResponse) error {
	waitCommandCh := make(chan struct{})

	go func() {
		if _, err := io.Copy(os.Stdout, attachResponse.Reader); err != nil {
			log.Err("failed to get command output:", err)
		}

		waitCommandCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-waitCommandCh:
	}

	return nil
}

// ProcessAttachResponse reads and processes the cmd output.
func ProcessAttachResponse(ctx context.Context, reader io.Reader, output io.Writer) error {
	var errBuf bytes.Buffer

	outputDone := make(chan error)

	go func() {
		// StdCopy de-multiplexes the stream into two writers.
		_, err := stdcopy.StdCopy(output, &errBuf, reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return errors.Wrap(err, "failed to copy output")
		}

		break

	case <-ctx.Done():
		return ctx.Err()
	}

	if errBuf.Len() > 0 {
		return errors.New(errBuf.String())
	}

	return nil
}
