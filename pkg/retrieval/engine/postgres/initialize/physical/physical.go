/*
2020 © Postgres.ai
*/

// Package physical provides jobs for physical initial operations.
package physical

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
)

const (
	// RestoreJobType defines the physical job type.
	RestoreJobType = "physical-restore"

	restoreContainerName = "retriever_physical_restore"
	restoreContainerPath = "/var/lib/postgresql/dblabdata"

	readyLogLine = "database system is ready to accept"
)

// RestoreJob describes a job for physical restoring.
type RestoreJob struct {
	name         string
	dockerClient *client.Client
	globalCfg    *dblabCfg.Global
	dbMarker     *dbmarker.Marker
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

	// GetRecoveryConfig returns a recovery config to restore data.
	GetRecoveryConfig() []byte
}

// NewJob creates a new physical restore job.
func NewJob(cfg config.JobConfig, docker *client.Client, global *dblabCfg.Global, marker *dbmarker.Marker) (*RestoreJob, error) {
	physicalJob := &RestoreJob{
		name:         cfg.Name,
		dockerClient: docker,
		globalCfg:    global,
		dbMarker:     marker,
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

	defer tools.RemoveContainer(ctx, r.dockerClient, cont.ID, tools.StopTimeout)

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", restoreContainerName, cont.ID))

	if err = r.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg("Running restore command")

	if err := tools.ExecCommand(ctx, r.dockerClient, cont.ID, types.ExecConfig{
		Cmd: r.restorer.GetRestoreCommand(),
	}); err != nil {
		return errors.Wrap(err, "failed to restore data")
	}

	log.Msg("Restoring job has been finished")

	if err := r.markDatabaseData(); err != nil {
		log.Err("Failed to mark database data: ", err)
	}

	pgVersion, err := tools.DetectPGVersion(r.globalCfg.DataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	if err := configuration.Run(r.globalCfg.DataDir); err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	if err := r.adjustRecoveryConfiguration(pgVersion, r.globalCfg.DataDir); err != nil {
		return err
	}

	// Set permissions.
	if err := tools.ExecCommand(ctx, r.dockerClient, cont.ID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", restoreContainerPath},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	// Start PostgreSQL instance.
	startCommand, err := r.dockerClient.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"postgres", "-D", restoreContainerPath},
		User:         defaults.Username,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	log.Msg("Running refresh command")

	attachResponse, err := r.dockerClient.ContainerExecAttach(ctx, startCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return errors.Wrap(err, "failed to attach to the exec command")
	}

	defer attachResponse.Close()

	if err := isDatabaseReady(attachResponse.Reader); err != nil {
		return errors.Wrap(err, "failed to refresh data")
	}

	log.Msg("Running restore command")

	return nil
}

func isDatabaseReady(input io.Reader) error {
	scanner := bufio.NewScanner(input)

	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for scanner.Scan() {
		select {
		case <-timer.C:
			return errors.New("timeout exceeded")
		default:
		}

		text := scanner.Text()

		if strings.Contains(text, readyLogLine) {
			return nil
		}

		fmt.Println(text)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return errors.New("not found")
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

func (r *RestoreJob) markDatabaseData() error {
	if err := r.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return r.dbMarker.SaveConfig(&dbmarker.Config{DataType: dbmarker.PhysicalDataType})
}

func (r *RestoreJob) adjustRecoveryConfiguration(pgVersion, pgDataDir string) error {
	// Remove postmaster.pid.
	if err := os.Remove(path.Join(pgDataDir, "postmaster.pid")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "failed to remove postmaster.pid")
	}

	// Truncate pg_ident.conf.
	if err := tools.TouchFile(path.Join(pgDataDir, "pg_ident.conf")); err != nil {
		return errors.Wrap(err, "failed to truncate pg_ident.conf")
	}

	// Replication mode.
	var recoveryFilename string

	if len(r.restorer.GetRecoveryConfig()) == 0 {
		return nil
	}

	version, err := strconv.Atoi(pgVersion)
	if err != nil {
		return errors.Wrap(err, "failed to parse PostgreSQL version")
	}

	const pgVersion12 = 12

	if version >= pgVersion12 {
		if err := tools.TouchFile(path.Join(pgDataDir, "standby.signal")); err != nil {
			return err
		}

		recoveryFilename = "postgresql.conf"
	} else {
		recoveryFilename = "recovery.conf"
	}

	recoveryFile, err := os.OpenFile(path.Join(pgDataDir, recoveryFilename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer func() { _ = recoveryFile.Close() }()

	if _, err := recoveryFile.Write(r.restorer.GetRecoveryConfig()); err != nil {
		return err
	}

	return nil
}
