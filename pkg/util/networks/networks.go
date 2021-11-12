/*
2021 © Postgres.ai
*/

// Package networks describes custom network elements.
package networks

import (
	"context"
	"fmt"

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
	networkPrefix = "dle_network_"
)

// Setup creates a new internal Docker network and connects container to it.
func Setup(ctx context.Context, dockerCLI *client.Client, instanceID, containerID string) (string, error) {
	networkName := getNetworkName(instanceID)

	log.Dbg("Discovering internal network:", networkName)

	networkResource, err := dockerCLI.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
	if err == nil {
		if err := dockerCLI.NetworkConnect(ctx, networkResource.ID, containerID, &network.EndpointSettings{}); err != nil {
			return "", err
		}

		return networkResource.ID, nil
	}

	log.Dbg("Internal network not found:", err.Error())
	log.Dbg("Creating a new internal network:", networkName)

	internalNetwork, err := dockerCLI.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Labels: map[string]string{
			"instance": instanceID,
			"app":      DLEApp,
			"type":     InternalType,
		},
		Attachable:     true,
		Internal:       true,
		CheckDuplicate: true,
	})
	if err != nil {
		return "", err
	}

	log.Dbg("A new internal network has been created:", internalNetwork.ID)

	if err := dockerCLI.NetworkConnect(ctx, internalNetwork.ID, containerID, &network.EndpointSettings{}); err != nil {
		return "", err
	}

	return internalNetwork.ID, nil
}

// Stop disconnect all containers from the network and removes it.
func Stop(dockerCLI *client.Client, internalNetworkID, containerName string) {
	log.Dbg("Disconnecting DLE container from the internal network:", containerName)

	if err := dockerCLI.NetworkDisconnect(context.Background(), internalNetworkID, containerName, true); err != nil {
		log.Errf(err.Error())
		return
	}

	log.Dbg("DLE container has been disconnected from the internal network:", containerName)

	networkInspect, err := dockerCLI.NetworkInspect(context.Background(), internalNetworkID, types.NetworkInspectOptions{})
	if err != nil {
		log.Errf(err.Error())
		return
	}

	if len(networkInspect.Containers) == 0 {
		log.Dbg("No containers connected to the internal network. Removing network:", internalNetworkID)

		if err := dockerCLI.NetworkRemove(context.Background(), internalNetworkID); err != nil {
			log.Errf(err.Error())
			return
		}

		log.Dbg("The internal network has been removed:", internalNetworkID)
	}
}

// Connect connects a container to an internal Docker network.
func Connect(ctx context.Context, dockerCLI *client.Client, instanceID, containerID string) error {
	networkName := getNetworkName(instanceID)

	log.Dbg("Discovering internal network:", networkName)

	networkResource, err := dockerCLI.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
	if err != nil {
		return fmt.Errorf("internal network not found: %w", err)
	}

	if err := dockerCLI.NetworkConnect(ctx, networkResource.ID, containerID, &network.EndpointSettings{}); err != nil {
		return err
	}

	log.Dbg(fmt.Sprintf("Container %s has been connected to %s", instanceID, networkName))

	return nil
}

func getNetworkName(instanceID string) string {
	return networkPrefix + instanceID
}
