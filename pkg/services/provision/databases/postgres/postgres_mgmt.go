/*
2019 © Postgres.ai
*/

package postgres

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

// ResetPasswordsQuery provides a template for a reset password query.
const ResetPasswordsQuery = `do $$
declare
  rec record;
  sql text;
begin
  for rec in
    select * from pg_roles where rolcanlogin{{OPTIONAL_WHERE}}
  loop
    sql := format(
      'alter role %I password %L',
      rec.rolname,
      md5(random()::text || clock_timestamp())
    );

    raise debug 'SQL: %', sql;

    execute sql;
  end loop;
end
$$;
`

// ResetPasswordsQueryWhere provides a template for a reset password where clause.
const ResetPasswordsQueryWhere = ` and rolname not in (%s)`

// ResetAllPasswords defines a method for resetting password of all Postgres users.
func ResetAllPasswords(r runners.Runner, c *resources.AppConfig, whitelistUsers []string) error {
	optionalWhere := ""

	if len(whitelistUsers) > 0 {
		for i, user := range whitelistUsers {
			if i != 0 {
				optionalWhere += ", "
			}

			optionalWhere += fmt.Sprintf("'%s'", user)
		}

		optionalWhere = fmt.Sprintf(ResetPasswordsQueryWhere, optionalWhere)
	}

	query := strings.Replace(ResetPasswordsQuery,
		"{{OPTIONAL_WHERE}}", optionalWhere, 1)

	out, err := runSimpleSQL(query, c)
	if err != nil {
		return errors.Wrap(err, "failed to run psql")
	}

	log.Dbg("ResetAllPasswords:", out)

	return nil
}

// CreateUser defines a method for creation of Postgres user.
func CreateUser(r runners.Runner, c *resources.AppConfig, username string, password string) error {
	query := fmt.Sprintf("create user \"%s\" with password '%s' login superuser;",
		username, password)

	out, err := runSimpleSQL(query, c)
	if err != nil {
		return errors.Wrap(err, "failed to run psql")
	}

	log.Dbg("AddUser:", out)

	return nil
}
