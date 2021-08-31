/*
2021 © Postgres.ai
*/

// Package networks describes custom network elements.
package networks

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

const (
	// DLEApp contains name of the Database Lab Engine network.
	DLEApp = "DLE"

	// InternalType contains name of the internal network type.
	InternalType = "internal"

	// networkPrefix defines a distinctive prefix for internal DLE networks.
	networkPrefix = "network_"
)

// Setup creates a new internal Docker network and connects container to it.
func Setup(ctx context.Context, dockerCLI *client.Client, instanceID, containerID string) (*types.NetworkCreateResponse, error) {
	networkID := networkPrefix + instanceID

	internalNetwork, err := dockerCLI.NetworkCreate(ctx, networkID, types.NetworkCreate{
		Labels: map[string]string{
			"instance": instanceID,
			"app":      DLEApp,
			"type":     InternalType,
		},
		Attachable: true,
		Internal:   true,
	})
	if err != nil {
		return nil, err
	}

	log.Dbg("New network: ", internalNetwork.ID)

	if err := dockerCLI.NetworkConnect(ctx, internalNetwork.ID, containerID, &network.EndpointSettings{}); err != nil {
		return nil, err
	}

	return &internalNetwork, nil
}

// Stop disconnect all containers from the network and removes it.
func Stop(dockerCLI *client.Client, internalNetworkID string) {
	networkInspect, err := dockerCLI.NetworkInspect(context.Background(), internalNetworkID, types.NetworkInspectOptions{})
	if err != nil {
		log.Errf(err.Error())
		return
	}

	for _, resource := range networkInspect.Containers {
		log.Dbg("Disconnecting container: ", resource.Name)

		if err := dockerCLI.NetworkDisconnect(context.Background(), internalNetworkID, resource.Name, true); err != nil {
			log.Errf(err.Error())
			return
		}

		log.Dbg("Container has been disconnected: ", resource.Name)
	}

	if err := dockerCLI.NetworkRemove(context.Background(), internalNetworkID); err != nil {
		log.Errf(err.Error())
		return
	}
}
