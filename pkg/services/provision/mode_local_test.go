package provision

import (
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPortChecker struct{}

func (m mockPortChecker) checkPortAvailability(_ string, _ uint) error {
	return nil
}

func TestPortAllocation(t *testing.T) {
	p := &Provisioner{
		mu: &sync.Mutex{},
		config: &Config{
			PortPool: PortPool{
				From: 6000,
				To:   6002,
			},
		},
		portChecker: &mockPortChecker{},
	}

	// Initialize port pool.
	require.NoError(t, p.initPortPool())

	// Allocate a new port.
	port, err := p.allocatePort()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Allocate one more port.
	_, err = p.allocatePort()
	require.NoError(t, err)

	// Impossible allocate a new port.
	_, err = p.allocatePort()
	assert.IsType(t, errors.Cause(err), &NoRoomError{})
	assert.EqualError(t, err, "session cannot be started because there is no room: no available ports")

	// Free port and allocate a new one.
	require.NoError(t, p.FreePort(port))
	port, err = p.allocatePort()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, p.config.PortPool.From)
	assert.LessOrEqual(t, port, p.config.PortPool.To)

	// Try to free a non-existing port.
	err = p.FreePort(1)
	assert.EqualError(t, err, "port 1 is out of bounds of the port pool")
}
