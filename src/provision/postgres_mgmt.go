/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"strings"

	"../log"
)

const RESET_PASSWORDS_QUERY = `do $$
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

const RESET_PASSWORDS_QUERY_WHERE = ` and rolname not in (%s)`

func PostgresResetAllPasswords(r Runner, c *PgConfig, whitelistUsers []string) error {
	optionalWhere := ""
	if len(whitelistUsers) > 0 {
		for i, user := range whitelistUsers {
			if i != 0 {
				optionalWhere += ", "
			}
			optionalWhere += fmt.Sprintf("'%s'", user)
		}
		optionalWhere = fmt.Sprintf(RESET_PASSWORDS_QUERY_WHERE, optionalWhere)
	}

	query := strings.Replace(RESET_PASSWORDS_QUERY,
		"{{OPTIONAL_WHERE}}", optionalWhere, 1)
	out, err := runPsql(r, query, c, false, true)
	if err != nil {
		log.Err("ResetAllPasswords:", err)
		return err
	}

	log.Dbg("ResetAllPasswords:", out)
	return nil
}

func PostgresCreateUser(r Runner, c *PgConfig, username string, password string) error {
	query := fmt.Sprintf("create user %s with password '%s' login superuser;",
		username, password)

	out, err := runPsql(r, query, c, false, true)
	if err != nil {
		log.Err("AddUser:", err)
		return err
	}

	log.Dbg("AddUser:", out)
	return nil

}
