/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
)

const (
	// DumpJobType declares a job type for logical dumping.
	DumpJobType = "logical-dump"

	// Defines dump options.
	dumpContainerPrefix = "dblab_ld_"

	// Defines dump source types.
	sourceTypeLocal  = "local"
	sourceTypeRemote = "remote"
	sourceTypeRDS    = "rds"

	// reservePort defines reserve port in case of a local dump.
	reservePort = 9999

	// Container network modes.
	networkModeDefault = container.NetworkMode("default")
	networkModeHost    = container.NetworkMode("host")
)

// DumpJob declares a job for logical dumping.
type DumpJob struct {
	name         string
	dockerClient *client.Client
	globalCfg    *dblabCfg.Global
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
	RDS        *RDSConfig `yaml:"rds"`
}

type dumpJobConfig struct {
	db Connection
}

// dumper describes the interface to prepare environment for a logical dump.
type dumper interface {
	// GetEnvVariables returns dumper environment variables.
	GetCmdEnvVariables() []string

	// GetMounts returns dumper volume configurations for mounting.
	GetMounts() []mount.Mount

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
	ForceInit bool `yaml:"forceInit"`
}

// NewDumpJob creates a new DumpJob.
func NewDumpJob(cfg config.JobConfig, docker *client.Client, global *dblabCfg.Global, marker *dbmarker.Marker) (*DumpJob, error) {
	dumpJob := &DumpJob{
		name:         cfg.Name,
		dockerClient: docker,
		globalCfg:    global,
		dbMarker:     marker,
		dbMark: &dbmarker.Config{
			DataType: dbmarker.LogicalDataType,
		},
	}

	if err := options.Unmarshal(cfg.Options, &dumpJob.DumpOptions); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	if err := dumpJob.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid logical dump job")
	}

	dumpJob.setDefaults()

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
			return errors.New("the RDS configuration section must not be empty when using the RDS source type")
		}

		dumper, err := newRDSDumper(d.Source.RDS)
		if err != nil {
			return errors.Wrap(err, "failed to create an RDS dumper")
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

// Run starts the job.
func (d *DumpJob) Run(ctx context.Context) (err error) {
	log.Msg(fmt.Sprintf("Run job: %s. Options: %v", d.Name(), d.DumpOptions))

	isEmpty, err := tools.IsEmptyDirectory(d.globalCfg.DataDir)
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

	hostConfig, err := d.buildHostConfig()
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	pwd, err := password.Generate(tools.PasswordLength, tools.PasswordMinDigits, tools.PasswordMinSymbols, false, true)
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	cont, err := d.dockerClient.ContainerCreate(ctx, d.buildContainerConfig(pwd), hostConfig, &network.NetworkingConfig{},
		d.dumpContainerName(),
	)
	if err != nil {
		log.Err(err)

		return errors.Wrapf(err, "failed to create container %q", d.dumpContainerName())
	}

	defer tools.RemoveContainer(ctx, d.dockerClient, cont.ID, tools.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, d.dockerClient, d.dumpContainerName())
		}
	}()

	if err := d.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container %q", d.dumpContainerName())
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", d.dumpContainerName(), cont.ID))

	if err := tools.CheckContainerReadiness(ctx, d.dockerClient, cont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	if err := d.setupConnectionOptions(ctx); err != nil {
		return errors.Wrap(err, "failed to setup connection options")
	}

	dumpCommand := d.buildLogicalDumpCommand()
	log.Msg("Running dump command", dumpCommand)

	execCommand, err := d.dockerClient.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          dumpCommand,
		Env:          d.getExecEnvironmentVariables(),
	})

	if err != nil {
		return errors.Wrap(err, "failed to create dump command")
	}

	if len(d.Partial.Tables) > 0 {
		log.Msg("Partial dump will be run. Tables for dumping: ", strings.Join(d.Partial.Tables, ", "))
	}

	if err := d.performDumpCommand(ctx, os.Stdout, cont.ID, execCommand.ID); err != nil {
		return errors.Wrap(err, "failed to dump a database")
	}

	if d.DumpOptions.Restore != nil {
		if err := d.markDatabaseData(); err != nil {
			return errors.Wrap(err, "failed to mark the created dump")
		}

		if err := recalculateStats(ctx, d.dockerClient, cont.ID, buildAnalyzeCommand(Connection{
			DBName:   d.config.db.DBName,
			Username: defaults.Username,
		}, d.DumpOptions.ParallelJobs)); err != nil {
			return errors.Wrap(err, "failed to recalculate statistics after restore")
		}
	}

	log.Msg("Dumping job has been finished")

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

func (d *DumpJob) performDumpCommand(ctx context.Context, cmdOutput io.Writer, contID, commandID string) error {
	if d.DumpOptions.Restore != nil {
		d.dbMark.DataStateAt = time.Now().Format(tools.DataStateAtFormat)
	}

	execAttach, err := d.dockerClient.ContainerExecAttach(ctx, commandID, types.ExecStartCheck{})
	if err != nil {
		return err
	}
	defer execAttach.Close()

	if err := tools.ProcessAttachResponse(ctx, execAttach.Reader, cmdOutput); err != nil {
		return err
	}

	if err := tools.InspectCommandResponse(ctx, d.dockerClient, contID, commandID); err != nil {
		return errors.Wrap(err, "failed to exec the dump command")
	}

	return nil
}

func (d *DumpJob) getEnvironmentVariables(password string) []string {
	envs := []string{
		"POSTGRES_PASSWORD=" + password,
	}

	// Avoid initialization of PostgreSQL directory in case of preparing of a dump.
	if d.DumpOptions.Restore != nil {
		envs = append(envs, "PGDATA="+d.globalCfg.DataDir)
	}

	if d.DumpOptions.Source.Type == sourceTypeLocal && d.DumpOptions.Source.Connection.Port == defaults.Port {
		log.Msg(fmt.Sprintf("The default PostgreSQL port is busy, trying to use an alternative one: %d", reservePort))
		envs = append(envs, "PGPORT="+strconv.Itoa(reservePort))
	}

	return envs
}

func (d *DumpJob) buildContainerConfig(password string) *container.Config {
	return &container.Config{
		Labels:      map[string]string{"label": tools.DBLabControlLabel},
		Env:         d.getEnvironmentVariables(password),
		Image:       d.DockerImage,
		Healthcheck: health.GetConfig(),
	}
}

func (d *DumpJob) buildHostConfig() (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{
		Mounts:      d.getMountVolumes(),
		NetworkMode: d.getContainerNetworkMode(),
	}

	if err := tools.AddVolumesToHostConfig(hostConfig, d.globalCfg.DataDir); err != nil {
		return nil, err
	}

	return hostConfig, nil
}

// getMountVolumes returns a list of mount volumes.
func (d *DumpJob) getMountVolumes() []mount.Mount {
	mounts := d.dumper.GetMounts()

	if d.DumpOptions.DumpLocation != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: filepath.Dir(d.DumpOptions.DumpLocation),
			Target: filepath.Dir(d.DumpOptions.DumpLocation),
		})
	}

	return mounts
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
	format := "custom"

	if d.DumpOptions.ParallelJobs > defaultParallelJobs {
		format = "directory"
	}

	optionalArgs := map[string]string{
		"--host":     d.config.db.Host,
		"--port":     strconv.Itoa(d.config.db.Port),
		"--username": d.config.db.Username,
		"--dbname":   d.config.db.DBName,
		"--jobs":     strconv.Itoa(d.DumpOptions.ParallelJobs),
	}

	dumpCmd := append([]string{"pg_dump", "--create", "--format", format}, prepareCmdOptions(optionalArgs)...)

	for _, table := range d.Partial.Tables {
		dumpCmd = append(dumpCmd, "--table", table)
	}

	// Define if restore directly or export to dump location.
	if d.DumpOptions.Restore != nil {
		dumpCmd = append(dumpCmd, d.buildLogicalRestoreCommand()...)
		cmd := strings.Join(dumpCmd, " ")

		log.Dbg(cmd)

		return []string{"sh", "-c", cmd}
	}

	dumpCmd = append(dumpCmd, "--file", d.DumpOptions.DumpLocation)

	return dumpCmd
}

func (d *DumpJob) buildLogicalRestoreCommand() []string {
	restoreCmd := []string{"|", "pg_restore", "--username", defaults.Username, "--create", "--dbname", defaults.DBName, "--no-privileges"}

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
