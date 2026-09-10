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
			Name:  "branch",
			Usage: "list, create, delete, or switch branches",
			Description: "Without arguments, lists branches. A bare positional BRANCH_NAME creates a branch, so a name\n" +
				"   that looks like a verb is better spelled out: use `branch create list` to create a branch named\n" +
				"   `list`, and the subcommands below for everything else.",
			Action: branchAction,
			Before: rejectBareFormFlags,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "delete",
					Usage:   "delete the branch with the given name; same as the delete subcommand",
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
				&cli.StringFlag{
					Name:    "protected",
					Usage:   "update deletion protection of BRANCH_NAME: 'true'=default, minutes or 30m/2h/7d, 0=forever, 'false'=off",
					Aliases: []string{"p"},
				},
			},
			ArgsUsage: "[BRANCH_NAME]",
			Subcommands: []*cli.Command{
				{
					Name:            "list",
					Aliases:         []string{"ls"},
					Usage:           "list branches",
					Action:          listBranches,
					HideHelpCommand: true,
				},
				{
					Name:            "create",
					Usage:           "create a new branch",
					Action:          create,
					ArgsUsage:       "BRANCH_NAME",
					HideHelpCommand: true,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "parent-branch",
							Usage: "specify branch name as starting point for new branch; cannot be used together with --snapshot-id",
						},
						&cli.StringFlag{
							Name:  "snapshot-id",
							Usage: "specify snapshot ID is starting point for new branch; cannot be used together with --parent-branch",
						},
					},
				},
				{
					Name:            "delete",
					Aliases:         []string{"rm"},
					Usage:           "delete a branch",
					Action:          deleteBranch,
					ArgsUsage:       "BRANCH_NAME",
					HideHelpCommand: true,
				},
				{
					Name:            "switch",
					Usage:           "switch to a specified branch",
					Action:          switchBranch,
					ArgsUsage:       "BRANCH_NAME",
					HideHelpCommand: true,
				},
			},
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
