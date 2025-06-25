/*
2020 © Postgres.ai
*/

// Package snapshot provides snapshot management commands.
package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
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

	cloneID := cliCtx.String("clone-id")

	var commandResponse []byte

	if cloneID != "" {
		commandResponse, err = createFromClone(cliCtx, dblabClient)
	} else {
		commandResponse, err = createOnPool(cliCtx, dblabClient)
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// createOnPool runs a request to create a new snapshot.
func createOnPool(cliCtx *cli.Context, client *dblabapi.Client) ([]byte, error) {
	snapshotRequest := types.SnapshotCreateRequest{
		PoolName: cliCtx.String("pool"),
	}

	snapshot, err := client.CreateSnapshot(cliCtx.Context, snapshotRequest)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(snapshot, "", "    ")
}

// createFromClone runs a request to create a new snapshot from clone.
func createFromClone(cliCtx *cli.Context, client *dblabapi.Client) ([]byte, error) {
	cloneID := cliCtx.String("clone-id")
	message := cliCtx.String("message")

	snapshotRequest := types.SnapshotCloneCreateRequest{
		CloneID: cloneID,
		Message: message,
	}

	snapshot, err := client.CreateSnapshotFromClone(cliCtx.Context, snapshotRequest)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(snapshot, "", "    ")
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
		return errors.Unwrap(err)
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "Deleted snapshot '%s'\n", snapshotID)

	return err
}
