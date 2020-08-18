/*
2020 © Postgres.ai
*/

// TODO(akartasov): Refactor tools package: divide to specific subpackages.

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
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

const (
	maxValuesToReturn     = 1
	essentialLogsInterval = "10s"

	// StopTimeout defines a container stop timeout.
	StopTimeout = time.Minute

	// SyncInstanceContainerPrefix defines a sync container name.
	SyncInstanceContainerPrefix = "dblab_sync_"

	// DBLabControlLabel defines a label to mark service containers.
	DBLabControlLabel = "dblab_control"
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

// AddVolumesToHostConfig adds volumes to container host configuration depends on process environment.
func AddVolumesToHostConfig(hostConfig *container.HostConfig, dataDir string) error {
	hostInfo, err := host.Info()
	if err != nil {
		return errors.Wrap(err, "failed to get host info")
	}

	log.Dbg("Virtualization system: ", hostInfo.VirtualizationSystem)

	if hostInfo.VirtualizationRole == "guest" {
		hostConfig.VolumesFrom = []string{hostInfo.Hostname}
	} else {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: dataDir,
			Target: dataDir,
		})
	}

	return nil
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

// PrintContainerLogs prints container output.
func PrintContainerLogs(ctx context.Context, dockerClient *client.Client, containerName string) {
	containerLogs, err := dockerClient.ContainerLogs(ctx, containerName, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      essentialLogsInterval,
		Details:    true,
	})

	if err != nil {
		log.Err("Failed to get container logs", err)
		return
	}

	if err := ProcessAttachResponse(ctx, containerLogs, os.Stdout); err != nil {
		log.Err("Failed to process attach response: ", err)
	}
}

// RemoveContainer stops and removes container.
func RemoveContainer(ctx context.Context, dockerClient *client.Client, containerID string, stopTimeout time.Duration) {
	log.Msg(fmt.Sprintf("Removing container ID: %v", containerID))

	if err := dockerClient.ContainerStop(ctx, containerID, pointer.ToDuration(stopTimeout)); err != nil {
		log.Err("Failed to stop container: ", err)
	}

	if err := dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		log.Err("Failed to remove container: ", err)

		return
	}

	log.Msg(fmt.Sprintf("Container %q has been removed", containerID))
}

// PullImage pulls a Docker image.
func PullImage(ctx context.Context, dockerClient *client.Client, image string) error {
	pullOutput, err := dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to pull image %q", image)
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
