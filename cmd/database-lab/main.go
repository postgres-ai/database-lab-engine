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

	"github.com/docker/docker/client"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/platform"
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
	// Load CLI options.
	if _, err := parseArgs(); err != nil {
		if flags.WroteHelp(err) {
			return
		}

		log.Fatal("Args parse error:", err)
	}

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to parse config"))
	}

	log.DEBUG = cfg.Debug
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

	if cfg.Provision.ModeLocal.MountDir != "" {
		cfg.Global.MountDir = cfg.Provision.ModeLocal.MountDir
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a cloning service to provision new clones.
	provisionSvc, err := provision.New(ctx, cfg.Provision)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, `error in the "provision" section of the config`))
	}

	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	// Create a new retrieval service to prepare a data directory and start snapshotting.
	retrievalSvc, err := retrieval.New(cfg, dockerCLI, provisionSvc.ThinCloneManager())
	if err != nil {
		log.Fatal("Failed to build a retrieval service:", err)
	}

	if err := retrievalSvc.Run(ctx); err != nil {
		log.Fatal("Failed to run the data retrieval service:", err)
	}

	cloningSvc, err := cloning.New(&cfg.Cloning, provisionSvc)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to init a new cloning service"))
	}

	if err = cloningSvc.Run(ctx); err != nil {
		log.Fatalf(err)
	}

	// Create a platform service to verify Platform tokens.
	platformSvc := platform.New(cfg.Platform)
	if err := platformSvc.Init(ctx); err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to create a new platform service"))
	}

	if len(opts.VerificationToken) > 0 {
		cfg.Server.VerificationToken = opts.VerificationToken
	}

	// Start the Database Lab.
	server := srv.NewServer(&cfg.Server, cloningSvc, platformSvc)
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
