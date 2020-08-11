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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools/health"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/thinclones"
)

const (
	// PhysicalInitialType declares a job type for preparing a physical snapshot.
	PhysicalInitialType = "physical-snapshot"

	pre                  = "_pre"
	pgDataContainerDir   = "/var/lib/postgresql/pgdata"
	promoteContainerName = "dblab_promote"

	// Defines container health check options.
	hcPromoteInterval = 2 * time.Second
	hcPromoteRetries  = 100
)

// PhysicalInitial describes a job for preparing a physical initial snapshot.
type PhysicalInitial struct {
	name         string
	cloneManager thinclones.Manager
	options      PhysicalOptions
	globalCfg    *dblabCfg.Global
	dbMarker     *dbmarker.Marker
	dbMark       *dbmarker.Config
	dockerClient *client.Client
}

// PhysicalOptions describes options for a physical initialization job.
type PhysicalOptions struct {
	Promote             bool              `yaml:"promote"`
	PreprocessingScript string            `yaml:"preprocessingScript"`
	Configs             map[string]string `yaml:"configs"`
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

	return p, nil
}

// Name returns a name of the job.
func (p *PhysicalInitial) Name() string {
	return p.name
}

// Run starts the job.
func (p *PhysicalInitial) Run(ctx context.Context) error {
	p.dbMark.DataStateAt = extractDataStateAt(p.dbMarker)

	// Snapshot data.
	preDataStateAt := time.Now().Format(tools.DataStateAtFormat)
	cloneName := fmt.Sprintf("clone%s_%s", pre, preDataStateAt)

	// Promotion.
	if p.options.Promote {
		// Prepare pre-snapshot.
		snapshotName, err := p.cloneManager.CreateSnapshot("", preDataStateAt+pre)
		if err != nil {
			return errors.Wrap(err, "failed to create a snapshot")
		}

		defer func() {
			if err != nil {
				if errDestroy := p.cloneManager.DestroySnapshot(snapshotName); errDestroy != nil {
					log.Err(fmt.Sprintf("Failed to destroy the %q snapshot: %v", snapshotName, err))
				}
			}
		}()

		if err := p.cloneManager.CreateClone(cloneName, snapshotName); err != nil {
			return errors.Wrap(err, "failed to create a pre clone")
		}

		defer func() {
			if err != nil {
				if errDestroy := p.cloneManager.DestroyClone(cloneName); errDestroy != nil {
					log.Err(fmt.Sprintf("Failed to destroy the %q clone: %v", cloneName, err))
				}
			}
		}()

		if err := p.promoteInstance(ctx, path.Join(p.globalCfg.MountDir, cloneName)); err != nil {
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

func (p *PhysicalInitial) promoteInstance(ctx context.Context, clonePath string) error {
	log.Msg("Promote the Postgres instance.")

	if err := configuration.Run(clonePath); err != nil {
		return errors.Wrap(err, "failed to enforce configs")
	}

	// Apply users configs.
	if err := applyUsersConfigs(p.options.Configs, path.Join(clonePath, "postgresql.conf")); err != nil {
		return err
	}

	pgVersion, err := tools.DetectPGVersion(p.globalCfg.DataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	// Adjust recovery configuration.
	if err := p.adjustRecoveryConfiguration(pgVersion, clonePath); err != nil {
		return errors.Wrap(err, "failed to adjust recovery configuration")
	}

	promoteImage := fmt.Sprintf("postgresai/sync-instance:%s", pgVersion)

	if err := tools.PullImage(ctx, p.dockerClient, promoteImage); err != nil {
		return errors.Wrap(err, "failed to scan image response")
	}

	// Run promotion container.
	cont, err := p.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{"label": "dblab_control"},
			Env: []string{
				"PGDATA=" + pgDataContainerDir,
				"POSTGRES_HOST_AUTH_METHOD=trust",
			},
			Image: promoteImage,
			Healthcheck: health.GetConfig(
				health.OptionInterval(hcPromoteInterval),
				health.OptionRetries(hcPromoteRetries),
			),
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: clonePath,
					Target: pgDataContainerDir,
					BindOptions: &mount.BindOptions{
						Propagation: mount.PropagationRShared,
					},
				},
			},
		},
		&network.NetworkingConfig{},
		promoteContainerName,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	defer tools.RemoveContainer(ctx, p.dockerClient, cont.ID, tools.StopTimeout)

	if err := p.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", promoteContainerName, cont.ID))

	// Set permissions.
	if err := tools.ExecCommand(ctx, p.dockerClient, cont.ID, types.ExecConfig{
		Cmd: []string{"chown", "-R", "postgres", pgDataContainerDir},
	}); err != nil {
		return errors.Wrap(err, "failed to set permissions")
	}

	// Start PostgreSQL instance.
	startCommand, err := p.dockerClient.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
		Cmd:  []string{"postgres", "-D", pgDataContainerDir},
		User: defaults.Username,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	log.Msg("Running PostgreSQL instance")

	if _, err := p.dockerClient.ContainerExecAttach(ctx, startCommand.ID, types.ExecStartCheck{Tty: true}); err != nil {
		return errors.Wrap(err, "failed to attach to the exec command")
	}

	if err := tools.CheckContainerReadiness(ctx, p.dockerClient, cont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	shouldBePromoted, err := p.checkRecovery(ctx, cont.ID)
	if err != nil {
		return errors.Wrap(err, "failed to read response of the exec command")
	}

	log.Msg("Should be promoted: ", shouldBePromoted)

	// Detect dataStateAt.
	if p.dbMark.DataStateAt == "" && shouldBePromoted == "t" {
		extractedDataStateAt, err := p.extractDataStateAt(ctx, cont.ID)
		if err != nil {
			return errors.Wrap(err,
				`Failed to get data_state_at: PGDATA should be promoted, but pg_last_xact_replay_timestamp() returns empty result.
				Check if pg_data is correct, or explicitly define DATA_STATE_AT via an environment variable.`)
		}

		if extractedDataStateAt == p.dbMark.DataStateAt {
			log.Msg(fmt.Sprintf(`The previous snapshot already contains the latest data: %s. Skip taking a new snapshot.`,
				p.dbMark.DataStateAt))

			return nil
		}

		p.dbMark.DataStateAt = extractedDataStateAt
	}

	log.Msg("Data state at: ", p.dbMark.DataStateAt)

	// Promote PGDATA.
	if shouldBePromoted == "t" {
		if err := p.runPromoteCommand(ctx, cont.ID); err != nil {
			return errors.Wrap(err, "failed to promote PGDATA")
		}
	}

	// Checkpoint.
	if err := p.checkpoint(ctx, cont.ID); err != nil {
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

	version, err := strconv.Atoi(pgVersion)
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

func (p *PhysicalInitial) runPromoteCommand(ctx context.Context, containerID string) error {
	promoteCommand := []string{"pg_ctl", "-D", pgDataContainerDir, "-W", "promote"}

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
