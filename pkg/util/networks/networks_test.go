/*
2021 © Postgres.ai
*/

package networks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalNetworks(t *testing.T) {
	t.Run("test internal network naming", func(t *testing.T) {
		instanceID := "testInstanceID"

		assert.Equal(t, "dle_network_testInstanceID", getNetworkName(instanceID))
	})
}
