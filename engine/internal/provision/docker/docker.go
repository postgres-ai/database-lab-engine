/*
2020 © Postgres.ai
*/

// Package docker provides an interface to work with Docker containers.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// LabelClone specifies the container label used to identify clone containers.
	LabelClone = "dblab_clone"

	// referenceKey uses as a filtering key to identify image tag.
	referenceKey = "reference"
)

var systemVolumes = []string{"/sys", "/lib", "/proc"}

// imagePullProgress describes the progress of pulling the container image.
type imagePullProgress struct {
	Status   string `json:"status"`
	Progress string `json:"progress"`
}

// RunContainer runs specified container.
func RunContainer(r runners.Runner, c *resources.AppConfig) error {
	hostInfo, err := host.Info()
	if err != nil {
		return errors.Wrap(err, "failed to get host info")
	}

	unixSocketCloneDir, volumes := createDefaultVolumes(c)

	log.Dbg(fmt.Sprintf("Host info: %#v", hostInfo))

	if hostInfo.VirtualizationRole == "guest" {
		// build custom mounts rely on mounts of the Database Lab instance if it's running inside Docker container.
		// we cannot use --volumes-from because it removes the ZFS mount point.
		volumes, err = getMountVolumes(r, c, hostInfo.Hostname)
		if err != nil {
			return errors.Wrap(err, "failed to detect container volumes")
		}
	}

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
		publishPorts(c.ProvisionHosts, instancePort),
		"--env", "PGDATA=" + c.DataDir(),
		"--env", "PG_UNIX_SOCKET_DIR=" + unixSocketCloneDir,
		"--env", "PG_SERVER_PORT=" + instancePort,
		strings.Join(volumes, " "),
		fmt.Sprintf("--label %s='%s'", LabelClone, c.Pool.Name),
		strings.Join(containerFlags, " "),
		// The image can come from a request (the clone upgrade override) and reaches this through
		// a /bin/bash -c string, so it is passed as data rather than as shell syntax.
		runners.Quote(c.DockerImage),
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

// UpgradeContainerConfig describes a one-off pg_upgrade run against a clone data directory.
type UpgradeContainerConfig struct {
	Image string
	Name  string
	// User is the numeric "uid:gid" owning the clone data directory. pg_upgrade refuses to run
	// as root and has to write every file it links, so the value is resolved from the directory
	// itself instead of assumed from the engine process or the image.
	User string
	// Env holds already formatted "NAME=value" pairs for the upgrade script.
	Env []string
}

// RunUpgradeContainer runs the upgrade image against a clone data directory and waits for it to
// exit, returning the container exit code so the caller can map it to a recovery action. An exit
// code is a normal outcome here, not an error: only a failure to run the container at all is.
//
// The container is detached and waited on through ctx, which is what bounds a pg_upgrade that
// never finishes. It is removed before this returns whatever ended the wait, because the caller
// decides what to do next by reading the directory the container writes into, and because a
// container left behind would take the name the next attempt needs.
//
// Volumes are derived exactly like clone containers derive them, so a Database Lab instance that
// itself runs inside Docker mounts the pool through its own mounts rather than a host path.
func RunUpgradeContainer(ctx context.Context, dockerClient *client.Client, r runners.Runner,
	c *resources.AppConfig, cfg UpgradeContainerConfig) (int, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get host info")
	}

	_, volumes := createDefaultVolumes(c)

	if hostInfo.VirtualizationRole == "guest" {
		volumes, err = getMountVolumes(r, c, hostInfo.Hostname)
		if err != nil {
			return 0, errors.Wrap(err, "failed to detect container volumes")
		}
	}

	envFlags := make([]string, 0, len(cfg.Env)+1)
	for _, envPair := range append(cfg.Env, "HOME="+c.CloneDir()) {
		envFlags = append(envFlags, "--env "+runners.Quote(envPair))
	}

	// pg_upgrade writes pg_upgrade_output.d and its temporary sockets into the working
	// directory, which would otherwise be the image's "/".
	dockerRunCmd := strings.Join([]string{
		"docker run",
		"--detach",
		"--name", cfg.Name,
		"--user", cfg.User,
		"--workdir", runners.Quote(c.CloneDir()),
		strings.Join(envFlags, " "),
		strings.Join(volumes, " "),
		runners.Quote(cfg.Image),
	}, " ")

	if _, err := r.Run(dockerRunCmd, true); err != nil {
		return 0, errors.Wrap(err, "failed to run the upgrade container")
	}

	defer func() {
		if _, err := RemoveContainer(r, cfg.Name); err != nil {
			log.Dbg("failed to remove the upgrade container:", err)
		}
	}()

	return waitForUpgradeContainer(ctx, dockerClient, cfg.Name)
}

// waitForUpgradeContainer blocks until the upgrade container exits or ctx ends. The condition has
// to be "not running" rather than "next exit": the container is started before the wait is
// requested, and a short run that has already exited by then would never produce a next exit.
func waitForUpgradeContainer(ctx context.Context, dockerClient *client.Client, name string) (int, error) {
	waiting := dockerClient.ContainerWait(ctx, name, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	select {
	case result := <-waiting.Result:
		return int(result.StatusCode), nil

	case err := <-waiting.Error:
		return 0, errors.Wrapf(err, "failed to wait for the upgrade container %s", name)

	case <-ctx.Done():
		return 0, errors.Wrapf(ctx.Err(), "the upgrade container %s did not finish in time", name)
	}
}

func publishPorts(provisionHosts string, instancePort string) string {
	if provisionHosts == "" {
		return fmt.Sprintf("--publish %[1]s:%[1]s", instancePort)
	}

	pub := []string{}

	for _, s := range strings.Split(provisionHosts, ",") {
		pub = append(pub, "--publish", fmt.Sprintf("%[1]s:%[2]s:%[2]s", s, instancePort))
	}

	return strings.Join(pub, " ")
}

func createDefaultVolumes(c *resources.AppConfig) (string, []string) {
	unixSocketCloneDir := c.Pool.SocketCloneDir(c.CloneName)

	// directly mount PGDATA if Database Lab is running without any virtualization.
	volumes := []string{
		fmt.Sprintf("--volume %s:%s", c.CloneDir(), c.CloneDir()),
		fmt.Sprintf("--volume %s:%s", unixSocketCloneDir, unixSocketCloneDir),
	}

	return unixSocketCloneDir, volumes
}

func getMountVolumes(r runners.Runner, c *resources.AppConfig, containerID string) ([]string, error) {
	inspectCmd := "docker inspect -f '{{ json .Mounts }}' " + containerID

	var mountPoints []container.MountPoint

	out, err := r.Run(inspectCmd, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container mounts")
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &mountPoints); err != nil {
		return nil, errors.Wrap(err, "failed to interpret mount paths")
	}

	return buildVolumesFromMountPoints(c, mountPoints), nil
}

func buildVolumesFromMountPoints(c *resources.AppConfig, mountPoints []container.MountPoint) []string {
	unixSocketCloneDir := c.Pool.SocketCloneDir(c.CloneName)
	mounts := tools.GetMountsFromMountPoints(c.CloneDir(), mountPoints)
	volumes := make([]string, 0, len(mounts))

	for _, mountPoint := range mountPoints {
		// add an extra mount for socket directories.
		if strings.HasPrefix(unixSocketCloneDir, mountPoint.Destination) {
			volumes = append(volumes, buildSocketMount(unixSocketCloneDir, mountPoint.Source, mountPoint.Destination))
			break
		}
	}

	for _, mount := range mounts {
		// exclude system and non-data volumes from a clone container.
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

	return os.Chmod(socketCloneDir, 0777)
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
	dockerListCmd := fmt.Sprintf(`docker container ls --filter "label=%s=%s" --all --format '{{.Names}}'`,
		LabelClone, clonePool)

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
func PrepareImage(ctx context.Context, docker *client.Client, dockerImage string) error {
	imageExists, err := ImageExists(ctx, docker, dockerImage)
	if err != nil {
		return fmt.Errorf("cannot check docker image existence: %w", err)
	}

	if imageExists {
		return nil
	}

	if err := PullImage(ctx, docker, dockerImage); err != nil {
		return fmt.Errorf("cannot pull docker image: %w", err)
	}

	return nil
}

// pgMajorEnv is the environment variable the PostgreSQL images carry their major version in.
// The pg-upgrade image inherits it from its base image, so it names the major its pg_upgrade
// binaries actually produce - which an image tag can only claim.
const pgMajorEnv = "PG_MAJOR"

// ImagePGMajor reports the PostgreSQL major an image carries binaries for, read from PG_MAJOR in
// the image config. A zero major means the image is there but declares no such variable, which
// leaves the caller with the tag as its only evidence; that is a different answer from an error,
// which says nothing about the image at all.
func ImagePGMajor(ctx context.Context, docker *client.Client, dockerImage string) (int, error) {
	inspection, err := docker.ImageInspect(ctx, dockerImage)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect image %q: %w", dockerImage, err)
	}

	if inspection.Config == nil {
		return 0, nil
	}

	return pgMajorFromEnv(inspection.Config.Env), nil
}

// pgMajorFromEnv picks the PostgreSQL major out of an image's environment, or zero when there is
// none to read. Docker keeps the entries in the order they were declared and a later assignment
// shadows an earlier one, so the last one decides - including when it is the unparsable one,
// because that is the value a container would see.
func pgMajorFromEnv(env []string) int {
	major := 0

	for _, entry := range env {
		value, found := strings.CutPrefix(entry, pgMajorEnv+"=")
		if !found {
			continue
		}

		major, _ = strconv.Atoi(strings.TrimSpace(value))
	}

	if major < 0 {
		return 0
	}

	return major
}

// ImageExists checks existence of Docker image.
func ImageExists(ctx context.Context, docker *client.Client, dockerImage string) (bool, error) {
	filterArgs := make(client.Filters)
	filterArgs.Add(referenceKey, dockerImage)

	list, err := docker.ImageList(ctx, client.ImageListOptions{
		All:     false,
		Filters: filterArgs,
	})

	if err != nil {
		return false, fmt.Errorf("failed to list images: %w", err)
	}

	return len(list.Items) > 0, nil
}

// PullImage pulls Docker image from DockerHub registry.
func PullImage(ctx context.Context, docker *client.Client, dockerImage string) error {
	pullResponse, err := docker.ImagePull(ctx, dockerImage, client.ImagePullOptions{})

	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// reading output of image pulling, without reading pull will not be performed
	decoder := json.NewDecoder(pullResponse)

	for {
		var pullResult imagePullProgress
		if err := decoder.Decode(&pullResult); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("failed to pull image: %w", err)
		}

		log.Dbg("Image pulling progress", pullResult.Status, pullResult.Progress)
	}

	err = pullResponse.Close()

	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return nil
}

// IsContainerRunning checks if specified container is running.
func IsContainerRunning(ctx context.Context, docker *client.Client, containerName string) (bool, error) {
	inspection, err := docker.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to inpect container: %w", err)
	}

	return inspection.Container.State.Running, nil
}
