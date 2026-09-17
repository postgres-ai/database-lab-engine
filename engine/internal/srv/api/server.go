/*
2026 © Postgres.ai
*/

package api

import (
	"net/http"
	"time"
)

const (
	// readHeaderTimeout bounds how long a client may take to send the request line and headers.
	readHeaderTimeout = 10 * time.Second
	// readTimeout bounds reading the whole request, body included; every body the API accepts
	// is a small JSON document.
	readTimeout = 60 * time.Second
	// idleTimeout bounds how long a keep-alive connection may sit between requests.
	idleTimeout = 120 * time.Second
)

// NewServer builds an HTTP server with the read and idle timeouts every engine listener uses.
// There is no write timeout on purpose: clone creation, observation, and the log stream keep a
// response open for as long as the work takes. The log websocket is hijacked, and the upgrade
// clears the connection deadline, so the read timeout never cuts it either.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}
}
