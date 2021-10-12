/*
2020 © Postgres.ai
*/

package snapshot

import (
	"github.com/urfave/cli/v2"
)

// CommandList returns available commands for a snapshot management.
func CommandList() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "snapshot",
			Usage: "manage snapshots",
			Subcommands: []*cli.Command{
				{
					Name:   "list",
					Usage:  "list all existing snapshots",
					Action: list,
				},
			},
		},
	}
}
