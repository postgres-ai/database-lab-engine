/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"
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
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

const (
	// DumpJobType declares a job type for logical dumping.
	DumpJobType = "logicalDump"

	// Defines dump options.
	dumpContainerPrefix = "dblab_ld_"

	// Defines dump source types.
	sourceTypeLocal  = "local"
	sourceTypeRemote = "remote"
	sourceTypeRDS    = "rdsIam"

	// reservePort defines reserve port in case of a local dump.
	reservePort = 9999

	// Container network modes.
	networkModeDefault = container.NetworkMode("default")
	networkModeHost    = container.NetworkMode("host")

	// PostgreSQL pg_dump formats.
	customFormat    = "custom"
	directoryFormat = "directory"
)

// DumpJob declares a job for logical dumping.
type DumpJob struct {
	name         string
	dockerClient *client.Client
	fsPool       *resources.Pool
	globalCfg    *global.Config
	config       dumpJobConfig
	dumper       dumper
	dbMarker     *dbmarker.Marker
	dbMark       *dbmarker.Config
	DumpOptions
}

// DumpOptions defines a logical dump options.
type DumpOptions struct {
	DumpLocation string            `yaml:"dumpLocation"`
	DockerImage  string            `yaml:"dockerImage"`
	Connection   Connection        `yaml:"connection"`
	Source       Source            `yaml:"source"`
	Partial      Partial           `yaml:"partial"`
	ParallelJobs int               `yaml:"parallelJobs"`
	Restore      *ImmediateRestore `yaml:"immediateRestore,omitempty"`
}

// Source describes source of data to dump.
type Source struct {
	Type       string     `yaml:"type"`
	Connection Connection `yaml:"connection"`
	RDS        *RDSConfig `yaml:"rdsIam"`
}

type dumpJobConfig struct {
	db Connection
}

// dumper describes the interface to prepare environment for a logical dump.
type dumper interface {
	// GetEnvVariables returns dumper environment variables.
	GetCmdEnvVariables() []string

	// SetConnectionOptions sets connection options for dumping.
	SetConnectionOptions(context.Context, *Connection) error
}

// Connection provides connection options.
type Connection struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ImmediateRestore contains options for direct data restore without saving the dump file on disk.
type ImmediateRestore struct {
	ForceInit bool              `yaml:"forceInit"`
	Configs   map[string]string `yaml:"configs"`
}

// NewDumpJob creates a new DumpJob.
func NewDumpJob(jobCfg config.JobConfig, global *global.Config) (*DumpJob, error) {
	dumpJob := &DumpJob{
		name:         jobCfg.Spec.Name,
		dockerClient: jobCfg.Docker,
		fsPool:       jobCfg.FSPool,
		globalCfg:    global,
		dbMarker:     jobCfg.Marker,
		dbMark: &dbmarker.Config{
			DataType: dbmarker.LogicalDataType,
		},
	}

	if err := dumpJob.Reload(jobCfg.Spec.Options); err != nil {
		return nil, errors.Wrap(err, "failed to load job config")
	}

	if err := dumpJob.setupDumper(); err != nil {
		return nil, errors.Wrap(err, "failed to set up a dump helper")
	}

	return dumpJob, nil
}

func (d *DumpJob) validate() error {
	if d.Restore != nil && d.ParallelJobs > 1 {
		return errors.New(`parallel backup not supported for the direct restore. 
Either set 'numberOfJobs' equals to 1 or disable the restore section`)
	}

	return nil
}

func (d *DumpJob) setDefaults() {
	// TODO: Default yaml values in tags.
	if d.DumpOptions.Source.Connection.Port == 0 {
		d.DumpOptions.Source.Connection.Port = defaults.Port
	}

	if d.DumpOptions.Source.Connection.Username == "" {
		d.DumpOptions.Source.Connection.Username = defaults.Username
	}

	if d.DumpOptions.ParallelJobs == 0 {
		d.DumpOptions.ParallelJobs = defaultParallelJobs
	}
}

// setupDumper sets up a tool to perform physical restoring.
func (d *DumpJob) setupDumper() error {
	switch d.Source.Type {
	case sourceTypeLocal, sourceTypeRemote, "":
		d.dumper = newDefaultDumper()
		return nil

	case sourceTypeRDS:
		if d.Source.RDS == nil {
			return errors.New("the RDS IAM configuration section must not be empty when using the RDS IAM source type")
		}

		dumper, err := newRDSDumper(d.Source.RDS)
		if err != nil {
			return errors.Wrap(err, "failed to create an RDS IAM dumper")
		}

		d.dumper = dumper

		return nil
	}

	return errors.Errorf("unknown source type given: %v", d.Source.Type)
}

func (d *DumpJob) dumpContainerName() string {
	return dumpContainerPrefix + d.globalCfg.InstanceID
}

// Name returns a name of the job.
func (d *DumpJob) Name() string {
	return d.name
}

// Reload reloads job configuration.
func (d *DumpJob) Reload(cfg map[string]interface{}) (err error) {
	if err := options.Unmarshal(cfg, &d.DumpOptions); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration options")
	}

	if err := d.validate(); err != nil {
		return errors.Wrap(err, "invalid logical dump job")
	}

	d.setDefaults()

	return nil
}

// Run starts the job.
func (d *DumpJob) Run(ctx context.Context) (err error) {
	log.Msg("Run job: ", d.Name())

	isEmpty, err := tools.IsEmptyDirectory(d.fsPool.DataDir())
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if d.DumpOptions.Restore != nil && !isEmpty {
		if !d.DumpOptions.Restore.ForceInit {
			return errors.New("the data directory is not empty. Use 'forceInit' or empty the data directory")
		}

		log.Msg("The data directory is not empty. Existing data may be overwritten.")
	}

	if err := tools.PullImage(ctx, d.dockerClient, d.DockerImage); err != nil {
		return errors.Wrap(err, "failed to scan pulling image response")
	}

	if err := os.MkdirAll(d.DumpOptions.DumpLocation, 0666); err != nil {
		return errors.Wrap(err, "failed to create a location directory")
	}

	hostConfig, err := d.buildHostConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	dumpCont, err := d.dockerClient.ContainerCreate(ctx, d.buildContainerConfig(pwd), hostConfig, &network.NetworkingConfig{},
		d.dumpContainerName(),
	)
	if err != nil {
		log.Err(err)

		return errors.Wrapf(err, "failed to create container %q", d.dumpContainerName())
	}

	defer tools.RemoveContainer(ctx, d.dockerClient, dumpCont.ID, cont.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, d.dockerClient, d.dumpContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", d.dumpContainerName(), dumpCont.ID))

	if err := d.dockerClient.ContainerStart(ctx, dumpCont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container %q", d.dumpContainerName())
	}

	if err := d.setupConnectionOptions(ctx); err != nil {
		return errors.Wrap(err, "failed to setup connection options")
	}

	dataDir := d.fsPool.DataDir()

	if d.DumpOptions.Restore != nil && len(d.DumpOptions.Restore.Configs) > 0 {
		if err := updateConfigs(ctx, d.dockerClient, dataDir, dumpCont.ID, d.DumpOptions.Restore.Configs); err != nil {
			return errors.Wrap(err, "failed to update configs")
		}
	}

	log.Msg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, d.dockerClient, dumpCont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	dumpCommand := d.buildLogicalDumpCommand()
	log.Msg("Running dump command", dumpCommand)

	if len(d.Partial.Tables) > 0 {
		log.Msg("Partial dump will be run. Tables for dumping: ", strings.Join(d.Partial.Tables, ", "))
	}

	if d.DumpOptions.DumpLocation != "" && d.DumpOptions.Restore == nil {
		if out, err := tools.ExecCommandWithOutput(ctx, d.dockerClient, dumpCont.ID, types.ExecConfig{
			Cmd: []string{"rm", "-rf", path.Join(d.DumpOptions.DumpLocation, "*")},
		}); err != nil {
			log.Dbg(out)
			return errors.Wrap(err, "failed to clean up dump location")
		}
	}

	if err := d.performDumpCommand(ctx, dumpCont.ID, types.ExecConfig{
		Cmd: dumpCommand,
		Env: d.getExecEnvironmentVariables(),
	}); err != nil {
		return errors.Wrap(err, "failed to dump a database")
	}

	if d.DumpOptions.Restore != nil {
		if err := d.markDatabaseData(); err != nil {
			return errors.Wrap(err, "failed to mark the created dump")
		}

		analyzeCmd := buildAnalyzeCommand(
			Connection{Username: d.globalCfg.Database.User(), DBName: d.globalCfg.Database.Name()},
			d.DumpOptions.ParallelJobs,
		)

		log.Msg("Running analyze command: ", analyzeCmd)

		if err := tools.ExecCommand(ctx, d.dockerClient, dumpCont.ID, types.ExecConfig{Cmd: analyzeCmd}); err != nil {
			return errors.Wrap(err, "failed to recalculate statistics after restore")
		}
	}

	log.Msg("Dumping job has been finished")

	return nil
}

func updateConfigs(ctx context.Context, dockerClient *client.Client, dataDir, dumpContID string, configs map[string]string) error {
	isEmpty, err := tools.IsEmptyDirectory(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if isEmpty {
		if err := tools.ExecCommand(ctx, dockerClient, dumpContID, types.ExecConfig{
			Cmd: []string{"chown", "-R", "postgres", dataDir},
		}); err != nil {
			return errors.Wrap(err, "failed to set permissions")
		}

		if err := tools.InitDB(ctx, dockerClient, dumpContID, dataDir); err != nil {
			return errors.Wrap(err, "failed to init Postgres")
		}

		log.Dbg("Initialized")

		if err := tools.StartPostgres(ctx, dockerClient, dumpContID, dataDir, tools.DefaultStopTimeout); err != nil {
			return errors.Wrap(err, "failed to init Postgres")
		}

		log.Dbg("Started Postgres")
	}

	log.Msg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, dockerClient, dumpContID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	tools.StopContainer(ctx, dockerClient, dumpContID, cont.StopTimeout)

	pgConfigFile := path.Join(dataDir, pgconfig.PgConfName)
	if err := pgconfig.AppendExtraConf(pgConfigFile, configs); err != nil {
		return errors.Wrapf(err, "failed to append extra configuration to %s ", pgConfigFile)
	}

	if err := dockerClient.ContainerStart(ctx, dumpContID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container %q", dumpContID)
	}

	return nil
}

// setupConnectionOptions prepares connection options to perform a logical dump.
func (d *DumpJob) setupConnectionOptions(ctx context.Context) error {
	d.config.db = d.DumpOptions.Source.Connection

	if err := d.dumper.SetConnectionOptions(ctx, &d.config.db); err != nil {
		return errors.Wrap(err, "failed to set connection options")
	}

	return nil
}

func (d *DumpJob) performDumpCommand(ctx context.Context, contID string, commandCfg types.ExecConfig) error {
	if d.DumpOptions.Restore != nil {
		d.dbMark.DataStateAt = time.Now().Format(tools.DataStateAtFormat)
	}

	_, err := tools.ExecCommandWithOutput(ctx, d.dockerClient, contID, commandCfg)

	return err
}

func (d *DumpJob) getEnvironmentVariables(password string) []string {
	envs := []string{
		"POSTGRES_PASSWORD=" + password,
	}

	// Avoid initialization of PostgreSQL directory in case of preparing of a dump.
	if d.DumpOptions.Restore != nil {
		envs = append(envs, "PGDATA="+d.fsPool.DataDir())
	}

	// TODO: fix it.
	envs = append(envs, "PGDATA="+d.fsPool.DataDir())

	if d.DumpOptions.Source.Type == sourceTypeLocal && d.DumpOptions.Source.Connection.Port == defaults.Port {
		log.Msg(fmt.Sprintf("The default PostgreSQL port is busy, trying to use an alternative one: %d", reservePort))
		envs = append(envs, "PGPORT="+strconv.Itoa(reservePort))
	}

	return envs
}

func (d *DumpJob) buildContainerConfig(password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabDumpLabel,
			cont.DBLabInstanceIDLabel: d.globalCfg.InstanceID,
		},
		Env:         d.getEnvironmentVariables(password),
		Image:       d.DockerImage,
		Healthcheck: health.GetConfig(d.globalCfg.Database.User(), d.globalCfg.Database.Name()),
	}
}

func (d *DumpJob) buildHostConfig(ctx context.Context) (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{
		NetworkMode: d.getContainerNetworkMode(),
	}

	if err := tools.AddVolumesToHostConfig(ctx, d.dockerClient, hostConfig, d.fsPool.DataDir()); err != nil {
		return nil, err
	}

	hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: d.DumpOptions.DumpLocation,
		Target: d.DumpOptions.DumpLocation,
	})

	return hostConfig, nil
}

func (d *DumpJob) getContainerNetworkMode() container.NetworkMode {
	networkMode := networkModeDefault

	if d.Source.Type == sourceTypeLocal {
		networkMode = networkModeHost
	}

	return networkMode
}

func (d *DumpJob) getExecEnvironmentVariables() []string {
	execEnvs := append(os.Environ(), d.dumper.GetCmdEnvVariables()...)

	if d.config.db.Password != "" && os.Getenv("PGPASSWORD") == "" {
		execEnvs = append(execEnvs, "PGPASSWORD="+d.config.db.Password)
	}

	return execEnvs
}

func (d *DumpJob) buildLogicalDumpCommand() []string {
	optionalArgs := map[string]string{
		"--host":     d.config.db.Host,
		"--port":     strconv.Itoa(d.config.db.Port),
		"--username": d.config.db.Username,
		"--dbname":   d.config.db.DBName,
		"--jobs":     strconv.Itoa(d.DumpOptions.ParallelJobs),
	}

	dumpCmd := append([]string{"pg_dump", "--create"}, prepareCmdOptions(optionalArgs)...)

	for _, table := range d.Partial.Tables {
		dumpCmd = append(dumpCmd, "--table", table)
	}

	// Define if restore directly or export to dump location.
	if d.DumpOptions.Restore != nil {
		dumpCmd = append(dumpCmd, "--format", customFormat)
		dumpCmd = append(dumpCmd, d.buildLogicalRestoreCommand(d.DumpOptions.Source.Connection.DBName)...)
		cmd := strings.Join(dumpCmd, " ")

		log.Dbg(cmd)

		return []string{"sh", "-c", cmd}
	}

	dumpCmd = append(dumpCmd, "--format", directoryFormat, "--file", d.DumpOptions.DumpLocation)

	return dumpCmd
}

func (d *DumpJob) buildLogicalRestoreCommand(dbName string) []string {
	restoreCmd := []string{"|", "pg_restore", "--username", d.globalCfg.Database.User(), "--dbname", defaults.DBName,
		"--no-privileges", "--no-owner"}

	if dbName != defaults.DBName {
		// To avoid recreating of the default database.
		restoreCmd = append(restoreCmd, "--create")
	}

	if d.Restore.ForceInit {
		restoreCmd = append(restoreCmd, "--clean", "--if-exists")
	}

	return restoreCmd
}

func prepareCmdOptions(options map[string]string) []string {
	cmdOptions := []string{}

	for optionKey, optionValue := range options {
		if optionValue != "" {
			cmdOptions = append(cmdOptions, optionKey, optionValue)
		}
	}

	return cmdOptions
}

func (d *DumpJob) markDatabaseData() error {
	if err := d.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return d.dbMarker.SaveConfig(d.dbMark)
}
