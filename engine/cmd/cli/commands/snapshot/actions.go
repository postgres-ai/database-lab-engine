/*
2020 © Postgres.ai
*/

// Package snapshot provides snapshot management commands.
package snapshot

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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

// create runs a request to create a new snapshot.
func create(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	snapshotRequest := types.SnapshotCreateRequest{
		PoolName: cliCtx.String("pool"),
	}

	snapshot, err := dblabClient.CreateSnapshot(cliCtx.Context, snapshotRequest)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// deleteSnapshot runs a request to delete existing snapshot.
func deleteSnapshot(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	snapshotID := cliCtx.Args().First()

	snapshotRequest := types.SnapshotDestroyRequest{
		SnapshotID: snapshotID,
	}

	if err := dblabClient.DeleteSnapshot(cliCtx.Context, snapshotRequest); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "The snapshot has been successfully deleted: %s\n", snapshotID)

	return err
}
