/*
2020 © Postgres.ai
*/

// Package snapshot provides components for preparing initial snapshots.
package snapshot

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/client"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/diagnostic"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/query"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	// LogicalSnapshotType declares a job type for preparing a logical initial snapshot.
	LogicalSnapshotType = "logicalSnapshot"

	patchContainerPrefix = "dblab_patch_"
)

// LogicalInitial describes a job for preparing a logical initial snapshot.
type LogicalInitial struct {
	name           string
	cloneManager   pool.FSManager
	tm             *telemetry.Agent
	fsPool         *resources.Pool
	dockerClient   *client.Client
	options        LogicalOptions
	globalCfg      *global.Config
	engineProps    *global.EngineProps
	dbMarker       *dbmarker.Marker
	queryProcessor *query.Processor
}

// LogicalOptions describes options for a logical initialization job.
type LogicalOptions struct {
	DataPatching        DataPatching      `yaml:"dataPatching"`
	PreprocessingScript string            `yaml:"preprocessingScript"`
	Configs             map[string]string `yaml:"configs"`
	Schedule            Scheduler         `yaml:"schedule"`
}

// DataPatching allows executing queries to transform data before snapshot taking.
type DataPatching struct {
	DockerImage        string                 `yaml:"dockerImage"`
	QueryPreprocessing query.PreprocessorCfg  `yaml:"queryPreprocessing"`
	ContainerConfig    map[string]interface{} `yaml:"containerConfig"`
}

// NewLogicalInitialJob creates a new logical initial job.
func NewLogicalInitialJob(cfg config.JobConfig, global *global.Config, engineProps *global.EngineProps, cloneManager pool.FSManager,
	tm *telemetry.Agent) (*LogicalInitial, error) {
	li := &LogicalInitial{
		name:         cfg.Spec.Name,
		cloneManager: cloneManager,
		fsPool:       cfg.FSPool,
		dockerClient: cfg.Docker,
		globalCfg:    global,
		engineProps:  engineProps,
		dbMarker:     cfg.Marker,
		tm:           tm,
	}

	if err := li.Reload(cfg.Spec.Options); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	if qp := li.options.DataPatching.QueryPreprocessing; qp.QueryPath != "" || qp.Inline != "" {
		li.queryProcessor = query.NewQueryProcessor(cfg.Docker, qp, global.Database.Name(), global.Database.User())
	}

	return li, nil
}

// Name returns a name of the job.
func (s *LogicalInitial) Name() string {
	return s.name
}

// patchContainerName returns container name.
func (s *LogicalInitial) patchContainerName() string {
	return patchContainerPrefix + s.engineProps.InstanceID
}

// Reload reloads job configuration.
func (s *LogicalInitial) Reload(cfg map[string]interface{}) (err error) {
	return options.Unmarshal(cfg, &s.options)
}

// ReportActivity reports the current job activity.
func (s *LogicalInitial) ReportActivity(_ context.Context) (*activity.Activity, error) {
	return &activity.Activity{}, nil
}

// Run starts the job.
func (s *LogicalInitial) Run(ctx context.Context) error {
	if s.options.PreprocessingScript != "" {
		if err := runPreprocessingScript(s.options.PreprocessingScript); err != nil {
			return err
		}
	}

	if err := s.touchConfigFiles(); err != nil {
		return errors.Wrap(err, "failed to create PostgreSQL configuration files")
	}

	dataDir := s.fsPool.DataDir()

	// Run basic PostgreSQL configuration.
	cfgManager, err := pgconfig.NewCorrector(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to create a config manager")
	}

	// Apply snapshot-specific configs.
	if err := cfgManager.ApplySnapshot(s.options.Configs); err != nil {
		return errors.Wrap(err, "failed to store PostgreSQL configs for the snapshot")
	}

	if s.queryProcessor != nil {
		if err := s.runPreprocessingQueries(ctx, dataDir); err != nil {
			return errors.Wrap(err, "failed to run preprocessing queries")
		}
	}

	log.Dbg("Cleaning up old snapshots from a dataset")

	if _, err := s.cloneManager.CleanupSnapshots(0); err != nil {
		return errors.Wrap(err, "failed to destroy old snapshots")
	}

	dataStateAt := extractDataStateAt(s.dbMarker)

	if _, err := s.cloneManager.CreateSnapshot("", dataStateAt); err != nil {
		var existsError *thinclones.SnapshotExistsError
		if errors.As(err, &existsError) {
			log.Msg("Skip snapshotting: ", existsError.Error())
			return err
		}

		return errors.Wrap(err, "failed to create a snapshot")
	}

	if err := s.markDatabaseData(dataStateAt); err != nil {
		return errors.Wrap(err, "failed to mark logical data")
	}

	s.tm.SendEvent(ctx, telemetry.SnapshotCreatedEvent, telemetry.SnapshotCreated{})

	return nil
}

func (s *LogicalInitial) markDatabaseData(dataStateAt string) error {
	if dataStateAt != "" {
		return nil
	}

	if err := s.dbMarker.CreateConfig(); err != nil {
		return errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return s.dbMarker.SaveConfig(&dbmarker.Config{
		DataType:    dbmarker.LogicalDataType,
		DataStateAt: time.Now().Format(util.DataStateAtFormat),
	})
}

func (s *LogicalInitial) touchConfigFiles() error {
	dataDir := s.fsPool.DataDir()

	if err := tools.TouchFile(path.Join(dataDir, "postgresql.conf")); err != nil {
		return err
	}

	return tools.TouchFile(path.Join(dataDir, "pg_hba.conf"))
}

func (s *LogicalInitial) runPreprocessingQueries(ctx context.Context, dataDir string) error {
	pgVersion, err := tools.DetectPGVersion(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	patchImage := s.options.DataPatching.DockerImage
	if patchImage == "" {
		patchImage = fmt.Sprintf("postgresai/extended-postgres:%g", pgVersion)
	}

	if err := tools.PullImage(ctx, s.dockerClient, patchImage); err != nil {
		return errors.Wrap(err, "failed to scan image pulling response")
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate PostgreSQL password")
	}

	hostConfig, err := cont.BuildHostConfig(ctx, s.dockerClient, s.fsPool.DataDir(), s.options.DataPatching.ContainerConfig)
	if err != nil {
		return errors.Wrap(err, "failed to build container host config")
	}

	// Run patch container.
	containerID, err := tools.CreateContainerIfMissing(ctx, s.dockerClient, s.patchContainerName(),
		s.buildContainerConfig(dataDir, patchImage, pwd), hostConfig)

	if err != nil {
		return fmt.Errorf("failed to create container %w", err)
	}

	defer tools.RemoveContainer(ctx, s.dockerClient, containerID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, s.dockerClient, s.patchContainerName())
			tools.PrintLastPostgresLogs(ctx, s.dockerClient, s.patchContainerName(), dataDir)

			filterArgs := filters.NewArgs(
				filters.KeyValuePair{Key: "label",
					Value: fmt.Sprintf("%s=%s", cont.DBLabControlLabel, cont.DBLabPatchLabel)})

			if err := diagnostic.CollectDiagnostics(ctx, s.dockerClient, filterArgs, s.patchContainerName(), dataDir); err != nil {
				log.Err("failed to collect container diagnostics", err)
			}
		}
	}()

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", s.patchContainerName(), containerID))

	if err := s.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg("Starting PostgreSQL and waiting for readiness")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, s.patchContainerName()))

	if err := tools.CheckContainerReadiness(ctx, s.dockerClient, containerID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	if err := s.queryProcessor.ApplyPreprocessingQueries(ctx, containerID); err != nil {
		return errors.Wrap(err, "failed to run preprocessing queries")
	}

	if err := tools.RunCheckpoint(ctx, s.dockerClient, containerID, s.globalCfg.Database.User(), s.globalCfg.Database.Name()); err != nil {
		return errors.Wrap(err, "failed to run checkpoint before stop")
	}

	return nil
}

func (s *LogicalInitial) buildContainerConfig(clonePath, patchImage, password string) *container.Config {
	hcInterval := health.DefaultRestoreInterval
	hcRetries := health.DefaultRestoreRetries

	return &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabPatchLabel,
			cont.DBLabInstanceIDLabel: s.engineProps.InstanceID,
			cont.DBLabEngineNameLabel: s.engineProps.ContainerName,
		},
		Env: []string{
			"PGDATA=" + clonePath,
			"POSTGRES_PASSWORD=" + password,
		},
		Image: patchImage,
		Healthcheck: health.GetConfig(
			s.globalCfg.Database.User(),
			s.globalCfg.Database.Name(),
			health.OptionInterval(hcInterval),
			health.OptionRetries(hcRetries),
		),
	}
}
