/*
2020 © Postgres.ai
*/

package logical

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
)

const (
	// DumpJobType declares a job type for logical dumping.
	DumpJobType = "logical-dump"

	// Defines dump options.
	dumpContainerName        = "retrieval_logical_dump"
	dumpContainerDir         = "/tmp"
	dumpContainerStopTimeout = 10 * time.Second

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
	DumpFile     string         `yaml:"dumpLocation"`
	DockerImage  string         `yaml:"dockerImage"`
	Connection   Connection     `yaml:"connection"`
	Source       Source         `yaml:"source"`
	Partial      Partial        `yaml:"partial"`
	ParallelJobs int            `yaml:"parallelJobs"`
	Restore      *DirectRestore `yaml:"restore,omitempty"`
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

// DirectRestore contains options for direct data restoring.
type DirectRestore struct {
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

	if err := dumpJob.dbMarker.CreateConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to create a DBMarker config of the database")
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
		d.DumpOptions.Source.Connection.Port = defaultPort
	}

	if d.DumpOptions.Source.Connection.Username == "" {
		d.DumpOptions.Source.Connection.Username = defaultUsername
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

// Name returns a name of the job.
func (d *DumpJob) Name() string {
	return d.name
}

// Run starts the job.
func (d *DumpJob) Run(ctx context.Context) error {
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

	cont, err := d.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Env:         d.getEnvironmentVariables(),
			Image:       d.DockerImage,
			Healthcheck: getContainerHealthConfig(),
		},
		&container.HostConfig{
			Mounts:      d.getMountVolumes(),
			NetworkMode: d.getContainerNetworkMode(),
		},
		&network.NetworkingConfig{},
		dumpContainerName,
	)
	if err != nil {
		log.Err(err)

		return errors.Wrap(err, "failed to create container")
	}

	defer func() {
		if err := d.dockerClient.ContainerStop(ctx, cont.ID, pointer.ToDuration(dumpContainerStopTimeout)); err != nil {
			log.Err("Failed to stop a dump container: ", err)
		}

		if err := d.dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			log.Err("Failed to remove container: ", err)

			return
		}

		log.Msg(fmt.Sprintf("Stop container: %s. ID: %v", dumpContainerName, cont.ID))
	}()

	if err := d.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", dumpContainerName, cont.ID))

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
		return errors.Wrap(err, "failed to create an exec command")
	}

	if len(d.Partial.Tables) > 0 {
		log.Msg("Partial dump will be run. Tables for dumping: ", strings.Join(d.Partial.Tables, ", "))
	}

	var output io.Writer = os.Stdout

	if d.DumpOptions.DumpFile != "" {
		dumpFile, err := os.Create(d.getDumpContainerPath())
		if err != nil {
			return errors.Wrap(err, "failed to create file")
		}

		defer func() {
			if err := dumpFile.Close(); err != nil {
				log.Err("failed to close dump file", err)
			}
		}()

		output = dumpFile
	}

	if err := d.performDumpCommand(ctx, output, execCommand.ID); err != nil {
		return errors.Wrap(err, "failed to dump a database")
	}

	if err := tools.InspectCommandResponse(ctx, d.dockerClient, cont.ID, execCommand.ID); err != nil {
		return errors.Wrap(err, "failed to exec the dump command")
	}

	if err := d.dbMarker.SaveConfig(d.dbMark); err != nil {
		return errors.Wrap(err, "failed to mark the created dump")
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

func (d *DumpJob) performDumpCommand(ctx context.Context, cmdOutput io.Writer, commandID string) error {
	if d.DumpOptions.Restore != nil {
		d.dbMark.DataStateAt = time.Now().Format(tools.DataStateAtFormat)
	}

	execAttach, err := d.dockerClient.ContainerExecAttach(ctx, commandID, types.ExecStartCheck{})
	if err != nil {
		return err
	}
	defer execAttach.Close()

	// read the cmd output
	var errBuf bytes.Buffer

	outputDone := make(chan error)

	go func() {
		// StdCopy de-multiplexes the stream into two writers.
		_, err = stdcopy.StdCopy(cmdOutput, &errBuf, execAttach.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return errors.Wrap(err, "failed to copy output")
		}

		break

	case <-ctx.Done():
		return ctx.Err()
	}

	if errBuf.Len() > 0 {
		return errors.New(errBuf.String())
	}

	return nil
}

func (d *DumpJob) getDumpContainerPath() string {
	return d.DumpFile
}

func (d *DumpJob) getEnvironmentVariables() []string {
	envs := []string{
		"PGDATA=" + pgDataContainerDir,
		"POSTGRES_HOST_AUTH_METHOD=trust",
	}

	if d.DumpOptions.Source.Type == sourceTypeLocal && d.DumpOptions.Source.Connection.Port == defaultPort {
		envs = append(envs, "PGPORT="+strconv.Itoa(reservePort))
	}

	return envs
}

// getMountVolumes returns a list of mount volumes.
func (d *DumpJob) getMountVolumes() []mount.Mount {
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: d.globalCfg.DataDir,
			Target: pgDataContainerDir,
		},
	}

	if d.DumpOptions.DumpFile != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: filepath.Dir(d.DumpOptions.DumpFile),
			Target: dumpContainerDir,
		})
	}

	// Add dump specific mounts.
	mounts = append(mounts, d.dumper.GetMounts()...)

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
	execEnvs := d.dumper.GetCmdEnvVariables()

	pgPassword := d.config.db.Password

	if pgPassword == "" && os.Getenv("PGPASSWORD") != "" {
		pgPassword = os.Getenv("PGPASSWORD")
	}

	if pgPassword != "" {
		execEnvs = append(execEnvs, "PGPASSWORD="+pgPassword)
	}

	return execEnvs
}

func (d *DumpJob) buildLogicalDumpCommand() []string {
	dumpCmd := []string{"pg_dump", "-C", "-Fc"}

	optionalArgs := map[string]string{
		"-h": d.config.db.Host,
		"-p": strconv.Itoa(d.config.db.Port),
		"-U": d.config.db.Username,
		"-d": d.config.db.DBName,
		"-j": strconv.Itoa(d.DumpOptions.ParallelJobs),
	}
	dumpCmd = append(dumpCmd, prepareCmdOptions(optionalArgs)...)

	for _, table := range d.Partial.Tables {
		dumpCmd = append(dumpCmd, "-t", table)
	}

	if d.DumpOptions.Restore != nil {
		dumpCmd = append(dumpCmd, d.buildLogicalRestoreCommand()...)
		cmd := strings.Join(dumpCmd, " ")

		log.Dbg(cmd)

		return []string{"sh", "-c", cmd}
	}

	return dumpCmd
}

func (d *DumpJob) buildLogicalRestoreCommand() []string {
	restoreCmd := []string{"|", "pg_restore", "-U", defaultUsername, "-C", "-d", defaultDBName, "--no-privileges"}

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
