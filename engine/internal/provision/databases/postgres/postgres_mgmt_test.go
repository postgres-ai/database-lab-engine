/*
2021 © Postgres.ai
*/

package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSuperuserQuery_CreateVsAlter(t *testing.T) {
	testCases := []struct {
		name     string
		user     string
		pwd      string
		exists   bool
		contains string
	}{
		{name: "new user creates", user: "dbadmin", pwd: "pass123", exists: false, contains: "create user"},
		{name: "existing user alters", user: "dbadmin", pwd: "pass123", exists: true, contains: "alter role"},
		{name: "both include superuser", user: "admin", pwd: "x", exists: false, contains: "superuser"},
		{name: "alter also includes superuser", user: "admin", pwd: "x", exists: true, contains: "superuser"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := superuserQuery(tc.user, tc.pwd, tc.exists)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestRestrictedUserQuery_CreateVsAlter(t *testing.T) {
	testCases := []struct {
		name     string
		user     string
		pwd      string
		exists   bool
		contains string
	}{
		{name: "new user creates", user: "app", pwd: "secret", exists: false, contains: "create user"},
		{name: "existing user alters", user: "app", pwd: "secret", exists: true, contains: "alter role"},
		{name: "both include login", user: "app", pwd: "x", exists: false, contains: "login"},
		{name: "alter also includes login", user: "app", pwd: "x", exists: true, contains: "login"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := restrictedUserQuery(tc.user, tc.pwd, tc.exists)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestRestrictedUserQuery_DoesNotContainSuperuser(t *testing.T) {
	assert.NotContains(t, restrictedUserQuery("u", "p", false), "superuser")
	assert.NotContains(t, restrictedUserQuery("u", "p", true), "superuser")
}

func TestSuperuserQuery_ContainsLogin(t *testing.T) {
	assert.Contains(t, superuserQuery("u", "p", false), "login")
	assert.Contains(t, superuserQuery("u", "p", true), "login")
}

func TestRestrictedObjectsQuery_ContainsAllObjectTypes(t *testing.T) {
	query := restrictedObjectsQuery("testuser")

	assert.Contains(t, query, "pg_namespace")
	assert.Contains(t, query, "pg_type")
	assert.Contains(t, query, "pg_class")
	assert.Contains(t, query, "pg_proc")
	assert.Contains(t, query, "pg_ts_dict")
}

func TestRestrictedObjectsQuery_FinalizesPendingDetach(t *testing.T) {
	query := restrictedObjectsQuery("testuser")

	versionGuard := strings.Index(query, "current_setting('server_version_num')::int >= 140000")
	pendingFilter := strings.Index(query, "where i.inhdetachpending")
	finalize := strings.Index(query, "detach partition %I.%I finalize")
	changeOwner := strings.Index(query, "'alter %s %I.%I owner to %I;'")

	require.NotEqual(t, -1, versionGuard)
	require.NotEqual(t, -1, changeOwner)
	assert.Less(t, versionGuard, pendingFilter, "the pending detach lookup must stay behind the version guard")
	assert.Less(t, pendingFilter, finalize)
	assert.Less(t, finalize, changeOwner, "the detach must be completed before relation ownership is changed")
}

func TestRestrictedUserOwnershipQuery_ContainsDatabaseIteration(t *testing.T) {
	query := restrictedUserOwnershipQuery("user1", "pwd")
	assert.Contains(t, query, "pg_catalog.pg_database")
	assert.Contains(t, query, "alter database")
}
