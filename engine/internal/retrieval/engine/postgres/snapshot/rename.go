/*
2024 © Postgres.ai
*/

package snapshot

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/diagnostic"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const renameContainerPrefix = "dblab_rename_"

// runDatabaseRename renames databases using ALTER DATABASE in a temporary container.
func runDatabaseRename(
	ctx context.Context,
	dockerClient *client.Client,
	engineProps *global.EngineProps,
	globalCfg *global.Config,
	dataDir string,
	renames map[string]string,
) error {
	if len(renames) == 0 {
		return nil
	}

	connDB := globalCfg.Database.Name()

	if err := validateDatabaseRenames(renames, connDB); err != nil {
		return err
	}

	pgVersion, err := tools.DetectPGVersion(dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect postgres version")
	}

	image := fmt.Sprintf("postgresai/extended-postgres:%g", pgVersion)

	if err := tools.PullImage(ctx, dockerClient, image); err != nil {
		return errors.Wrap(err, "failed to pull image for database rename")
	}

	pwd, err := tools.GeneratePassword()
	if err != nil {
		return errors.Wrap(err, "failed to generate password")
	}

	hostConfig, err := cont.BuildHostConfig(ctx, dockerClient, dataDir, nil)
	if err != nil {
		return errors.Wrap(err, "failed to build host config")
	}

	containerName := renameContainerPrefix + engineProps.InstanceID

	containerID, err := tools.CreateContainerIfMissing(ctx, dockerClient, containerName,
		&container.Config{
			Labels: map[string]string{
				cont.DBLabControlLabel:    cont.DBLabRenameLabel,
				cont.DBLabInstanceIDLabel: engineProps.InstanceID,
				cont.DBLabEngineNameLabel: engineProps.ContainerName,
			},
			Env: []string{
				"PGDATA=" + dataDir,
				"POSTGRES_PASSWORD=" + pwd,
			},
			Image: image,
			Healthcheck: health.GetConfig(
				globalCfg.Database.User(),
				connDB,
			),
		},
		hostConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create rename container: %w", err)
	}

	defer tools.RemoveContainer(ctx, dockerClient, containerID, cont.StopPhysicalTimeout)

	defer func() {
		if err != nil {
			tools.PrintContainerLogs(ctx, dockerClient, containerName)
			tools.PrintLastPostgresLogs(ctx, dockerClient, containerName, dataDir)

			filterArgs := filters.NewArgs(
				filters.KeyValuePair{Key: "label",
					Value: fmt.Sprintf("%s=%s", cont.DBLabControlLabel, cont.DBLabRenameLabel)})

			if diagErr := diagnostic.CollectDiagnostics(ctx, dockerClient, filterArgs, containerName, dataDir); diagErr != nil {
				log.Err("failed to collect rename container diagnostics", diagErr)
			}
		}
	}()

	log.Msg(fmt.Sprintf("Running rename container: %s. ID: %v", containerName, containerID))

	if err = dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start rename container")
	}

	log.Msg("Waiting for rename container readiness")
	log.Msg(fmt.Sprintf("View logs using the command: %s %s", tools.ViewLogsCmd, containerName))

	if err = tools.CheckContainerReadiness(ctx, dockerClient, containerID); err != nil {
		return errors.Wrap(err, "rename container readiness check failed")
	}

	for oldName, newName := range renames {
		log.Msg(fmt.Sprintf("Renaming database %q to %q", oldName, newName))

		cmd := buildRenameCommand(globalCfg.Database.User(), connDB, oldName, newName)

		output, execErr := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, container.ExecOptions{Cmd: cmd})
		if execErr != nil {
			err = errors.Wrapf(execErr, "failed to rename database %q to %q", oldName, newName)
			return err
		}

		log.Msg("Rename result: ", output)
	}

	if err = tools.RunCheckpoint(ctx, dockerClient, containerID, globalCfg.Database.User(), connDB); err != nil {
		return errors.Wrap(err, "failed to run checkpoint after rename")
	}

	if err = tools.StopPostgres(ctx, dockerClient, containerID, dataDir, tools.DefaultStopTimeout); err != nil {
		return errors.Wrap(err, "failed to stop postgres after rename")
	}

	return nil
}

func buildRenameCommand(username, connDB, oldName, newName string) []string {
	return []string{
		"psql",
		"-U", username,
		"-d", connDB,
		"-XAtc", fmt.Sprintf(`ALTER DATABASE "%s" RENAME TO "%s"`, oldName, newName),
	}
}

func validateDatabaseRenames(renames map[string]string, connDB string) error {
	for oldName := range renames {
		if oldName == connDB {
			return fmt.Errorf("cannot rename database %q: it is used as the connection database", oldName)
		}
	}

	return nil
}
