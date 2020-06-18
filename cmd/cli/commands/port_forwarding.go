/*
2020 © Postgres.ai
*/

// Package commands provides general resources for CLI.
package commands

import (
	"fmt"
	"net"
	"net/url"

	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"

	"gitlab.com/postgres-ai/database-lab/pkg/portfwd"
)

// BuildTunnel creates a new instance of SSH tunnel.
func BuildTunnel(cliCtx *cli.Context, remoteHost string) (*portfwd.SSHTunnel, error) {
	remoteURL, err := url.Parse(cliCtx.String(URLKey))
	if err != nil {
		return nil, err
	}

	localEndpoint := forwardingLocalEndpoint(remoteURL, cliCtx.String(FwLocalPortKey))

	serverURL, err := url.Parse(cliCtx.String(FwServerURLKey))
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: serverURL.User.Username(),
		Auth: []ssh.AuthMethod{
			portfwd.SSHAgent(),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always accept key.
			return nil
		},
	}

	tunnel := portfwd.NewTunnel(localEndpoint, serverURL.Host, remoteHost, sshConfig)

	return tunnel, nil
}

func forwardingLocalEndpoint(remoteURL *url.URL, localPort string) string {
	if localPort == "" {
		localPort = remoteURL.Port()
	}

	return fmt.Sprintf("%s:%s", "127.0.0.1", localPort)
}

// BuildHostname builds a hostname string.
func BuildHostname(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

// CheckForwardingServerURL checks if the forwarding server URL is set.
func CheckForwardingServerURL(cliCtx *cli.Context) error {
	if cliCtx.String(FwServerURLKey) == "" {
		return NewActionError(`Forwarding server URL is required. 
Use a global configuration flag 'forwarding-server-url' or set up the environment variable 'DBLAB_FORWARDING_SERVER_URL'.`)
	}

	return nil
}
