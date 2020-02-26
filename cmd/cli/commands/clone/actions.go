/*
2020 © Postgres.ai
*/

// Package clone provides clones management commands.
package clone

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

// list runs a request to list clones of an instance.
func list() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		list, err := dblabClient.ListClones(cliCtx.Context)
		if err != nil {
			return err
		}

		commandResponse, err := json.MarshalIndent(list, "", "    ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return err
	}
}

// create runs a request to create a new clone.
func create() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		cloneRequest := types.CloneCreateRequest{
			ID:        cliCtx.String("id"),
			Project:   cliCtx.String("project"),
			Protected: cliCtx.Bool("protected"),
			DB: &types.DatabaseRequest{
				Username: cliCtx.String("username"),
				Password: cliCtx.String("password"),
			},
		}

		if cliCtx.IsSet("snapshot-id") {
			cloneRequest.Snapshot = &types.SnapshotCloneFieldRequest{ID: cliCtx.String("snapshot-id")}
		}

		var clone *models.Clone

		if cliCtx.Bool("async") {
			clone, err = dblabClient.CreateCloneAsync(cliCtx.Context, cloneRequest)
		} else {
			clone, err = dblabClient.CreateClone(cliCtx.Context, cloneRequest)
		}

		if err != nil {
			return err
		}

		commandResponse, err := json.MarshalIndent(clone, "", "    ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return err
	}
}

// status runs a request to get clone info.
func status() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		clone, err := dblabClient.GetClone(cliCtx.Context, cliCtx.Args().First())
		if err != nil {
			return err
		}

		commandResponse, err := json.MarshalIndent(clone, "", "    ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return err
	}
}

// update runs a request to update an existing clone.
func update() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		updateRequest := types.CloneUpdateRequest{
			Protected: cliCtx.Bool("protected"),
		}

		cloneID := cliCtx.Args().First()

		clone, err := dblabClient.UpdateClone(cliCtx.Context, cloneID, updateRequest)
		if err != nil {
			return err
		}

		commandResponse, err := json.MarshalIndent(clone, "", "    ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

		return err
	}
}

// reset runs a request to reset clone.
func reset() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		cloneID := cliCtx.Args().First()

		if cliCtx.Bool("async") {
			err = dblabClient.ResetCloneAsync(cliCtx.Context, cloneID)
		} else {
			err = dblabClient.ResetClone(cliCtx.Context, cloneID)
		}

		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The clone has been successfully reset: %s\n", cloneID)

		return err
	}
}

// destroy runs a request to destroy clone.
func destroy() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		cloneID := cliCtx.Args().First()

		if cliCtx.Bool("async") {
			err = dblabClient.DestroyCloneAsync(cliCtx.Context, cloneID)
		} else {
			err = dblabClient.DestroyClone(cliCtx.Context, cloneID)
		}

		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The clone has been successfully destroyed: %s\n", cloneID)

		return err
	}
}
