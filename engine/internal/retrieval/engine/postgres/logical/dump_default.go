/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"
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

func (d *defaultDumper) GetDatabaseListQuery(username string) string {
	return fmt.Sprintf(`select datname from pg_catalog.pg_database 
	where not datistemplate and has_database_privilege('%s', datname, 'CONNECT')`, username)
}
