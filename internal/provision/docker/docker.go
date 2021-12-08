/*
2020 © Postgres.ai
*/

// Package docker provides an interface to work with Docker containers.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
)

const (
	labelClone = "dblab_clone"
)

var systemVolumes = []string{"/sys", "/lib", "/proc"}

// RunContainer runs specified container.
func RunContainer(r runners.Runner, c *resources.AppConfig) error {
	hostInfo, err := host.Info()
	if err != nil {
		return errors.Wrap(err, "failed to get host info")
	}

	// Directly mount PGDATA if Database Lab is running without any virtualization.
	volumes := []string{fmt.Sprintf("--volume %s:%s", c.DataDir(), c.DataDir())}

	if hostInfo.VirtualizationRole == "guest" {
		// Build custom mounts rely on mounts of the Database Lab instance if it's running inside Docker container.
		// We cannot use --volumes-from because it removes the ZFS mount point.
		volumes, err = getMountVolumes(r, c, hostInfo.Hostname)
		if err != nil {
			return errors.Wrap(err, "failed to detect container volumes")
		}
	}

	unixSocketCloneDir := c.Pool.SocketCloneDir(c.CloneName)

	if err := createSocketCloneDir(unixSocketCloneDir); err != nil {
		return errors.Wrap(err, "failed to create socket clone directory")
	}

	containerFlags := make([]string, 0, len(c.ContainerConf))
	for flagName, flagValue := range c.ContainerConf {
		containerFlags = append(containerFlags, fmt.Sprintf("--%s=%s", flagName, flagValue))
	}

	// TODO (akartasov): use Docker client instead of command execution.
	instancePort := strconv.Itoa(int(c.Port))
	dockerRunCmd := strings.Join([]string{
		"docker run",
		"--name", c.CloneName,
		"--detach",
		"--publish", fmt.Sprintf("%[1]s:%[1]s", instancePort),
		"--env", "PGDATA=" + c.DataDir(),
		strings.Join(volumes, " "),
		"--label", labelClone,
		"--label", c.Pool.Name,
		strings.Join(containerFlags, " "),
		c.DockerImage,
		"-p", instancePort,
		"-k", unixSocketCloneDir,
	}, " ")

	if _, err := r.Run(dockerRunCmd, true); err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	dockerConnectCmd := strings.Join([]string{"docker network connect", c.NetworkID, c.CloneName}, " ")

	if _, err := r.Run(dockerConnectCmd, true); err != nil {
		return errors.Wrap(err, "failed to connect container to the internal DLE network")
	}

	return nil
}

func getMountVolumes(r runners.Runner, c *resources.AppConfig, containerID string) ([]string, error) {
	inspectCmd := "docker inspect -f '{{ json .Mounts }}' " + containerID

	var mountPoints []types.MountPoint

	out, err := r.Run(inspectCmd, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container mounts")
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &mountPoints); err != nil {
		return nil, errors.Wrap(err, "failed to interpret mount paths")
	}

	return buildVolumesFromMountPoints(c, mountPoints), nil
}

func buildVolumesFromMountPoints(c *resources.AppConfig, mountPoints []types.MountPoint) []string {
	unixSocketCloneDir := c.Pool.SocketCloneDir(c.CloneName)
	mounts := tools.GetMountsFromMountPoints(c.DataDir(), mountPoints)
	volumes := make([]string, 0, len(mounts))

	for _, mountPoint := range mountPoints {
		// Add an extra mount for socket directories.
		if strings.HasPrefix(unixSocketCloneDir, mountPoint.Destination) {
			volumes = append(volumes, buildSocketMount(unixSocketCloneDir, mountPoint.Source, mountPoint.Destination))
			break
		}
	}

	for _, mount := range mounts {
		// Exclude system and non-data volumes from a clone container.
		if isSystemVolume(mount.Source) || !strings.HasPrefix(mount.Source, c.Pool.MountDir) {
			continue
		}

		volume := fmt.Sprintf("--volume %s:%s", mount.Source, mount.Target)

		if mount.BindOptions != nil && mount.BindOptions.Propagation != "" {
			volume += ":" + string(mount.BindOptions.Propagation)
		}

		volumes = append(volumes, volume)
	}

	return volumes
}

func isSystemVolume(source string) bool {
	for _, sysVolume := range systemVolumes {
		if strings.HasPrefix(source, sysVolume) {
			return true
		}
	}

	return false
}

// buildSocketMount builds a socket directory mounting rely on dataDir mounting.
func buildSocketMount(socketDir, hostDataDir, destinationDir string) string {
	socketPath := strings.TrimPrefix(socketDir, destinationDir)
	hostSocketDir := path.Join(hostDataDir, socketPath)

	return fmt.Sprintf("--volume %s:%s:rshared", hostSocketDir, socketDir)
}

func createSocketCloneDir(socketCloneDir string) error {
	if err := os.RemoveAll(socketCloneDir); err != nil {
		return err
	}

	if err := os.MkdirAll(socketCloneDir, 0777); err != nil {
		return err
	}

	if err := os.Chmod(socketCloneDir, 0777); err != nil {
		return err
	}

	return nil
}

// StopContainer stops specified container.
func StopContainer(r runners.Runner, c *resources.AppConfig) (string, error) {
	dockerStopCmd := "docker container stop " + c.CloneName

	return r.Run(dockerStopCmd, false)
}

// RemoveContainer removes specified container.
func RemoveContainer(r runners.Runner, cloneName string) (string, error) {
	dockerRemoveCmd := "docker container rm --force --volumes " + cloneName

	return r.Run(dockerRemoveCmd, false)
}

// ListContainers lists container names.
func ListContainers(r runners.Runner, clonePool string) ([]string, error) {
	dockerListCmd := fmt.Sprintf(`docker container ls --filter "label=%s" --filter "label=%s" --all --format '{{.Names}}'`,
		labelClone, clonePool)

	out, err := r.Run(dockerListCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers")
	}

	out = strings.TrimSpace(out)
	if len(out) == 0 {
		return []string{}, nil
	}

	return strings.Split(out, "\n"), nil
}

// GetLogs gets logs from specified container.
func GetLogs(r runners.Runner, c *resources.AppConfig, sinceRelMins uint) (string, error) {
	dockerLogsCmd := "docker logs " + c.CloneName + " " +
		"--since " + strconv.FormatUint(uint64(sinceRelMins), 10) + "m " +
		"--timestamps"

	return r.Run(dockerLogsCmd, true)
}

// Exec executes command on specified container.
func Exec(r runners.Runner, c *resources.AppConfig, cmd string) (string, error) {
	dockerExecCmd := "docker exec " + c.CloneName + " " + cmd

	return r.Run(dockerExecCmd, true)
}

// PrepareImage prepares a Docker image to use.
func PrepareImage(runner runners.Runner, dockerImage string) error {
	imageExists, err := ImageExists(runner, dockerImage)
	if err != nil {
		return fmt.Errorf("cannot check docker image existence: %w", err)
	}

	if imageExists {
		return nil
	}

	if err := PullImage(runner, dockerImage); err != nil {
		return fmt.Errorf("cannot pull docker image: %w", err)
	}

	return nil
}

// ImageExists checks existence of Docker image.
func ImageExists(r runners.Runner, dockerImage string) (bool, error) {
	dockerListImagesCmd := "docker images " + dockerImage + " --quiet"

	out, err := r.Run(dockerListImagesCmd, true)
	if err != nil {
		return false, fmt.Errorf("failed to list images: %w", err)
	}

	return len(strings.TrimSpace(out)) > 0, nil
}

// PullImage pulls Docker image from DockerHub registry.
func PullImage(r runners.Runner, dockerImage string) error {
	dockerPullImageCmd := "docker pull " + dockerImage

	if _, err := r.Run(dockerPullImageCmd, true); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	return nil
}

// IsContainerRunning checks if specified container is running.
func IsContainerRunning(ctx context.Context, docker *client.Client, containerName string) (bool, error) {
	inspection, err := docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return false, fmt.Errorf("failed to inpect container: %w", err)
	}

	return inspection.State.Running, nil
}
