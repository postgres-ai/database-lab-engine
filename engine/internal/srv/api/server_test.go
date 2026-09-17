/*
2026 © Postgres.ai
*/

package api

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	handler := http.NewServeMux()
	srv := NewServer("127.0.0.1:2345", handler)

	assert.Equal(t, "127.0.0.1:2345", srv.Addr)
	assert.Equal(t, handler, srv.Handler)
	assert.Equal(t, readHeaderTimeout, srv.ReadHeaderTimeout)
	assert.Equal(t, readTimeout, srv.ReadTimeout)
	assert.Equal(t, idleTimeout, srv.IdleTimeout)
	assert.Zero(t, srv.WriteTimeout, "streaming endpoints must not be cut by a write timeout")
}

// serveWithShortTimeouts starts a server built by NewServer with every timeout shortened, so a
// test can prove which connections the timeouts reach without waiting a minute.
func serveWithShortTimeouts(t *testing.T, handler http.Handler, timeout time.Duration) net.Addr {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := NewServer(listener.Addr().String(), handler)
	srv.ReadHeaderTimeout = timeout
	srv.ReadTimeout = timeout
	srv.IdleTimeout = timeout

	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() { _ = srv.Close() })

	return listener.Addr()
}

func TestNewServer_ReadTimeoutClosesASilentClient(t *testing.T) {
	addr := serveWithShortTimeouts(t, http.NotFoundHandler(), 200*time.Millisecond)

	conn, err := net.Dial("tcp", addr.String())
	require.NoError(t, err)

	defer conn.Close()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

	_, err = conn.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF, "a client that never sends its request line is dropped")
}

func TestNewServer_ReadTimeoutDoesNotCutAHijackedWebsocket(t *testing.T) {
	const timeout = 200 * time.Millisecond

	upgrader := websocket.Upgrader{}
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	})

	addr := serveWithShortTimeouts(t, echo, timeout)

	conn, response, err := websocket.DefaultDialer.Dial("ws://"+addr.String()+"/instance/logs", nil)
	require.NoError(t, err)

	defer conn.Close()
	defer response.Body.Close()

	// stay quiet for longer than every server timeout, then use the connection
	time.Sleep(3 * timeout)

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("still here")))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

	_, message, err := conn.ReadMessage()
	require.NoError(t, err, "the upgrade clears the connection deadline, so the read timeout must not apply")
	assert.Equal(t, "still here", string(message))
}
