package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/clone"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/config"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/global"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/instance"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/snapshot"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/templates"
	dblabLog "gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

func main() {
	app := &cli.App{
		Version: version.GetVersion(),
		CommandNotFound: func(c *cli.Context, command string) {
			fmt.Fprintf(c.App.Writer, "[ERROR] Command %q not found.\n", command)
		},
		Before: loadEnvironmentParams,
		Commands: joinCommands(
			// Config commands.
			global.List(),

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
			&cli.DurationFlag{
				Name:    "request-timeout",
				Usage:   "allow changing requests timeout",
				EnvVars: []string{"DBLAB_REQUEST_TIMEOUT"},
			},
			&cli.StringFlag{
				Name:    "forwarding-server-url",
				Usage:   "forwarding server URL of Database Lab instance",
				EnvVars: []string{"DBLAB_CLI_FORWARDING_SERVER_URL"},
			},
			&cli.StringFlag{
				Name:    "forwarding-local-port",
				Usage:   "local port for forwarding to the Database Lab instance",
				EnvVars: []string{"DBLAB_CLI_FORWARDING_LOCAL_PORT"},
			},
			&cli.StringFlag{
				Name:    "identity-file",
				Usage:   "select a file from which the identity (private key) for public key authentication is read",
				EnvVars: []string{"DBLAB_CLI_IDENTITY_FILE"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "run in debug mode",
				EnvVars: []string{"DBLAB_CLI_DEBUG"},
			},
		},
		EnableBashCompletion: true,
	}

	adoptTemplates()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// loadEnvironmentParams loads environment params from a Database Lab's config file.
func loadEnvironmentParams(c *cli.Context) error {
	dblabLog.SetDebug(c.IsSet("debug"))

	filename, err := config.GetFilename()
	if err != nil {
		return err
	}

	cfg, err := config.Load(filename)
	if err != nil {
		// Failed to load config, skip auto-loading environment keys.
		return nil
	}

	currentEnv := cfg.CurrentEnvironment
	if env, ok := cfg.Environments[currentEnv]; ok {
		if !c.IsSet(commands.URLKey) {
			if err := c.Set(commands.URLKey, env.URL); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.TokenKey) {
			if err := c.Set(commands.TokenKey, env.Token); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.InsecureKey) {
			if err := c.Set(commands.InsecureKey, strconv.FormatBool(env.Insecure)); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.RequestTimeoutKey) {
			if err := c.Set(commands.RequestTimeoutKey, env.RequestTimeout.String()); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.FwServerURLKey) {
			if err := c.Set(commands.FwServerURLKey, env.Forwarding.ServerURL); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.FwLocalPortKey) {
			if err := c.Set(commands.FwLocalPortKey, env.Forwarding.LocalPort); err != nil {
				return err
			}
		}

		if !c.IsSet(commands.IdentityFileKey) {
			if err := c.Set(commands.IdentityFileKey, env.Forwarding.IdentityFile); err != nil {
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
	cli.AppHelpTemplate = templates.CustomAppHelpTemplate + templates.SupportProjectTemplate
	cli.CommandHelpTemplate = templates.CustomCommandHelpTemplate
	cli.SubcommandHelpTemplate = templates.CustomSubcommandHelpTemplate
}
