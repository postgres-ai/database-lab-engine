/*
2020 © Postgres.ai
*/

// Package docker provides an interface to work with Docker containers.
package docker

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

const (
	labelClone = "dblab-clone"
)

// RunContainer runs specified container.
func RunContainer(r runners.Runner, c *resources.AppConfig) (string, error) {
	dockerRunCmd := "docker run " +
		"--name " + c.CloneName + " " +
		"--detach " +
		"--publish " + strconv.FormatUint(uint64(c.Port), 10) + ":5432 " +
		"--env PGDATA=/var/lib/postgresql/pgdata " +
		"--volume " + c.Datadir + ":/var/lib/postgresql/pgdata " +
		"--volume " + c.UnixSocketCloneDir + ":/var/run/postgresql " +
		"--label " + labelClone + " " +
		c.DockerImage

	return r.Run(dockerRunCmd, true)
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
func ListContainers(r runners.Runner) ([]string, error) {
	dockerListCmd := "docker container ls " +
		"--filter \"label=" + labelClone + "\" " +
		"--all --quiet"

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