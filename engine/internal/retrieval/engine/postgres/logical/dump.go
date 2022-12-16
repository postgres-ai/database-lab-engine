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
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/diagnostic"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// DumpJobType declares a job type for logical dumping.
	DumpJobType = "logicalDump"

	// Defines dump options.
	dumpContainerPrefix = "dblab_ld_"

	// Defines a temporary directory path as a PGDATA to dump a database without immediate restore.
	tmpDBLabPGDataDir = "/tmp/dblab_dump"

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
	plainFormat     = "plain"
	directoryFormat = "directory"
)

// DumpJob declares a job for logical dumping.
type DumpJob struct {
	name         string
	dockerClient *client.Client
	fsPool       *resources.Pool
	globalCfg    *global.Config
	engineProps  global.EngineProps
	config       dumpJobConfig
	dumper       dumper
	dbMarker     *dbmarker.Marker
	dbMark       *dbmarker.Config
	DumpOptions
}

// DumpOptions defines a logical dump options.
type DumpOptions struct {
	DumpLocation    string                    `yaml:"dumpLocation"`
	DockerImage     string                    `yaml:"dockerImage"`
	ContainerConfig map[string]interface{}    `yaml:"containerConfig"`
	Connection      Connection                `yaml:"connection"`
	Source          Source                    `yaml:"source"`
	Databases       map[string]DumpDefinition `yaml:"databases"`
	ParallelJobs    int                       `yaml:"parallelJobs"`
	Restore         ImmediateRestore          `yaml:"immediateRestore"`
	CustomOptions   []string                  `yaml:"customOptions"`
}

// Source describes source of data to dump.
type Source struct {
	Type       string     `yaml:"type"`
	Connection Connection `yaml:"connection"`
	RDS        *RDSConfig `yaml:"rdsIam"`
}

// DumpDefinition describes a database for dumping.
type DumpDefinition struct {
	Tables        []string        `yaml:"tables"`
	ExcludeTables []string        `yaml:"excludeTables"`
	Format        string          `yaml:"format"`
	Compression   compressionType `yaml:"compression"`
	dbName        string
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

	// GetDatabaseListQuery provides the query to get the list of databases for dumping.
	GetDatabaseListQuery(username string) string
}

// Connection provides connection options.
type Connection struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	Username string `yaml:"username"`
	Password string `yaml:"password" json:"-"`
}

// ImmediateRestore contains options for direct data restore without saving the dump file on disk.
type ImmediateRestore struct {
	Enabled       bool              `yaml:"enabled"`
	ForceInit     bool              `yaml:"forceInit"`
	Configs       map[string]string `yaml:"configs"`
	CustomOptions []string          `yaml:"customOptions"`
}

// NewDumpJob creates a new DumpJob.
func NewDumpJob(jobCfg config.JobConfig, global *global.Config, engineProps global.EngineProps) (*DumpJob, error) {
	dumpJob := &DumpJob{
		name:         jobCfg.Spec.Name,
		dockerClient: jobCfg.Docker,
		fsPool:       jobCfg.FSPool,
		globalCfg:    global,
		engineProps:  engineProps,
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
	if d.Restore.Enabled && d.ParallelJobs > 1 {
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

	d.config.db = d.DumpOptions.Source.Connection
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
	return dumpContainerPrefix + d.engineProps.InstanceID
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

// ReportActivity reports the current job activity.
func (d *DumpJob) ReportActivity(ctx context.Context) (*activity.Activity, error) {
	dbConnection := d.config.db
	dbConnection.Password = d.getPassword()

	pgeList, err := dbSourceActivity(ctx, dbConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to get source activity: %w", err)
	}

	jobActivity := &activity.Activity{
		Source: pgeList,
	}

	if d.DumpOptions.Restore.Enabled {
		pgEvents, err := pgContainerActivity(ctx, d.dockerClient, d.dumpContainerName(), d.globalCfg.Database)
		if err != nil {
			log.Err(err)
			return jobActivity, fmt.Errorf("failed to get activity for target container: %w", err)
		}

		jobActivity.Target = pgEvents
	}

	return jobActivity, nil
}

// Run starts the job.
func (d *DumpJob) Run(ctx context.Context) (err error) {
	log.Msg("Run job: ", d.Name())

	dataDir := d.fsPool.DataDir()

	isEmpty, err := tools.IsEmptyDirectory(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if d.DumpOptions.Restore.Enabled && !isEmpty {
		if !d.DumpOptions.Restore.ForceInit {
			return errors.New("the data directory is not empty. Use 'forceInit' or empty the data directory")
		}

		log.Msg("The data directory is not empty. Existing data may be overwritten.")

		if err := updateConfigs(dataDir, d.DumpOptions.Restore.Configs); err != nil {
			return fmt.Errorf("failed to update configs: %w", err)
		}
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

	containerID, err := tools.CreateContainerIfMissing(ctx, d.dockerClient, d.dumpContainerName(), d.buildContainerConfig(pwd), hostConfig)

	if err != nil {
		return fmt.Errorf("failed to create container %q %w", d.dumpContainerName(), err)
	}

	defer tools.RemoveContainer(ctx, d.dockerClient, containerID, cont.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, d.dockerClient, d.dumpContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", d.dumpContainerName(), containerID))

	if err := d.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		collectDiagnostics(ctx, d.dockerClient, d.dumpContainerName(), dataDir)
		return errors.Wrapf(err, "failed to start container %q", d.dumpContainerName())
	}

	if err := d.setupConnectionOptions(ctx); err != nil {
		collectDiagnostics(ctx, d.dockerClient, d.dumpContainerName(), dataDir)
		return errors.Wrap(err, "failed to setup connection options")
	}

	log.Msg("Waiting for container readiness")

	if err := tools.MakeDir(ctx, d.dockerClient, containerID, tmpDBLabPGDataDir); err != nil {
		return err
	}

	pgDataDir := tmpDBLabPGDataDir
	if d.DumpOptions.Restore.Enabled {
		pgDataDir = dataDir
	}

	if err := tools.CheckContainerReadiness(ctx, d.dockerClient, containerID); err != nil {
		var errHealthCheck *tools.ErrHealthCheck
		if !errors.As(err, &errHealthCheck) {
			return errors.Wrap(err, "failed to readiness check")
		}

		var configs map[string]string

		if d.DumpOptions.Restore.Enabled {
			configs = d.DumpOptions.Restore.Configs
		}

		if err := setupPGData(ctx, d.dockerClient, pgDataDir, containerID, configs); err != nil {
			collectDiagnostics(ctx, d.dockerClient, d.dumpContainerName(), pgDataDir)
			return errors.Wrap(err, "failed to set up Postgres data")
		}
	}

	dbList := d.Databases

	if len(dbList) == 0 {
		dbList, err = d.getDBList(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to extract the list of databases for dumping")
		}
	}

	if err := d.cleanupDumpLocation(ctx, containerID, dbList); err != nil {
		return err
	}

	for dbName, dbDetails := range dbList {
		if err := d.dumpDatabase(ctx, containerID, dbName, dbDetails); err != nil {
			return errors.Wrapf(err, "failed to dump the database %s", dbName)
		}
	}

	if d.DumpOptions.Restore.Enabled {
		if err := d.markDatabaseData(); err != nil {
			return errors.Wrap(err, "failed to mark the created dump")
		}

		analyzeCmd := buildAnalyzeCommand(
			Connection{Username: d.globalCfg.Database.User(), DBName: d.globalCfg.Database.Name()},
			d.DumpOptions.ParallelJobs,
		)

		log.Msg("Running analyze command: ", analyzeCmd)

		if err := tools.ExecCommand(ctx, d.dockerClient, containerID, types.ExecConfig{
			Cmd: analyzeCmd,
			Env: []string{"PGAPPNAME=" + dleRetrieval},
		}); err != nil {
			return errors.Wrap(err, "failed to recalculate statistics after restore")
		}

		if err := tools.RunCheckpoint(ctx, d.dockerClient, containerID, d.globalCfg.Database.User(), d.globalCfg.Database.Name()); err != nil {
			return errors.Wrap(err, "failed to run checkpoint before stop")
		}

		if err := tools.StopPostgres(ctx, d.dockerClient, containerID, dataDir, tools.DefaultStopTimeout); err != nil {
			return errors.Wrap(err, "failed to stop Postgres instance")
		}
	}

	return nil
}

func collectDiagnostics(ctx context.Context, client *client.Client, postgresName, dataDir string) {
	filterArgs := filters.NewArgs(
		filters.KeyValuePair{Key: "label",
			Value: fmt.Sprintf("%s=%s", cont.DBLabControlLabel, cont.DBLabDumpLabel)})

	if err := diagnostic.CollectDiagnostics(ctx, client, filterArgs, postgresName, dataDir); err != nil {
		log.Err("Failed to collect container diagnostics", err)
	}
}

func (d *DumpJob) getDBList(ctx context.Context) (map[string]DumpDefinition, error) {
	dbList := make(map[string]DumpDefinition)

	connStr := db.ConnectionString(d.config.db.Host, strconv.Itoa(d.config.db.Port), d.config.db.Username, d.config.db.DBName, d.getPassword())

	querier, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	rows, err := querier.Query(ctx, d.dumper.GetDatabaseListQuery(d.config.db.Username))
	if err != nil {
		return nil, fmt.Errorf("failed to perform query listing databases: %w", err)
	}

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan next row in database list result set: %w", err)
		}

		dbList[dbName] = DumpDefinition{}
	}

	return dbList, nil
}

func (d *DumpJob) getPassword() string {
	pwd := os.Getenv("PGPASSWORD")

	if d.config.db.Password != "" && os.Getenv("PGPASSWORD") == "" {
		pwd = d.config.db.Password
	}

	return pwd
}

func (d *DumpJob) cleanupDumpLocation(ctx context.Context, dumpContID string, dbList map[string]DumpDefinition) error {
	if d.DumpOptions.DumpLocation == "" || d.DumpOptions.Restore.Enabled {
		return nil
	}

	if len(dbList) == 0 {
		return nil
	}

	cleanupCmd := []string{"rm", "-rf"}

	for dbName := range dbList {
		cleanupCmd = append(cleanupCmd, path.Join(d.DumpOptions.DumpLocation, dbName))
	}

	log.Msg("Running cleanup command: ", cleanupCmd)

	if out, err := tools.ExecCommandWithOutput(ctx, d.dockerClient, dumpContID, types.ExecConfig{
		Tty: true,
		Cmd: cleanupCmd,
	}); err != nil {
		log.Dbg(out)
		return errors.Wrap(err, "failed to clean up dump location")
	}

	return nil
}

func (d *DumpJob) dumpDatabase(ctx context.Context, dumpContID, dbName string, dumpDefinition DumpDefinition) error {
	dumpCommand := d.buildLogicalDumpCommand(dbName, dumpDefinition)

	if len(dumpDefinition.Tables) > 0 ||
		len(dumpDefinition.ExcludeTables) > 0 {
		log.Msg("Partial dump")

		if len(dumpDefinition.Tables) > 0 {
			log.Msg("Including tables: ", strings.Join(dumpDefinition.Tables, ", "))
		}

		if len(dumpDefinition.ExcludeTables) > 0 {
			log.Msg("Excluding tables: ", strings.Join(dumpDefinition.ExcludeTables, ", "))
		}
	}

	log.Msg("Running dump command: ", dumpCommand)

	if output, err := d.performDumpCommand(ctx, dumpContID, types.ExecConfig{
		Tty: true,
		Cmd: dumpCommand,
		Env: d.getExecEnvironmentVariables(),
	}); err != nil {
		log.Dbg(output)
		return errors.Wrap(err, "failed to dump a database")
	}

	log.Msg(fmt.Sprintf("Dumping job for the database %q has been finished", dbName))

	return nil
}

func setupPGData(ctx context.Context, dockerClient *client.Client, dataDir, dumpContID string, configs map[string]string) error {
	entryList, err := tools.LsContainerDirectory(ctx, dockerClient, dumpContID, dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	// Already initialized.
	if len(entryList) != 0 {
		return nil
	}

	if err := tools.ExecCommand(ctx, dockerClient, dumpContID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", dataDir},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	if err := tools.InitDB(ctx, dockerClient, dumpContID); err != nil {
		return errors.Wrap(err, "failed to init Postgres")
	}

	log.Dbg("Database has been initialized")

	if err := updateConfigs(dataDir, configs); err != nil {
		return fmt.Errorf("failed to update configs: %w", err)
	}

	if err := tools.StartPostgres(ctx, dockerClient, dumpContID, tools.DefaultStopTimeout); err != nil {
		return errors.Wrap(err, "failed to init Postgres")
	}

	log.Dbg("Postgres has been started")

	return nil
}

func updateConfigs(dataDir string, configs map[string]string) error {
	if len(configs) == 0 {
		return nil
	}

	cfgManager, err := pgconfig.NewCorrector(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to create a config manager")
	}

	if err := cfgManager.AppendGeneralConfig(configs); err != nil {
		return errors.Wrap(err, "failed to append general configuration")
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

func (d *DumpJob) performDumpCommand(ctx context.Context, contID string, commandCfg types.ExecConfig) (string, error) {
	if d.DumpOptions.Restore.Enabled {
		d.dbMark.DataStateAt = time.Now().Format(tools.DataStateAtFormat)
	}

	return tools.ExecCommandWithOutput(ctx, d.dockerClient, contID, commandCfg)
}

func (d *DumpJob) getEnvironmentVariables(password string) []string {
	envs := []string{
		"POSTGRES_PASSWORD=" + password,
	}

	// Avoid initialization of PostgreSQL in the DataDir in case of preparing of a dump without immediate restore.
	pgData := tmpDBLabPGDataDir

	if d.DumpOptions.Restore.Enabled {
		pgData = d.fsPool.DataDir()
	}

	envs = append(envs, "PGDATA="+pgData)

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
			cont.DBLabInstanceIDLabel: d.engineProps.InstanceID,
			cont.DBLabEngineNameLabel: d.engineProps.ContainerName,
		},
		Env:         d.getEnvironmentVariables(password),
		Image:       d.DockerImage,
		Healthcheck: health.GetConfig(d.globalCfg.Database.User(), d.globalCfg.Database.Name()),
	}
}

func (d *DumpJob) buildHostConfig(ctx context.Context) (*container.HostConfig, error) {
	hostConfig, err := cont.BuildHostConfig(ctx, d.dockerClient, d.fsPool.DataDir(), d.DumpOptions.ContainerConfig)
	if err != nil {
		return nil, err
	}

	hostConfig.NetworkMode = d.getContainerNetworkMode()

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

	// Set unlimited statement_timeout for the dump session
	// because there is a risk of dump failure due to exceeding the statement_timeout.
	//
	// PGAPPNAME marks the dumping command to detect retrieval events in pg_stat_activity.
	execEnvs = append(execEnvs, "PGOPTIONS=-c statement_timeout=0", "PGAPPNAME="+dleRetrieval)

	return execEnvs
}

func (d *DumpJob) buildLogicalDumpCommand(dbName string, dump DumpDefinition) []string {
	// don't use map here, it creates inconsistency in the order of arguments
	dumpCmd := []string{"pg_dump", "--create"}

	if d.config.db.Host != "" {
		dumpCmd = append(dumpCmd, "--host", d.config.db.Host)
	}

	if d.config.db.Port > 0 {
		dumpCmd = append(dumpCmd, "--port", strconv.Itoa(d.config.db.Port))
	}

	if d.config.db.Username != "" {
		dumpCmd = append(dumpCmd, "--username", d.config.db.Username)
	}

	if dbName != "" {
		dumpCmd = append(dumpCmd, "--dbname", dbName)
	}

	if d.DumpOptions.ParallelJobs > 0 {
		dumpCmd = append(dumpCmd, "--jobs", strconv.Itoa(d.DumpOptions.ParallelJobs))
	}

	for _, table := range dump.Tables {
		dumpCmd = append(dumpCmd, "--table", table)
	}

	for _, table := range dump.ExcludeTables {
		dumpCmd = append(dumpCmd, "--exclude-table", table)
	}

	dumpCmd = append(dumpCmd, d.DumpOptions.CustomOptions...)

	// Define if restore directly or export to dump location.
	if d.DumpOptions.Restore.Enabled {
		dumpCmd = append(dumpCmd, "--format", customFormat)
		dumpCmd = append(dumpCmd, d.buildLogicalRestoreCommand(dbName)...)
		cmd := strings.Join(dumpCmd, " ")

		log.Dbg(cmd)

		return []string{"sh", "-c", cmd}
	}

	dumpCmd = append(dumpCmd, "--format", directoryFormat, "--file", path.Join(d.DumpOptions.DumpLocation, dbName))

	return dumpCmd
}

func (d *DumpJob) buildLogicalRestoreCommand(dbName string) []string {
	restoreCmd := []string{"|", "pg_restore", "--username", d.globalCfg.Database.User(), "--dbname", defaults.DBName}

	if dbName != defaults.DBName {
		// To avoid recreating of the default database.
		restoreCmd = append(restoreCmd, "--create")
	}

	if d.Restore.ForceInit {
		restoreCmd = append(restoreCmd, "--clean", "--if-exists")
	}

	restoreCmd = append(restoreCmd, d.DumpOptions.Restore.CustomOptions...)

	return restoreCmd
}

func (d *DumpJob) markDatabaseData() error {
	if err := d.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return d.dbMarker.SaveConfig(d.dbMark)
}
