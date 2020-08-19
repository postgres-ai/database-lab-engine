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
	"github.com/sethvargo/go-password/password"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools/fs"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	// RestoreJobType defines the physical job type.
	RestoreJobType = "physical-restore"

	restoreContainerPrefix = "dblab_phr_"
	readyLogLine           = "database system is ready to accept"
	defaultPgConfigsDir    = "default"
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
	Tool         string            `yaml:"tool"`
	DockerImage  string            `yaml:"dockerImage"`
	Envs         map[string]string `yaml:"envs"`
	WALG         walgOptions       `yaml:"walg"`
	CustomTool   customOptions     `yaml:"customTool"`
	SyncInstance bool              `yaml:"syncInstance"`
}

// restorer describes the interface of tools for physical restore.
type restorer interface {
	// GetEnvVariables returns restorer environment variables.
	GetEnvVariables() []string

	// GetMounts returns restorer volume configurations for mounting.
	GetMounts() []mount.Mount

	// GetRestoreCommand returns a command to restore data.
	GetRestoreCommand() string

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
		return newWalg(r.globalCfg.DataDir, r.WALG), nil

	case customTool:
		return newCustomTool(r.CustomTool), nil
	}

	return nil, errors.Errorf("unknown restore tool given: %v", tool)
}

func (r *RestoreJob) restoreContainerName() string {
	return restoreContainerPrefix + r.globalCfg.InstanceID
}

// Name returns a name of the job.
func (r *RestoreJob) Name() string {
	return r.name
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) (err error) {
	log.Msg(fmt.Sprintf("Run job: %s. Options: %v", r.Name(), r.CopyOptions))

	defer func() {
		if err != nil && r.CopyOptions.SyncInstance {
			if syncErr := r.runSyncInstance(ctx); syncErr != nil {
				log.Err("Failed to run sync instance", syncErr)
			}
		}
	}()

	isEmpty, err := tools.IsEmptyDirectory(r.globalCfg.DataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if !isEmpty {
		log.Msg("Data directory is not empty. Skipping physical restore.")

		return nil
	}

	if err := tools.PullImage(ctx, r.dockerClient, r.CopyOptions.DockerImage); err != nil {
		return errors.Wrap(err, "failed to scan image pulling response")
	}

	contID, err := r.startContainer(ctx, r.restoreContainerName())
	if err != nil {
		return errors.Wrapf(err, "failed to create container: %s", r.restoreContainerName())
	}

	defer tools.RemoveContainer(ctx, r.dockerClient, contID, tools.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, r.dockerClient, r.restoreContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", r.restoreContainerName(), contID))

	if err = r.dockerClient.ContainerStart(ctx, contID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container: %v", contID)
	}

	log.Msg("Running restore command")

	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"bash", "-c", r.restorer.GetRestoreCommand()},
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

	// Prepare configuration files.
	sourceConfigDir, err := util.GetConfigPath(path.Join(defaultPgConfigsDir, pgVersion))
	if err != nil {
		return errors.Wrap(err, "cannot get path to default configs")
	}

	if err := fs.CopyDirectoryContent(sourceConfigDir, r.globalCfg.DataDir); err != nil {
		return errors.Wrap(err, "failed to set default configuration files")
	}

	if err := configuration.Run(r.globalCfg.DataDir); err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	if err := r.adjustRecoveryConfiguration(pgVersion, r.globalCfg.DataDir); err != nil {
		return err
	}

	// Set permissions.
	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", r.globalCfg.DataDir},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	// Start PostgreSQL instance.
	startCommand, err := r.dockerClient.ContainerExecCreate(ctx, contID, startingPostgresConfig(r.globalCfg.DataDir))

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	log.Msg("Running refresh command")

	attachResponse, err := r.dockerClient.ContainerExecAttach(ctx, startCommand.ID, types.ExecStartCheck{})
	if err != nil {
		return errors.Wrap(err, "failed to attach to the exec command")
	}

	defer attachResponse.Close()

	if err := isDatabaseReady(attachResponse.Reader); err != nil {
		return errors.Wrap(err, "failed to refresh data")
	}

	log.Msg("Refresh command has been finished")

	return nil
}

func (r *RestoreJob) startContainer(ctx context.Context, containerName string) (string, error) {
	hostConfig, err := r.buildHostConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to build container host config")
	}

	pwd, err := password.Generate(tools.PasswordLength, tools.PasswordMinDigits, tools.PasswordMinSymbols, false, true)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	syncInstance, err := r.dockerClient.ContainerCreate(ctx,
		r.buildContainerConfig(pwd),
		hostConfig,
		&network.NetworkingConfig{},
		containerName,
	)

	if err != nil {
		return "", errors.Wrap(err, "failed to start sync container")
	}

	if err = r.dockerClient.ContainerStart(ctx, syncInstance.ID, types.ContainerStartOptions{}); err != nil {
		return "", errors.Wrap(err, "failed to start sync container")
	}

	return syncInstance.ID, nil
}

func startingPostgresConfig(pgDataDir string) types.ExecConfig {
	return types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"postgres", "-D", pgDataDir},
		User:         defaults.Username,
	}
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

	return errors.New("database instance is not running")
}

func (r *RestoreJob) syncInstanceName() string {
	return tools.SyncInstanceContainerPrefix + r.globalCfg.InstanceID
}

func (r *RestoreJob) runSyncInstance(ctx context.Context) error {
	syncContainer, err := r.dockerClient.ContainerInspect(ctx, r.syncInstanceName())
	if err != nil && !client.IsErrNotFound(err) {
		return errors.Wrap(err, "failed to inspect sync container")
	}

	if syncContainer.ContainerJSONBase != nil {
		if syncContainer.State.Running {
			log.Msg("Sync instance is already running")
			return nil
		}

		log.Msg("Removing non-running sync instance")

		tools.RemoveContainer(ctx, r.dockerClient, syncContainer.ID, tools.StopTimeout)
	}

	log.Msg("Starting sync instance: ", r.syncInstanceName())

	syncInstanceID, err := r.startContainer(ctx, r.syncInstanceName())
	if err != nil {
		return err
	}

	startSyncCommand, err := r.dockerClient.ContainerExecCreate(ctx, syncInstanceID, startingPostgresConfig(r.globalCfg.DataDir))
	if err != nil {
		return errors.Wrap(err, "failed to create exec command")
	}

	if err = r.dockerClient.ContainerExecStart(ctx, startSyncCommand.ID, types.ExecStartCheck{Tty: true}); err != nil {
		return errors.Wrap(err, "failed to attach to exec command")
	}

	if err := tools.InspectCommandResponse(ctx, r.dockerClient, startSyncCommand.ID, startSyncCommand.ID); err != nil {
		return errors.Wrap(err, "failed to perform exec command")
	}

	return nil
}

func (r *RestoreJob) getEnvironmentVariables(password string) []string {
	envVariables := append([]string{
		"POSTGRES_PASSWORD=" + password,
		"PGDATA=" + r.globalCfg.DataDir,
	}, r.restorer.GetEnvVariables()...)

	for env, value := range r.Envs {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", env, value))
	}

	return envVariables
}

func (r *RestoreJob) buildContainerConfig(password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{"label": tools.DBLabControlLabel},
		Env:    r.getEnvironmentVariables(password),
		Image:  r.DockerImage,
	}
}

func (r *RestoreJob) buildHostConfig() (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{
		Mounts: r.restorer.GetMounts(),
	}

	if err := tools.AddVolumesToHostConfig(hostConfig, r.globalCfg.DataDir); err != nil {
		return nil, err
	}

	return hostConfig, nil
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

	version, err := strconv.ParseFloat(pgVersion, 64)
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
