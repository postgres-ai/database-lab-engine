/*
2020 © Postgres.ai
*/

// Package instance provides instance management commands.
package instance

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v2/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

// status runs a request to get status of the instance.
func status(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	body, err := dblabClient.StatusRaw(cliCtx.Context)
	if err != nil {
		return err
	}

	defer func() { _ = body.Close() }()

	var instanceStatusView *models.InstanceStatusView

	if err := json.NewDecoder(body).Decode(&instanceStatusView); err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(instanceStatusView, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// health runs a request to get health info of the instance.
func health(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	list, err := dblabClient.Health(cliCtx.Context)
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
