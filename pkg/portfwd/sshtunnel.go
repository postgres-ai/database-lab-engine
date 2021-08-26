/*
2019 © Postgres.ai
*/

// Package portfwd provides an SSH port forwarder.
package portfwd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

// SSHTunnel describes a tunnel structure.
type SSHTunnel struct {
	Endpoints  Endpoints
	Config     *ssh.ClientConfig
	serverConn *ssh.Client
	listener   net.Listener
	stop       chan struct{}
}

// Endpoints contains addresses of an SSH tunnel components.
type Endpoints struct {
	Local  string
	Server string
	Remote string
}

// NewTunnel create a new tunnel.
func NewTunnel(localEndpoint, serverEndpoint, remoteEndpoint string, sshConfig *ssh.ClientConfig) *SSHTunnel {
	return &SSHTunnel{
		Config: sshConfig,
		Endpoints: Endpoints{
			Local:  localEndpoint,
			Server: serverEndpoint,
			Remote: remoteEndpoint,
		},
		stop: make(chan struct{}, 1),
	}
}

// Open starts an ssh tunnel.
func (tunnel *SSHTunnel) Open() error {
	log.Dbg("Opening local connection...", tunnel.Endpoints.Local)

	listener, err := net.Listen("tcp", tunnel.Endpoints.Local)
	if err != nil {
		return errors.Wrapf(err, "failed to start local listener: %v", tunnel.Endpoints.Local)
	}

	tunnel.listener = listener

	log.Dbg("Opening server connection...", tunnel.Endpoints.Server)

	serverConn, err := ssh.Dial("tcp", tunnel.Endpoints.Server, tunnel.Config)
	if err != nil {
		return errors.Wrapf(err, "failed to start ssh server connection: %q", tunnel.Endpoints.Server)
	}

	tunnel.serverConn = serverConn

	return nil
}

// Listen waits for and processes connections to the SSH tunnel.
func (tunnel *SSHTunnel) Listen(ctx context.Context) error {
	if tunnel.listener == nil || tunnel.serverConn == nil {
		return errors.New("connections are not ready")
	}

	for {
		conn, err := tunnel.listener.Accept()
		if err != nil {
			return err
		}

		remoteConn, err := tunnel.serverConn.Dial("tcp", tunnel.Endpoints.Remote)
		if err != nil {
			return errors.Wrapf(err, "failed to start remote dial connection: %v", err)
		}

		tunnel.forward(conn, remoteConn)

		select {
		case <-ctx.Done():
			return nil

		case <-tunnel.stop:
			return nil

		default:
		}
	}
}

// Stop stops the SSH tunnel.
func (tunnel *SSHTunnel) Stop() error {
	tunnel.stop <- struct{}{}

	return tunnel.close()
}

func (tunnel *SSHTunnel) close() error {
	if tunnel.listener != nil {
		if err := tunnel.listener.Close(); err != nil {
			return err
		}

		log.Dbg("Close local connection", tunnel.Endpoints.Local)
	}

	if tunnel.serverConn != nil {
		if err := tunnel.serverConn.Close(); err != nil {
			return err
		}

		log.Dbg("Close server connection", tunnel.Endpoints.Server)
	}

	return nil
}

func (tunnel *SSHTunnel) forward(localConn, remoteConn net.Conn) {
	defer func() {
		if err := remoteConn.Close(); err != nil && err != io.EOF {
			log.Errf("failed to close remote connection: %v", err)
		}
	}()

	chWait := make(chan struct{})

	copyConn := func(writer, reader net.Conn) {
		if _, err := io.Copy(writer, reader); err != nil {
			log.Errf("io.Copy error: %s", err)
		}

		chWait <- struct{}{}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)

	<-chWait
}

// SSHAgent returns an auth method that runs the given function to obtain a list of key pairs.
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}

	return nil
}

// ReadAuthFromIdentityFile reads an identity file and returns an AuthMethod that uses the given key pairs.
func ReadAuthFromIdentityFile(identityFilename string) (ssh.AuthMethod, error) {
	log.Dbg(fmt.Sprintf("Read identity file %q", identityFilename))

	privateKey, err := os.ReadFile(identityFilename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read an identity file")
	}

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse a private key")
	}

	return ssh.PublicKeys(signer), nil
}
