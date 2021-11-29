/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/pgtool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	// PhysicalSnapshotType declares a job type for preparing a physical snapshot.
	PhysicalSnapshotType = "physicalSnapshot"

	pre                    = "_pre"
	promoteContainerPrefix = "dblab_promote_"

	supportedSysctlPrefix = "fs.mqueue."

	checkpointTimestampLabel = "Time of latest checkpoint:"

	restoreCommandOption = "restore_command"
	targetActionOption   = "recovery_target_action"
	promoteTargetAction  = "promote"

	// WAL parsing constants.
	walNameLen  = 24
	pgVersion10 = 10
)

var defaultRecoveryCfg = map[string]string{
	"recovery_target":        "immediate",
	"recovery_target_action": promoteTargetAction,
}

// supportedSysctls describes supported sysctls for Promote Docker image.
var supportedSysctls = map[string]struct{}{
	"kernel.msgmax":          {},
	"kernel.msgmnb":          {},
	"kernel.msgmni":          {},
	"kernel.sem":             {},
	"kernel.shmall":          {},
	"kernel.shmmax":          {},
	"kernel.shmmni":          {},
	"kernel.shm_rmid_forced": {},
}

// PhysicalInitial describes a job for preparing a physical initial snapshot.
type PhysicalInitial struct {
	name           string
	cloneManager   pool.FSManager
	fsPool         *resources.Pool
	options        PhysicalOptions
	globalCfg      *global.Config
	engineProps    global.EngineProps
	dbMarker       *dbmarker.Marker
	dbMark         *dbmarker.Config
	dockerClient   *client.Client
	scheduler      *cron.Cron
	schedulerCtx   context.Context
	promotionMutex sync.Mutex
	queryProcessor *queryProcessor
	tm             *telemetry.Agent
}

// PhysicalOptions describes options for a physical initialization job.
type PhysicalOptions struct {
	SkipStartSnapshot   bool              `yaml:"skipStartSnapshot"`
	Promotion           Promotion         `yaml:"promotion"`
	PreprocessingScript string            `yaml:"preprocessingScript"`
	Configs             map[string]string `yaml:"configs"`
	Sysctls             map[string]string `yaml:"sysctls"`
	Envs                map[string]string `yaml:"envs"`
	Scheduler           *Scheduler        `yaml:"scheduler"`
}

// Promotion describes promotion options.
type Promotion struct {
	Enabled            bool                   `yaml:"enabled"`
	DockerImage        string                 `yaml:"dockerImage"`
	ContainerConfig    map[string]interface{} `yaml:"containerConfig"`
	HealthCheck        HealthCheck            `yaml:"healthCheck"`
	QueryPreprocessing QueryPreprocessing     `yaml:"queryPreprocessing"`
	Configs            map[string]string      `yaml:"configs"`
	Recovery           map[string]string      `yaml:"recovery"`
}

// HealthCheck describes health check options of a promotion.
type HealthCheck struct {
	Interval   int64 `yaml:"interval"`
	MaxRetries int   `yaml:"maxRetries"`
}

// Scheduler provides scheduler options.
type Scheduler struct {
	Snapshot  ScheduleSpec `yaml:"snapshot"`
	Retention ScheduleSpec `yaml:"retention"`
}

// ScheduleSpec defines options to set up scheduler components.
type ScheduleSpec struct {
	Timetable string `yaml:"timetable"`
	Limit     int    `yaml:"limit"`
}

// QueryPreprocessing defines query preprocessing options.
type QueryPreprocessing struct {
	QueryPath          string `yaml:"queryPath"`
	MaxParallelWorkers int    `yaml:"maxParallelWorkers"`
}

// syncState defines state of a sync instance.
type syncState struct {
	DSA string
	Err error
}

// NewPhysicalInitialJob creates a new physical initial job.
func NewPhysicalInitialJob(cfg config.JobConfig, global *global.Config, engineProps global.EngineProps, cloneManager pool.FSManager,
	tm *telemetry.Agent) (*PhysicalInitial, error) {
	p := &PhysicalInitial{
		name:         cfg.Spec.Name,
		cloneManager: cloneManager,
		fsPool:       cfg.FSPool,
		globalCfg:    global,
		engineProps:  engineProps,
		dbMarker:     cfg.Marker,
		dbMark:       &dbmarker.Config{DataType: dbmarker.PhysicalDataType},
		dockerClient: cfg.Docker,
		tm:           tm,
	}

	if err := p.loadConfig(cfg.Spec.Options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	if err := p.validateConfig(); err != nil {
		return nil, errors.Wrap(err, "invalid physicalSnapshot configuration")
	}

	if p.options.Promotion.QueryPreprocessing.QueryPath != "" {
		p.queryProcessor = newQueryProcessor(cfg.Docker, global.Database.Name(), global.Database.User(),
			p.options.Promotion.QueryPreprocessing.QueryPath,
			p.options.Promotion.QueryPreprocessing.MaxParallelWorkers)
	}

	p.setupScheduler()

	return p, nil
}

func (p *PhysicalInitial) setupScheduler() {
	if !p.hasSchedulingOptions() {
		return
	}

	p.scheduler = cron.New()
}

func (p *PhysicalInitial) validateConfig() error {
	notSupportedSysctls := []string{}

	for sysctl := range p.options.Sysctls {
		if _, ok := supportedSysctls[sysctl]; !ok && !strings.HasPrefix(sysctl, supportedSysctlPrefix) {
			notSupportedSysctls = append(notSupportedSysctls, sysctl)
		}
	}

	if len(notSupportedSysctls) > 0 {
		return errors.Errorf("Docker does not support following kernel parameters (sysctls): %s",
			strings.Join(notSupportedSysctls, ", "))
	}

	if err := p.validateScheduler(); err != nil {
		return err
	}

	return nil
}

func (p *PhysicalInitial) hasSchedulingOptions() bool {
	return p.options.Scheduler != nil &&
		(p.options.Scheduler.Snapshot.Timetable != "" || p.options.Scheduler.Retention.Timetable != "")
}

func (p *PhysicalInitial) validateScheduler() error {
	if !p.hasSchedulingOptions() {
		return nil
	}

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	if _, err := specParser.Parse(p.options.Scheduler.Snapshot.Timetable); p.options.Scheduler.Snapshot.Timetable != "" && err != nil {
		return errors.Wrapf(err, "failed to parse schedule timetable %q", p.options.Scheduler.Snapshot.Timetable)
	}

	if _, err := specParser.Parse(p.options.Scheduler.Retention.Timetable); p.options.Scheduler.Retention.Timetable != "" && err != nil {
		return errors.Wrapf(err, "failed to parse retention timetable %q", p.options.Scheduler.Retention.Timetable)
	}

	return nil
}

// Name returns a name of the job.
func (p *PhysicalInitial) Name() string {
	return p.name
}

// Reload reloads job configuration.
func (p *PhysicalInitial) Reload(cfg map[string]interface{}) (err error) {
	if err := p.loadConfig(cfg); err != nil {
		return errors.Wrap(err, "failed to load job config")
	}

	p.reloadScheduler()

	return nil
}

func (p *PhysicalInitial) loadConfig(cfg map[string]interface{}) (err error) {
	if err := options.Unmarshal(cfg, &p.options); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration options")
	}

	return nil
}

func (p *PhysicalInitial) reloadScheduler() {
	if p.scheduler == nil {
		log.Msg("Skip schedule reloading because it has not been initialized")
		return
	}

	p.scheduler.Stop()

	for _, ent := range p.scheduler.Entries() {
		p.scheduler.Remove(ent.ID)
	}

	p.startScheduler(p.schedulerCtx)
}

// Run starts the job.
func (p *PhysicalInitial) Run(ctx context.Context) (err error) {
	p.schedulerCtx = ctx

	// Start scheduling after initial snapshot.
	defer p.startScheduler(p.schedulerCtx)

	if p.options.SkipStartSnapshot {
		log.Msg("Skip taking a snapshot at the start")

		return nil
	}

	return p.run(p.schedulerCtx)
}

func (p *PhysicalInitial) run(ctx context.Context) (err error) {
	select {
	case <-ctx.Done():
		if p.scheduler != nil {
			log.Msg("Stop automatic snapshots")
			p.scheduler.Stop()
		}

		return nil

	default:
	}

	p.dbMark.DataStateAt = extractDataStateAt(p.dbMarker)

	// Snapshot data.
	preDataStateAt := time.Now().Format(tools.DataStateAtFormat)
	cloneName := fmt.Sprintf("clone%s_%s", pre, preDataStateAt)

	defer func() {
		if _, ok := errors.Cause(err).(*skipSnapshotErr); ok {
			log.Msg(err.Error())
			err = nil
		}
	}()

	var syState syncState

	if p.options.Promotion.Enabled {
		syState.DSA, syState.Err = p.checkSyncInstance(ctx)

		if syState.Err != nil {
			log.Dbg(fmt.Sprintf("failed to check the sync instance before snapshotting: %v", syState),
				"Recovery configs will be applied on the promotion stage")
		}
	}

	// Prepare pre-snapshot.
	snapshotName, err := p.cloneManager.CreateSnapshot("", preDataStateAt+pre)
	if err != nil {
		return errors.Wrap(err, "failed to create snapshot")
	}

	defer func() {
		if err != nil {
			if errDestroy := p.cloneManager.DestroySnapshot(snapshotName); errDestroy != nil {
				log.Err(fmt.Sprintf("Failed to destroy the %q snapshot: %v", snapshotName, errDestroy))
			}
		}
	}()

	if err := p.cloneManager.CreateClone(cloneName, snapshotName); err != nil {
		return errors.Wrapf(err, "failed to create \"pre\" clone %s", cloneName)
	}

	defer func() {
		if err != nil {
			if errDestroy := p.cloneManager.DestroyClone(cloneName); errDestroy != nil {
				log.Err(fmt.Sprintf("Failed to destroy clone %q: %v", cloneName, errDestroy))
			}
		}
	}()

	// Promotion.
	if p.options.Promotion.Enabled {
		if err := p.promoteInstance(ctx, path.Join(p.fsPool.ClonesDir(), cloneName, p.fsPool.DataSubDir), syState); err != nil {
			return errors.Wrap(err, "failed to promote instance")
		}
	}

	// Transformation.
	if p.options.PreprocessingScript != "" {
		if err := runPreprocessingScript(p.options.PreprocessingScript); err != nil {
			return err
		}
	}

	// Mark database data.
	if err := p.markDatabaseData(); err != nil {
		return errors.Wrap(err, "failed to mark the prepared data")
	}

	// Create a snapshot.
	if _, err := p.cloneManager.CreateSnapshot(cloneName, p.dbMark.DataStateAt); err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	p.updateDataStateAt()

	p.tm.SendEvent(ctx, telemetry.SnapshotCreatedEvent, telemetry.SnapshotCreated{})

	return nil
}

func (p *PhysicalInitial) checkSyncInstance(ctx context.Context) (string, error) {
	log.Msg("Check the sync instance state: ", p.syncInstanceName())

	syncContainer, err := p.dockerClient.ContainerInspect(ctx, p.syncInstanceName())
	if err != nil {
		return "", err
	}

	if err := tools.CheckContainerReadiness(ctx, p.dockerClient, syncContainer.ID); err != nil {
		return "", errors.Wrap(err, "failed to readiness check")
	}

	log.Msg("Sync instance has been checked. It is running")

	if err := p.checkpoint(ctx, syncContainer.ID); err != nil {
		return "", errors.Wrap(err, "failed to make a checkpoint for sync instance")
	}

	extractedDataStateAt, err := p.getLastXActReplayTimestamp(ctx, syncContainer.ID)
	if err != nil {
		return "", errors.Wrap(err, `failed to get last replay timestamp from the sync instance`)
	}

	log.Msg("Sync instance data state at: ", extractedDataStateAt)

	return extractedDataStateAt, nil
}

func (p *PhysicalInitial) syncInstanceName() string {
	return cont.SyncInstanceContainerPrefix + p.engineProps.InstanceID
}

func (p *PhysicalInitial) startScheduler(ctx context.Context) {
	if p.scheduler == nil || !p.hasSchedulingOptions() {
		return
	}

	if p.options.Scheduler.Snapshot.Timetable != "" {
		if _, err := p.scheduler.AddFunc(p.options.Scheduler.Snapshot.Timetable, p.runAutoSnapshot(ctx)); err != nil {
			log.Err(errors.Wrap(err, "failed to schedule a new snapshot job"))
			return
		}
	}

	if p.options.Scheduler.Retention.Timetable != "" {
		if _, err := p.scheduler.AddFunc(p.options.Scheduler.Retention.Timetable,
			p.runAutoCleanup(p.options.Scheduler.Retention.Limit)); err != nil {
			log.Err(errors.Wrap(err, "failed to schedule a new cleanup job"))
			return
		}
	}

	p.scheduler.Start()

	log.Msg("Snapshot scheduler has been started")

	go p.waitToStopScheduler()
}

func (p *PhysicalInitial) waitToStopScheduler() {
	<-p.schedulerCtx.Done()

	if p.scheduler != nil {
		log.Msg("Stop snapshot scheduler")
		p.scheduler.Stop()
	}
}

func (p *PhysicalInitial) runAutoSnapshot(ctx context.Context) func() {
	return func() {
		if err := p.run(ctx); err != nil {
			log.Err(errors.Wrap(err, "failed to take a snapshot automatically"))
		}
	}
}

func (p *PhysicalInitial) runAutoCleanup(retentionLimit int) func() {
	return func() {
		if err := p.cleanupSnapshots(retentionLimit); err != nil {
			log.Err(errors.Wrap(err, "failed to clean up snapshots automatically"))
		}
	}
}

func (p *PhysicalInitial) promoteContainerName() string {
	return promoteContainerPrefix + p.engineProps.InstanceID
}

func (p *PhysicalInitial) promoteInstance(ctx context.Context, clonePath string, syState syncState) (err error) {
	p.promotionMutex.Lock()
	defer p.promotionMutex.Unlock()

	log.Msg("Promote the Postgres instance.")

	cfgManager, err := pgconfig.NewCorrector(clonePath)
	if err != nil {
		return errors.Wrap(err, "failed to init configs manager")
	}

	// Adjust recovery configuration.
	if err := cfgManager.AdjustRecoveryFiles(); err != nil {
		return errors.Wrap(err, "failed to adjust recovery configuration")
	}

	recoveryFileConfig, err := cfgManager.ReadRecoveryConfig()
	if err != nil {
		return errors.Wrap(err, "failed to read recovery configuration file")
	}

	if len(recoveryFileConfig) == 0 {
		if err := cfgManager.RemoveRecoveryConfig(); err != nil {
			return errors.Wrap(err, "failed to remove recovery config file")
		}
	}

	recoveryConfig := make(map[string]string)

	// Item 5. Remove a recovery file: https://gitlab.com/postgres-ai/database-lab/-/issues/236#note_513401256
	if syState.Err != nil {
		recoveryConfig = buildRecoveryConfig(recoveryFileConfig, p.options.Promotion.Recovery)

		if err := cfgManager.ApplyRecovery(recoveryFileConfig); err != nil {
			return errors.Wrap(err, "failed to apply recovery configuration")
		}
	} else if err := cfgManager.RemoveRecoveryConfig(); err != nil {
		log.Err(errors.Wrap(err, "failed to remove recovery config file"))
	}

	// Apply promotion configs.
	if promotionConfig := p.options.Promotion.Configs; len(promotionConfig) > 0 {
		if err := cfgManager.ApplyPromotion(p.options.Promotion.Configs); err != nil {
			return errors.Wrap(err, "failed to store prepared configuration")
		}
	}

	hostConfig, err := p.buildHostConfig(ctx, clonePath)
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	promoteImage := p.options.Promotion.DockerImage
	if promoteImage == "" {
		promoteImage = fmt.Sprintf("postgresai/extended-postgres:%g", cfgManager.GetPgVersion())
	}

	if err := tools.PullImage(ctx, p.dockerClient, promoteImage); err != nil {
		return errors.Wrap(err, "failed to scan image pulling response")
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	// Run promotion container.
	promoteCont, err := p.dockerClient.ContainerCreate(ctx,
		p.buildContainerConfig(clonePath, promoteImage, pwd, recoveryConfig[targetActionOption]),
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		p.promoteContainerName(),
	)

	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	defer tools.RemoveContainer(ctx, p.dockerClient, promoteCont.ID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, p.dockerClient, p.promoteContainerName())
			tools.PrintLastPostgresLogs(ctx, p.dockerClient, p.promoteContainerName(), clonePath)
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", p.promoteContainerName(), promoteCont.ID))

	if err := p.dockerClient.ContainerStart(ctx, promoteCont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	if syState.DSA == "" {
		dsa, err := p.getDSAFromWAL(ctx, cfgManager.GetPgVersion(), promoteCont.ID, clonePath)
		if err != nil {
			log.Dbg("cannot extract DSA form WAL files: ", err)
		}

		if dsa != "" {
			log.Msg("DataStateAt extracted from WAL files: ", dsa)

			syState.DSA = dsa
		}
	}

	log.Msg("Starting PostgreSQL and waiting for readiness")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, p.promoteContainerName()))

	if err := tools.CheckContainerReadiness(ctx, p.dockerClient, promoteCont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	shouldBePromoted, err := p.checkRecovery(ctx, promoteCont.ID)
	if err != nil {
		return errors.Wrap(err, "failed to check recovery mode")
	}

	log.Msg("Should be promoted: ", shouldBePromoted)

	// Detect dataStateAt.
	if shouldBePromoted == "t" {
		// Promote PGDATA.
		if err := p.runPromoteCommand(ctx, promoteCont.ID, clonePath); err != nil {
			return errors.Wrapf(err, "failed to promote PGDATA: %s", clonePath)
		}

		isInRecovery, err := p.checkRecovery(ctx, promoteCont.ID)
		if err != nil {
			return errors.Wrap(err, "failed to check recovery mode after promotion")
		}

		if isInRecovery != "f" {
			return errors.Errorf("PostgreSQL is in recovery, promotion has been failed: %s", clonePath)
		}
	}

	if err := p.markDSA(ctx, syState.DSA, promoteCont.ID, clonePath, cfgManager.GetPgVersion()); err != nil {
		return errors.Wrap(err, "failed to mark dataStateAt")
	}

	if p.queryProcessor != nil {
		if err := p.queryProcessor.applyPreprocessingQueries(ctx, promoteCont.ID); err != nil {
			return errors.Wrap(err, "failed to run preprocessing queries")
		}
	}

	// Checkpoint.
	if err := p.checkpoint(ctx, promoteCont.ID); err != nil {
		return err
	}

	if err := cfgManager.RemoveRecoveryConfig(); err != nil {
		return errors.Wrap(err, "failed to remove recovery config file")
	}

	if err := cfgManager.TruncateSyncConfig(); err != nil {
		return errors.Wrap(err, "failed to truncate sync config file")
	}

	if err := cfgManager.TruncatePromotionConfig(); err != nil {
		return errors.Wrap(err, "failed to truncate promotion config file")
	}

	// Apply configs to the snapshot.
	if err := cfgManager.ApplySnapshot(p.options.Configs); err != nil {
		return errors.Wrap(err, "failed to store prepared configuration")
	}

	if err := tools.StopPostgres(ctx, p.dockerClient, promoteCont.ID, clonePath, tools.DefaultStopTimeout); err != nil {
		log.Msg("Failed to stop Postgres", err)
		tools.PrintContainerLogs(ctx, p.dockerClient, promoteCont.ID)
	}

	return nil
}

func (p *PhysicalInitial) getDSAFromWAL(ctx context.Context, pgVersion float64, containerID, cloneDir string) (string, error) {
	log.Dbg(cloneDir)

	walDirectory := walDir(cloneDir, pgVersion)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		Cmd: []string{"ls", "-t", walDirectory},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to read the wal directory")
	}

	walFileList := strings.Fields(output)

	// Walk in the reverse order.
	for i := len(walFileList) - 1; i >= 0; i-- {
		fileName := walFileList[i]
		walFilePath := path.Join(walDirectory, fileName)

		log.Dbg("Look up into file: ", walFilePath)

		if len(fileName) != walNameLen {
			continue
		}

		dateTime := p.parseWAL(ctx, containerID, pgVersion, walFilePath)
		if dateTime != "" {
			return dateTime, nil
		}
	}

	log.Dbg("no found dataStateAt in WAL files")

	return "", nil
}

func walDir(cloneDir string, pgVersion float64) string {
	dir := "pg_wal"

	if pgVersion < pgVersion10 {
		dir = "pg_xlog"
	}

	return path.Join(cloneDir, dir)
}

func (p *PhysicalInitial) parseWAL(ctx context.Context, containerID string, pgVersion float64, walFilePath string) string {
	cmd := walCommand(pgVersion, walFilePath)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		Cmd: []string{"sh", "-c", cmd},
	})
	if err != nil {
		log.Dbg("failed to parse WAL: ", err)
		return ""
	}

	if output == "" {
		log.Dbg("empty timestamp output given")
		return ""
	}

	log.Dbg("Parse the line from a WAL file", output)

	return parseWALLine(output)
}

func walCommand(pgVersion float64, walFilePath string) string {
	walDumpUtil := "pg_waldump"

	if pgVersion < pgVersion10 {
		walDumpUtil = "pg_xlogdump"
	}

	return fmt.Sprintf("/usr/lib/postgresql/%g/bin/%s %s -r Transaction | tail -1", pgVersion, walDumpUtil, walFilePath)
}

func parseWALLine(line string) string {
	const (
		commitToken = "COMMIT"
		tokenLen    = len(commitToken)
		layout      = "2006-01-02 15:04:05.000000 MST"
	)

	commitIndex := strings.LastIndex(line, commitToken)
	if commitIndex == -1 {
		log.Dbg("timestamp not found", line)
		return ""
	}

	dateTimeString := strings.TrimSpace(line[commitIndex+tokenLen:])

	if idx := strings.IndexByte(dateTimeString, ';'); idx > 0 {
		dateTimeString = dateTimeString[:idx]
	}

	parsedDate, err := time.Parse(layout, dateTimeString)
	if err != nil {
		log.Dbg("failed to parse WAL time: ", dateTimeString)
		return ""
	}

	return parsedDate.Format(tools.DataStateAtFormat)
}

func buildRecoveryConfig(fileConfig, userRecoveryConfig map[string]string) map[string]string {
	recoveryConf := fileConfig

	if rc, ok := fileConfig[restoreCommandOption]; ok || rc != "" {
		for k, v := range defaultRecoveryCfg {
			recoveryConf[k] = v
		}
	}

	for k, v := range userRecoveryConfig {
		recoveryConf[k] = v
	}

	return recoveryConf
}

func (p *PhysicalInitial) markDSA(ctx context.Context, defaultDSA, containerID, dataDir string, pgVersion float64) error {
	extractedDataStateAt, err := p.extractDataStateAt(ctx, containerID, dataDir, pgVersion, defaultDSA)
	if err != nil {
		if defaultDSA == "" {
			return errors.Wrap(err, `failed to extract dataStateAt`)
		}

		log.Msg("failed to extract dataStateAt. Use value from the sync instance: ", defaultDSA)
		extractedDataStateAt = defaultDSA
	}

	log.Msg("Data state at: ", extractedDataStateAt)

	if p.dbMark.DataStateAt != "" && extractedDataStateAt == p.dbMark.DataStateAt {
		return newSkipSnapshotErr(fmt.Sprintf(
			`The previous snapshot already contains the latest data: %s. Skip taking a new snapshot.`,
			p.dbMark.DataStateAt))
	}

	p.dbMark.DataStateAt = extractedDataStateAt

	log.Msg("Mark data state at: ", p.dbMark.DataStateAt)

	return nil
}

func (p *PhysicalInitial) buildContainerConfig(clonePath, promoteImage, password, action string) *container.Config {
	hcPromotionInterval := health.DefaultRestoreInterval
	hcPromotionRetries := health.DefaultRestoreRetries

	if p.options.Promotion.HealthCheck.Interval != 0 {
		hcPromotionInterval = time.Duration(p.options.Promotion.HealthCheck.Interval) * time.Second
	}

	if p.options.Promotion.HealthCheck.MaxRetries != 0 {
		hcPromotionRetries = p.options.Promotion.HealthCheck.MaxRetries
	}

	hcOptions := []health.ContainerOption{
		health.OptionInterval(hcPromotionInterval),
		health.OptionRetries(hcPromotionRetries),
	}

	// Perform the custom health check in case of automatic promotion.
	if action == promoteTargetAction {
		testCommand := fmt.Sprintf("if [ \"`psql -U %s -d %s -XAtc \"select pg_is_in_recovery()\"`\" = \"f\" ];then true;else false;fi",
			p.globalCfg.Database.User(),
			p.globalCfg.Database.Name(),
		)
		hcOptions = append(hcOptions, health.OptionTest(testCommand))
	}

	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabPromoteLabel,
			cont.DBLabInstanceIDLabel: p.engineProps.InstanceID,
			cont.DBLabEngineNameLabel: p.engineProps.ContainerName,
		},
		Env:   p.getEnvironmentVariables(clonePath, password),
		Image: promoteImage,
		Healthcheck: health.GetConfig(
			p.globalCfg.Database.User(),
			p.globalCfg.Database.Name(),
			hcOptions...,
		),
	}
}

func (p *PhysicalInitial) getEnvironmentVariables(clonePath, password string) []string {
	envVariables := []string{
		"PGDATA=" + clonePath,
		"POSTGRES_PASSWORD=" + password,
	}

	// Add user-defined environment variables.
	for env, value := range p.options.Envs {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", env, value))
	}

	return envVariables
}

func (p *PhysicalInitial) buildHostConfig(ctx context.Context, clonePath string) (*container.HostConfig, error) {
	hostConfig, err := cont.BuildHostConfig(ctx, p.dockerClient, clonePath, p.options.Promotion.ContainerConfig)
	if err != nil {
		return nil, err
	}

	hostConfig.Sysctls = p.options.Sysctls

	return hostConfig, nil
}

func (p *PhysicalInitial) checkRecovery(ctx context.Context, containerID string) (string, error) {
	checkRecoveryCmd := []string{"psql",
		"-U", p.globalCfg.Database.User(),
		"-d", p.globalCfg.Database.Name(),
		"-XAtc", "select pg_is_in_recovery()",
	}

	log.Msg("Check recovery command", checkRecoveryCmd)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		Cmd: checkRecoveryCmd,
	})

	return output, err
}

func (p *PhysicalInitial) extractDataStateAt(ctx context.Context, containerID, dataDir string, pgVersion float64,
	defaultDSA string) (string, error) {
	output, err := p.getLastXActReplayTimestamp(ctx, containerID)
	if err != nil {
		log.Dbg("unable to get last replay timestamp from the promotion container: ", err)
	}

	if output != "" && err == nil {
		return output, nil
	}

	if defaultDSA != "" {
		log.Msg("failed to extract dataStateAt. Use value from the sync instance: ", defaultDSA)

		return defaultDSA, nil
	}

	// If the sync instance has not yet downloaded WAL when retrieving the default DSA, run it again.
	dsa, err := p.getDSAFromWAL(ctx, pgVersion, containerID, dataDir)
	if err != nil {
		log.Dbg("cannot extract DSA from WAL files in the promotion container: ", err)
	}

	if dsa != "" {
		log.Msg("Use dataStateAt value from the promotion WAL files: ", defaultDSA)

		return dsa, nil
	}

	log.Msg("The last replay timestamp and dataStateAt from the sync instance are not found. Extract the last checkpoint timestamp")

	response, err := pgtool.ReadControlData(ctx, p.dockerClient, containerID, dataDir, pgVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to read control data")
	}

	defer response.Close()

	output, err = getCheckPointTimestamp(ctx, response.Reader)
	if err != nil {
		return "", errors.Wrap(err, "failed to read control data")
	}

	return output, nil
}

func (p *PhysicalInitial) getLastXActReplayTimestamp(ctx context.Context, containerID string) (string, error) {
	extractionCommand := []string{"psql", "-U", p.globalCfg.Database.User(), "-d", p.globalCfg.Database.Name(), "-XAtc",
		"select to_char(pg_last_xact_replay_timestamp() at time zone 'UTC', 'YYYYMMDDHH24MISS')"}

	log.Msg("Running dataStateAt command", extractionCommand)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		Cmd:  extractionCommand,
		User: defaults.Username,
	})

	log.Msg("Extracted last replay timestamp: ", output)

	return output, err
}

func getCheckPointTimestamp(ctx context.Context, r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	checkpointTitleBytes := []byte(checkpointTimestampLabel)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		if bytes.HasPrefix(scanner.Bytes(), checkpointTitleBytes) {
			checkpointTimestamp := bytes.TrimSpace(bytes.TrimPrefix(scanner.Bytes(), checkpointTitleBytes))

			checkpointDate, err := dateparse.ParseStrict(string(checkpointTimestamp))
			if err != nil {
				return "", err
			}

			return checkpointDate.UTC().Format(util.DataStateAtFormat), nil
		}
	}

	return "", errors.New("checkpoint timestamp not found")
}

func (p *PhysicalInitial) runPromoteCommand(ctx context.Context, containerID, clonePath string) error {
	promoteCommand := []string{"pg_ctl", "-D", clonePath, "-w", "promote"}

	log.Msg("Running promote command", promoteCommand)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		User: defaults.Username,
		Cmd:  promoteCommand,
		Env: []string{
			fmt.Sprintf("PGCTLTIMEOUT=%d", p.options.Promotion.HealthCheck.MaxRetries*int(p.options.Promotion.HealthCheck.Interval)),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to promote instance")
	}

	log.Msg("Promotion result: ", output)

	return nil
}

func (p *PhysicalInitial) checkpoint(ctx context.Context, containerID string) error {
	commandCheckpoint := []string{"psql", "-U", p.globalCfg.Database.User(), "-d", p.globalCfg.Database.Name(), "-XAtc", "checkpoint"}
	log.Msg("Run checkpoint command", commandCheckpoint)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{Cmd: commandCheckpoint})
	if err != nil {
		return errors.Wrap(err, "failed to make checkpoint")
	}

	log.Msg("Checkpoint result: ", output)

	return nil
}

func (p *PhysicalInitial) markDatabaseData() error {
	if err := p.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return p.dbMarker.SaveConfig(p.dbMark)
}

// updateDataStateAt updates dataStateAt for in-memory representation of a storage pool.
func (p *PhysicalInitial) updateDataStateAt() {
	dsaTime, err := time.Parse(util.DataStateAtFormat, p.dbMark.DataStateAt)
	if err != nil {
		log.Err("Invalid value for DataStateAt: ", p.dbMark.DataStateAt)
		return
	}

	p.fsPool.SetDSA(dsaTime)
}

func (p *PhysicalInitial) cleanupSnapshots(retentionLimit int) error {
	select {
	case <-p.schedulerCtx.Done():
		log.Msg("Stop automatic snapshot cleanup")
		return nil
	default:
	}

	_, err := p.cloneManager.CleanupSnapshots(retentionLimit)
	if err != nil {
		return errors.Wrap(err, "failed to clean up snapshots")
	}

	return nil
}
