package provision

import (
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortAllocation(t *testing.T) {
	p := &provisionModeLocal{
		mu: &sync.Mutex{},
		provision: provision{
			config: Config{
				ModeLocal: ModeLocalConfig{
					PortPool: ModeLocalPortPool{
						From: 6000,
						To:   6002,
					},
				},
			},
		},
	}

	// Initialize port pool.
	require.NoError(t, p.initPortPool())

	// Allocate a new port.
	port, err := p.allocatePort()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, port, p.provision.config.ModeLocal.PortPool.From)
	assert.LessOrEqual(t, port, p.provision.config.ModeLocal.PortPool.To)

	// Allocate one more port.
	_, err = p.allocatePort()
	require.NoError(t, err)

	// Impossible allocate a new port.
	_, err = p.allocatePort()
	assert.IsType(t, errors.Cause(err), &NoRoomError{})
	assert.EqualError(t, err, "session cannot be started because there is no room: no available ports")

	// Free port and allocate a new one.
	require.NoError(t, p.freePort(port))
	port, err = p.allocatePort()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, p.provision.config.ModeLocal.PortPool.From)
	assert.LessOrEqual(t, port, p.provision.config.ModeLocal.PortPool.To)

	// Try to free a non-existing port.
	err = p.freePort(1)
	assert.EqualError(t, err, "port 1 is out of bounds of the port pool")
}
