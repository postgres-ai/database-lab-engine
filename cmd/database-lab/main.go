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
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/srv"
	"gitlab.com/postgres-ai/database-lab/version"
)

var opts struct {
	VerificationToken string `short:"t" long:"token" description:"API verification token" env:"VERIFICATION_TOKEN"`

	MountDir      string `long:"mount-dir" description:"clones data mount directory" env:"MOUNT_DIR"`
	UnixSocketDir string `long:"sockets-dir" description:"unix sockets directory for secure connection to clones" env:"UNIX_SOCKET_DIR"`
	DockerImage   string `long:"docker-image" description:"clones Docker image" env:"DOCKER_IMAGE"`

	ShowHelp func() error `long:"help" description:"Show this help message"`
}

func main() {
	log.Msg("Database Lab version: ", version.GetVersion())

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

	log.DEBUG = cfg.Global.Debug
	log.Dbg("Config loaded", cfg)

	// TODO(anatoly): Annotate envs in configs. Use different lib for flags/configs?
	if len(opts.MountDir) > 0 {
		cfg.Provision.Options.ClonesMountDir = opts.MountDir
	}

	if len(opts.UnixSocketDir) > 0 {
		cfg.Provision.Options.UnixSocketDir = opts.UnixSocketDir
	}

	if len(opts.DockerImage) > 0 {
		cfg.Provision.Options.DockerImage = opts.DockerImage
	}

	if cfg.Provision.Options.ClonesMountDir != "" {
		cfg.Global.ClonesMountDir = cfg.Provision.Options.ClonesMountDir
	}

	cfg.Global.InstanceID = xid.New().String()

	log.Msg("Database Lab Instance ID", cfg.Global.InstanceID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	// Create a cloning service to provision new clones.
	provisionSvc, err := provision.New(ctx, cfg.Provision, dockerCLI)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, `error in the "provision" section of the config`))
	}

	// Create a new retrieval service to prepare a data directory and start snapshotting.
	retrievalSvc, err := retrieval.New(cfg, dockerCLI, provisionSvc.ThinCloneManager())
	if err != nil {
		log.Fatal("Failed to build a retrieval service:", err)
	}

	if err := retrievalSvc.Run(ctx); err != nil {
		if cleanUpErr := cont.CleanUpServiceContainers(ctx, dockerCLI, cfg.Global.InstanceID); cleanUpErr != nil {
			log.Err("Failed to clean up service containers:", cleanUpErr)
		}

		log.Fatal("Failed to run the data retrieval service:", err)
	}

	cloningSvc := cloning.New(&cfg.Cloning, provisionSvc)
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
