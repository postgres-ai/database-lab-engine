/*
2020 © Postgres.ai
*/

// Package instance provides instance management commands.
package instance

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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

	engineHealth, err := dblabClient.Health(cliCtx.Context)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(engineHealth, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// refresh runs a request to initiate a full refresh.
func refresh(cliCtx *cli.Context) error {
	client, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	response, err := client.FullRefresh(cliCtx.Context)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, response.Message)

	return err
}
