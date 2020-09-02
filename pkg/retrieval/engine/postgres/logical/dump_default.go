/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
)

type defaultDumper struct{}

func newDefaultDumper() *defaultDumper {
	return &defaultDumper{}
}

func (d *defaultDumper) GetCmdEnvVariables() []string {
	return []string{}
}

func (d *defaultDumper) SetConnectionOptions(_ context.Context, _ *Connection) error {
	return nil
}
