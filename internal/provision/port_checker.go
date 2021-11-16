/*
2020 © Postgres.ai
*/

package provision

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
)

type portChecker interface {
	checkPortAvailability(host string, port uint) error
}

type localPortChecker struct{}

func (c *localPortChecker) checkPortAvailability(host string, port uint) error {
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))

	conn, err := net.DialTimeout("tcp", addr, portCheckingTimeout)
	if conn != nil {
		_ = conn.Close()
	}

	if err == nil {
		return errors.New("port already in use")
	}

	if opErr, ok := err.(*net.OpError); ok &&
		opErr != nil && opErr.Err != nil && opErr.Err.Error() == "connect: connection refused" {
		return nil
	}

	return err
}
