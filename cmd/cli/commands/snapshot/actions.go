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
)

// list runs a request to list snapshots of an instance.
func list() func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		dblabClient, err := commands.ClientByCLIContext(cliCtx)
		if err != nil {
			return err
		}

		list, err := dblabClient.ListSnapshots(cliCtx.Context)
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
