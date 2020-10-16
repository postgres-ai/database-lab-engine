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
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/sethvargo/go-password/password"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

const (
	// PhysicalInitialType declares a job type for preparing a physical snapshot.
	PhysicalInitialType = "physicalSnapshot"

	pre                    = "_pre"
	promoteContainerPrefix = "dblab_promote_"

	// Defines container health check options.
	hcPromoteInterval = 5 * time.Second
	hcPromoteRetries  = 200

	syncContainerStopTimeout = 2 * time.Minute
	supportedSysctlPrefix    = "fs.mqueue."
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
	cloneManager   thinclones.Manager
	options        PhysicalOptions
	globalCfg      *dblabCfg.Global
	dbMarker       *dbmarker.Marker
	dbMark         *dbmarker.Config
	dockerClient   *client.Client
	scheduler      *cron.Cron
	scheduleOnce   sync.Once
	promotionMutex sync.Mutex
}

// PhysicalOptions describes options for a physical initialization job.
type PhysicalOptions struct {
	Promote             bool              `yaml:"promote"`
	DockerImage         string            `yaml:"dockerImage"`
	PreprocessingScript string            `yaml:"preprocessingScript"`
	Configs             map[string]string `yaml:"configs"`
	Sysctls             map[string]string `yaml:"sysctls"`
	Scheduler           *Scheduler        `yaml:"scheduler"`
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

// NewPhysicalInitialJob creates a new physical initial job.
func NewPhysicalInitialJob(cfg config.JobConfig, docker *client.Client, cloneManager thinclones.Manager,
	global *dblabCfg.Global, marker *dbmarker.Marker) (*PhysicalInitial, error) {
	p := &PhysicalInitial{
		name:         cfg.Name,
		cloneManager: cloneManager,
		globalCfg:    global,
		dbMarker:     marker,
		dbMark:       &dbmarker.Config{DataType: dbmarker.PhysicalDataType},
		dockerClient: docker,
	}

	if err := options.Unmarshal(cfg.Options, &p.options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	if err := p.validateConfig(); err != nil {
		return nil, errors.Wrap(err, "invalid physicalSnapshot configuration")
	}

	if err := p.setupScheduler(); err != nil {
		return nil, errors.Wrap(err, "failed to set up scheduler")
	}

	return p, nil
}

func (p *PhysicalInitial) setupScheduler() error {
	if p.options.Scheduler == nil ||
		p.options.Scheduler.Snapshot.Timetable == "" && p.options.Scheduler.Retention.Timetable == "" {
		return nil
	}

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	if _, err := specParser.Parse(p.options.Scheduler.Snapshot.Timetable); p.options.Scheduler.Snapshot.Timetable != "" && err != nil {
		return errors.Wrapf(err, "failed to parse schedule timetable %q", p.options.Scheduler.Snapshot.Timetable)
	}

	if _, err := specParser.Parse(p.options.Scheduler.Retention.Timetable); p.options.Scheduler.Retention.Timetable != "" && err != nil {
		return errors.Wrapf(err, "failed to parse retention timetable %q", p.options.Scheduler.Retention.Timetable)
	}

	p.scheduler = cron.New()

	return nil
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

	return nil
}

func (p *PhysicalInitial) syncInstanceName() string {
	return cont.SyncInstanceContainerPrefix + p.globalCfg.InstanceID
}

// Name returns a name of the job.
func (p *PhysicalInitial) Name() string {
	return p.name
}

// Run starts the job.
func (p *PhysicalInitial) Run(ctx context.Context) (err error) {
	// Start scheduling after initial snapshot.
	defer func() {
		p.scheduleOnce.Do(p.startScheduler(ctx))
	}()

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

	// Sync container management.
	syncContainer, err := p.dockerClient.ContainerInspect(ctx, p.syncInstanceName())
	if err != nil && !client.IsErrNotFound(err) {
		return errors.Wrap(err, "failed to inspect sync container")
	}

	if syncContainer.ContainerJSONBase != nil && syncContainer.State.Running {
		log.Msg("Stopping sync container before snapshotting")

		if err := p.dockerClient.ContainerStop(ctx, syncContainer.ID, pointer.ToDuration(syncContainerStopTimeout)); err != nil {
			return errors.Wrapf(err, "failed to stop %q", p.syncInstanceName())
		}

		defer func() {
			log.Msg("Starting sync container after snapshotting")

			if err := p.dockerClient.ContainerStart(ctx, syncContainer.ID, types.ContainerStartOptions{}); err != nil {
				log.Err(fmt.Sprintf("failed to start %q: %v", p.syncInstanceName(), err))
			}

			if err := tools.RunPostgres(ctx, p.dockerClient, syncContainer.ID, p.globalCfg.DataDir()); err != nil {
				log.Err(fmt.Sprintf("failed to start PostgreSQL instance inside %q: %v", p.syncInstanceName(), err))
			}
		}()
	}

	defer func() {
		if _, ok := err.(*skipSnapshotErr); ok {
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
				log.Err(fmt.Sprintf("Failed to destroy the %q snapshot: %v", snapshotName, err))
			}
		}
	}()

	if err := p.cloneManager.CreateClone(cloneName, snapshotName); err != nil {
		return errors.Wrapf(err, "failed to create \"pre\" clone %s", cloneName)
	}

	defer func() {
		if err != nil {
			if errDestroy := p.cloneManager.DestroyClone(cloneName); errDestroy != nil {
				log.Err(fmt.Sprintf("Failed to destroy clone %q: %v", cloneName, err))
			}
		}
	}()

	// Promotion.
	if p.options.Promote {
		if err := p.promoteInstance(ctx, path.Join(p.globalCfg.ClonesMountDir, cloneName, p.globalCfg.DataSubDir)); err != nil {
			return err
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

	return nil
}

func (p *PhysicalInitial) startScheduler(ctx context.Context) func() {
	if p.scheduler == nil || p.options.Scheduler == nil ||
		p.options.Scheduler.Snapshot.Timetable == "" && p.options.Scheduler.Retention.Timetable == "" {
		return func() {}
	}

	return func() {
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
	}
}

func (p *PhysicalInitial) runAutoSnapshot(ctx context.Context) func() {
	return func() {
		if err := p.Run(ctx); err != nil {
			log.Err(errors.Wrap(err, "failed to take a snapshot automatically"))

			log.Msg("Interrupt automatic snapshots")
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

	if err := configuration.Run(clonePath); err != nil {
		return errors.Wrap(err, "failed to enforce configs")
	}

	// Apply users configs.
	if err := applyUsersConfigs(p.options.Configs, path.Join(clonePath, "postgresql.conf")); err != nil {
		return err
	}

	pgVersion, err := tools.DetectPGVersion(clonePath)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	// Adjust recovery configuration.
	if err := p.adjustRecoveryConfiguration(pgVersion, clonePath); err != nil {
		return errors.Wrap(err, "failed to adjust recovery configuration")
	}

	hostConfig, err := p.buildHostConfig(ctx, clonePath)
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	promoteImage := p.options.DockerImage
	if promoteImage == "" {
		promoteImage = fmt.Sprintf("postgresai/sync-instance:%s", pgVersion)
	}

	if err := tools.PullImage(ctx, p.dockerClient, promoteImage); err != nil {
		return errors.Wrap(err, "failed to scan image pulling response")
	}

	pwd, err := password.Generate(tools.PasswordLength, tools.PasswordMinDigits, tools.PasswordMinSymbols, false, true)
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

	defer tools.RemoveContainer(ctx, p.dockerClient, promoteCont.ID, cont.StopTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, p.dockerClient, p.promoteContainerName())
		}
	}()

	if err := p.dockerClient.ContainerStart(ctx, promoteCont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", p.promoteContainerName(), promoteCont.ID))

	// Start PostgreSQL instance.
	if err := tools.RunPostgres(ctx, p.dockerClient, promoteCont.ID, clonePath); err != nil {
		return errors.Wrap(err, "failed to start PostgreSQL instance")
	}

	if err := tools.CheckContainerReadiness(ctx, p.dockerClient, promoteCont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	shouldBePromoted, err := p.checkRecovery(ctx, promoteCont.ID)
	if err != nil {
		return errors.Wrap(err, "failed to read response of the exec command")
	}

	log.Msg("Should be promoted: ", shouldBePromoted)

	// Detect dataStateAt.
	if shouldBePromoted == "t" {
		extractedDataStateAt, err := p.extractDataStateAt(ctx, promoteCont.ID)
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

		// Promote PGDATA.
		if err := p.runPromoteCommand(ctx, promoteCont.ID, clonePath); err != nil {
			return errors.Wrapf(err, "failed to promote PGDATA: %s", clonePath)
		}
	}

	// Checkpoint.
	if err := p.checkpoint(ctx, promoteCont.ID); err != nil {
		return err
	}

	return nil
}

func (p *PhysicalInitial) adjustRecoveryConfiguration(pgVersion, clonePGDataDir string) error {
	// Remove postmaster.pid.
	if err := os.Remove(path.Join(clonePGDataDir, "postmaster.pid")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "failed to remove postmaster.pid")
	}

	// Truncate pg_ident.conf.
	if err := tools.TouchFile(path.Join(clonePGDataDir, "pg_ident.conf")); err != nil {
		return errors.Wrap(err, "failed to truncate pg_ident.conf")
	}

	// Replication mode.
	var (
		replicationFilename string
		buffer              bytes.Buffer
	)

	version, err := strconv.ParseFloat(pgVersion, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse PostgreSQL version")
	}

	const pgVersion12 = 12

	if version >= pgVersion12 {
		replicationFilename = "standby.signal"
	} else {
		replicationFilename = "recovery.conf"

		buffer.WriteString("standby_mode = 'on'\n")
		buffer.WriteString("primary_conninfo = ''\n")
		buffer.WriteString("restore_command = ''\n")
	}

	if err := ioutil.WriteFile(path.Join(clonePGDataDir, replicationFilename), buffer.Bytes(), 0666); err != nil {
		return err
	}

	return nil
}

func (p *PhysicalInitial) buildContainerConfig(clonePath, promoteImage, password string) *container.Config {
	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabPromoteLabel,
			cont.DBLabInstanceIDLabel: p.globalCfg.InstanceID,
		},
		Env: []string{
			"PGDATA=" + clonePath,
			"POSTGRES_PASSWORD=" + password,
		},
		Image: promoteImage,
		Healthcheck: health.GetConfig(
			health.OptionInterval(hcPromoteInterval),
			health.OptionRetries(hcPromoteRetries),
		),
	}
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
	checkRecoveryCmd := []string{"psql", "-U", defaults.Username, "-XAtc", "select pg_is_in_recovery()"}
	log.Msg("Check recovery command", checkRecoveryCmd)

	execCommand, err := p.dockerClient.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          checkRecoveryCmd,
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to create exec command")
	}

	attachResponse, err := p.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return "", errors.Wrap(err, "failed to attach to exec command")
	}

	defer attachResponse.Close()

	return checkRecoveryModeResponse(attachResponse.Reader)
}

func checkRecoveryModeResponse(input io.Reader) (string, error) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		text := scanner.Text()

		fmt.Println(text)

		if text == "f" {
			return "f", nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "t", nil
}

func (p *PhysicalInitial) extractDataStateAt(ctx context.Context, containerID string) (string, error) {
	promoteCommand := []string{"psql", "-U", defaults.Username, "-d", defaults.DBName, "-XAtc",
		"select to_char(coalesce(pg_last_xact_replay_timestamp(), NOW()) at time zone 'UTC', 'YYYYMMDDHH24MISS')"}

	log.Msg("Running promote command", promoteCommand)

	execCommand, err := p.dockerClient.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          promoteCommand,
		User:         defaults.Username,
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to create exec command")
	}

	attachResponse, err := p.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return "", errors.Wrap(err, "failed to attach to exec command")
	}

	defer attachResponse.Close()

	return readDataStateAt(attachResponse.Reader)
}

func readDataStateAt(input io.Reader) (string, error) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		text := scanner.Text()

		fmt.Println(text)

		return text, nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

func (p *PhysicalInitial) runPromoteCommand(ctx context.Context, containerID, clonePath string) error {
	promoteCommand := []string{"pg_ctl", "-D", clonePath, "-W", "promote"}

	log.Msg("Running promote command", promoteCommand)

	if err := tools.ExecCommand(ctx, p.dockerClient, containerID, types.ExecConfig{
		User: defaults.Username,
		Cmd:  promoteCommand,
	}); err != nil {
		return errors.Wrap(err, "failed to promote instance")
	}

	return nil
}

func (p *PhysicalInitial) checkpoint(ctx context.Context, containerID string) error {
	commandCheckpoint := []string{"psql", "-U", defaults.Username, "-d", defaults.DBName, "-XAtc", "checkpoint"}
	log.Msg("Run checkpoint command", commandCheckpoint)

	execCommand, err := p.dockerClient.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          commandCheckpoint,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create exec command")
	}

	attachResponse, err := p.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return errors.Wrap(err, "failed to attach to exec command")
	}

	defer attachResponse.Close()

	if _, err = io.Copy(os.Stdout, attachResponse.Reader); err != nil {
		return errors.Wrap(err, "failed to read response of exec command")
	}

	return nil
}

func (p *PhysicalInitial) markDatabaseData() error {
	if err := p.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return p.dbMarker.SaveConfig(p.dbMark)
}

func (p *PhysicalInitial) cleanupSnapshots(retentionLimit int) error {
	_, err := p.cloneManager.CleanupSnapshots(retentionLimit)
	if err != nil {
		return errors.Wrap(err, "failed to clean up snapshots")
	}

	return nil
}
