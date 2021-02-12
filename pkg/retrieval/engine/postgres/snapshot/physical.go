/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	dblabCfg "gitlab.com/postgres-ai/database-lab/v2/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

const (
	// PhysicalInitialType declares a job type for preparing a physical snapshot.
	PhysicalInitialType = "physicalSnapshot"

	pre                    = "_pre"
	promoteContainerPrefix = "dblab_promote_"

	supportedSysctlPrefix = "fs.mqueue."
)

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
	globalCfg      *dblabCfg.Global
	dbMarker       *dbmarker.Marker
	dbMark         *dbmarker.Config
	dockerClient   *client.Client
	scheduler      *cron.Cron
	schedulerCtx   context.Context
	promotionMutex sync.Mutex
	queryProcessor *queryProcessor
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
	Enabled            bool               `yaml:"enabled"`
	DockerImage        string             `yaml:"dockerImage"`
	HealthCheck        HealthCheck        `yaml:"healthCheck"`
	QueryPreprocessing QueryPreprocessing `yaml:"queryPreprocessing"`
	Configs            map[string]string  `yaml:"configs"`
	Recovery           map[string]string  `yaml:"recovery"`
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

// NewPhysicalInitialJob creates a new physical initial job.
func NewPhysicalInitialJob(cfg config.JobConfig, global *dblabCfg.Global, cloneManager pool.FSManager) (*PhysicalInitial, error) {
	p := &PhysicalInitial{
		name:         cfg.Spec.Name,
		cloneManager: cloneManager,
		fsPool:       cfg.FSPool,
		globalCfg:    global,
		dbMarker:     cfg.Marker,
		dbMark:       &dbmarker.Config{DataType: dbmarker.PhysicalDataType},
		dockerClient: cfg.Docker,
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
		if err := p.promoteInstance(ctx, path.Join(p.fsPool.ClonesDir(), cloneName, p.fsPool.DataSubDir)); err != nil {
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

	return nil
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
	return promoteContainerPrefix + p.globalCfg.InstanceID
}

func (p *PhysicalInitial) promoteInstance(ctx context.Context, clonePath string) (err error) {
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

	if len(p.options.Promotion.Recovery) > 0 {
		if err := cfgManager.ApplyRecovery(p.options.Promotion.Recovery); err != nil {
			return errors.Wrap(err, "failed to apply recovery configuration")
		}
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
		p.buildContainerConfig(clonePath, promoteImage, pwd),
		hostConfig,
		&network.NetworkingConfig{},
		p.promoteContainerName(),
	)

	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	defer tools.RemoveContainer(ctx, p.dockerClient, promoteCont.ID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, p.dockerClient, p.promoteContainerName())
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", p.promoteContainerName(), promoteCont.ID))

	if err := p.dockerClient.ContainerStart(ctx, promoteCont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg("Starting PostgreSQL and waiting for readiness")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, p.promoteContainerName()))

	if err := tools.CheckContainerReadiness(ctx, p.dockerClient, promoteCont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	if err := p.markDSA(ctx, promoteCont.ID); err != nil {
		return errors.Wrap(err, "failed to mark dataStateAt")
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

	if p.queryProcessor != nil {
		if err := p.queryProcessor.applyPreprocessingQueries(ctx, promoteCont.ID); err != nil {
			return errors.Wrap(err, "failed to run preprocessing queries")
		}
	}

	// Checkpoint.
	if err := p.checkpoint(ctx, promoteCont.ID); err != nil {
		return err
	}

	if err := cfgManager.TruncateSyncConfig(); err != nil {
		return errors.Wrap(err, "failed to truncate a sync config file")
	}

	if err := cfgManager.TruncatePromotionConfig(); err != nil {
		return errors.Wrap(err, "failed to truncate a promotion config file")
	}

	// Apply configs to the snapshot.
	if err := cfgManager.ApplySnapshot(p.options.Configs); err != nil {
		return errors.Wrap(err, "failed to store prepared configuration")
	}

	return nil
}

func (p *PhysicalInitial) markDSA(ctx context.Context, containerID string) error {
	extractedDataStateAt, err := p.extractDataStateAt(ctx, containerID)
	if err != nil {
		return errors.Wrap(err,
			`Failed to get data_state_at: PGDATA should be promoted, but pg_last_xact_replay_timestamp() returns empty result.
				Check if pg_data is correct, or explicitly define DATA_STATE_AT via an environment variable.`)
	}

	log.Msg("Extracted Data state at: ", extractedDataStateAt)

	if p.dbMark.DataStateAt != "" && extractedDataStateAt == p.dbMark.DataStateAt {
		return newSkipSnapshotErr(fmt.Sprintf(
			`The previous snapshot already contains the latest data: %s. Skip taking a new snapshot.`,
			p.dbMark.DataStateAt))
	}

	p.dbMark.DataStateAt = extractedDataStateAt

	log.Msg("Data state at: ", p.dbMark.DataStateAt)

	return nil
}

func (p *PhysicalInitial) buildContainerConfig(clonePath, promoteImage, password string) *container.Config {
	hcPromotionInterval := health.DefaultRestoreInterval
	hcPromotionRetries := health.DefaultRestoreRetries

	if p.options.Promotion.HealthCheck.Interval != 0 {
		hcPromotionInterval = time.Duration(p.options.Promotion.HealthCheck.Interval) * time.Second
	}

	if p.options.Promotion.HealthCheck.MaxRetries != 0 {
		hcPromotionRetries = p.options.Promotion.HealthCheck.MaxRetries
	}

	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabPromoteLabel,
			cont.DBLabInstanceIDLabel: p.globalCfg.InstanceID,
		},
		Env:   p.getEnvironmentVariables(clonePath, password),
		Image: promoteImage,
		Healthcheck: health.GetConfig(
			p.globalCfg.Database.User(),
			p.globalCfg.Database.Name(),
			health.OptionInterval(hcPromotionInterval),
			health.OptionRetries(hcPromotionRetries),
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
	hostConfig := &container.HostConfig{
		Sysctls: p.options.Sysctls,
	}

	if err := tools.AddVolumesToHostConfig(ctx, p.dockerClient, hostConfig, clonePath); err != nil {
		return nil, err
	}

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

func (p *PhysicalInitial) extractDataStateAt(ctx context.Context, containerID string) (string, error) {
	extractionCommand := []string{"psql", "-U", p.globalCfg.Database.User(), "-d", p.globalCfg.Database.Name(), "-XAtc",
		"select to_char(coalesce(pg_last_xact_replay_timestamp(), NOW()) at time zone 'UTC', 'YYYYMMDDHH24MISS')"}

	log.Msg("Running dataStateAt command", extractionCommand)

	output, err := tools.ExecCommandWithOutput(ctx, p.dockerClient, containerID, types.ExecConfig{
		Cmd:  extractionCommand,
		User: defaults.Username,
	})

	return output, err
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
