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
		db := "postgres"
		query := restrictedUserQuery(user, pwd, db)

		assert.Contains(t, query, `create user "user1" with password 'pwd' login;`)
		assert.Contains(t, query, `new_owner := 'user1'`)

	})

	t.Run("special chars must be quoted", func(t *testing.T) {
		user := "user.test\""
		pwd := "pwd\\'--"
		db := "postgres"
		query := restrictedUserQuery(user, pwd, db)

		assert.Contains(t, query, `create user "user.test""" with password  E'pwd\\''--' login;`)
		assert.Contains(t, query, `new_owner := 'user.test"'`)
	})
}
