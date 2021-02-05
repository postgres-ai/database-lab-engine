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

	"gitlab.com/postgres-ai/database-lab/v2/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/observer"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/platform"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/srv"
	"gitlab.com/postgres-ai/database-lab/v2/version"
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

	runner := runners.NewLocalRunner(cfg.Provision.UseSudo)

	pm := pool.NewPoolManager(&cfg.PoolManager, runner)
	if err := pm.ReloadPools(); err != nil {
		log.Fatal(err.Error())
	}

	dockerCLI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	// Create a platform service to make requests to Platform.
	platformSvc, err := platform.New(ctx, cfg.Platform)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, "failed to create a new platform service"))
	}

	// Create a new retrieval service to prepare a data directory and start snapshotting.
	retrievalSvc := retrieval.New(cfg, dockerCLI, pm, runner)

	if err := retrievalSvc.Run(ctx); err != nil {
		if cleanUpErr := cont.CleanUpControlContainers(ctx, dockerCLI, cfg.Global.InstanceID); cleanUpErr != nil {
			log.Err("Failed to clean up service containers:", cleanUpErr)
		}

		log.Fatal("Failed to run the data retrieval service:", err)
	}

	defer retrievalSvc.Stop()

	dbCfg := &resources.DB{
		Username: cfg.Global.Database.User(),
		DBName:   cfg.Global.Database.Name(),
	}

	// Create a cloning service to provision new clones.
	provisionSvc, err := provision.New(ctx, &cfg.Provision, dbCfg, dockerCLI, pm)
	if err != nil {
		log.Fatalf(errors.WithMessage(err, `error in the "provision" section of the config`))
	}

	cloningSvc := cloning.New(&cfg.Cloning, provisionSvc)
	if err = cloningSvc.Run(ctx); err != nil {
		log.Fatal(err)
	}

	obs := observer.NewObserver(dockerCLI, &cfg.Observer, platformSvc.Client, pm.Active().Pool())

	server := srv.NewServer(&cfg.Server, obs, cloningSvc, platformSvc, dockerCLI)

	reloadCh := setReloadListener()

	go func() {
		for range reloadCh {
			log.Msg("Reloading configuration")

			if err := reloadConfig(ctx, instanceID, provisionSvc, retrievalSvc, pm, cloningSvc, platformSvc, server); err != nil {
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

	shutdownDatabaseLabEngine(shutdownCtx, dockerCLI, cfg.Global, pm.Active().Pool())
}

func loadConfiguration(instanceID string) (*config.Config, error) {
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	log.DEBUG = cfg.Global.Debug
	log.Dbg("Config loaded", cfg)

	cfg.Global.InstanceID = instanceID

	return cfg, nil
}

func reloadConfig(ctx context.Context, instanceID string, provisionSvc *provision.Provisioner, retrievalSvc *retrieval.Retrieval,
	pm *pool.Manager, cloningSvc cloning.Cloning, platformSvc *platform.Service, server *srv.Server) error {
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

	if err := pm.Reload(cfg.PoolManager); err != nil {
		return err
	}

	dbCfg := resources.DB{
		Username: cfg.Global.Database.User(),
		DBName:   cfg.Global.Database.Name(),
	}

	provisionSvc.Reload(cfg.Provision, dbCfg)
	retrievalSvc.Reload(ctx, cfg)
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

func shutdownDatabaseLabEngine(ctx context.Context, dockerCLI *client.Client, global config.Global, fsp *resources.Pool) {
	log.Msg("Stopping control containers")

	if err := cont.StopControlContainers(ctx, dockerCLI, global.InstanceID, fsp.DataDir()); err != nil {
		log.Err("Failed to stop control containers", err)
	}

	log.Msg("Control containers have been stopped")
}
