package zfs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
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
		poolName          = "datastore"
		preSnapshotSuffix = "_pre"
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
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
datastore/branch/main/cls19p20l4rc73bc2v9g
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
			},
		},
		{
			caseName: "multiple clones",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
datastore/branch/main/cls19p20l4rc73bc2v9g
datastore/branch/main/cls184a0l4rc73bc2v90
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
				"cls184a0l4rc73bc2v90",
			},
		},
		{
			caseName: "clone duplicate",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
datastore/branch/main/cls19p20l4rc73bc2v9g
datastore/branch/main/cls19p20l4rc73bc2v9g
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
			},
		},
		{
			caseName: "different pool",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
dblab_pool/branch/main/cls19p20l4rc73bc2v9g
datastore/branch/main/cls184a0l4rc73bc2v90
`,
			cloneNames: []string{
				"cls184a0l4rc73bc2v90",
			},
		},
		{
			caseName: "no matched clone",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
dblab_pool/branch/main/cls19p20l4rc73bc2v9g
`,
			cloneNames: []string{},
		},
	}

	for _, testCase := range testCases {
		m := Manager{
			runner: runnerMock{
				cmdOutput: testCase.cmdOutput,
			},
			config: Config{
				Pool:              resources.NewPool(poolName),
				PreSnapshotSuffix: preSnapshotSuffix,
			},
		}

		listClones, err := m.ListClonesNames()

		require.NoError(t, err, testCase.caseName)
		assert.Equal(t, testCase.cloneNames, listClones, testCase.caseName)
	}
}

func TestFailedListClones(t *testing.T) {
	m := Manager{
		runner: runnerMock{
			err: errors.New("runner error"),
		},
	}

	cloneNames, err := m.ListClonesNames()

	assert.Nil(t, cloneNames)
	assert.EqualError(t, err, "failed to list clones: runner error")
}

func TestBusySnapshotList(t *testing.T) {
	const preSnapshotSuffix = "_pre"
	m := Manager{config: Config{Pool: &resources.Pool{Name: "test_dblab_pool"}, PreSnapshotSuffix: preSnapshotSuffix}}

	out := `test_dblab_pool	-
test_dblab_pool/branch	-
test_dblab_pool/branch/main	-
test_dblab_pool/branch/main/clone_pre_20250403061908	-
test_dblab_pool/branch/main/clone_pre_20250403061908/r0	test_dblab_pool@snapshot_20250403061908_pre
test_dblab_pool/branch/main/clone_pre_20250403085500	-
test_dblab_pool/branch/main/clone_pre_20250403085500/r0	test_dblab_pool@snapshot_20250403085500_pre
test_dblab_pool/branch/main/clone_pre_20250403090000	-
test_dblab_pool/branch/main/clone_pre_20250403090000/r0	test_dblab_pool@snapshot_20250403090000_pre
test_dblab_pool/branch/main/clone_pre_20250403090500	-
test_dblab_pool/branch/main/clone_pre_20250403090500/r0	test_dblab_pool@snapshot_20250403090500_pre
test_dblab_pool/branch/main/cvn2j50n9i6s73as3k9g	-
test_dblab_pool/branch/main/cvn2j50n9i6s73as3k9g/r0	test_dblab_pool/branch/main/clone_pre_20250403061908/r0@snapshot_20250403061908
test_dblab_pool/branch/main/cvn2kdon9i6s73as3ka0	-
test_dblab_pool/branch/main/cvn2kdon9i6s73as3ka0/r0	test_dblab_pool/branch/new001@20250403062641
test_dblab_pool/branch/new001	test_dblab_pool/branch/main/cvn2j50n9i6s73as3k9g/r0@20250403062503
test_dblab_pool/branch/new001/cvn4n38n9i6s73as3kag	-
test_dblab_pool/branch/new001/cvn4n38n9i6s73as3kag/r0	test_dblab_pool/branch/new001@20250403062641
`
	expected := []string{
		"test_dblab_pool@snapshot_20250403061908_pre",
	}

	list := m.getBusySnapshotList(out)
	require.Len(t, list, len(expected))
	assert.ElementsMatch(t, list, expected)
}

func TestExcludingBusySnapshots(t *testing.T) {
	testCases := []struct {
		snapshotList []string
		result       string
	}{
		{
			snapshotList: []string{},
			result:       "",
		},
		{
			snapshotList: []string{"dblab_pool@snapshot_20210127133000_pre"},
			result:       "| grep -Ev 'dblab_pool@snapshot_20210127133000_pre' ",
		},
		{
			snapshotList: []string{"dblab_pool@snapshot_20210127133000_pre", "dblab_pool@snapshot_20210127123000_pre"},
			result:       "| grep -Ev 'dblab_pool@snapshot_20210127133000_pre|dblab_pool@snapshot_20210127123000_pre' ",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.result, excludeBusySnapshots(tc.snapshotList))
	}
}

func TestProcessingMappingOutput(t *testing.T) {
	out := `pgclusters     	           			/var/lib/postgresql/pools
pgclusters/dblab           			/var/lib/postgresql/pools/dblab
pgclusters/pgcluster1      			/var/lib/postgresql/pools/pgcluster1
pgclusters/pgcluster2      			/var/lib/postgresql/pools/pgcluster2
pgclusters/pgcluster5      			/var/lib/postgresql/pools/pgcluster5/data
datastore                           /var/lib/postgresql/pools/datastore
datastore/clone_pre_20210729130000  /var/lib/postgresql/pools/datastore/clones/clone_pre_20210729130000
datastore/dblab_clone_6000          /var/lib/postgresql/pools/datastore/clones/dblab_clone_6000
datastore/dblab_clone_6001          /var/lib/postgresql/pools/datastore/clones/dblab_clone_6001
poolnext 							/var/lib/postgresql/pools/pool5`
	mountDir := "/var/lib/postgresql/pools"
	expected := map[string]string{
		"dblab":      "pgclusters/dblab",
		"pgcluster1": "pgclusters/pgcluster1",
		"pgcluster2": "pgclusters/pgcluster2",
		"datastore":  "datastore",
		"pool5":      "poolnext",
	}

	poolMappings := processMappingOutput(out, mountDir)
	assert.Equal(t, len(expected), len(poolMappings))
	assert.Equal(t, expected, poolMappings)
}

func TestBuildingCommands(t *testing.T) {
	t.Run("Origin Command", func(t *testing.T) {
		command := buildOriginCommand("testClone")
		require.Equal(t, "zfs get -H -o value origin testClone", command)
	})

	t.Run("Snapshot Size Command", func(t *testing.T) {
		command := buildSizeCommand("testSnapshot")
		require.Equal(t, "zfs get -H -p -o value used testSnapshot", command)
	})
}

func TestSnapshotList(t *testing.T) {
	t.Run("Snapshot list", func(t *testing.T) {
		fsManager := NewFSManager(runnerMock{}, Config{Pool: &resources.Pool{Name: "testPool"}})

		require.Equal(t, 0, len(fsManager.SnapshotList()))

		snapshot := resources.Snapshot{ID: "test1"}
		fsManager.addSnapshotToList(snapshot)

		require.Equal(t, 1, len(fsManager.SnapshotList()))
		require.Equal(t, []resources.Snapshot{{ID: "test1"}}, fsManager.SnapshotList())

		snapshot2 := resources.Snapshot{ID: "test2"}
		fsManager.addSnapshotToList(snapshot2)

		snapshot3 := resources.Snapshot{ID: "test3"}
		fsManager.addSnapshotToList(snapshot3)

		require.Equal(t, 3, len(fsManager.SnapshotList()))
		require.Equal(t, []resources.Snapshot{{ID: "test3"}, {ID: "test2"}, {ID: "test1"}}, fsManager.SnapshotList())

		fsManager.removeSnapshotFromList("test2")

		require.Equal(t, 2, len(fsManager.SnapshotList()))
		require.Equal(t, []resources.Snapshot{{ID: "test3"}, {ID: "test1"}}, fsManager.SnapshotList())
	})
}
