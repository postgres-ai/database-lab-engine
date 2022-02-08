/*
2020 © Postgres.ai
*/

// Package commands provides general resources for CLI.
package commands

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"

	"gitlab.com/postgres-ai/database-lab/v3/internal/portfwd"
)

// BuildTunnel creates a new instance of SSH tunnel.
func BuildTunnel(cliCtx *cli.Context, remoteHost *url.URL) (*portfwd.SSHTunnel, error) {
	localEndpoint := forwardingLocalEndpoint(remoteHost, cliCtx.String(FwLocalPortKey))

	serverURL, err := url.Parse(cliCtx.String(FwServerURLKey))
	if err != nil {
		return nil, err
	}

	authMethod, err := getAuthMethod(cliCtx)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: serverURL.User.Username(),
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always accept key.
			return nil
		},
	}

	tunnel := portfwd.NewTunnel(localEndpoint, serverURL.Host, remoteHost.Host, sshConfig)

	return tunnel, nil
}

func forwardingLocalEndpoint(remoteURL *url.URL, localPort string) string {
	if localPort == "" {
		localPort = remoteURL.Port()
	}

	return fmt.Sprintf("%s:%s", "127.0.0.1", localPort)
}

func getAuthMethod(cliCtx *cli.Context) (ssh.AuthMethod, error) {
	if cliCtx.String(IdentityFileKey) != "" {
		authMethod, err := portfwd.ReadAuthFromIdentityFile(cliCtx.String(IdentityFileKey))
		if err != nil {
			return nil, err
		}

		return authMethod, nil
	}

	if sshAgent := portfwd.SSHAgent(); sshAgent != nil {
		return sshAgent, nil
	}

	return nil, errors.New("no auth method found. Either define `--identity-file` flag or add your certificate to the SSH agent")
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
