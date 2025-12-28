/*
2020 © Postgres.ai
*/

// Package instance provides instance management commands.
package instance

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/format"
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

	cfg := format.FromContext(cliCtx)

	if cfg.IsJSON() {
		return outputJSON(cliCtx.App.Writer, instanceStatusView)
	}

	return printInstanceStatus(cfg, instanceStatusView)
}

func outputJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, string(data))

	return err
}

func printInstanceStatus(cfg format.Config, instance *models.InstanceStatusView) error {
	w := cfg.Writer

	statusCode := ""
	if instance.Status != nil {
		statusCode = format.FormatStatus(string(instance.Status.Code), cfg.NoColor)
	}

	fmt.Fprintf(w, "Status:      %s\n", statusCode)

	if instance.Status != nil && instance.Status.Message != "" {
		fmt.Fprintf(w, "Message:     %s\n", instance.Status.Message)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Engine:")
	fmt.Fprintf(w, "  Version:     %s (%s)\n", instance.Engine.Version, instance.Engine.Edition)
	fmt.Fprintf(w, "  Instance ID: %s\n", instance.Engine.InstanceID)

	if instance.Engine.StartedAt != nil {
		fmt.Fprintf(w, "  Started:     %s (%s)\n",
			format.FormatTimeAbs(instance.Engine.StartedAt.Time),
			format.FormatTime(instance.Engine.StartedAt.Time))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Cloning:")
	fmt.Fprintf(w, "  Active clones:  %d\n", instance.Cloning.NumClones)
	fmt.Fprintf(w, "  Expected time:  %.2fs\n", instance.Cloning.ExpectedCloningTime)

	if len(instance.Pools) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Pools:")

		t := format.NewTable(w, cfg.NoColor)
		t.SetHeaders("NAME", "MODE", "STATUS", "DATA STATE", "USED", "FREE", "CLONES")

		for _, pool := range instance.Pools {
			dataState := ""
			if pool.DataStateAt != nil {
				dataState = format.FormatTime(pool.DataStateAt.Time)
			}

			poolStatus := format.FormatStatus(string(pool.Status), cfg.NoColor)

			t.Append([]string{
				pool.Name,
				pool.Mode,
				poolStatus,
				dataState,
				format.FormatBytes(uint64(pool.FileSystem.Used)),
				format.FormatBytes(uint64(pool.FileSystem.Free)),
				strconv.Itoa(len(pool.CloneList)),
			})
		}

		t.Render()
	}

	if instance.Retrieving.Mode != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Retrieving:")
		fmt.Fprintf(w, "  Mode:    %s\n", instance.Retrieving.Mode)
		fmt.Fprintf(w, "  Status:  %s\n", format.FormatStatus(string(instance.Retrieving.Status), cfg.NoColor))

		if instance.Retrieving.LastRefresh != nil {
			fmt.Fprintf(w, "  Last:    %s\n", format.FormatTime(instance.Retrieving.LastRefresh.Time))
		}

		if instance.Retrieving.NextRefresh != nil {
			fmt.Fprintf(w, "  Next:    %s\n", format.FormatTime(instance.Retrieving.NextRefresh.Time))
		}
	}

	return nil
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
