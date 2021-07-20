/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"context"
	"fmt"
	"io/ioutil"
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

	// dumpMetafile describes metafile name of a custom dump.
	dumpMetafile = "toc.dat"

	// prefixDBName describes a prefix of database name inside of a custom dump metafile.
	prefixDBName = "dbname:"
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
	DumpLocation string                  `yaml:"dumpLocation"`
	DockerImage  string                  `yaml:"dockerImage"`
	Databases    map[string]DBDefinition `yaml:"databases"`
	ForceInit    bool                    `yaml:"forceInit"`
	ParallelJobs int                     `yaml:"parallelJobs"`
	Configs      map[string]string       `yaml:"configs"`
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

	log.Msg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, r.dockerClient, restoreCont.ID); err != nil {
		var errHealthCheck *tools.ErrHealthCheck
		if !errors.As(err, &errHealthCheck) {
			return errors.Wrap(err, "failed to readiness check")
		}

		if err := setupPGData(ctx, r.dockerClient, dataDir, restoreCont.ID); err != nil {
			return errors.Wrap(err, "failed to set up Postgres data")
		}
	}

	if len(r.RestoreOptions.Configs) > 0 {
		if err := updateConfigs(ctx, r.dockerClient, dataDir, restoreCont.ID, r.RestoreOptions.Configs); err != nil {
			return errors.Wrap(err, "failed to update configs")
		}
	}

	dbList, err := r.getDBList(ctx, r.RestoreOptions.DumpLocation, dumpMetafile, restoreCont.ID)
	if err != nil {
		return err
	}

	log.Dbg("Database List to restore: ", dbList)

	for dbName, dbDefinition := range dbList {
		if err := r.restoreDB(ctx, restoreCont.ID, dbName, dbDefinition); err != nil {
			return errors.Wrap(err, "failed to restore a database")
		}
	}

	analyzeCmd := buildAnalyzeCommand(
		Connection{Username: r.globalCfg.Database.User(), DBName: r.globalCfg.Database.Name()},
		r.RestoreOptions.ParallelJobs,
	)

	log.Msg("Running analyze command: ", analyzeCmd)

	if err := tools.ExecCommand(ctx, r.dockerClient, restoreCont.ID, types.ExecConfig{Cmd: analyzeCmd}); err != nil {
		return errors.Wrap(err, "failed to recalculate statistics after restore")
	}

	if err := tools.StopPostgres(ctx, r.dockerClient, restoreCont.ID, dataDir, tools.DefaultStopTimeout); err != nil {
		return errors.Wrap(err, "failed to stop Postgres instance")
	}

	log.Msg("Restoring job has been finished")

	return nil
}

func (r *RestoreJob) getDBList(ctx context.Context, dumpLocation, dumpFileMeta, contID string) (map[string]DBDefinition, error) {
	if len(r.Databases) > 0 {
		return r.Databases, nil
	}

	dumpMetafilePath := path.Join(dumpLocation, dumpFileMeta)

	if _, err := os.Stat(dumpMetafilePath); err != nil {
		if os.IsNotExist(err) {
			return r.discoverDumpDirectories(dumpLocation)
		}

		return nil, err
	}

	return r.discoverDumpLocation(ctx, contID, dumpMetafilePath)
}

// discoverDumpDirectories discovers a dump location when a metafile of a custom dump format does not exist.
func (r *RestoreJob) discoverDumpDirectories(dumpLocation string) (map[string]DBDefinition, error) {
	fileInfos, err := ioutil.ReadDir(dumpLocation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to discover dumped directories")
	}

	dbList := map[string]DBDefinition{}

	for _, info := range fileInfos {
		if info.IsDir() {
			dbList[info.Name()] = DBDefinition{
				Format: directoryFormat,
			}
		}
	}

	return dbList, nil
}

// discoverDumpLocation discovers a dump location when a metafile exists in order to to extract a database name.
func (r *RestoreJob) discoverDumpLocation(ctx context.Context, contID, dumpMetaPath string) (map[string]DBDefinition, error) {
	extractDBNameCmd := fmt.Sprintf("pg_restore --list %s | grep %s | tr -d '[;]'", dumpMetaPath, prefixDBName)
	log.Msg("Extract a database name: ", extractDBNameCmd)

	outputLine, err := tools.ExecCommandWithOutput(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"bash", "-c", extractDBNameCmd},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find a database name to restore")
	}

	if outputLine == "" {
		return nil, errors.New("a database name to restore not found")
	}

	dbName := strings.TrimSpace(strings.TrimPrefix(outputLine, prefixDBName))
	dbList := map[string]DBDefinition{
		dbName: {Format: customFormat},
	}

	return dbList, nil
}

func (r *RestoreJob) restoreDB(ctx context.Context, contID, dbName string, dbDefinition DBDefinition) error {
	restoreCommand := r.buildLogicalRestoreCommand(dbName, dbDefinition)
	log.Msg("Running restore command: ", restoreCommand)

	if err := tools.ExecCommand(ctx, r.dockerClient, contID, types.ExecConfig{Cmd: restoreCommand}); err != nil {
		return errors.Wrap(err, "failed to exec restore command")
	}

	dumpLocation := r.getDumpLocation(dbDefinition.Format, dbName)

	if err := r.markDatabase(ctx, contID, dumpLocation); err != nil {
		return errors.Wrap(err, "failed to mark the database")
	}

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

func (r *RestoreJob) markDatabase(ctx context.Context, contID, dumpLocation string) error {
	dataStateAt, err := r.retrieveDataStateAt(ctx, contID, dumpLocation)
	if err != nil {
		log.Err("Failed to extract dataStateAt: ", err)
	}

	if dataStateAt != "" {
		r.dbMark.DataStateAt = dataStateAt
		log.Msg("Data state at: ", dataStateAt)
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

func (r *RestoreJob) retrieveDataStateAt(ctx context.Context, contID, dumpLocation string) (string, error) {
	restoreMetaCmd := []string{"sh", "-c", "pg_restore --list " + dumpLocation + " | head -n 10"}

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

func (r *RestoreJob) buildLogicalRestoreCommand(dbName string, definition DBDefinition) []string {
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

	if len(definition.Tables) > 0 {
		log.Msg("Partial restore will be run. Tables for restoring: ", strings.Join(definition.Tables, ", "))

		for _, table := range definition.Tables {
			restoreCmd = append(restoreCmd, "--table", table)
		}
	}

	restoreCmd = append(restoreCmd, r.getDumpLocation(definition.Format, dbName))

	return restoreCmd
}

func (r *RestoreJob) getDumpLocation(dumpFormat, dbName string) string {
	switch dumpFormat {
	case customFormat, plainFormat:
		return r.RestoreOptions.DumpLocation

	default:
		return path.Join(r.RestoreOptions.DumpLocation, dbName)
	}
}
