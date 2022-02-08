/*
2020 © Postgres.ai
*/

package global

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/templates"
)

// List provides commands for getting started.
func List() []*cli.Command {
	return []*cli.Command{
		{
			Name:               "init",
			Usage:              "initialize Database Lab CLI",
			CustomHelpTemplate: templates.CustomCommandHelpTemplate + templates.SupportProjectTemplate,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "environment-id",
					Usage:    "environment ID of Database Lab instance's API",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "url",
					Usage:    "URL of Database Lab instance's API",
					Required: true,
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
			Action: initCLI,
		},
		{
			Name:   "port-forward",
			Usage:  "start port forwarding to the Database Lab instance",
			Before: commands.CheckForwardingServerURL,
			Action: forward,
		},
	}
}
