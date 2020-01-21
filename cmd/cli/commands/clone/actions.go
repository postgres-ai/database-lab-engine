/*
2020 © Postgres.ai
*/

// Package clone provides clones management commands.
package clone

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/client"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
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

		cloneRequest := client.CreateRequest{
			Name:      cliCtx.String("name"),
			Project:   cliCtx.String("project"),
			Protected: cliCtx.Bool("protected"),
			DB: &client.DatabaseRequest{
				Username: cliCtx.String("username"),
				Password: cliCtx.String("password"),
			},
		}

		clone, err := dblabClient.CreateClone(cliCtx.Context, cloneRequest)
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

		updateRequest := client.UpdateRequest{
			Name:      cliCtx.String("name"),
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
		if err := dblabClient.ResetClone(cliCtx.Context, cloneID); err != nil {
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
		if err = dblabClient.DestroyClone(cliCtx.Context, cloneID); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cliCtx.App.Writer, "The clone has been successfully destroyed: %s\n", cloneID)

		return err
	}
}
