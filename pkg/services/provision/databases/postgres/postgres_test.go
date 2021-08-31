package postgres

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
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

		err := Stop(runner, p, "test_clone")

		assert.Equal(t, tc.err, errors.Cause(err))
	}
}
