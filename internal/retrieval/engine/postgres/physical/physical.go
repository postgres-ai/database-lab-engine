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
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v2/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/engine/postgres/tools/pgtool"
	"gitlab.com/postgres-ai/database-lab/v2/internal/retrieval/options"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

const (
	// RestoreJobType defines the physical job type.
	RestoreJobType = "physicalRestore"

	restoreContainerPrefix = "dblab_phr_"
)

var (
	// List of original parameters to synchronize on restore.
	originalParamsToRestore = map[string]string{
		"max_connections":        "max_connections",
		"max_prepared_xacts":     "max_prepared_transactions",
		"max_locks_per_xact":     "max_locks_per_transaction",
		"max_worker_processes":   "max_worker_processes",
		"track_commit_timestamp": "track_commit_timestamp",
		"max_wal_senders":        "max_wal_senders",
	}
)

// RestoreJob describes a job for physical restoring.
type RestoreJob struct {
	name         string
	dockerClient *client.Client
	fsPool       *resources.Pool
	globalCfg    *global.Config
	engineProps  global.EngineProps
	dbMarker     *dbmarker.Marker
	restorer     restorer
	CopyOptions
}

// CopyOptions describes options for physical copying.
type CopyOptions struct {
	Tool            string                 `yaml:"tool"`
	DockerImage     string                 `yaml:"dockerImage"`
	ContainerConfig map[string]interface{} `yaml:"containerConfig"`
	Envs            map[string]string      `yaml:"envs"`
	WALG            walgOptions            `yaml:"walg"`
	CustomTool      customOptions          `yaml:"customTool"`
	Sync            Sync                   `yaml:"sync"`
}

// Sync describes sync instance options.
type Sync struct {
	Enabled     bool              `yaml:"enabled"`
	HealthCheck HealthCheck       `yaml:"healthCheck"`
	Configs     map[string]string `yaml:"configs"`
	Recovery    map[string]string `yaml:"recovery"`
}

// HealthCheck describes health check options of a sync instance.
type HealthCheck struct {
	Interval   int64 `yaml:"interval"`
	MaxRetries int   `yaml:"maxRetries"`
}

// restorer describes the interface of tools for physical restore.
type restorer interface {
	// GetRestoreCommand returns a command to restore data.
	GetRestoreCommand() string

	// GetRecoveryConfig returns a recovery config to restore data.
	GetRecoveryConfig(version float64) map[string]string
}

// NewJob creates a new physical restore job.
func NewJob(cfg config.JobConfig, global *global.Config, engineProps global.EngineProps) (*RestoreJob, error) {
	physicalJob := &RestoreJob{
		name:         cfg.Spec.Name,
		dockerClient: cfg.Docker,
		globalCfg:    global,
		engineProps:  engineProps,
		dbMarker:     cfg.Marker,
		fsPool:       cfg.FSPool,
	}

	if err := physicalJob.Reload(cfg.Spec.Options); err != nil {
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
		return newWALG(r.fsPool.DataDir(), r.WALG), nil

	case customTool:
		return newCustomTool(r.CustomTool), nil
	}

	return nil, errors.Errorf("unknown restore tool given: %v", tool)
}

func (r *RestoreJob) restoreContainerName() string {
	return restoreContainerPrefix + r.engineProps.InstanceID
}

// Name returns a name of the job.
func (r *RestoreJob) Name() string {
	return r.name
}

// Reload reloads job configuration.
func (r *RestoreJob) Reload(cfg map[string]interface{}) (err error) {
	return options.Unmarshal(cfg, &r.CopyOptions)
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) (err error) {
	log.Msg("Run job: ", r.Name())

	defer func() {
		if err == nil && r.CopyOptions.Sync.Enabled {
			go func() {
				if syncErr := r.runSyncInstance(ctx); syncErr != nil {
					log.Err("Failed to run sync instance: ", syncErr)

					if ctx.Err() != nil {
						// if context was canceled
						// - we can't use shared context
						// - main routine will stop container
						return
					}

					tools.StopContainer(ctx, r.dockerClient, r.syncInstanceName(), time.Second)
				}
			}()
		}
	}()

	dataDir := r.fsPool.DataDir()

	isEmpty, err := tools.IsEmptyDirectory(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if !isEmpty {
		log.Msg("Data directory is not empty. Skipping physical restore.")

		return nil
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	contID, err := r.startContainer(ctx, r.restoreContainerName(), r.buildContainerConfig(cont.DBLabRestoreLabel, pwd))
	if err != nil {
		return err
	}

	defer tools.RemoveContainer(ctx, r.dockerClient, contID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, r.dockerClient, r.restoreContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", r.restoreContainerName(), contID))

	if err = r.dockerClient.ContainerStart(ctx, contID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container: %v", contID)
	}

	log.Msg("Running restore command: ", r.restorer.GetRestoreCommand())
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, r.restoreContainerName()))

	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"bash", "-c", r.restorer.GetRestoreCommand() + " >& /proc/1/fd/1"},
	}); err != nil {
		return errors.Wrap(err, "failed to restore data")
	}

	log.Msg("Restoring job has been finished")

	if err := r.markDatabaseData(); err != nil {
		log.Err("Failed to mark database data: ", err)
	}

	cfgManager, err := pgconfig.NewCorrector(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to create a config manager")
	}

	// Apply important pg_control configs.
	pgControlParams, err := r.getPgControlParams(ctx, contID, dataDir, cfgManager.GetPgVersion())
	if err != nil {
		return errors.Wrap(err, "failed to adjust by init parameters")
	}

	if err := cfgManager.ApplyPgControl(pgControlParams); err != nil {
		return errors.Wrap(err, "failed to adjust pg_control parameters")
	}

	// Apply sync instance configs.
	if syncConfig := r.CopyOptions.Sync.Configs; len(syncConfig) > 0 {
		if err := cfgManager.ApplySync(syncConfig); err != nil {
			return errors.Wrap(err, "cannot update sync instance configs")
		}
	}

	// Adjust recovery configuration.
	if err := cfgManager.AdjustRecoveryFiles(); err != nil {
		return err
	}

	if err := cfgManager.ApplyRecovery(r.buildRecoveryConf(cfgManager.GetPgVersion())); err != nil {
		return err
	}

	// Set permissions.
	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", dataDir},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	log.Msg("Configuration has been finished")

	return nil
}

func (r *RestoreJob) startContainer(ctx context.Context, containerName string, containerConfig *container.Config) (string, error) {
	hostConfig, err := cont.BuildHostConfig(ctx, r.dockerClient, r.fsPool.DataDir(), r.CopyOptions.ContainerConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to build container host config")
	}

	if err := tools.PullImage(ctx, r.dockerClient, r.CopyOptions.DockerImage); err != nil {
		return "", err
	}

	newContainer, err := r.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, &network.NetworkingConfig{}, nil,
		containerName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create container %s", containerName)
	}

	if err = r.dockerClient.ContainerStart(ctx, newContainer.ID, types.ContainerStartOptions{}); err != nil {
		return "", errors.Wrapf(err, "failed to start container %s", containerName)
	}

	return newContainer.ID, nil
}

func (r *RestoreJob) syncInstanceName() string {
	return cont.SyncInstanceContainerPrefix + r.engineProps.InstanceID
}

func (r *RestoreJob) runSyncInstance(ctx context.Context) (err error) {
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

	syncInstanceConfig, err := r.buildSyncInstanceConfig()
	if err != nil {
		return errors.Wrap(err, "failed to build a sync instance config")
	}

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, r.dockerClient, r.syncInstanceName())
			tools.PrintLastPostgresLogs(ctx, r.dockerClient, r.syncInstanceName(), r.fsPool.DataDir())
		}
	}()

	log.Msg("Starting sync instance: ", r.syncInstanceName())

	syncInstanceID, err := r.startContainer(ctx, r.syncInstanceName(), syncInstanceConfig)
	if err != nil {
		return err
	}

	log.Msg("Starting PostgreSQL and waiting for readiness")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, r.syncInstanceName()))

	if err := tools.CheckContainerReadiness(ctx, r.dockerClient, syncInstanceID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	log.Msg("Sync instance has been running")

	return nil
}

func (r *RestoreJob) buildSyncInstanceConfig() (*container.Config, error) {
	pwd, err := tools.GeneratePassword()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	hcInterval := health.DefaultRestoreInterval
	hcRetries := health.DefaultRestoreRetries

	if r.CopyOptions.Sync.HealthCheck.Interval != 0 {
		hcInterval = time.Duration(r.CopyOptions.Sync.HealthCheck.Interval) * time.Second
	}

	if r.CopyOptions.Sync.HealthCheck.MaxRetries != 0 {
		hcRetries = r.CopyOptions.Sync.HealthCheck.MaxRetries
	}

	return r.buildContainerConfigWithHealthCheck(cont.DBLabSyncLabel, pwd,
		health.OptionInterval(hcInterval), health.OptionRetries(hcRetries)), nil
}

func (r *RestoreJob) buildContainerConfigWithHealthCheck(label, password string, hcOptions ...health.ContainerOption) *container.Config {
	containerCfg := r.buildContainerConfig(label, password)
	containerCfg.Healthcheck = health.GetConfig(r.globalCfg.Database.User(), r.globalCfg.Database.Name(), hcOptions...)

	return containerCfg
}

func (r *RestoreJob) buildContainerConfig(label, password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    label,
			cont.DBLabInstanceIDLabel: r.engineProps.InstanceID,
			cont.DBLabEngineNameLabel: r.engineProps.ContainerName,
		},
		Env:   r.getEnvironmentVariables(password),
		Image: r.CopyOptions.DockerImage,
	}
}

func (r *RestoreJob) getEnvironmentVariables(password string) []string {
	// Pass Database Lab environment variables.
	envVariables := append(os.Environ(), []string{
		"POSTGRES_PASSWORD=" + password,
		"PGDATA=" + r.fsPool.DataDir(),
	}...)

	// Add user-defined environment variables.
	for env, value := range r.Envs {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", env, value))
	}

	return envVariables
}

func (r *RestoreJob) buildRecoveryConf(pgVersion float64) map[string]string {
	recoveryConf := r.restorer.GetRecoveryConfig(pgVersion)

	// Add user-defined recovery configuration.
	if len(r.Sync.Recovery) > 0 {
		for recoveryKey, recoveryValue := range r.Sync.Recovery {
			recoveryConf[recoveryKey] = recoveryValue
		}
	}

	return recoveryConf
}

func (r *RestoreJob) markDatabaseData() error {
	if err := r.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return r.dbMarker.SaveConfig(&dbmarker.Config{DataType: dbmarker.PhysicalDataType})
}

func (r *RestoreJob) getPgControlParams(ctx context.Context, contID, dataDir string, pgVersion float64) (map[string]string, error) {
	log.Msg("Check pg_controldata configuration options")

	attachResponse, err := pgtool.ReadControlData(ctx, r.dockerClient, contID, dataDir, pgVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to attach to the exec command")
	}

	defer attachResponse.Close()

	return extractControlDataParams(ctx, attachResponse.Reader)
}

func extractControlDataParams(ctx context.Context, read io.Reader) (map[string]string, error) {
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
