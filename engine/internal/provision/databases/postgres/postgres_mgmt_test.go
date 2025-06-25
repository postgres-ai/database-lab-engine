/*
2021 © Postgres.ai
*/

package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuperuserQuery(t *testing.T) {
	const (
		user     = "user1"
		userTest = "user.test\""
		pwd      = "pwd"
		pwdQuote = "pwd\\'--"
	)

	t.Run("username and password must be quoted", func(t *testing.T) {
		assert.Equal(t, `create user "user1" with password 'pwd' login superuser;`, superuserQuery(user, pwd, false))
	})

	t.Run("username and password must be quoted", func(t *testing.T) {
		assert.Equal(t, `alter role "user1" with password 'pwd' login superuser;`, superuserQuery(user, pwd, true))
	})

	t.Run("special chars must be quoted", func(t *testing.T) {

		assert.Equal(t, `create user "user.test""" with password  E'pwd\\''--' login superuser;`,
			superuserQuery(userTest, pwdQuote, false))
	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		assert.Equal(t, `alter role "user.test""" with password  E'pwd\\''--' login superuser;`,
			superuserQuery(userTest, pwdQuote, true))
	})
}

func TestRestrictedUserQuery(t *testing.T) {
	t.Run("username and password must be quoted", func(t *testing.T) {
		user := "user1"
		pwd := "pwd"
		query := restrictedUserQuery(user, pwd, false)

		assert.Contains(t, query, `create user "user1" with password 'pwd' login;`)
	})

	t.Run("username and password must be quoted", func(t *testing.T) {
		user := "user1"
		pwd := "pwd"
		query := restrictedUserQuery(user, pwd, true)

		assert.Contains(t, query, `alter role "user1" with password 'pwd' login;`)
	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		query := restrictedUserQuery(user, pwd, false)

		assert.Contains(t, query, `create user "user.test""" with password  E'pwd\\''--' login;`)
	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		query := restrictedUserQuery(user, pwd, true)

		assert.Contains(t, query, `alter role "user.test""" with password  E'pwd\\''--' login;`)
	})
}

func TestRestrictedUserOwnershipQuery(t *testing.T) {
	t.Run("username and password must be quoted", func(t *testing.T) {
		user := "user1"
		pwd := "pwd"
		query := restrictedUserOwnershipQuery(user, pwd)

		assert.Contains(t, query, `new_owner := 'user1'`)
	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		query := restrictedUserOwnershipQuery(user, pwd)

		assert.Contains(t, query, `new_owner := 'user.test"'`)
	})

	t.Run("change owner of all databases", func(t *testing.T) {
		user := "user.test"
		pwd := "pwd"
		query := restrictedUserOwnershipQuery(user, pwd)

		assert.Contains(t, query, `select datname from pg_catalog.pg_database where not datistemplat`)
	})
}
