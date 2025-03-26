/*
2020 © Postgres.ai
*/

package branch

import (
	"github.com/urfave/cli/v2"
)

// List provides commands for getting started.
func List() []*cli.Command {
	return []*cli.Command{
		{
			Name:   "branch",
			Usage:  "list, create, or delete branches",
			Action: list,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "delete",
					Aliases: []string{"d"},
				},
				&cli.StringFlag{
					Name:  "parent-branch",
					Usage: "specify branch name as starting point for new branch; cannot be used together with --snapshot-id",
				},
				&cli.StringFlag{
					Name:  "snapshot-id",
					Usage: "specify snapshot ID is starting point for new branch; cannot be used together with --parent-branch",
				},
			},
			ArgsUsage: "BRANCH_NAME",
		},
		{
			Name:   "switch",
			Usage:  "switch to a specified branch",
			Action: switchBranch,
		},
		{
			Name:   "commit",
			Usage:  "create a new snapshot containing the current state of data and the given log message describing the changes",
			Action: commit,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "clone-id",
					Usage: "clone ID",
				},
				&cli.StringFlag{
					Name:    "message",
					Usage:   "use the given message as the commit message",
					Aliases: []string{"m"},
				},
			},
		},
		{
			Name:      "log",
			Usage:     "shows the snapshot logs",
			Action:    history,
			ArgsUsage: "BRANCH_NAME",
		},
	}
}
