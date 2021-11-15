/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
)

type defaultDumper struct {
	c *Connection
}

func newDefaultDumper() *defaultDumper {
	return &defaultDumper{}
}

func (d *defaultDumper) GetCmdEnvVariables() []string {
	return []string{}
}

func (d *defaultDumper) SetConnectionOptions(_ context.Context, c *Connection) error {
	d.c = c
	return nil
}

func (d *defaultDumper) GetDatabaseListQuery() string {
	return "select datname from pg_catalog.pg_database where not datistemplate"
}
