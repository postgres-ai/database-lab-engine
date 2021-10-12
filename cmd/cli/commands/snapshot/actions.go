/*
2020 © Postgres.ai
*/

// Package snapshot provides snapshot management commands.
package snapshot

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v2/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

// list runs a request to list snapshots of an instance.
func list(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	body, err := dblabClient.ListSnapshotsRaw(cliCtx.Context)
	if err != nil {
		return err
	}

	defer func() { _ = body.Close() }()

	var snapshotListView []*models.SnapshotView

	if err := json.NewDecoder(body).Decode(&snapshotListView); err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(snapshotListView, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}
