/*
2020 © Postgres.ai
*/

package instance

import (
	"github.com/urfave/cli/v2"
)

// CommandList returns available commands for an instance management.
func CommandList() []*cli.Command {
	return []*cli.Command{
		{
			Name:   "instance",
			Usage:  "Displays instance info",
			Action: cli.ShowSubcommandHelp,
			Subcommands: []*cli.Command{
				{
					Name:   "status",
					Usage:  "display instance's status",
					Action: status(),
				},
			},
		},
	}
}
