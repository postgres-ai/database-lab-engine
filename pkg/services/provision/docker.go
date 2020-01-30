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
	labelClone = "dblab-clone"
)

// DockerRunContainer runs specified container.
func DockerRunContainer(r Runner, c *PgConfig) (string, error) {
	dockerRunCmd := "sudo docker run " +
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
	dockerStopCmd := "sudo docker container stop " + c.CloneName

	return r.Run(dockerStopCmd, true)
}

// DockerRemoveContainer removes specified container.
func DockerRemoveContainer(r Runner, c *PgConfig) (string, error) {
	dockerRemoveCmd := "sudo docker container rm " + c.CloneName

	return r.Run(dockerRemoveCmd, true)
}

// DockerListContainers lists containers.
func DockerListContainers(r Runner) ([]string, error) {
	dockerListCmd := "sudo docker container ls " +
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
	dockerLogsCmd := "sudo docker logs " + c.CloneName + " " +
		"--since " + strconv.FormatUint(uint64(sinceRelMins), 10) + "m " +
		"--timestamps"

	return r.Run(dockerLogsCmd, true)
}

// DockerExec executes command on specified container.
func DockerExec(r Runner, c *PgConfig, cmd string) (string, error) {
	dockerExecCmd := "sudo docker exec " + c.CloneName + " " + cmd

	return r.Run(dockerExecCmd, true)
}

// DockerImageExists checks existence of Docker image.
func DockerImageExists(r Runner, dockerImage string) (bool, error) {
	dockerListImagesCmd := "sudo docker images " + dockerImage + " --quiet"

	out, err := r.Run(dockerListImagesCmd, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to list images")
	}

	return len(strings.TrimSpace(out)) > 0, nil
}

// DockerPullImage pulls Docker image from DockerHub registry.
func DockerPullImage(r Runner, dockerImage string) error {
	dockerPullImageCmd := "sudo docker pull " + dockerImage

	_, err := r.Run(dockerPullImageCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to pull images")
	}

	return err
}
