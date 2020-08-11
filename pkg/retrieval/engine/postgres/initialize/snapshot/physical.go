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
	"os"
	"os/exec"
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

	adjustingConfigsScript = "/scripts/adjust_configs.sh"
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
	Promote             bool   `yaml:"promote"`
	PreprocessingScript string `yaml:"preprocessingScript"`
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

	if err := p.dbMarker.CreateConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to create a DBMarker config of the database")
	}

	return p, nil
}

// Name returns a name of the job.
func (p *PhysicalInitial) Name() string {
	return p.name
}

// Run starts the job.
func (p *PhysicalInitial) Run(ctx context.Context) error {
	// TODO(akartasov): Automated basic Postgres configuration: https://gitlab.com/postgres-ai/database-lab/-/issues/141
	p.dbMark.DataStateAt = extractDataStateAt(p.dbMarker)

	// Promotion.
	if p.options.Promote {
		if err := p.promoteInstance(ctx); err != nil {
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
	if err := p.dbMarker.SaveConfig(p.dbMark); err != nil {
		return errors.Wrap(err, "failed to mark the prepared data")
	}

	// Create a snapshot.
	if _, err := p.cloneManager.CreateSnapshot(p.dbMark.DataStateAt); err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	return nil
}

func (p *PhysicalInitial) promoteInstance(ctx context.Context) error {
	log.Msg("Promote the Postgres instance.")

	// Pre snapshot.
	preDataStateAt := time.Now().Format(tools.DataStateAtFormat)

	snapshotName, err := p.cloneManager.CreateSnapshot(preDataStateAt + pre)
	if err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	defer func() {
		if err := p.cloneManager.DestroySnapshot(snapshotName); err != nil {
			log.Err(fmt.Sprintf("Failed to destroy the %q snapshot: %v", snapshotName, err))
		}
	}()

	cloneName := fmt.Sprintf("clone%s_%s", pre, preDataStateAt)

	if err := p.cloneManager.CreateClone(cloneName, snapshotName); err != nil {
		return errors.Wrap(err, "failed to create a pre clone")
	}

	defer func() {
		if err := p.cloneManager.DestroyClone(cloneName); err != nil {
			log.Err(fmt.Sprintf("Failed to destroy the %q clone: %v", cloneName, err))
		}
	}()

	pgVersion, err := p.detectPGVersion()
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	// Adjust configuration.
	err = p.adjustConfiguration(pgVersion, cloneName)
	if err != nil {
		return errors.Wrap(err, "failed to adjust configuration")
	}

	promoteImage := fmt.Sprintf("postgres:%s-alpine", pgVersion)

	if err := tools.PullImage(ctx, p.dockerClient, promoteImage); err != nil {
		return errors.Wrap(err, "failed to scan image response")
	}

	// Run promotion container.
	cont, err := p.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{"label": "dblab_control"},
			Env: []string{
				"PGDATA=" + pgDataContainerDir,
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
					Source: p.globalCfg.DataDir,
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
		p.dbMark.DataStateAt, err = p.extractDataStateAt(ctx, cont.ID)
		if err != nil {
			return errors.Wrap(err,
				`Failed to get data_state_at: PGDATA should be promoted, but pg_last_xact_replay_timestamp() returns empty result.
				Check if pg_data is correct, or explicitly define DATA_STATE_AT via an environment variable.`)
		}
	}

	log.Msg("Data state at: ", p.dbMark.DataStateAt)

	// Promote PGDATA.
	if shouldBePromoted == "t" {
		if err := p.runPromoteCommand(ctx, cont.ID); err != nil {
			return errors.Wrap(err, "failed to promote PGDATA")
		}
	}

	// Checkpoint.
	return p.checkpoint(ctx, cont.ID)
}

func (p *PhysicalInitial) detectPGVersion() (string, error) {
	version, err := exec.Command("cat", fmt.Sprintf(`%s/PG_VERSION`, p.globalCfg.DataDir)).CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(version)), nil
}

func (p *PhysicalInitial) adjustConfiguration(pgVersion, cloneName string) error {
	fmt.Println("Adjust configuration for PostgreSQL " + pgVersion)

	scriptDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "failed to get script path")
	}

	configCmd := exec.Command("bash", scriptDir+adjustingConfigsScript)
	configCmd.Env = []string{
		"PG_VER=" + pgVersion,
		"CLONE_NAME=" + cloneName,
	}

	adjustConfigs, err := configCmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "failed to run adjust command")
	}

	fmt.Println(string(adjustConfigs))

	return nil
}

func (p *PhysicalInitial) checkRecovery(ctx context.Context, containerID string) (string, error) {
	checkRecoveryCmd := []string{"psql", "-U", "postgres", "-XAtc", "select pg_is_in_recovery()"}
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
	promoteCommand := []string{"psql", "-U", "postgres", "-d", "postgres", "-XAtc",
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

	execCommand, err := p.dockerClient.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          promoteCommand,
		User:         defaults.Username,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create exec command")
	}

	attachResponse, err := p.dockerClient.ContainerExecAttach(ctx, execCommand.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return errors.Wrap(err, "failed to attach to exec command")
	}

	defer attachResponse.Close()

	return tools.ProcessAttachResponse(ctx, attachResponse.Reader, os.Stdout)
}

func (p *PhysicalInitial) checkpoint(ctx context.Context, containerID string) error {
	commandCheckpoint := []string{"psql", "-U", "postgres", "-d", "postgres", "-XAtc", "checkpoint"}
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
