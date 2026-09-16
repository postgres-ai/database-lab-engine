package postgres

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
)

type MockRunner struct {
	mock.Mock
	output string
	err    error
}

func (m *MockRunner) Run(cmd string, _ ...bool) (string, error) {
	m.Called(cmd)

	err := m.err
	if m.err != nil {
		err = runners.NewRunnerError(cmd, m.err.Error(), m.err)
	}

	return m.output, err
}

func TestGetPgConnStr(t *testing.T) {
	testCases := []struct {
		host     string
		dbname   string
		username string
		port     uint
		expected string
	}{
		{host: "localhost", dbname: "testdb", username: "postgres", port: 5432, expected: "host=localhost port=5432 dbname=testdb user=postgres "},
		{host: "", dbname: "mydb", username: "admin", port: 6000, expected: "port=6000 dbname=mydb user=admin "},
		{host: "127.0.0.1", dbname: "postgres", username: "user1", port: 6100, expected: "host=127.0.0.1 port=6100 dbname=postgres user=user1 "},
		{host: "/var/run/postgresql", dbname: "db", username: "u", port: 5433, expected: "host=/var/run/postgresql port=5433 dbname=db user=u "},
	}

	for _, tc := range testCases {
		result := getPgConnStr(tc.host, tc.dbname, tc.username, tc.port)
		assert.Equal(t, tc.expected, result)
	}
}

func TestGetPgConnStr_EmptyHostOmitted(t *testing.T) {
	result := getPgConnStr("", "db", "user", 5432)
	assert.NotContains(t, result, "host=")
	assert.Contains(t, result, "port=5432")
	assert.Contains(t, result, "dbname=db")
	assert.Contains(t, result, "user=user")
}

func TestGetPgConnStr_ComponentOrder(t *testing.T) {
	result := getPgConnStr("host1", "db1", "user1", 5432)
	hostIdx := strings.Index(result, "host=")
	portIdx := strings.Index(result, "port=")
	dbnameIdx := strings.Index(result, "dbname=")
	userIdx := strings.Index(result, "user=")

	require.True(t, hostIdx < portIdx, "host should come before port")
	require.True(t, portIdx < dbnameIdx, "port should come before dbname")
	require.True(t, dbnameIdx < userIdx, "dbname should come before user")
}

func TestUserExistsQuery(t *testing.T) {
	testCases := []struct {
		username string
		expected string
	}{
		{username: "admin", expected: "select exists (select from pg_roles where rolname = 'admin')"},
		{username: "user.test", expected: "select exists (select from pg_roles where rolname = 'user.test')"},
		{username: "user'inject", expected: "select exists (select from pg_roles where rolname = 'user''inject')"},
	}

	for _, tc := range testCases {
		result := userExistsQuery(tc.username)
		assert.Equal(t, tc.expected, result)
	}
}

func TestRestrictedObjectsQuery(t *testing.T) {
	t.Run("replaces username placeholders", func(t *testing.T) {
		query := restrictedObjectsQuery("testuser")
		assert.Contains(t, query, `'testuser'`)
		assert.Contains(t, query, `"testuser"`)
		assert.NotContains(t, query, "@usernameStr")
		assert.NotContains(t, query, "@username")
	})

	t.Run("special characters in username are quoted", func(t *testing.T) {
		query := restrictedObjectsQuery("user'inject")
		assert.Contains(t, query, `'user''inject'`)
		assert.NotContains(t, query, "@usernameStr")
	})

	t.Run("contains schema ownership change", func(t *testing.T) {
		query := restrictedObjectsQuery("admin")
		assert.Contains(t, query, "pg_namespace")
		assert.Contains(t, query, "alter schema")
	})

	t.Run("contains function ownership change", func(t *testing.T) {
		query := restrictedObjectsQuery("admin")
		assert.Contains(t, query, "pg_proc")
		assert.Contains(t, query, "alter")
	})

	t.Run("grants select on pg_stat_activity", func(t *testing.T) {
		query := restrictedObjectsQuery("admin")
		assert.Contains(t, query, fmt.Sprintf("grant select on pg_stat_activity to %q", "admin"))
	})
}

func TestRestrictedUserOwnershipQuery_ReplacesAllPlaceholders(t *testing.T) {
	query := restrictedUserOwnershipQuery("testuser", "secret")
	assert.NotContains(t, query, "@usernameStr")
	assert.NotContains(t, query, "@username")
	assert.NotContains(t, query, "@password")
}

func TestRemoveContainers(t *testing.T) {
	p := &resources.Pool{}
	testCases := []struct {
		output string
		err    error
	}{
		{
			output: "",
			err:    nil,
		},
		{
			err:    runners.RunnerError{Msg: "test fail"},
			output: "Unknown error",
		},
		{
			err:    nil,
			output: "Error: No such container:",
		},
	}

	for _, tc := range testCases {
		runner := &MockRunner{
			output: tc.output,
			err:    tc.err,
		}
		runner.On("Run",
			mock.MatchedBy(
				func(cmd string) bool {
					return strings.HasPrefix(cmd, "docker container rm --force --volumes ")
				})).
			Return("", tc.err).
			On("Run",
				mock.MatchedBy(
					func(cmd string) bool {
						return strings.HasPrefix(cmd, "rm -rf ")
					})).
			Return("", nil)

		err := Stop(runner, p, "test_clone", "6200")

		assert.Equal(t, tc.err, errors.Cause(err))
	}
}

func TestStartTimeoutError(t *testing.T) {
	queryErr := errors.New("connection refused")

	testCases := []struct {
		name          string
		recoveryState string
		queryErr      error
		wantContains  string
	}{
		{name: "still in recovery", recoveryState: "t", queryErr: nil, wantContains: `still in recovery (pg_is_in_recovery: "t")`},
		{name: "no status at all", recoveryState: "", queryErr: nil, wantContains: `pg_is_in_recovery: ""`},
		{name: "status check failed", recoveryState: "", queryErr: queryErr, wantContains: "last status check failed: connection refused"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := startTimeoutError(tc.recoveryState, tc.queryErr)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "postgres start timeout")
			assert.Contains(t, err.Error(), tc.wantContains)

			if tc.queryErr != nil {
				assert.ErrorIs(t, err, tc.queryErr)
			}
		})
	}
}
