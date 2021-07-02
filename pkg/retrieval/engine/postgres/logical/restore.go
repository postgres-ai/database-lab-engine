/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

const (
	// RestoreJobType declares a job type for logical dumping.
	RestoreJobType = "logicalRestore"

	// const defines restore options.
	restoreContainerPrefix = "dblab_lr_"

	// defaultParallelJobs declares a default number of parallel jobs for logical dump and restore.
	defaultParallelJobs = 1
)

// RestoreJob defines a logical restore job.
type RestoreJob struct {
	name         string
	dockerClient *client.Client
	fsPool       *resources.Pool
	globalCfg    *global.Config
	dbMarker     *dbmarker.Marker
	dbMark       *dbmarker.Config
	RestoreOptions
}

// RestoreOptions defines a logical restore options.
type RestoreOptions struct {
	DumpLocation string            `yaml:"dumpLocation"`
	DockerImage  string            `yaml:"dockerImage"`
	DBName       string            `yaml:"dbname"`
	ForceInit    bool              `yaml:"forceInit"`
	ParallelJobs int               `yaml:"parallelJobs"`
	Partial      Partial           `yaml:"partial"`
	Configs      map[string]string `yaml:"configs"`
}

// Partial defines tables and rules for a partial logical restore.
type Partial struct {
	Tables []string `yaml:"tables"`
}

// NewJob create a new logical restore job.
func NewJob(cfg config.JobConfig, global *global.Config) (*RestoreJob, error) {
	restoreJob := &RestoreJob{
		name:         cfg.Spec.Name,
		dockerClient: cfg.Docker,
		fsPool:       cfg.FSPool,
		globalCfg:    global,
		dbMarker:     cfg.Marker,
		dbMark:       &dbmarker.Config{DataType: dbmarker.LogicalDataType},
	}

	if err := restoreJob.Reload(cfg.Spec.Options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	restoreJob.setDefaults()

	return restoreJob, nil
}

func (r *RestoreJob) setDefaults() {
	// TODO: Default yaml values in tags.
	if r.ParallelJobs == 0 {
		r.ParallelJobs = defaultParallelJobs
	}
}

func (r *RestoreJob) restoreContainerName() string {
	return restoreContainerPrefix + r.globalCfg.InstanceID
}

// Name returns a name of the job.
func (r *RestoreJob) Name() string {
	return r.name
}

// Reload reloads job configuration.
func (r *RestoreJob) Reload(cfg map[string]interface{}) (err error) {
	if err := options.Unmarshal(cfg, &r.RestoreOptions); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration options")
	}

	r.setDefaults()

	return nil
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) (err error) {
	log.Msg("Run job: ", r.Name())

	isEmpty, err := tools.IsEmptyDirectory(r.fsPool.DataDir())
	if err != nil {
		return errors.Wrapf(err, "failed to explore the data directory %q", r.fsPool.DataDir())
	}

	if !isEmpty {
		if !r.ForceInit {
			return errors.Errorf("the data directory %q is not empty. Use 'forceInit' or empty the data directory",
				r.fsPool.DataDir())
		}

		log.Msg(fmt.Sprintf("The data directory %q is not empty. Existing data may be overwritten.", r.fsPool.DataDir()))
	}

	if err := tools.PullImage(ctx, r.dockerClient, r.RestoreOptions.DockerImage); err != nil {
		return errors.Wrap(err, "failed to scan image pulling response")
	}

	hostConfig, err := r.buildHostConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	restoreCont, err := r.dockerClient.ContainerCreate(ctx,
		r.buildContainerConfig(pwd),
		hostConfig,
		&network.NetworkingConfig{},
		r.restoreContainerName(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create container %q", r.restoreContainerName())
	}

	defer tools.RemoveContainer(ctx, r.dockerClient, restoreCont.ID, cont.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, r.dockerClient, r.restoreContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", r.restoreContainerName(), restoreCont.ID))

	if err := r.dockerClient.ContainerStart(ctx, restoreCont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container %q", r.restoreContainerName())
	}

	dataDir := r.fsPool.DataDir()

	if len(r.RestoreOptions.Configs) > 0 {
		if err := updateConfigs(ctx, r.dockerClient, dataDir, restoreCont.ID, r.RestoreOptions.Configs); err != nil {
			return errors.Wrap(err, "failed to update configs")
		}
	}

	log.Msg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, r.dockerClient, restoreCont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	restoreCommand := r.buildLogicalRestoreCommand(r.DBName)
	log.Msg("Running restore command: ", restoreCommand)

	if len(r.Partial.Tables) > 0 {
		log.Msg("Partial restore will be run. Tables for restoring: ", strings.Join(r.Partial.Tables, ", "))
	}

	if err := tools.ExecCommand(ctx, r.dockerClient, restoreCont.ID, types.ExecConfig{Cmd: restoreCommand}); err != nil {
		return errors.Wrap(err, "failed to exec restore command")
	}

	if err := r.markDatabase(ctx, restoreCont.ID); err != nil {
		return errors.Wrap(err, "failed to mark the database")
	}

	analyzeCmd := buildAnalyzeCommand(
		Connection{Username: r.globalCfg.Database.User(), DBName: r.globalCfg.Database.Name()},
		r.RestoreOptions.ParallelJobs,
	)

	log.Msg("Running analyze command: ", analyzeCmd)

	if err := tools.ExecCommand(ctx, r.dockerClient, restoreCont.ID, types.ExecConfig{Cmd: analyzeCmd}); err != nil {
		return errors.Wrap(err, "failed to recalculate statistics after restore")
	}

	log.Msg("Restoring job has been finished")

	return nil
}

func (r *RestoreJob) buildContainerConfig(password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabRestoreLabel,
			cont.DBLabInstanceIDLabel: r.globalCfg.InstanceID,
		},
		Env: append(os.Environ(), []string{
			"PGDATA=" + r.fsPool.DataDir(),
			"POSTGRES_PASSWORD=" + password,
		}...),
		Image:       r.RestoreOptions.DockerImage,
		Healthcheck: health.GetConfig(r.globalCfg.Database.User(), r.globalCfg.Database.Name()),
	}
}

func (r *RestoreJob) buildHostConfig(ctx context.Context) (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{}

	if err := tools.AddVolumesToHostConfig(ctx, r.dockerClient, hostConfig, r.fsPool.DataDir()); err != nil {
		return nil, err
	}

	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: r.RestoreOptions.DumpLocation,
		Target: r.RestoreOptions.DumpLocation,
	})

	return hostConfig, nil
}

func (r *RestoreJob) markDatabase(ctx context.Context, contID string) error {
	dataStateAt, err := r.retrieveDataStateAt(ctx, contID)
	if err != nil {
		log.Err("Failed to extract dataStateAt: ", err)
	}

	if dataStateAt != "" {
		r.dbMark.DataStateAt = dataStateAt
	}

	if err := r.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	if err := r.dbMarker.SaveConfig(r.dbMark); err != nil {
		return errors.Wrap(err, "failed to mark the database")
	}

	r.updateDataStateAt()

	return nil
}

func (r *RestoreJob) retrieveDataStateAt(ctx context.Context, contID string) (string, error) {
	restoreMetaCmd := []string{"sh", "-c", "pg_restore --list " + r.RestoreOptions.DumpLocation + " | head -n 10"}

	log.Dbg("Running a restore metadata command: ", restoreMetaCmd)

	execCommand, err := r.dockerClient.ContainerExecCreate(ctx, contID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          restoreMetaCmd,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create a restore metadata command")
	}

	execAttach, err := r.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{})
	if err != nil {
		return "", errors.Wrap(err, "failed to exec a restore metadata command")
	}

	defer execAttach.Close()

	dataStateAt, err := tools.DiscoverDataStateAt(execAttach.Reader)
	if err != nil {
		return "", err
	}

	return dataStateAt, nil
}

// updateDataStateAt updates dataStateAt for in-memory representation of a storage pool.
func (r *RestoreJob) updateDataStateAt() {
	dsaTime, err := time.Parse(util.DataStateAtFormat, r.dbMark.DataStateAt)
	if err != nil {
		log.Err("Invalid value for DataStateAt: ", r.dbMark.DataStateAt)
		return
	}

	r.fsPool.SetDSA(dsaTime)
}

func (r *RestoreJob) buildLogicalRestoreCommand(dbName string) []string {
	restoreCmd := []string{"pg_restore", "--username", r.globalCfg.Database.User(), "--dbname", defaults.DBName,
		"--no-privileges", "--no-owner"}

	if dbName != defaults.DBName {
		// To avoid recreating of the default database.
		restoreCmd = append(restoreCmd, "--create")
	}

	if r.ForceInit {
		restoreCmd = append(restoreCmd, "--clean", "--if-exists")
	}

	restoreCmd = append(restoreCmd, "--jobs", strconv.Itoa(r.ParallelJobs))

	for _, table := range r.Partial.Tables {
		restoreCmd = append(restoreCmd, "--table", table)
	}

	restoreCmd = append(restoreCmd, r.RestoreOptions.DumpLocation)

	return restoreCmd
}
