/*
2020 © Postgres.ai
*/

package logical

import (
	"context"

	"github.com/docker/docker/api/types/mount"
)

type defaultDumper struct{}

func newDefaultDumper() *defaultDumper {
	return &defaultDumper{}
}

func (d *defaultDumper) GetCmdEnvVariables() []string {
	return []string{}
}

func (d *defaultDumper) GetMounts() []mount.Mount {
	return []mount.Mount{}
}

func (d *defaultDumper) SetConnectionOptions(_ context.Context, _ *Connection) error {
	return nil
}
