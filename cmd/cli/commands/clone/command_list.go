/*
2020 © Postgres.ai
*/

package clone

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v2/cmd/cli/commands"
)

const (
	cloneResetLatestFlag     = "latest"
	cloneResetSnapshotIDFlag = "snapshot-id"
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
				Action: create,
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
					&cli.BoolFlag{
						Name:  "restricted",
						Usage: "create a user with restricted permissions",
					},
					&cli.StringFlag{
						Name:  "db-name",
						Usage: "database available to the user with restricted permissions",
					},
					&cli.StringFlag{
						Name:  "id",
						Usage: "clone ID (optional)",
					},
					&cli.StringFlag{
						Name:  "snapshot-id",
						Usage: "snapshot ID (optional)",
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
					&cli.StringSliceFlag{
						Name:  "extra-config",
						Usage: "set an extra database configuration for the clone. An example: statement_timeout='1s'",
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
					&cli.BoolFlag{
						Name:  cloneResetLatestFlag,
						Usage: "reset clone to the latest available snapshot",
					},
					&cli.StringFlag{
						Name:  cloneResetSnapshotIDFlag,
						Usage: "snapshot ID used when resetting clone's state",
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
			{
				Name:      "start-observation",
				Usage:     "[EXPERIMENTAL] start clone state monitoring",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    startObservation,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "observation-interval",
						Usage:   "interval of metric gathering and output (in seconds)",
						EnvVars: []string{"DBLAB_OBSERVATION_INTERVAL"},
					},
					&cli.IntFlag{
						Name:    "max-lock-duration",
						Usage:   "maximum allowed duration for locks (in seconds)",
						EnvVars: []string{"DBLAB_MAX_LOCK_DURATION"},
					},
					&cli.IntFlag{
						Name:    "max-duration",
						Usage:   "maximum allowed duration for observation (in seconds)",
						EnvVars: []string{"DBLAB_MAX_DURATION"},
					},
					&cli.StringSliceFlag{
						Name:  "tags",
						Usage: "set tags for the observation session. An example: branch=patch-1",
					},
					&cli.StringFlag{
						Name:  "db-name",
						Usage: "database name to observe",
					},
				},
			},
			{
				Name:      "stop-observation",
				Usage:     "[EXPERIMENTAL] summarize clone monitoring and check results",
				ArgsUsage: "CLONE_ID",
				Before:    checkCloneIDBefore,
				Action:    stopObservation,
			},
			{
				Name:   "summary-observation",
				Usage:  "[EXPERIMENTAL] display summary of an observation session",
				Action: summaryObservation,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "clone-id",
						Usage:    "clone ID",
						Required: true,
						EnvVars:  []string{"DBLAB_OBSERVATION_CLONE_ID"},
					},
					&cli.StringFlag{
						Name:     "session-id",
						Usage:    "observing session ID",
						Required: true,
						EnvVars:  []string{"DBLAB_OBSERVATION_SESSION_ID"},
					},
				},
			},
			{
				Name:   "download-artifact",
				Usage:  "[EXPERIMENTAL] download artifact of an observation session",
				Action: downloadArtifact,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "clone-id",
						Usage:    "clone ID",
						Required: true,
						EnvVars:  []string{"DBLAB_OBSERVATION_CLONE_ID"},
					},
					&cli.StringFlag{
						Name:     "session-id",
						Usage:    "observing session ID",
						Required: true,
						EnvVars:  []string{"DBLAB_OBSERVATION_SESSION_ID"},
					},
					&cli.StringFlag{
						Name:     "artifact-type",
						Usage:    "artifact type to download",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "output",
						Usage:   "write an artifact to file (optional)",
						Aliases: []string{"o"},
					},
				},
			},
			{
				Name:  "port-forward",
				Usage: "start port forwarding to clone",
				Before: func(ctxCli *cli.Context) error {
					if err := checkCloneIDBefore(ctxCli); err != nil {
						return err
					}

					if err := commands.CheckForwardingServerURL(ctxCli); err != nil {
						return err
					}

					return nil
				},
				Action: forward,
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
