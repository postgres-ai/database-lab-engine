/*
2020 © Postgres.ai
*/

// Package physical provides jobs for physical initial operations.
package physical

import (
	"bufio"
	"bytes"
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
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/fs"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	// RestoreJobType defines the physical job type.
	RestoreJobType = "physicalRestore"

	restoreContainerPrefix = "dblab_phr_"
	readyLogLine           = "database system is ready to accept"
	defaultPgConfigsDir    = "default"
)

var (
	// List of original parameters to synchronize on restore.
	originalParamsToRestore = map[string]string{
		"max_connections":        "max_connections",
		"max_prepared_xacts":     "max_prepared_transactions",
		"max_locks_per_xact":     "max_locks_per_transaction",
		"max_worker_processes":   "max_worker_processes",
		"track_commit_timestamp": "track_commit_timestamp",
	}
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
	// GetRestoreCommand returns a command to restore data.
	GetRestoreCommand() string

	// GetRecoveryConfig returns a recovery config to restore data.
	GetRecoveryConfig(version float64) []byte
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
		return newWALG(r.globalCfg.DataDir(), r.WALG), nil

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
		if err == nil && r.CopyOptions.SyncInstance {
			if syncErr := r.runSyncInstance(ctx); syncErr != nil {
				log.Err("Failed to run sync instance", syncErr)
			}
		}
	}()

	dataDir := r.globalCfg.DataDir()

	isEmpty, err := tools.IsEmptyDirectory(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if !isEmpty {
		log.Msg("Data directory is not empty. Skipping physical restore.")

		return nil
	}

	contID, err := r.startContainer(ctx, r.restoreContainerName(), cont.DBLabRestoreLabel)
	if err != nil {
		return errors.Wrapf(err, "failed to create container: %s", r.restoreContainerName())
	}

	defer tools.RemoveContainer(ctx, r.dockerClient, contID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, r.dockerClient, r.restoreContainerName(), err)
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", r.restoreContainerName(), contID))

	if err = r.dockerClient.ContainerStart(ctx, contID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container: %v", contID)
	}

	log.Msg("Running restore command: ", r.restorer.GetRestoreCommand())

	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"bash", "-c", r.restorer.GetRestoreCommand() + " >& /proc/1/fd/1"},
	}); err != nil {
		return errors.Wrap(err, "failed to restore data")
	}

	log.Msg("Restoring job has been finished")

	if err := r.markDatabaseData(); err != nil {
		log.Err("Failed to mark database data: ", err)
	}

	pgVersion, err := tools.DetectPGVersion(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	// Prepare configuration files.
	sourceConfigDir, err := util.GetConfigPath(path.Join(defaultPgConfigsDir, pgVersion))
	if err != nil {
		return errors.Wrap(err, "cannot get path to default configs")
	}

	if err := fs.CopyDirectoryContent(sourceConfigDir, dataDir); err != nil {
		return errors.Wrap(err, "failed to set default configuration files")
	}

	if err := configuration.Run(dataDir); err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	if err := r.adjustRecoveryConfiguration(pgVersion, dataDir); err != nil {
		return err
	}

	// Set permissions.
	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", dataDir},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	// Apply important initial configs.
	if err := r.applyInitParams(ctx, contID, pgVersion, dataDir); err != nil {
		return errors.Wrap(err, "failed to adjust by init parameters")
	}

	log.Msg("Configuration has been finished")

	return nil
}

func (r *RestoreJob) startContainer(ctx context.Context, containerName, containerLabel string) (string, error) {
	hostConfig, err := r.buildHostConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to build container host config")
	}

	pwd, err := password.Generate(tools.PasswordLength, tools.PasswordMinDigits, tools.PasswordMinSymbols, false, true)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	if err := tools.PullImage(ctx, r.dockerClient, r.CopyOptions.DockerImage); err != nil {
		return "", errors.Wrap(err, "failed to scan image pulling response")
	}

	syncInstance, err := r.dockerClient.ContainerCreate(ctx,
		r.buildContainerConfig(pwd, containerLabel),
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

func (r *RestoreJob) syncInstanceName() string {
	return cont.SyncInstanceContainerPrefix + r.globalCfg.InstanceID
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

		tools.RemoveContainer(ctx, r.dockerClient, syncContainer.ID, cont.StopPhysicalTimeout)
	}

	log.Msg("Starting sync instance: ", r.syncInstanceName())

	syncInstanceID, err := r.startContainer(ctx, r.syncInstanceName(), cont.DBLabSyncLabel)
	if err != nil {
		return err
	}

	log.Msg("Starting PostgreSQL")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, r.syncInstanceName()))

	if err := tools.RunPostgres(ctx, r.dockerClient, syncInstanceID, r.globalCfg.DataDir()); err != nil {
		return errors.Wrap(err, "failed to start PostgreSQL instance")
	}

	logs, err := r.dockerClient.ContainerLogs(ctx, syncInstanceID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get container logs")
	}

	defer func() { _ = logs.Close() }()

	if err := isDatabaseReady(logs); err != nil {
		return errors.Wrap(err, "failed to run PostgreSQL")
	}

	return nil
}

func isDatabaseReady(input io.Reader) error {
	scanner := bufio.NewScanner(input)

	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

LOOP:
	for {
		select {
		case <-timer.C:
			return errors.New("timeout exceeded")
		default:
			if !scanner.Scan() {
				break LOOP
			}

			timer.Reset(time.Minute)
		}

		text := scanner.Text()

		if strings.Contains(text, readyLogLine) {
			log.Msg(text)
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return errors.New("database instance is not running")
}

func (r *RestoreJob) getEnvironmentVariables(password string) []string {
	// Pass Database Lab environment variables.
	envVariables := append(os.Environ(), []string{
		"POSTGRES_PASSWORD=" + password,
		"PGDATA=" + r.globalCfg.DataDir(),
	}...)

	// Add user-defined environment variables.
	for env, value := range r.Envs {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", env, value))
	}

	return envVariables
}

func (r *RestoreJob) buildContainerConfig(password, label string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    label,
			cont.DBLabInstanceIDLabel: r.globalCfg.InstanceID,
		},
		Env:   r.getEnvironmentVariables(password),
		Image: r.CopyOptions.DockerImage,
	}
}

func (r *RestoreJob) buildHostConfig(ctx context.Context) (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{}

	if err := tools.AddVolumesToHostConfig(ctx, r.dockerClient, hostConfig, r.globalCfg.DataDir()); err != nil {
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

	version, err := strconv.ParseFloat(pgVersion, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse PostgreSQL version")
	}

	if len(r.restorer.GetRecoveryConfig(version)) == 0 {
		return nil
	}

	if version >= defaults.PGVersion12 {
		if err := tools.TouchFile(path.Join(pgDataDir, "standby.signal")); err != nil {
			return err
		}

		recoveryFilename = "postgresql.conf"
	} else {
		recoveryFilename = "recovery.conf"
	}

	return appendConfigFile(path.Join(pgDataDir, recoveryFilename), r.restorer.GetRecoveryConfig(version))
}

func (r *RestoreJob) applyInitParams(ctx context.Context, contID, pgVersion, dataDir string) error {
	initConfCmd, err := r.dockerClient.ContainerExecCreate(ctx, contID, pgControlDataConfig(dataDir, pgVersion))

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	log.Msg("Check initial configs")

	attachResponse, err := r.dockerClient.ContainerExecAttach(ctx, initConfCmd.ID, types.ExecStartCheck{})
	if err != nil {
		return errors.Wrap(err, "failed to attach to the exec command")
	}

	defer attachResponse.Close()

	initParams, err := extractInitParams(ctx, attachResponse.Reader)
	if err != nil {
		return err
	}

	return appendInitConfigs(initParams, dataDir)
}

func pgControlDataConfig(pgDataDir, pgVersion string) types.ExecConfig {
	command := fmt.Sprintf("/usr/lib/postgresql/%s/bin/pg_controldata", pgVersion)

	return types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{command, "-D", pgDataDir},
		User:         defaults.Username,
		Env:          os.Environ(),
	}
}

func extractInitParams(ctx context.Context, read io.Reader) (map[string]string, error) {
	extractedConfigs := make(map[string]string)
	scanner := bufio.NewScanner(read)

	const settingSuffix = " setting:"

	for scanner.Scan() {
		if ctx.Err() != nil {
			return extractedConfigs, ctx.Err()
		}

		responseLine := scanner.Text()

		for param, configName := range originalParamsToRestore {
			extractedName := param + settingSuffix

			if !strings.HasPrefix(responseLine, extractedName) {
				continue
			}

			value := strings.TrimSpace(strings.TrimPrefix(responseLine, extractedName))

			extractedConfigs[configName] = value
		}

		if len(originalParamsToRestore) == len(extractedConfigs) {
			break
		}
	}

	return extractedConfigs, nil
}

func appendInitConfigs(initConfiguration map[string]string, pgDataDir string) error {
	if len(initConfiguration) == 0 {
		return nil
	}

	buffer := bytes.NewBuffer([]byte("\n"))

	for key, value := range initConfiguration {
		buffer.WriteString(fmt.Sprintf("%s = '%s'\n", key, value))
	}

	return appendConfigFile(path.Join(pgDataDir, "postgresql.conf"), buffer.Bytes())
}

func appendConfigFile(file string, data []byte) error {
	configFile, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer func() { _ = configFile.Close() }()

	if _, err := configFile.Write(data); err != nil {
		return err
	}

	return nil
}
