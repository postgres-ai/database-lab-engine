/*
2020 © Postgres.ai
*/

// Package physical provides jobs for physical initial operations.
package physical

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
)

const (
	// RestoreJobType defines the physical job type.
	RestoreJobType = "physical-restore"

	restoreContainerName = "retriever_physical_restore"
	restoreContainerPath = "/var/lib/postgresql/dblabdata"
)

// RestoreJob describes a job for physical restoring.
type RestoreJob struct {
	name         string
	dockerClient *client.Client
	globalCfg    *dblabCfg.Global
	restorer     restorer
	CopyOptions
}

// CopyOptions describes options for physical copying.
type CopyOptions struct {
	Tool        string            `yaml:"tool"`
	DockerImage string            `yaml:"dockerImage"`
	Envs        map[string]string `yaml:"envs"`
	WALG        walgOptions       `yaml:"walg"`
	CustomTool  customOptions     `yaml:"customTool"`
}

// restorer describes the interface of tools for physical restore.
type restorer interface {
	// GetEnvVariables returns restorer environment variables.
	GetEnvVariables() []string

	// GetMounts returns restorer volume configurations for mounting.
	GetMounts() []mount.Mount

	// GetRestoreCommand returns a command to restore data.
	GetRestoreCommand() []string
}

// NewJob creates a new physical restore job.
func NewJob(cfg config.JobConfig, docker *client.Client, global *dblabCfg.Global) (*RestoreJob, error) {
	physicalJob := &RestoreJob{
		name:         cfg.Name,
		dockerClient: docker,
		globalCfg:    global,
	}

	if err := options.Unmarshal(cfg.Options, &physicalJob.CopyOptions); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	restorer, err := physicalJob.getRestorer(physicalJob.Tool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init restorer")
	}

	physicalJob.restorer = restorer

	return physicalJob, nil
}

// getRestorer builds a tool to perform physical restoring.
func (r *RestoreJob) getRestorer(tool string) (restorer, error) {
	switch tool {
	case walgTool:
		return newWalg(r.WALG), nil

	case customTool:
		return newCustomTool(r.CustomTool), nil
	}

	return nil, errors.Errorf("unknown restore tool given: %v", tool)
}

// Name returns a name of the job.
func (r *RestoreJob) Name() string {
	return r.name
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) error {
	log.Msg(fmt.Sprintf("Run job: %s. Options: %v", r.Name(), r.CopyOptions))

	isEmpty, err := tools.IsEmptyDirectory(r.globalCfg.DataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if !isEmpty {
		return errors.New("the data directory is not empty. Clean the data directory before continue")
	}

	cont, err := r.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Env:   r.getEnvironmentVariables(),
			Image: r.DockerImage,
		},
		&container.HostConfig{
			Mounts: r.getMountVolumes(),
		},
		&network.NetworkingConfig{},
		restoreContainerName,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	defer func() {
		if err := r.dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			log.Err("Failed to remove container: ", err)

			return
		}

		log.Msg(fmt.Sprintf("Stop container: %s. ID: %v", restoreContainerName, cont.ID))
	}()

	if err = r.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", restoreContainerName, cont.ID))

	execCommand, err := r.dockerClient.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          r.restorer.GetRestoreCommand(),
	})

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	log.Msg("Running restore command")

	attachResponse, err := r.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return errors.Wrap(err, "failed to attach to the exec command")
	}

	defer attachResponse.Close()

	if err := waitForCommandResponse(ctx, attachResponse); err != nil {
		return errors.Wrap(err, "failed to exec the command")
	}

	if err := tools.InspectCommandResponse(ctx, r.dockerClient, cont.ID, execCommand.ID); err != nil {
		return errors.Wrap(err, "failed to exec the restore command")
	}

	log.Msg("Restoring job has been finished")

	return nil
}

func (r *RestoreJob) getEnvironmentVariables() []string {
	envVariables := append([]string{"POSTGRES_PASSWORD=password"}, r.restorer.GetEnvVariables()...)

	for env, value := range r.Envs {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", env, value))
	}

	return envVariables
}

func (r *RestoreJob) getMountVolumes() []mount.Mount {
	mounts := append(
		[]mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: r.globalCfg.DataDir,
				Target: restoreContainerPath,
				BindOptions: &mount.BindOptions{
					Propagation: mount.PropagationRShared,
				},
			},
		},
		r.restorer.GetMounts()...)

	return mounts
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
