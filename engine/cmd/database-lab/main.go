/*
2019 © Postgres.ai
*/

// TODO(anatoly):
// - Validate configs in all components.
// - Tests.

// Package main contains the starting point of the DLE server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/billing"
	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/diagnostic"
	"gitlab.com/postgres-ai/database-lab/v3/internal/embeddedui"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/ws"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	shutdownTimeout = 30 * time.Second
	contactSupport  = "If you have problems or questions, " +
		"please contact Postgres.ai: https://postgres.ai/contact"
)

func main() {
	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatal(errors.WithMessage(err, "failed to parse config"))
	}

	logFilter := log.GetFilter()
	logFilter.ReloadLogRegExp([]string{cfg.Server.VerificationToken, cfg.Platform.AccessToken, cfg.Platform.OrgKey})

	config.ApplyGlobals(cfg)

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if err != nil {
			log.Msg(contactSupport)
		}
	}()

	engProps, err := getEngineProperties(ctx, docker, cfg)
	if err != nil {
		log.Err("failed to get Database Lab Engine properties:", err.Error())
		return
	}

	log.Msg("Database Lab Instance ID:", engProps.InstanceID)
	log.Msg("Database Lab Engine version:", version.GetVersion())

	// Create a platform service to make requests to Platform.
	platformSvc, err := platform.New(ctx, cfg.Platform, engProps.InstanceID)
	if err != nil {
		log.Errf(err.Error())
		return
	}

	if cfg.Server.VerificationToken == "" {
		log.Warn("Verification Token is empty. Database Lab Engine is insecure")
	}

	runner := runners.NewLocalRunner(cfg.Provision.UseSudo)

	internalNetworkID, err := networks.Setup(ctx, docker, engProps.InstanceID, engProps.ContainerName)
	if err != nil {
		log.Errf(err.Error())
		return
	}

	defer networks.Stop(docker, internalNetworkID, engProps.ContainerName)

	dbCfg := &resources.DB{
		Username: cfg.Global.Database.User(),
		DBName:   cfg.Global.Database.Name(),
	}

	tm := telemetry.New(platformSvc, engProps.InstanceID)

	pm := pool.NewPoolManager(&cfg.PoolManager, runner)
	if err = pm.ReloadPools(); err != nil {
		log.Err(err.Error())
	}

	// Create a new retrieval service to prepare a data directory and start snapshotting.
	retrievalSvc, err := retrieval.New(cfg, &engProps, docker, pm, tm, runner)
	if err != nil {
		log.Errf(errors.WithMessage(err, `error in the "retrieval" section of config`).Error())
		return
	}

	// Create a cloning service to provision new clones.
	networkGateway := getNetworkGateway(docker, internalNetworkID)

	provisioner, err := provision.New(ctx, &cfg.Provision, dbCfg, docker, pm, engProps.InstanceID, internalNetworkID, networkGateway)
	if err != nil {
		log.Errf(errors.WithMessage(err, `error in the "provision" section of the config`).Error())
	}

	tokenHolder, err := ws.NewTokenKeeper()
	if err != nil {
		log.Errf(errors.WithMessage(err, `failed to init WebSockets Token Manager`).Error())
	}

	go tokenHolder.RunCleaningUp(ctx)

	observingChan := make(chan string, 1)

	emergencyShutdown := func() {
		cancel()

		shutdownDatabaseLabEngine(context.Background(), docker, &cfg.Global.Database, engProps.InstanceID, pm.First())
	}

	cloningSvc := cloning.NewBase(&cfg.Cloning, provisioner, tm, observingChan)
	if err = cloningSvc.Run(ctx); err != nil {
		log.Err(err)
		emergencyShutdown()

		return
	}

	obs := observer.NewObserver(docker, &cfg.Observer, pm)
	billingSvc := billing.New(platformSvc.Client, &engProps, pm)

	go removeObservingClones(observingChan, obs)

	embeddedUI := embeddedui.New(cfg.EmbeddedUI, engProps, runner, docker)

	logCleaner := diagnostic.NewLogCleaner()

	reloadConfigFn := func(server *srv.Server) error {
		return reloadConfig(
			ctx,
			engProps,
			provisioner,
			billingSvc,
			retrievalSvc,
			pm,
			cloningSvc,
			platformSvc,
			embeddedUI,
			server,
			logCleaner,
			logFilter,
		)
	}

	server := srv.NewServer(&cfg.Server, &cfg.Global, &engProps, docker, cloningSvc, provisioner, retrievalSvc, platformSvc,
		billingSvc, obs, pm, tm, tokenHolder, logFilter, embeddedUI, reloadConfigFn)

	server.InitHandlers()

	go func() {
		if err := server.Run(); err != nil {
			log.Msg(err)
		}
	}()

	if cfg.EmbeddedUI.Enabled {
		go func() {
			if err := embeddedUI.Run(ctx); err != nil {
				log.Err("Failed to start embedded UI container:", err.Error())
				return
			}
		}()
	}

	if err := provisioner.Init(); err != nil {
		log.Err(err)
		emergencyShutdown()

		return
	}

	systemMetrics := billing.GetSystemMetrics(pm)

	tm.SendEvent(ctx, telemetry.EngineStartedEvent, telemetry.EngineStarted{
		EngineVersion: version.GetVersion(),
		DBEngine:      cfg.Global.Engine,
		DBVersion:     provisioner.DetectDBVersion(),
		Pools:         pm.CollectPoolStat(),
		Restore:       retrievalSvc.ReportState(),
		System:        systemMetrics,
	})

	if err := billingSvc.RegisterInstance(ctx, systemMetrics); err != nil {
		log.Msg("Skip registering instance:", err)
	}

	log.Msg("DBLab Edition:", engProps.GetEdition())

	shutdownCh := setShutdownListener()

	go setReloadListener(ctx, engProps, provisioner, billingSvc,
		retrievalSvc, pm, cloningSvc, platformSvc,
		embeddedUI, server,
		logCleaner, logFilter)

	go billingSvc.CollectUsage(ctx, systemMetrics)

	if err := retrievalSvc.Run(ctx); err != nil {
		log.Err("Failed to run the data retrieval service:", err)
		log.Msg(contactSupport)
	}

	defer retrievalSvc.Stop()

	if err := logCleaner.ScheduleLogCleanupJob(cfg.Diagnostic); err != nil {
		log.Err("Failed to schedule a cleanup job of the diagnostic logs collector", err)
	}

	<-shutdownCh
	cancel()

	ctxBackground := context.Background()

	shutdownCtx, shutdownCancel := context.WithTimeout(ctxBackground, shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Msg(err)
	}

	shutdownDatabaseLabEngine(ctxBackground, docker, &cfg.Global.Database, engProps.InstanceID, pm.First())
	cloningSvc.SaveClonesState()
	logCleaner.StopLogCleanupJob()
	tm.SendEvent(ctxBackground, telemetry.EngineStoppedEvent, telemetry.EngineStopped{Uptime: server.Uptime()})
}

func getNetworkGateway(docker *client.Client, internalNetworkID string) string {
	gateway := ""

	networkResource, err := docker.NetworkInspect(context.Background(), internalNetworkID, types.NetworkInspectOptions{})
	if err != nil {
		log.Err(err.Error())
		return gateway
	}

	if len(networkResource.IPAM.Config) > 0 {
		gateway = networkResource.IPAM.Config[0].Gateway
	}

	return gateway
}

func getEngineProperties(ctx context.Context, docker *client.Client, cfg *config.Config) (global.EngineProps, error) {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		return global.EngineProps{}, errors.New("hostname is empty")
	}

	dleContainer, err := docker.ContainerInspect(ctx, hostname)
	if err != nil {
		return global.EngineProps{}, fmt.Errorf("failed to inspect DLE container: %w", err)
	}

	instanceID, err := config.LoadInstanceID()
	if err != nil {
		return global.EngineProps{}, fmt.Errorf("failed to load instance ID: %w", err)
	}

	infra := os.Getenv("DLE_COMPUTING_INFRASTRUCTURE")
	if infra == "" {
		infra = global.LocalInfra
	}

	engProps := global.EngineProps{
		InstanceID:     instanceID,
		ContainerName:  strings.Trim(dleContainer.Name, "/"),
		Infrastructure: infra,
		EnginePort:     cfg.Server.Port,
	}

	return engProps, nil
}

func reloadConfig(ctx context.Context, engProp global.EngineProps, provisionSvc *provision.Provisioner, billingSvc *billing.Billing,
	retrievalSvc *retrieval.Retrieval, pm *pool.Manager, cloningSvc *cloning.Base, platformSvc *platform.Service,
	embeddedUI *embeddedui.UIManager, server *srv.Server, cleaner *diagnostic.Cleaner, filtering *log.Filtering) error {
	cfg, err := config.LoadConfiguration()
	if err != nil {
		return err
	}

	filtering.ReloadLogRegExp([]string{cfg.Server.VerificationToken, cfg.Platform.AccessToken, cfg.Platform.OrgKey})
	config.ApplyGlobals(cfg)

	if err := provision.IsValidConfig(cfg.Provision); err != nil {
		return err
	}

	newRetrievalConfig, err := retrieval.ValidateConfig(&cfg.Retrieval)
	if err != nil {
		return err
	}

	newPlatformSvc, err := platform.New(ctx, cfg.Platform, engProp.InstanceID)
	if err != nil {
		return err
	}

	if err := pm.Reload(cfg.PoolManager); err != nil {
		return err
	}

	if err := embeddedUI.Reload(ctx, cfg.EmbeddedUI); err != nil {
		return err
	}

	if err := cleaner.ScheduleLogCleanupJob(cfg.Diagnostic); err != nil {
		return err
	}

	dbCfg := resources.DB{
		Username: cfg.Global.Database.User(),
		DBName:   cfg.Global.Database.Name(),
	}

	provisionSvc.Reload(cfg.Provision, dbCfg)
	retrievalSvc.Reload(ctx, newRetrievalConfig)
	cloningSvc.Reload(cfg.Cloning)
	platformSvc.Reload(newPlatformSvc)
	billingSvc.Reload(newPlatformSvc.Client)
	server.Reload(cfg.Server)

	return nil
}

func setReloadListener(ctx context.Context, engProp global.EngineProps, provisionSvc *provision.Provisioner, billingSvc *billing.Billing,
	retrievalSvc *retrieval.Retrieval, pm *pool.Manager, cloningSvc *cloning.Base, platformSvc *platform.Service,
	embeddedUI *embeddedui.UIManager, server *srv.Server, cleaner *diagnostic.Cleaner, logFilter *log.Filtering) {
	reloadCh := make(chan os.Signal, 1)
	signal.Notify(reloadCh, syscall.SIGHUP)

	for range reloadCh {
		log.Msg("Reloading configuration")

		if err := reloadConfig(ctx, engProp,
			provisionSvc, billingSvc, retrievalSvc,
			pm, cloningSvc,
			platformSvc,
			embeddedUI, server,
			cleaner, logFilter); err != nil {
			log.Err("Failed to reload configuration:", err)

			continue
		}

		log.Msg("Configuration has been reloaded")
	}
}

func setShutdownListener() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	return c
}

func shutdownDatabaseLabEngine(ctx context.Context, docker *client.Client, dbCfg *global.Database, instanceID string, fsm pool.FSManager) {
	log.Msg("Stopping auxiliary containers")

	if err := cont.StopControlContainers(ctx, docker, dbCfg, instanceID, fsm); err != nil {
		log.Err("Failed to stop control containers", err)
	}

	if err := cont.CleanUpSatelliteContainers(ctx, docker, instanceID); err != nil {
		log.Err("Failed to stop satellite containers", err)
	}

	log.Msg("Auxiliary containers have been stopped")
}

func removeObservingClones(obsCh chan string, obs *observer.Observer) {
	for cloneID := range obsCh {
		obs.RemoveObservingClone(cloneID)
	}
}
