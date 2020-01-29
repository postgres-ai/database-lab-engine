/*
2020 © Postgres.ai
*/

package provision

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
<<<<<<< HEAD
	labelClone = "dblab-clone"
=======
	SUDO = "" // "sudo "
>>>>>>> feat: Dockerfile for dblab-server
)

// DockerRunContainer runs specified container.
func DockerRunContainer(r Runner, c *PgConfig) (string, error) {
	dockerRunCmd := SUDO + "docker run " +
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

// DockerStopContainer stops specified container.
func DockerStopContainer(r Runner, c *PgConfig) (string, error) {
	dockerStopCmd := SUDO + "docker container stop " + c.CloneName

	return r.Run(dockerStopCmd, true)
}

// DockerRemoveContainer removes specified container.
func DockerRemoveContainer(r Runner, c *PgConfig) (string, error) {
	dockerRemoveCmd := SUDO + "docker container rm " + c.CloneName

	return r.Run(dockerRemoveCmd, true)
}

// DockerListContainers lists containers.
func DockerListContainers(r Runner) ([]string, error) {
	dockerListCmd := SUDO + "docker container ls " +
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

// DockerGetLogs gets logs from specified container.
func DockerGetLogs(r Runner, c *PgConfig, sinceRelMins uint) (string, error) {
	dockerLogsCmd := SUDO + "docker logs " + c.CloneName + " " +
		"--since " + strconv.FormatUint(uint64(sinceRelMins), 10) + "m " +
		"--timestamps"

	return r.Run(dockerLogsCmd, true)
}

// DockerExec executes command on specified container.
func DockerExec(r Runner, c *PgConfig, cmd string) (string, error) {
	dockerExecCmd := SUDO + "docker exec " + c.CloneName + " " + cmd

	return r.Run(dockerExecCmd, true)
}

// DockerImageExists checks existence of Docker image.
func DockerImageExists(r Runner, dockerImage string) (bool, error) {
	dockerListImagesCmd := SUDO + "docker images " + dockerImage + " --quiet"

	out, err := r.Run(dockerListImagesCmd, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to list images")
	}

	return len(strings.TrimSpace(out)) > 0, nil
}

// DockerPullImage pulls Docker image from DockerHub registry.
func DockerPullImage(r Runner, dockerImage string) error {
	dockerPullImageCmd := SUDO + "docker pull " + dockerImage

	_, err := r.Run(dockerPullImageCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to pull images")
	}

	return err
}
