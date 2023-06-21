/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/query"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	// RestoreJobType declares a job type for logical dumping.
	RestoreJobType = "logicalRestore"

	// const defines restore options.
	restoreContainerPrefix = "dblab_lr_"

	// defaultParallelJobs declares a default number of parallel jobs for logical dump and restore.
	defaultParallelJobs = 1

	// dumpMetafile defines metafile name of a directory dump.
	dumpMetafile = "toc.dat"

	// prefixDBName defines a prefix of database name inside of a custom dump metafile.
	prefixDBName = "dbname:"

	// prefixConnectDB defines a prefix for connection to a database construction.
	prefixConnectDB = `\connect `

	// prefixCreateTable defines a prefix for creation table query.
	prefixCreateTable = "CREATE TABLE "

	// templateCreateDB provides a template for preparing database creation queries.
	templateCreateDB = `
create database "@database" with template = template0 encoding = 'utf8';
alter database "@database" owner to "@username"; 
`
)

var (
	// errDBNameNotFound occurs when could not detect the name of the database in the provided dump.
	errDBNameNotFound = errors.New("database name not found")

	// errInvalidDump occurs when invalid dump found in the provided location.
	errInvalidDump = errors.New("invalid dump provided")

	// filenameFormatter replaces all non-word characters to an underscore.
	filenameFormatter = regexp.MustCompile(`\W`)
)

// RestoreJob defines a logical restore job.
type RestoreJob struct {
	name              string
	dockerClient      *client.Client
	fsPool            *resources.Pool
	globalCfg         *global.Config
	engineProps       *global.EngineProps
	dbMarker          *dbmarker.Marker
	dbMark            *dbmarker.Config
	queryProcessor    *query.Processor
	isDumpLocationDir bool
	RestoreOptions
}

// RestoreOptions defines a logical restore options.
type RestoreOptions struct {
	DumpLocation       string                    `yaml:"dumpLocation"`
	DockerImage        string                    `yaml:"dockerImage"`
	ContainerConfig    map[string]interface{}    `yaml:"containerConfig"`
	Databases          map[string]DumpDefinition `yaml:"databases"`
	IgnoreErrors       bool                      `yaml:"ignoreErrors"`
	ParallelJobs       int                       `yaml:"parallelJobs"`
	Configs            map[string]string         `yaml:"configs"`
	QueryPreprocessing query.PreprocessorCfg     `yaml:"queryPreprocessing"`
	CustomOptions      []string                  `yaml:"customOptions"`
	SkipPolicies       bool                      `yaml:"skipPolicies"`
}

// Partial defines tables and rules for a partial logical restore.
type Partial struct {
	Tables []string `yaml:"tables"`
}

// NewJob create a new logical restore job.
func NewJob(cfg config.JobConfig, global *global.Config, engineProps *global.EngineProps) (*RestoreJob, error) {
	restoreJob := &RestoreJob{
		name:         cfg.Spec.Name,
		dockerClient: cfg.Docker,
		fsPool:       cfg.FSPool,
		globalCfg:    global,
		engineProps:  engineProps,
		dbMarker:     cfg.Marker,
		dbMark:       &dbmarker.Config{DataType: dbmarker.LogicalDataType},
	}

	if err := restoreJob.Reload(cfg.Spec.Options); err != nil {
		return nil, errors.Wrap(err, "failed to load configuration options")
	}

	restoreJob.setDefaults()

	if qp := restoreJob.RestoreOptions.QueryPreprocessing; qp.QueryPath != "" || qp.Inline != "" {
		restoreJob.queryProcessor = query.NewQueryProcessor(cfg.Docker, qp, global.Database.Name(), global.Database.User())
	}

	return restoreJob, nil
}

func (r *RestoreJob) setDefaults() {
	// TODO: Default yaml values in tags.
	if r.ParallelJobs == 0 {
		r.ParallelJobs = defaultParallelJobs
	}
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
	if err := options.Unmarshal(cfg, &r.RestoreOptions); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration options")
	}

	r.setDefaults()

	stat, err := os.Stat(r.RestoreOptions.DumpLocation)
	if err != nil {
		return errors.Wrap(err, "dumpLocation not found")
	}

	r.isDumpLocationDir = stat.IsDir()

	return nil
}

// ReportActivity reports the current job activity.
func (r *RestoreJob) ReportActivity(ctx context.Context) (*activity.Activity, error) {
	pgEvents, err := pgContainerActivity(ctx, r.dockerClient, r.restoreContainerName(), r.globalCfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity for target container: %w", err)
	}

	jobActivity := &activity.Activity{
		Target: pgEvents,
	}

	return jobActivity, nil
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) (err error) {
	log.Msg("Run job: ", r.Name())

	dataDir := r.fsPool.DataDir()

	isEmpty, err := tools.IsEmptyDirectory(dataDir)
	if err != nil {
		return fmt.Errorf("failed to explore the data directory %q: %w", dataDir, err)
	}

	if !isEmpty {
		log.Warn(fmt.Sprintf("The data directory %q is not empty. Existing data will be overwritten.", dataDir))
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

	containerID, err := tools.CreateContainerIfMissing(ctx, r.dockerClient, r.restoreContainerName(), r.buildContainerConfig(pwd), hostConfig)

	if err != nil {
		return fmt.Errorf("failed to create container %q %w", r.restoreContainerName(), err)
	}

	defer tools.RemoveContainer(ctx, r.dockerClient, containerID, cont.StopTimeout)

	defer func() {
		if err != nil {
			collectDiagnostics(ctx, r.dockerClient, r.restoreContainerName(), r.fsPool.DataDir())
			tools.PrintContainerLogs(ctx, r.dockerClient, r.restoreContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", r.restoreContainerName(), containerID))

	if err := r.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "failed to start container %q", r.restoreContainerName())
	}

	log.Msg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, r.dockerClient, containerID); err != nil {
		var errHealthCheck *tools.ErrHealthCheck
		if !errors.As(err, &errHealthCheck) {
			return errors.Wrap(err, "failed to readiness check")
		}

		if err := setupPGData(ctx, r.dockerClient, dataDir, containerID, r.RestoreOptions.Configs); err != nil {
			return errors.Wrap(err, "failed to set up Postgres data")
		}
	}

	if err := r.queryProcessor.ApplyPreprocessingQueries(ctx, containerID); err != nil {
		return errors.Wrap(err, "failed to run preprocessing queries")
	}

	dbList, err := r.getDBList(ctx, containerID)
	if err != nil {
		return err
	}

	log.Dbg("Database List to restore: ", dbList)

	for dbName, dbDefinition := range dbList {
		if err := r.restoreDB(ctx, containerID, dbName, dbDefinition); err != nil {
			return errors.Wrap(err, "failed to restore a database")
		}
	}

	analyzeCmd := buildAnalyzeCommand(
		Connection{Username: r.globalCfg.Database.User(), DBName: r.globalCfg.Database.Name()},
		r.RestoreOptions.ParallelJobs,
	)

	log.Msg("Running analyze command: ", analyzeCmd)

	if err := tools.ExecCommand(ctx, r.dockerClient, containerID, types.ExecConfig{
		Cmd: analyzeCmd,
		Env: []string{"PGAPPNAME=" + dleRetrieval},
	}); err != nil {
		return errors.Wrap(err, "failed to recalculate statistics after restore")
	}

	if err := tools.RunCheckpoint(ctx, r.dockerClient, containerID, r.globalCfg.Database.User(), r.globalCfg.Database.Name()); err != nil {
		return errors.Wrap(err, "failed to run checkpoint before stop")
	}

	if err := tools.StopPostgres(ctx, r.dockerClient, containerID, dataDir, tools.DefaultStopTimeout); err != nil {
		return errors.Wrap(err, "failed to stop Postgres instance")
	}

	log.Msg("Restoring job has been finished")

	return nil
}

func (r *RestoreJob) buildHostConfig(ctx context.Context) (*container.HostConfig, error) {
	hostConfig, err := cont.BuildHostConfig(ctx, r.dockerClient, r.fsPool.DataDir(), r.RestoreOptions.ContainerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build container host config: %w", err)
	}

	if r.RestoreOptions.DumpLocation != "" && !isAlreadyMounted(hostConfig.Mounts, r.RestoreOptions.DumpLocation) {
		hostConfig.Mounts = append(hostConfig.Mounts,
			mount.Mount{
				Type:   mount.TypeBind,
				Source: r.RestoreOptions.DumpLocation,
				Target: r.RestoreOptions.DumpLocation,
				BindOptions: &mount.BindOptions{
					Propagation: mount.PropagationRShared,
				},
			},
		)

		log.Dbg("Mount dump location", r.RestoreOptions.DumpLocation)
	}

	return hostConfig, nil
}

func (r *RestoreJob) getDBList(ctx context.Context, contID string) (map[string]DumpDefinition, error) {
	if len(r.Databases) > 0 {
		return r.Databases, nil
	}

	if !r.isDumpLocationDir {
		dbDefinition, err := r.exploreDumpFile(ctx, contID, r.RestoreOptions.DumpLocation)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find a database to restore in dump files")
		}

		if dbDefinition == nil {
			return nil, errors.Wrap(err, "database definition not found")
		}

		return map[string]DumpDefinition{
			filepath.Base(r.RestoreOptions.DumpLocation): *dbDefinition,
		}, nil
	}

	return r.discoverDumpLocation(ctx, contID)
}

// discoverDumpLocation explores dump file to identify its type.
func (r *RestoreJob) exploreDumpFile(ctx context.Context, contID, dumpPath string) (*DumpDefinition, error) {
	// Detect if dump is custom.
	dbName, err := r.extractDBNameFromDump(ctx, contID, dumpPath)
	if err != nil {
		if !errors.Is(err, errInvalidDump) {
			return nil, err
		}

		log.Dbg(dumpPath + " is not a dump file in the custom format")
	}

	if dbName != "" {
		return &DumpDefinition{
			Format: customFormat,
			dbName: dbName,
		}, nil
	}

	// Identify type of compression if plain-text dump is archived.
	dbDefinition := &DumpDefinition{
		Format:      plainFormat,
		Compression: getCompressionType(dumpPath),
	}

	if dbDefinition.Compression != noCompression {
		return dbDefinition, nil
	}

	// Extract database name from plain-text dump.
	dbName, err = r.parsePlainFile(dumpPath)
	if err != nil {
		if errors.Is(err, errDBNameNotFound) {
			return dbDefinition, nil
		}

		return nil, err
	}

	dbDefinition.dbName = dbName

	return dbDefinition, nil
}

// extractDBNameFromDump discovers dump to extract the database name.
func (r *RestoreJob) extractDBNameFromDump(ctx context.Context, contID, dumpPath string) (string, error) {
	extractDBNameCmd := fmt.Sprintf("pg_restore --list %s | grep %s | tr -d '[;]'", dumpPath, prefixDBName)
	log.Msg("Extract database name: ", extractDBNameCmd)

	outputLine, err := tools.ExecCommandWithOutput(ctx, r.dockerClient, contID, types.ExecConfig{
		Cmd: []string{"bash", "-c", extractDBNameCmd},
	})
	if err != nil {
		return "", errInvalidDump
	}

	if outputLine == "" {
		return "", errors.New("database name to restore not found")
	}

	dbName := strings.TrimSpace(strings.TrimPrefix(outputLine, prefixDBName))

	return dbName, nil
}

func (r *RestoreJob) parsePlainFile(dumpPath string) (string, error) {
	f, err := os.Open(dumpPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open dump file")
	}

	defer func() { _ = f.Close() }()

	connectPrefix := []byte(prefixConnectDB)
	tablePrefix := []byte(prefixCreateTable)

	sc := bufio.NewScanner(f)

	for sc.Scan() {
		if bytes.HasPrefix(sc.Bytes(), connectPrefix) {
			nameCandidate := bytes.TrimSpace(bytes.TrimPrefix(sc.Bytes(), connectPrefix))

			if len(nameCandidate) > 0 {
				return string(nameCandidate), nil
			}

			log.Dbg("Cannot parse database name from string: ", sc.Text())

			break
		}

		if bytes.HasPrefix(sc.Bytes(), tablePrefix) {
			// The dump does not contain the database name.
			return "", errDBNameNotFound
		}
	}

	return "", errors.Errorf("unknown format of the dump file: %v", dumpPath)
}

// discoverDumpLocation discovers dump location to find databases ready to restore.
func (r *RestoreJob) discoverDumpLocation(ctx context.Context, contID string) (map[string]DumpDefinition, error) {
	dbList := make(map[string]DumpDefinition)

	// Check the dumpLocation directory.
	if dumpDefinition, err := r.getDirectoryDumpDefinition(ctx, contID, r.RestoreOptions.DumpLocation); err == nil {
		dbList[""] = dumpDefinition // empty string because of the root directory.

		return dbList, nil
	}

	log.Msg(fmt.Sprintf("Directory dump not found in %q", r.RestoreOptions.DumpLocation))

	fileInfos, err := os.ReadDir(r.RestoreOptions.DumpLocation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to discover dump location")
	}

	for _, info := range fileInfos {
		log.Dbg("Explore: ", info.Name())

		if info.IsDir() {
			dumpDirectory := path.Join(r.RestoreOptions.DumpLocation, info.Name())

			dumpDefinition, err := r.getDirectoryDumpDefinition(ctx, contID, dumpDirectory)
			if err != nil {
				log.Msg(fmt.Sprintf("Dump not found: %v. Skip directory: %s", err, info.Name()))
				continue
			}

			dbList[info.Name()] = dumpDefinition

			log.Msg("Found the directory dump: ", info.Name())

			continue
		}

		dumpDefinition, err := r.exploreDumpFile(ctx, contID, path.Join(r.RestoreOptions.DumpLocation, info.Name()))
		if err != nil {
			log.Dbg(fmt.Sprintf("Skip file %q due to failure to find a database to restore: %v", info.Name(), err))
			continue
		}

		if dumpDefinition == nil {
			log.Dbg(fmt.Sprintf("Skip file %q because the database definition is empty", info.Name()))
			continue
		}

		dbList[info.Name()] = *dumpDefinition

		log.Msg(fmt.Sprintf("Found the %s dump file: %s", dumpDefinition.Format, info.Name()))
	}

	return dbList, nil
}

func (r *RestoreJob) getDirectoryDumpDefinition(ctx context.Context, contID, dumpDir string) (DumpDefinition, error) {
	dumpMetafilePath := path.Join(dumpDir, dumpMetafile)

	if _, err := os.Stat(dumpMetafilePath); err != nil {
		log.Msg(fmt.Sprintf("TOC file not found: %v. Skip directory: %s", err, dumpDir))
		return DumpDefinition{}, err
	}

	log.Msg(fmt.Sprintf("TOC file has been found: %q", dumpMetafilePath))

	dbName, err := r.extractDBNameFromDump(ctx, contID, dumpDir)
	if err != nil {
		log.Err("Invalid dump: ", err)
		return DumpDefinition{}, errors.Wrap(err, "invalid database name")
	}

	return DumpDefinition{
		Format: directoryFormat,
		dbName: dbName,
	}, nil
}

func (r *RestoreJob) restoreDB(ctx context.Context, contID, dbName string, dbDefinition DumpDefinition) error {
	// The dump contains no database creation requests, so create a new database by ourselves.
	if dbDefinition.Format == plainFormat && dbDefinition.dbName == "" {
		if err := r.prepareDB(ctx, contID, dbName); err != nil {
			return errors.Wrapf(err, "failed to prepare database for dump: %s", dbName)
		}
	}

	var (
		tmpListFile *os.File
		err         error
	)

	if r.RestoreOptions.SkipPolicies {
		tmpListFile, err = os.CreateTemp(r.getDumpLocation(dbDefinition.Format, dbDefinition.dbName), dbName+"-list-file*")
		if err != nil {
			return fmt.Errorf("failed to create temporary list file: %w", err)
		}

		defer func() {
			if err := os.Remove(tmpListFile.Name()); err != nil {
				log.Dbg("Cannot remove temporary file:", err)
			}
		}()
		defer func() { _ = tmpListFile.Close() }()

		dumpLocation := r.getDumpLocation(dbDefinition.Format, dbDefinition.dbName)

		if dbDefinition.Format != directoryFormat {
			dumpLocation = r.RestoreOptions.DumpLocation
		}

		preCmd := []string{"bash", "-c", `pg_restore -l ` + dumpLocation + " | grep -v POLICY > " + tmpListFile.Name()}

		log.Msg("Running preparatory command to create list file for "+dbName, preCmd)

		output, err := tools.ExecCommandWithOutput(ctx, r.dockerClient, contID, types.ExecConfig{
			Tty: true,
			Cmd: preCmd,
			Env: []string{"PGAPPNAME=" + dleRetrieval},
		})

		if err != nil {
			log.Dbg(output)

			return fmt.Errorf("failed to perform preparatory command: %w", err)
		}
	}

	restoreCommand := r.buildLogicalRestoreCommand(dbName, dbDefinition, tmpListFile)
	log.Msg("Running restore command for "+dbName, restoreCommand)

	output, err := tools.ExecCommandWithOutput(ctx, r.dockerClient, contID, types.ExecConfig{
		Tty: true,
		Cmd: restoreCommand,
		Env: []string{"PGAPPNAME=" + dleRetrieval},
	})

	if err != nil && !r.RestoreOptions.IgnoreErrors {
		log.Err("Restore command failed: ", output)

		return fmt.Errorf("failed to exec restore command: %w. Output: %s", err, output)
	}

	if output != "" {
		log.Dbg("Output of the restore command: ", output)
	}

	if err := r.defineDSA(ctx, dbDefinition, contID, dbName); err != nil {
		log.Err("Failed to define DataStateAt: ", err)
	}

	if err := r.markDatabase(); err != nil {
		return errors.Wrap(err, "failed to mark the database")
	}

	return nil
}

// prepareDB creates a new database if it does not exist in the dump file.
func (r *RestoreJob) prepareDB(ctx context.Context, contID, dbName string) error {
	log.Dbg("The dump has a plain-text format with an empty database name. Creating a database for the dump:", dbName)

	replacer := strings.NewReplacer(
		"@database", formatDBName(dbName),
		"@username", r.globalCfg.Database.User())
	creationSQL := replacer.Replace(templateCreateDB)

	tempFile, err := os.CreateTemp("", "createdb_"+dbName+"_*.sql")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	if err := os.WriteFile(tempFile.Name(), []byte(creationSQL), 0666); err != nil {
		return err
	}

	dstPath := path.Join("/tmp", dbName)

	// Copy the temporary file to the restore container.
	if err := r.prepareArchive(ctx, contID, tempFile, dstPath); err != nil {
		return errors.Wrap(err, "failed to copy auxiliary file")
	}

	cmd := []string{"psql", "--username", r.globalCfg.Database.User(), "--dbname", defaults.DBName, "--file", dstPath}
	log.Msg("Run command", cmd)

	if out, err := tools.ExecCommandWithOutput(ctx, r.dockerClient, contID, types.ExecConfig{Tty: true, Cmd: cmd}); err != nil {
		log.Dbg("Command output: ", out)
		return errors.Wrap(err, "failed to exec restore command")
	}

	return nil
}

func (r *RestoreJob) prepareArchive(ctx context.Context, contID string, tempFile *os.File, dstPath string) error {
	srcInfo, err := archive.CopyInfoSourcePath(tempFile.Name(), false)
	if err != nil {
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return err
	}

	defer func() { _ = srcArchive.Close() }()

	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, archive.CopyInfo{Path: dstPath})
	if err != nil {
		return err
	}

	defer func() { _ = preparedArchive.Close() }()

	if err := r.dockerClient.CopyToContainer(ctx, contID, dstDir, preparedArchive, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
		CopyUIDGID:                true,
	}); err != nil {
		log.Err(err)

		return errors.Wrap(err, "failed to copy auxiliary file")
	}

	return nil
}

// formatDBName extracts a database name from a file name and adjusts it.
func formatDBName(fileName string) string {
	return filenameFormatter.ReplaceAllString(strings.TrimSuffix(fileName, filepath.Ext(fileName)), "_")
}

func (r *RestoreJob) buildContainerConfig(password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabRestoreLabel,
			cont.DBLabInstanceIDLabel: r.engineProps.InstanceID,
			cont.DBLabEngineNameLabel: r.engineProps.ContainerName,
		},
		Env: append(os.Environ(), []string{
			"PGDATA=" + r.fsPool.DataDir(),
			"POSTGRES_PASSWORD=" + password,
		}...),
		Image:       r.RestoreOptions.DockerImage,
		Healthcheck: health.GetConfig(r.globalCfg.Database.User(), r.globalCfg.Database.Name()),
	}
}

func (r *RestoreJob) defineDSA(ctx context.Context, dbDefinition DumpDefinition, contID, dbName string) error {
	if dbDefinition.Format == plainFormat {
		// dataStateAt cannot be found, but we have to mark data.
		r.dbMark.DataStateAt = time.Now().Format(util.DataStateAtFormat)
		return nil
	}

	dumpLocation := r.getDumpLocation(dbDefinition.Format, dbName)

	dataStateAt, err := r.retrieveDataStateAt(ctx, contID, dumpLocation)
	if err != nil {
		return fmt.Errorf("failed to extract dataStateAt: %w", err)
	}

	if dataStateAt != "" {
		r.dbMark.DataStateAt = dataStateAt
		log.Msg("Data state at: ", dataStateAt)
	}

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

func (r *RestoreJob) markDatabase() error {
	if err := r.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	if err := r.dbMarker.SaveConfig(r.dbMark); err != nil {
		return errors.Wrap(err, "failed to mark the database")
	}

	r.updateDataStateAt()

	return nil
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

func (r *RestoreJob) buildLogicalRestoreCommand(dumpName string, definition DumpDefinition, listFile *os.File) []string {
	if definition.Format == plainFormat {
		return r.buildPlainTextCommand(dumpName, definition)
	}

	return r.buildPGRestoreCommand(dumpName, definition, listFile)
}

func (r *RestoreJob) buildPlainTextCommand(dumpName string, definition DumpDefinition) []string {
	dbName := defaults.DBName

	// It means a required database has been created in the previous step.
	if definition.dbName == "" {
		dbName = formatDBName(dumpName)
	}

	if len(definition.Tables) > 0 {
		log.Msg("Partial restore is not available for plain-text dump")
	}

	if r.ParallelJobs > 1 {
		log.Msg("Parallel restore is not available for plain-text dump. It is always single-threaded")
	}

	return []string{
		"sh", "-c", fmt.Sprintf("%s %s | psql --username %s --dbname %s", getReadingArchiveCommand(definition.Compression),
			r.getDumpLocation(definition.Format, dumpName), r.globalCfg.Database.User(), dbName),
	}
}

func (r *RestoreJob) buildPGRestoreCommand(dumpName string, definition DumpDefinition, listFile *os.File) []string {
	// Using the default database name because the database for connection must exist.
	restoreCmd := []string{"pg_restore", "--username", r.globalCfg.Database.User(), "--dbname", defaults.DBName}

	if definition.dbName != defaults.DBName {
		// To avoid recreating of the default database.
		restoreCmd = append(restoreCmd, "--create")
	}

	restoreCmd = append(restoreCmd, "--jobs", strconv.Itoa(r.ParallelJobs))

	if len(definition.Tables) > 0 {
		log.Msg("Partial restore will be run. Tables for restoring: ", strings.Join(definition.Tables, ", "))

		for _, table := range definition.Tables {
			restoreCmd = append(restoreCmd, "--table", table)
		}
	}

	restoreCmd = append(restoreCmd, r.getDumpLocation(definition.Format, dumpName))

	customOptions := r.RestoreOptions.CustomOptions

	// Skip policies: https://gitlab.com/postgres-ai/database-lab/-/merge_requests/769
	if listFile != nil {
		restoreCmd = append(restoreCmd, fmt.Sprintf("--use-list=%s", listFile.Name()))

		customOptions = []string{}

		for _, customOption := range r.RestoreOptions.CustomOptions {
			// Exclude -L (--use-list) in customOptions to avoid conflicts if skipping policies is enabled.
			if strings.HasPrefix(customOption, "-L") || strings.HasPrefix(customOption, "--use-list") {
				continue
			}

			customOptions = append(customOptions, customOption)
		}
	}

	restoreCmd = append(restoreCmd, customOptions...)

	return restoreCmd
}

func (r *RestoreJob) getDumpLocation(dumpFormat, dbName string) string {
	switch dumpFormat {
	case customFormat, plainFormat:
		if r.isDumpLocationDir {
			return path.Join(r.RestoreOptions.DumpLocation, dbName)
		}

		return r.RestoreOptions.DumpLocation

	default:
		return path.Join(r.RestoreOptions.DumpLocation, dbName)
	}
}
