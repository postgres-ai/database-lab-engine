/*
2020 © Postgres.ai
*/

package config

import (
	"errors"

	"github.com/urfave/cli/v2"
)

// CommandList returns available commands for a CLI config management.
func CommandList() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "config",
			Usage: "configure CLI environments",
			Subcommands: []*cli.Command{
				{
					Name:      "create",
					Usage:     "create new CLI environment",
					ArgsUsage: "ENVIRONMENT_ID",
					Action:    createEnvironment(),
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "url",
							Usage:    "URL of Database Lab instance's API",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "token",
							Usage:    "verification token of Database Lab instance",
						},
						&cli.BoolFlag{
							Name:  "insecure",
							Usage: "allow insecure server connections when using SSL",
						},
						&cli.DurationFlag{
							Name:  "request-timeout",
							Usage: "allow changing requests timeout",
						},
						&cli.StringFlag{
							Name:  "forwarding-server-url",
							Usage: "forwarding server URL of Database Lab instance",
						},
						&cli.StringFlag{
							Name:  "forwarding-local-port",
							Usage: "local port for forwarding to the Database Lab instance",
						},
						&cli.StringFlag{
							Name:  "identity-file",
							Usage: "select a file from which the identity (private key) for public key authentication is read",
						},
					},
				},
				{
					Name:      "update",
					Usage:     "update an existing CLI environment",
					ArgsUsage: "ENVIRONMENT_ID",
					Action:    updateEnvironment(),
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "url",
							Usage: "URL of Database Lab instance's API",
						},
						&cli.StringFlag{
							Name:  "token",
							Usage: "verification token of Database Lab instance",
						},
						&cli.BoolFlag{
							Name:  "insecure",
							Usage: "allow insecure server connections when using SSL",
						},
						&cli.DurationFlag{
							Name:  "request-timeout",
							Usage: "allow changing requests timeout",
						},
						&cli.StringFlag{
							Name:  "forwarding-server-url",
							Usage: "forwarding server URL of Database Lab instance",
						},
						&cli.StringFlag{
							Name:  "forwarding-local-port",
							Usage: "local port for forwarding to the Database Lab instance",
						},
						&cli.StringFlag{
							Name:  "identity-file",
							Usage: "select a file from which the identity (private key) for public key authentication is read",
						},
					},
				},
				{
					Name:      "view",
					Usage:     "view status of CLI environment",
					ArgsUsage: "[ENVIRONMENT_ID]",
					Action:    view(),
				},
				{
					Name:   "list",
					Usage:  "display list of all available CLI environments",
					Action: list(),
				},
				{
					Name:      "switch",
					Usage:     "switch to another CLI environment",
					ArgsUsage: "ENVIRONMENT_ID",
					Action:    switchEnvironment(),
				},
				{
					Name:      "remove",
					Usage:     "remove CLI environment",
					ArgsUsage: "ENVIRONMENT_ID",
					Action:    removeEnvironment(),
				},
			},
		},
	}
}

func checkEnvironmentIDBefore(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("ENVIRONMENT_ID argument is required.") //nolint
	}

	return nil
}
