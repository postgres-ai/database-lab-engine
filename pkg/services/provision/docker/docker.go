/*
2020 © Postgres.ai
*/

// Package docker provides an interface to work with Docker containers.
package docker

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

const (
	labelClone = "dblab_clone"
)

// RunContainer runs specified container.
func RunContainer(r runners.Runner, c *resources.AppConfig) (string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", errors.Wrap(err, "failed to get host info")
	}

	// Directly mount PGDATA if Database Lab is running without any virtualization.
	socketVolume := fmt.Sprintf("--volume %s:%s", c.Datadir, c.Datadir)

	if hostInfo.VirtualizationRole == "guest" {
		// Use volumes from the Database Lab instance if it's running inside Docker container.
		socketVolume = "--volumes-from=" + hostInfo.Hostname
	}

	if err := createSocketCloneDir(c.UnixSocketCloneDir); err != nil {
		return "", errors.Wrap(err, "failed to create socket clone directory")
	}

	dockerRunCmd := "docker run " +
		"--name " + c.CloneName + " " +
		"--detach " +
		"--publish " + strconv.Itoa(int(c.Port)) + ":5432 " +
		"--env PGDATA=" + c.Datadir + " " + socketVolume + " " +
		"--label " + labelClone + " " +
		"--label " + c.ClonePool + " " +
		c.DockerImage + " -k " + c.UnixSocketCloneDir

	return r.Run(dockerRunCmd, true)
}

func createSocketCloneDir(socketCloneDir string) error {
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

	return r.Run(dockerStopCmd, true)
}

// RemoveContainer removes specified container.
func RemoveContainer(r runners.Runner, c *resources.AppConfig) (string, error) {
	dockerRemoveCmd := "docker container rm --force " + c.CloneName

	return r.Run(dockerRemoveCmd, true)
}

// ListContainers lists containers.
func ListContainers(r runners.Runner, clonePool string) ([]string, error) {
	dockerListCmd := fmt.Sprintf(`docker container ls --filter "label=%s" --filter "label=%s" --all --quiet`,
		labelClone, clonePool)

	out, err := r.Run(dockerListCmd, true)
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

// ImageExists checks existence of Docker image.
func ImageExists(r runners.Runner, dockerImage string) (bool, error) {
	dockerListImagesCmd := "docker images " + dockerImage + " --quiet"

	out, err := r.Run(dockerListImagesCmd, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to list images")
	}

	return len(strings.TrimSpace(out)) > 0, nil
}

// PullImage pulls Docker image from DockerHub registry.
func PullImage(r runners.Runner, dockerImage string) error {
	dockerPullImageCmd := "docker pull " + dockerImage

	_, err := r.Run(dockerPullImageCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to pull images")
	}

	return err
}
