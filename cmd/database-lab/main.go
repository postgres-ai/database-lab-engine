/*
2019 © Postgres.ai
*/

// TODO(anatoly):
// - Validate configs in all components.
// - Tests.
// - Don't kill clones on shutdown/start.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
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

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	log.Msg("Database Lab version: ", version.GetVersion())

	instanceID := xid.New().String()

	log.Msg("Database Lab Instance ID:", instanceID)

	cfg, err := loadConfiguration(instanceID)
	if err != nil {
		log.Fatal(errors.WithMessage(err, "failed to parse config"))
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
		log.Fatal(errors.WithMessage(err, `error in the "provision" section of the config`))
	}

	// Create a new retrieval service to prepare a data directory and start snapshotting.
	retrievalSvc, err := retrieval.New(cfg, dockerCLI, provisionSvc.ThinCloneManager())
	if err != nil {
		log.Fatal("Failed to build a retrieval service:", err)
	}

	if err := retrievalSvc.Run(ctx); err != nil {
		if cleanUpErr := cont.CleanUpControlContainers(ctx, dockerCLI, cfg.Global.InstanceID); cleanUpErr != nil {
			log.Err("Failed to clean up service containers:", cleanUpErr)
		}

		log.Fatal("Failed to run the data retrieval service:", err)
	}

	cloningSvc := cloning.New(&cfg.Cloning, provisionSvc)
	if err = cloningSvc.Run(ctx); err != nil {
		log.Fatal(err)
	}

	// Create a platform service to make requests to Platform.
	platformSvc, err := platform.New(ctx, cfg.Platform)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to create a new platform service"))
	}

	cfg.Observer.CloneDir = cfg.Provision.Options.ClonesMountDir
	cfg.Observer.DataSubDir = cfg.Global.DataSubDir
	cfg.Observer.SocketDir = cfg.Provision.Options.UnixSocketDir

	server := srv.NewServer(&cfg.Server, &cfg.Observer, cloningSvc, platformSvc, dockerCLI)

	reloadCh := setReloadListener()

	go func() {
		for range reloadCh {
			log.Msg("Reloading configuration")

			if err := reloadConfig(ctx, instanceID, provisionSvc, retrievalSvc, cloningSvc, platformSvc, server); err != nil {
				log.Err("Failed to reload configuration", err)
			}

			log.Msg("Configuration has been reloaded")
		}
	}()

	shutdownCh := setShutdownListener()

	server.InitHandlers()

	go func() {
		// Start the Database Lab.
		if err = server.Run(); err != nil {
			log.Msg(err)
		}
	}()

	<-shutdownCh
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Msg(err)
	}

	shutdownDatabaseLabEngine(shutdownCtx, dockerCLI, cfg.Global)
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

func reloadConfig(ctx context.Context, instanceID string, provisionSvc provision.Provision, retrievalSvc *retrieval.Retrieval,
	cloningSvc cloning.Cloning, platformSvc *platform.Service, server *srv.Server) error {
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

	newPlatformSvc, err := platform.New(ctx, cfg.Platform)
	if err != nil {
		return err
	}

	provisionSvc.Reload(cfg.Provision)
	retrievalSvc.Reload(cfg)
	cloningSvc.Reload(cfg.Cloning)
	platformSvc.Reload(newPlatformSvc)
	server.Reload(cfg.Server)

	return nil
}

func setReloadListener() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	return c
}

func setShutdownListener() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	return c
}

func shutdownDatabaseLabEngine(ctx context.Context, dockerCLI *client.Client, global config.Global) {
	log.Msg("Stopping control containers")

	if err := cont.StopControlContainers(ctx, dockerCLI, global.InstanceID, global.DataDir()); err != nil {
		log.Err("Failed to stop control containers", err)
	}

	log.Msg("Control containers have been stopped")
}
