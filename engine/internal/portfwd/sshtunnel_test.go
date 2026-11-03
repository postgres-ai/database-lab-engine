package portfwd

import (
	"io"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type panicConn struct {
	net.Conn
}

func (panicConn) Read([]byte) (int, error) { panic("broken connection") }

func TestForward_ReturnsWhenCopyPanics(t *testing.T) {
	local, peer := net.Pipe()
	defer func() { _ = peer.Close() }()

	remote, remotePeer := net.Pipe()
	defer func() { _ = remotePeer.Close() }()

	done := make(chan struct{})

	go func() {
		(&SSHTunnel{}).forward(panicConn{Conn: local}, remote)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("forward did not return after a panic in the copy goroutine")
	}
}

func TestForward_ClosesBothSidesWhenRemoteEnds(t *testing.T) {
	client, local := net.Pipe()
	defer func() { _ = client.Close() }()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))

	remote, server := net.Pipe()

	done := make(chan struct{})

	go func() {
		(&SSHTunnel{}).forward(local, remote)
		close(done)
	}()

	require.NoError(t, server.Close())

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("forward did not return after the remote side ended")
	}

	_, err := client.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF)

	assert.Eventually(t, func() bool {
		stacks := make([]byte, 1<<20)

		return !strings.Contains(string(stacks[:runtime.Stack(stacks, true)]), "SSHTunnel).forward")
	}, 5*time.Second, 10*time.Millisecond)
}
