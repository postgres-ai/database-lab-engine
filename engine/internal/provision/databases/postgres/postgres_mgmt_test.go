/*
2021 © Postgres.ai
*/

package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuperuserQuery(t *testing.T) {
	t.Run("username and password must be quoted", func(t *testing.T) {
		user := "user1"
		pwd := "pwd"
		assert.Equal(t, `create user "user1" with password 'pwd' login superuser;`, superuserQuery(user, pwd))
	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		assert.Equal(t, `create user "user.test""" with password  E'pwd\\''--' login superuser;`, superuserQuery(user, pwd))
	})
}

func TestRestrictedUserQuery(t *testing.T) {
	t.Run("username and password must be quoted", func(t *testing.T) {
		user := "user1"
		pwd := "pwd"
		query := restrictedUserQuery(user, pwd)

		assert.Contains(t, query, `create user "user1" with password 'pwd' login;`)
		assert.Contains(t, query, `new_owner := 'user1'`)

	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		query := restrictedUserQuery(user, pwd)

		assert.Contains(t, query, `create user "user.test""" with password  E'pwd\\''--' login;`)
		assert.Contains(t, query, `new_owner := 'user.test"'`)
	})

	t.Run("change owner of all databases", func(t *testing.T) {
		user := "user.test"
		pwd := "pwd"
		query := restrictedUserQuery(user, pwd)

		assert.Contains(t, query, `select datname from pg_catalog.pg_database where not datistemplat`)
	})

}
