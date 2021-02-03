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
func ResetAllPasswords(c *resources.AppConfig, whitelistUsers []string) error {
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
func CreateUser(c *resources.AppConfig, username, password string, restricted bool) error {
	var query string

	if restricted {
		query = restrictedUserQuery(username, password, c.DB.DBName)
	} else {
		query = superuserQuery(username, password)
	}

	out, err := runSimpleSQL(query, c)
	if err != nil {
		return errors.Wrap(err, "failed to run psql")
	}

	log.Dbg("AddUser:", out)

	return nil
}

func superuserQuery(username, password string) string {
	return fmt.Sprintf(`create user "%s" with password '%s' login superuser;`, username, password)
}

const restrictedTemplate = `
-- create new user 
create user %[1]s with password '%s' createdb;

-- grant all privileges in the database 
grant all privileges on database %s to %[1]s;

-- grant all on all objects in all schemas in the database
do $$
begin
  -- grant usage on all schemas in the database
  execute (
    select string_agg(format('grant usage on schema %%I to %[1]s', nspname), '; ')
    from pg_namespace
    where nspname <> 'information_schema'
    and nspname not like 'pg\_%%'
  );
   
  -- grant all on all tables in all schemas in database
  execute (
    select string_agg(format('grant all on all tables in schema %%I to %[1]s', nspname), '; ')
    from pg_namespace
    where nspname <> 'information_schema'
    and nspname not like 'pg\_%%'
  );

  -- grant all on all sequences in all custom schemas in the database
  execute (
    select string_agg(format('grant all on all sequences in schema %%I to %[1]s', nspname), '; ')
    from pg_namespace
    where nspname <> 'information_schema'
    and nspname not like 'pg\_%%'
  );

  -- grant all on all functions in all schemas in the database
  execute (
    select string_agg(format('grant all on all functions in schema %%I to %[1]s', nspname), '; ')
    from pg_namespace
    where nspname <> 'information_schema'
    and nspname not like 'pg\_%%'
  );
end $$; 
`

func restrictedUserQuery(username, password, database string) string {
	return fmt.Sprintf(restrictedTemplate, username, password, database)
}
