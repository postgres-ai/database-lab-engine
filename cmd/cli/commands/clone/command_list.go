/*
2020 © Postgres.ai
*/

package clone

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
)

// CommandList returns available commands for a clones management.
func CommandList() []*cli.Command {
	return []*cli.Command{{
		Name:  "clone",
		Usage: "manages clones",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "list all existing clones",
				Action: list(),
			},
			{
				Name:      "status",
				Usage:     "display clone's information",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    status(),
			},
			{
				Name:   "create",
				Usage:  "create new clone",
				Action: create(),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "username",
						Usage:    "database username",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "password",
						Usage:    "database password",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "id",
						Usage: "clone ID (optional)",
					},
					&cli.StringFlag{
						Name:  "snapshot-id",
						Usage: "snapshot ID (optional)",
					},
					&cli.StringFlag{
						Name:  "project",
						Usage: "project name (optional)",
					},
					&cli.BoolFlag{
						Name:    "protected",
						Usage:   "mark instance as protected from deletion",
						Aliases: []string{"p"},
					},
					&cli.BoolFlag{
						Name:    "async",
						Usage:   "run the command asynchronously",
						Aliases: []string{"a"},
					},
				},
			},
			{
				Name:      "update",
				Usage:     "update existing clone",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    update(),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "protected",
						Usage:   "mark instance as protected from deletion",
						Aliases: []string{"p"},
					},
				},
			},
			{
				Name:      "reset",
				Usage:     "reset clone's state",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    reset(),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "async",
						Usage:   "run the command asynchronously",
						Aliases: []string{"a"},
					},
				},
			},
			{
				Name:      "destroy",
				Usage:     "destroy clone",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    destroy(),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "async",
						Usage:   "run the command asynchronously",
						Aliases: []string{"a"},
					},
				},
			},
		},
	}}
}

func checkCloneIDBefore(c *cli.Context) error {
	if c.NArg() == 0 {
		return commands.NewActionError("CLONE_ID argument is required")
	}

	return nil
}
