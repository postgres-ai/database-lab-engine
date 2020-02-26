/*
2020 © Postgres.ai
*/

package initialization

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/templates"
)

// GlobalList provides commands for getting started.
func GlobalList() []*cli.Command {
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
					Name:     "token",
					Usage:    "verification token of Database Lab instance",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  "insecure",
					Usage: "allow insecure server connections when using SSL",
				},
			},
			Action: initCLI,
		},
	}
}
