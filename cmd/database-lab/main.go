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
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/observer"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/srv"
	"gitlab.com/postgres-ai/database-lab/version"
)

func main() {
	log.Msg("Database Lab version: ", version.GetVersion())

	instanceID := xid.New().String()

	log.Msg("Database Lab Instance ID:", instanceID)

	cfg, err := loadConfiguration(instanceID)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to parse config"))
	}

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

	observerCfg := &observer.Config{
		CloneDir:   cfg.Provision.Options.ClonesMountDir,
		DataSubDir: cfg.Global.DataSubDir,
		SocketDir:  cfg.Provision.Options.UnixSocketDir,
	}

	server := srv.NewServer(&cfg.Server, observerCfg, cloningSvc, platformSvc, dockerCLI)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	go func() {
		for range c {
			log.Msg("Reloading configuration")

			if err := reloadConfig(instanceID, provisionSvc, retrievalSvc, cloningSvc, platformSvc, server); err != nil {
				log.Err("Failed to reload configuration", err)
			}

			log.Msg("Configuration has been reloaded")
		}
	}()

	// Start the Database Lab.
	if err = server.Run(); err != nil {
		log.Fatalf(err)
	}
}

func loadConfiguration(instanceID string) (*config.Config, error) {
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	log.DEBUG = cfg.Global.Debug
	log.Dbg("Config loaded", cfg)

	if cfg.Provision.Options.ClonesMountDir != "" {
		cfg.Global.ClonesMountDir = cfg.Provision.Options.ClonesMountDir
	}

	cfg.Global.InstanceID = instanceID

	return cfg, nil
}

func reloadConfig(instanceID string, provisionSvc provision.Provision, retrievalSvc *retrieval.Retrieval, cloningSvc cloning.Cloning,
	platformSvc *platform.Service, server *srv.Server) error {
	cfg, err := loadConfiguration(instanceID)
	if err != nil {
		return err
	}

	if err := provision.IsValidConfig(cfg.Provision); err != nil {
		return err
	}

	if err := retrieval.IsValidConfig(cfg); err != nil {
		return err
	}

	provisionSvc.Reload(cfg.Provision)
	retrievalSvc.Reload(cfg)
	cloningSvc.Reload(cfg.Cloning)
	platformSvc.Reload(cfg.Platform)
	server.Reload(cfg.Server)

	return nil
}
