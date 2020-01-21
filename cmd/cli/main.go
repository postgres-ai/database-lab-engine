package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/clone"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/config"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/initialization"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/instance"
	"gitlab.com/postgres-ai/database-lab/cmd/cli/commands/snapshot"
)

const (
	applicationName = "Database Lab CLI"
	version         = "v0.0.1"
	website         = "https://postgres.ai"
	supportEmail    = "team@postgres.ai"
)

func main() {
	app := &cli.App{
		Name:    applicationName,
		Version: version,
		Usage:   fmt.Sprintf("version %s", version),
		CommandNotFound: func(c *cli.Context, command string) {
			fmt.Fprintf(c.App.Writer, "Command %q not found.\n", command)
		},
		Before: loadEnvironmentParams,
		Commands: joinCommands(
			// Getting started.
			initialization.GlobalList(),

			// Database Lab API.
			clone.CommandList(),
			instance.CommandList(),
			snapshot.CommandList(),

			// CLI config.
			config.CommandList(),
		),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "url",
				Usage:   "URL (with port, if needed) of Database Lab instance's API",
				EnvVars: []string{"DBLAB_INSTANCE_URL"},
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "verification token of Database Lab instance",
				EnvVars: []string{"DBLAB_VERIFICATION_TOKEN"},
			},
			&cli.BoolFlag{
				Name:    "insecure",
				Aliases: []string{"k"},
				Usage:   "allow insecure server connections when using SSL",
				EnvVars: []string{"DBLAB_INSECURE_SKIP_VERIFY"},
			},
		},
	}

	adoptTemplates()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// loadEnvironmentParams loads environment params from a Database Lab's config file.
func loadEnvironmentParams(c *cli.Context) error {
	filename, err := config.GetFilename()
	if err != nil {
		return err
	}

	cfg, err := config.Load(filename)
	if err != nil {
		return err
	}

	currentEnv := cfg.CurrentEnvironment
	if env, ok := cfg.Environments[currentEnv]; ok {
		if c.String(commands.URLKey) == "" {
			if err := c.Set(commands.URLKey, env.URL); err != nil {
				return err
			}
		}

		if c.String(commands.TokenKey) == "" {
			if err := c.Set(commands.TokenKey, env.Token); err != nil {
				return err
			}
		}
	}

	return nil
}

func joinCommands(cliGroups ...[]*cli.Command) []*cli.Command {
	// There are at least len(cliGroups) elements.
	cliCommands := make([]*cli.Command, 0, len(cliGroups))

	for _, cliGroup := range cliGroups {
		cliCommands = append(cliCommands, cliGroup...)
	}

	return cliCommands
}

func adoptTemplates() {
	const adaptiveTemplate = `%s
CONTACT US: %s, %s

`

	cli.AppHelpTemplate = fmt.Sprintf(adaptiveTemplate, cli.AppHelpTemplate, website, supportEmail)
}
