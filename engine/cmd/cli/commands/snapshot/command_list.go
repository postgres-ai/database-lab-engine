/*
2020 © Postgres.ai
*/

package snapshot

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
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
				{
					Name:   "create",
					Usage:  "create a snapshot",
					Action: create,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "pool",
							Usage: "pool name",
						},
						&cli.StringFlag{
							Name:  "clone-id",
							Usage: "create a snapshot from existing clone",
						},
					},
				},
				{
					Name:      "delete",
					Usage:     "delete existing snapshot",
					Action:    deleteSnapshot,
					ArgsUsage: "SNAPSHOT_ID",
					Before:    checkSnapshotIDBefore,
				},
			},
		},
	}
}

func checkSnapshotIDBefore(c *cli.Context) error {
	if c.NArg() == 0 {
		return commands.NewActionError("SNAPSHOT_ID argument is required")
	}

	return nil
}
