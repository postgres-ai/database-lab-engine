/*
2020 © Postgres.ai
*/

package logical

import (
	"context"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
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
	return db.GetDatabaseListQuery(username)
}
