/*
2026 © Postgres.ai
*/

// Package localinstall provides a CLI front end to configure logical retrieval
// from a source database: it probes the source and applies the proposed config
// without the UI.
package localinstall

import (
	"github.com/urfave/cli/v2"
)

// CommandList returns the local-install command group.
func CommandList() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "local-install",
			Usage: "probe a source database and configure logical retrieval from the terminal",
			Description: "Probes the source database (POST /admin/probe-source), shows the proposed\n" +
				"configuration, and applies it (POST /admin/config) after confirmation.\n\n" +
				"Examples:\n" +
				"  # discrete fields, prompt for the password, start retrieval:\n" +
				"  dblab local-install --source-url postgresql://postgres@db.example.com:5432/app --start\n\n" +
				"  # managed provider requiring TLS — the full connection string is preserved:\n" +
				"  dblab local-install \\\n" +
				"    --source-url 'postgresql://app@db.rds.amazonaws.com:5432/app?sslmode=require' \\\n" +
				"    --password \"$PGPASSWORD\" --yes --start",
			Action: localInstall,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "source-url",
					Usage:    "source libpq connection string (postgresql:// URI or keyword/value DSN); must not embed a password",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "password",
					Usage: "source database password; prompted for if omitted and a TTY is present",
				},
				&cli.StringFlag{
					Name:  "provider",
					Usage: "override the detected managed-Postgres provider key shown in the preview",
				},
				&cli.StringFlag{
					Name:  "docker-image",
					Usage: "override the resolved docker image (full reference); wins over the engine-resolved image",
				},
				&cli.StringFlag{
					Name:  "docker-tag",
					Usage: "override only the tag of the resolved docker image",
				},
				&cli.StringFlag{
					Name:  "shared-buffers",
					Usage: "override the recommended shared_buffers value",
				},
				&cli.StringSliceFlag{
					Name:  "dbname",
					Usage: "database to dump (repeatable); defaults to the probed database",
				},
				&cli.BoolFlag{
					Name:  "start",
					Usage: "trigger a full refresh after applying even when retrieval is not pending",
				},
				&cli.BoolFlag{
					Name:  "no-start",
					Usage: "never trigger a full refresh after applying",
				},
				&cli.BoolFlag{
					Name:    "yes",
					Aliases: []string{"y"},
					Usage:   "apply without the confirmation prompt",
				},
			},
		},
	}
}
