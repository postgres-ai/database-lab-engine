/*
2021 © Postgres.ai
*/

package networks

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

func TestInternalNetworks(t *testing.T) {
	t.Run("test internal network naming", func(t *testing.T) {
		instanceID := "testInstanceID"

		assert.Equal(t, "dle_network_testInstanceID", getNetworkName(instanceID))
	})
}

func TestIfContainerConnected(t *testing.T) {
	t.Run("test if container connected", func(t *testing.T) {
		resource := types.NetworkResource{
			Containers: map[string]types.EndpointResource{
				"testID": {Name: "test_server"},
			},
		}
		testCases := []struct {
			containerName string
			result        bool
		}{
			{
				containerName: "test_server",
				result:        true,
			},
			{
				containerName: "not_connected_server",
				result:        false,
			},
		}

		for _, tc := range testCases {
			assert.Equal(t, tc.result, hasContainerConnected(resource, tc.containerName))
		}
	})
}
