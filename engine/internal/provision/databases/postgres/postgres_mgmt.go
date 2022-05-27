/*
2019 © Postgres.ai
*/

package postgres

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
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

	out, err := runSimpleSQL(query, getPgConnStr(c.Host, c.DB.DBName, c.DB.Username, c.Port))
	if err != nil {
		return errors.Wrap(err, "failed to run psql")
	}

	log.Dbg("ResetAllPasswords:", out)

	return nil
}

// selectAllDatabases provides a query to list available databases.
const selectAllDatabases = "select datname from pg_catalog.pg_database where not datistemplate"

// CreateUser defines a method for creation of Postgres user.
func CreateUser(c *resources.AppConfig, user resources.EphemeralUser) error {
	var query string

	dbName := c.DB.DBName
	if user.AvailableDB != "" {
		dbName = user.AvailableDB
	}

	if user.Restricted {
		// create restricted user
		query = restrictedUserQuery(user.Name, user.Password)
		out, err := runSimpleSQL(query, getPgConnStr(c.Host, dbName, c.DB.Username, c.Port))

		if err != nil {
			return fmt.Errorf("failed to create restricted user: %w", err)
		}

		log.Dbg("Restricted user has been created: ", out)

		// set restricted user as owner for database objects
		databaseList, err := runSQLSelectQuery(selectAllDatabases, getPgConnStr(c.Host, dbName, c.DB.Username, c.Port))

		if err != nil {
			return fmt.Errorf("failed list all databases: %w", err)
		}

		for _, database := range databaseList {
			query = restrictedObjectsQuery(user.Name)
			out, err = runSimpleSQL(query, getPgConnStr(c.Host, database, c.DB.Username, c.Port))

			if err != nil {
				return fmt.Errorf("failed to run objects restrict query: %w", err)
			}

			log.Dbg("Objects restriction applied", database, out)
		}
	} else {
		query = superuserQuery(user.Name, user.Password)

		out, err := runSimpleSQL(query, getPgConnStr(c.Host, dbName, c.DB.Username, c.Port))
		if err != nil {
			return fmt.Errorf("failed to create superuser: %w", err)
		}

		log.Dbg("Super user has been created: ", out)
	}

	return nil
}

func superuserQuery(username, password string) string {
	return fmt.Sprintf(`create user %s with password %s login superuser;`, pq.QuoteIdentifier(username), pq.QuoteLiteral(password))
}

const restrictionUserCreationTemplate = `
-- create a new user 
create user @username with password @password login;
do $$
declare
  new_owner text;
  object_type record;
  r record;
begin
  new_owner := @usernameStr;

  -- Changing owner of all databases
  for r in select datname from pg_catalog.pg_database where not datistemplate loop
    raise debug 'Changing owner of %', r.datname;
    execute format(
      'alter database %s owner to %s;',
      r.datname,
      new_owner
    );
  end loop;
end
$$;
`

const restrictionTemplate = `
do $$
declare
  new_owner text;
  object_type record;
  r record;
begin
  new_owner := @usernameStr;

  -- Schemas
  -- allow working with all schemas
  for r in select * from pg_namespace loop
    raise debug 'Changing ownership of schema % to %',
                r.nspname, new_owner;
    execute format(
      'alter schema %I owner to %I;',
      r.nspname,
      new_owner
    );
  end loop;

  -- Types and Domains
  -- d: domain (assuming that ALTER TYPE will be equivalent to ALTER DOMAIN)
  -- e: enum
  -- r: range
  -- m: multirange
  for r in
    select n.nspname, t.typname
    from pg_type t
    join pg_namespace n on
      n.oid = t.typnamespace
      and not n.nspname in ('pg_catalog', 'information_schema')
      and t.typtype in ('d', 'e', 'r', 'm')
    order by t.typname
  loop
      raise debug 'Changing ownership of type %.% to %',
                   r.nspname, r.typname, new_owner;
      execute format(
        'alter type %I.%I owner to %I;',
        r.nspname,
        r.typname,
        new_owner
      );
  end loop;

  -- Relations
  -- c: composite type
  -- p: partitioned table
  -- i: index
  -- r: table
  -- v: view
  -- m: materialized view
  -- S: sequence
  for object_type in
    select
      unnest('{type,table,table,view,materialized view,sequence}'::text[]) type_name,
      unnest('{c,p,r,v,m,S}'::text[]) code
  loop
    for r in 
      execute format(
        $sql$
          select n.nspname, c.relname
          from pg_class c
          join pg_namespace n on
            n.oid = c.relnamespace
            and not n.nspname in ('pg_catalog', 'information_schema')
            and c.relkind = %L
          order by c.relname
        $sql$,
        object_type.code
      )
    loop 
      raise debug 'Changing ownership of % %.% to %',
                   object_type.type_name, r.nspname, r.relname, new_owner;
      execute format(
        'alter %s %I.%I owner to %I;',
        object_type.type_name,
        r.nspname,
        r.relname,
        new_owner
      );
    end loop;
  end loop;

  -- Functions and Procedures, 
  for r in 
    select
      p.prokind,
      p.proname,
      n.nspname,
      pg_catalog.pg_get_function_identity_arguments(p.oid) as args
    from pg_catalog.pg_namespace as n
    join pg_catalog.pg_proc as p on p.pronamespace = n.oid
    where not n.nspname in ('pg_catalog', 'information_schema')
    and p.proname not ilike 'dblink%' -- We do not want dblink to be involved (exclusion)
    and p.prokind in ('f', 'p', 'a', 'w')
  loop
    raise debug 'Changing ownership of function %.%(%) to %', 
                r.nspname, r.proname, r.args, new_owner;
    execute format(
      'alter %s %I.%I(%s) owner to %I', -- todo: check support CamelStyle r.args,
      case r.prokind
        when 'f' then 'function'
        when 'w' then 'function'
        when 'p' then 'procedure'
        when 'a' then 'aggregate'
        else 'unknown'
      end,
      r.nspname,
      r.proname,
      r.args,
      new_owner
    );
  end loop;

  -- full text search dictionary
  -- TODO: text search configuration
  for r in 
    select * 
    from pg_catalog.pg_namespace n
    join pg_catalog.pg_ts_dict d on d.dictnamespace = n.oid
    where not n.nspname in ('pg_catalog', 'information_schema')
  loop
    raise debug 'Changing ownership of text search dictionary %.% to %', 
                 r.nspname, r.dictname, new_owner;
    execute format(
      'alter text search dictionary %I.%I owner to %I',
      r.nspname,
      r.dictname,
      new_owner
    );
  end loop;

  -- domain
  for r in 
     select typname, nspname
     from pg_catalog.pg_type
     join pg_catalog.pg_namespace on pg_namespace.oid = pg_type.typnamespace
     where typtype = 'd' and not nspname in ('pg_catalog', 'information_schema')
  loop
    raise debug 'Changing ownership of domain %.% to %', 
                 r.nspname, r.typname, new_owner;
    execute format(
      'alter domain %I.%I owner to %I',
      r.nspname,
      r.typname,
      new_owner
    );
  end loop;

  grant select on pg_stat_activity to @username;
end
$$;
`

func restrictedUserQuery(username, password string) string {
	repl := strings.NewReplacer(
		"@usernameStr", pq.QuoteLiteral(username),
		"@username", pq.QuoteIdentifier(username),
		"@password", pq.QuoteLiteral(password),
	)

	return repl.Replace(restrictionUserCreationTemplate)
}

func restrictedObjectsQuery(username string) string {
	repl := strings.NewReplacer(
		"@usernameStr", pq.QuoteLiteral(username),
		"@username", pq.QuoteIdentifier(username),
	)

	return repl.Replace(restrictionTemplate)
}
