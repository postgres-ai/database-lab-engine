package zfs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runnerMock struct {
	cmdOutput string
	err       error
}

func (r runnerMock) Run(string, ...bool) (string, error) {
	return r.cmdOutput, r.err
}

func TestListClones(t *testing.T) {
	const (
		pool        = "datastore"
		clonePrefix = "dblab_clone_"
	)

	testCases := []struct {
		caseName   string
		cmdOutput  string
		cloneNames []string
	}{
		{
			caseName:   "empty output",
			cloneNames: []string{},
		},
		{
			caseName: "single clone",
			cmdOutput: `datastore/clone_pre_20200831030000
datastore/dblab_clone_6000
`,
			cloneNames: []string{
				"dblab_clone_6000",
			},
		},
		{
			caseName: "multiple clones",
			cmdOutput: `datastore/clone_pre_20200831030000
datastore/dblab_clone_6000
datastore/dblab_clone_6001
`,
			cloneNames: []string{
				"dblab_clone_6000",
				"dblab_clone_6001",
			},
		},
		{
			caseName: "clone duplicate",
			cmdOutput: `datastore/clone_pre_20200831030000
datastore/dblab_clone_6000
datastore/dblab_clone_6000
`,
			cloneNames: []string{
				"dblab_clone_6000",
			},
		},
		{
			caseName: "different pool",
			cmdOutput: `datastore/clone_pre_20200831030000
dblab_pool/dblab_clone_6001
datastore/dblab_clone_6000
`,
			cloneNames: []string{
				"dblab_clone_6000",
			},
		},
		{
			caseName: "no matched clone",
			cmdOutput: `datastore/clone_pre_20200831030000
dblab_pool/dblab_clone_6001
`,
			cloneNames: []string{},
		},
	}

	for _, testCase := range testCases {
		runner := runnerMock{
			cmdOutput: testCase.cmdOutput,
		}

		listClones, err := ListClones(runner, pool, clonePrefix)

		require.NoError(t, err, testCase.caseName)
		assert.Equal(t, testCase.cloneNames, listClones, testCase.caseName)
	}
}

func TestFailedListClones(t *testing.T) {
	runner := runnerMock{
		err: errors.New("runner error"),
	}

	cloneNames, err := ListClones(runner, "pool", "dblab_clone_")

	assert.Nil(t, cloneNames)
	assert.EqualError(t, err, "failed to list clones: runner error")
}
