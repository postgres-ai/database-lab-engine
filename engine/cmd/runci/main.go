// Package main contains the starting point of the CI Checker tool.
package main

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/runci"
	"gitlab.com/postgres-ai/database-lab/v3/internal/runci/source"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

func main() {
	dockerCLI, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := runci.LoadConfiguration()
	if err != nil {
		log.Errf("failed to load config: %v", err)
		return
	}

	log.SetDebug(cfg.App.Debug)
	log.Dbg("Config loaded: ", cfg)

	if cfg.App.VerificationToken == "" {
		log.Err("migration checker is insecure since verification token is empty")
		return
	}

	networkID := discoverNetwork(ctx, cfg, dockerCLI)
	if networkID != "" {
		hostname := os.Getenv("HOSTNAME")
		if hostname == "" {
			log.Errf("hostname is empty")
			return
		}

		if err := dockerCLI.NetworkConnect(context.Background(), networkID, hostname, &network.EndpointSettings{}); err != nil {
			log.Errf(err.Error())
			return
		}

		defer func() {
			if err := dockerCLI.NetworkDisconnect(context.Background(), networkID, hostname, true); err != nil {
				log.Errf(err.Error())
				return
			}
		}()
	}

	// Create a platform service to make requests to Platform.
	// Instance ID is not defined for the Run CI service.
	platformSvc, err := platform.New(ctx, cfg.Platform, "")
	if err != nil {
		log.Errf(errors.WithMessage(err, "failed to create a new platform service").Error())
		return
	}

	dleClient, err := dblabapi.NewClient(dblabapi.Options{
		Host:              cfg.DLE.URL,
		VerificationToken: cfg.DLE.VerificationToken,
	})

	if err != nil {
		log.Errf("failed to create a Database Lab client: %v", err)
		return
	}

	if err := os.MkdirAll(source.RepoDir, 0666); err != nil {
		log.Errf("failed to create a directory to download archives: %v", err)
		return
	}

	defer func() {
		if err := os.RemoveAll(source.RepoDir); err != nil {
			log.Errf("failed to remove the directory with source code archives: %v", err)
			return
		}
	}()

	codeProvider := source.NewCodeProvider(ctx, &cfg.Source)

	srv := runci.NewServer(cfg, dleClient, platformSvc, codeProvider, dockerCLI, networkID)

	if err := srv.Run(); err != nil {
		log.Msg(err)
	}
}

func discoverNetwork(ctx context.Context, cfg *runci.Config, dockerCLI *client.Client) string {
	parsedURL, err := url.Parse(cfg.DLE.URL)
	if err != nil {
		log.Errf("invalid DLE URL in the config")
		return ""
	}

	// External hostname.
	if strings.Contains(parsedURL.Host, ".") {
		return ""
	}

	inspection, err := dockerCLI.ContainerInspect(ctx, tools.TrimPort(parsedURL.Host))
	if err != nil {
		log.Errf(err.Error())
		return ""
	}

	log.Dbg("ContainerInspect: ", inspection.ID)
	log.Dbg("ContainerInspect: ", inspection.NetworkSettings.Networks)

	networkID := ""

	for networkLabel, endpointSettings := range inspection.NetworkSettings.Networks {
		if strings.HasPrefix(networkLabel, networks.NetworkPrefix) {
			networkResource, err := dockerCLI.NetworkInspect(ctx, endpointSettings.NetworkID, network.InspectOptions{})
			if err != nil {
				log.Err(err)
				continue
			}

			networkApp := networkResource.Labels["app"]
			networkType := networkResource.Labels["type"]

			if networkApp == networks.DLEApp && networkType == networks.InternalType {
				networkID = endpointSettings.NetworkID
				break
			}
		}
	}

	log.Dbg("Network ID: ", networkID)

	return networkID
}
