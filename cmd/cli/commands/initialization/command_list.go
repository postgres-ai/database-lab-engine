/*
2020 © Postgres.ai
*/

package initialization

import "github.com/urfave/cli/v2"

// GlobalList provides commands for getting started.
func GlobalList() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "init",
			Usage: "initialize Database Lab CLI",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "environment_id",
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
			},
			Action: initCLI,
		},
	}
}
