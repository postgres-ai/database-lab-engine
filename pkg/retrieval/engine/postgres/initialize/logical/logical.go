/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
)

func recalculateStats(ctx context.Context, dockerClient *client.Client, contID string, analyzeCmd []string) error {
	log.Msg("Running analyze command: ", analyzeCmd)

	exec, err := dockerClient.ContainerExecCreate(ctx, contID, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          analyzeCmd,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	if err := dockerClient.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{Tty: true}); err != nil {
		return errors.Wrap(err, "failed to run the exec command")
	}

	if err := tools.InspectCommandResponse(ctx, dockerClient, contID, exec.ID); err != nil {
		return errors.Wrap(err, "failed to exec the restore command")
	}

	return nil
}

func buildAnalyzeCommand(conn Connection) []string {
	analyzeCmd := []string{
		"psql",
		"-U", conn.Username,
		"-d", conn.DBName,
		"-c", "vacuum freeze analyze;",
	}

	return analyzeCmd
}
