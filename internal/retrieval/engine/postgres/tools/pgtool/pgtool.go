/*
2021 © Postgres.ai
*/

// Package pgtool provides tools to run PostgreSQL-specific commands.
package pgtool

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// ReadControlData reads a control data file.
func ReadControlData(ctx context.Context, d *client.Client, contID, dataDir string, pgVersion float64) (types.HijackedResponse, error) {
	controlDataCmd, err := d.ContainerExecCreate(ctx, contID, pgControlDataConfig(dataDir, pgVersion))

	if err != nil {
		return types.HijackedResponse{}, errors.Wrap(err, "failed to create an exec command")
	}

	attachResponse, err := d.ContainerExecAttach(ctx, controlDataCmd.ID, types.ExecStartCheck{})
	if err != nil {
		return types.HijackedResponse{}, errors.Wrap(err, "failed to attach to the exec command")
	}

	return attachResponse, nil
}

func pgControlDataConfig(pgDataDir string, pgVersion float64) types.ExecConfig {
	command := fmt.Sprintf("/usr/lib/postgresql/%g/bin/pg_controldata", pgVersion)

	return types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{command, "-D", pgDataDir},
		Env:          os.Environ(),
	}
}
