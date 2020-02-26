/*
2019 © Postgres.ai
*/

// TODO(anatoly):
// - Validate configs in all components.
// - Tests.
// - Graceful shutdown.
// - Don't kill clones on shutdown/start.

package main

import (
	"bytes"
	"context"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/srv"
)

var opts struct {
	VerificationToken string `short:"t" long:"token" description:"API verification token" env:"VERIFICATION_TOKEN"`

	MountDir      string `long:"mount-dir" description:"clones data mount directory" env:"MOUNT_DIR"`
	UnixSocketDir string `long:"sockets-dir" description:"unix sockets directory for secure connection to clones" env:"UNIX_SOCKET_DIR"`
	DockerImage   string `long:"docker-image" description:"clones Docker image" env:"DOCKER_IMAGE"`

	ShowHelp func() error `long:"help" description:"Show this help message"`
}

func main() {
	ctx := context.Background()

	// Load CLI options.
	if _, err := parseArgs(); err != nil {
		if flags.WroteHelp(err) {
			return
		}

		log.Fatal("Args parse error:", err)
	}

	log.DEBUG = true

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to parse config"))
	}

	log.Dbg("Config loaded", cfg)

	// TODO(anatoly): Annotate envs in configs. Use different lib for flags/configs?
	if len(opts.MountDir) > 0 {
		cfg.Provision.ModeLocal.MountDir = opts.MountDir
	}

	if len(opts.UnixSocketDir) > 0 {
		cfg.Provision.ModeLocal.UnixSocketDir = opts.UnixSocketDir
	}

	if len(opts.DockerImage) > 0 {
		cfg.Provision.ModeLocal.DockerImage = opts.DockerImage
	}

	provisionSvc, err := provision.NewProvision(ctx, cfg.Provision)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, `error in "provision" config`))
	}

	cloningSvc, err := cloning.NewCloning(&cfg.Cloning, provisionSvc)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to init a new cloning service"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = cloningSvc.Run(ctx); err != nil {
		log.Fatalf(err)
	}

	if len(opts.VerificationToken) > 0 {
		cfg.Server.VerificationToken = opts.VerificationToken
	}

	server := srv.NewServer(&cfg.Server, cloningSvc)
	if err = server.Run(); err != nil {
		log.Fatalf(err)
	}
}

func parseArgs() ([]string, error) {
	var parser = flags.NewParser(&opts, flags.Default & ^flags.HelpFlag)

	// jessevdk/go-flags lib doesn't allow to use short flag -h because
	// it's binded to usage help. We need to hack it a bit to use -h
	// for as a hostname option.
	// See https://github.com/jessevdk/go-flags/issues/240
	opts.ShowHelp = func() error {
		var b bytes.Buffer

		parser.WriteHelp(&b)

		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: b.String(),
		}
	}

	return parser.Parse()
}
