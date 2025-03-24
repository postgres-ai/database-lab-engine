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
	m := Manager{config: Config{Pool: &resources.Pool{Name: "dblab_pool"}, PreSnapshotSuffix: preSnapshotSuffix}}

	out := `dblab_pool	-
	dblab_pool/clone_pre_20210127105215	dblab_pool@snapshot_20210127105215_pre
	dblab_pool/clone_pre_20210127113000	dblab_pool@snapshot_20210127113000_pre
	dblab_pool/clone_pre_20210127120000	dblab_pool@snapshot_20210127120000_pre
	dblab_pool/clone_pre_20210127123000	dblab_pool@snapshot_20210127123000_pre
	dblab_pool/clone_pre_20210127130000	dblab_pool@snapshot_20210127130000_pre
	dblab_pool/clone_pre_20210127133000	dblab_pool@snapshot_20210127133000_pre
	dblab_pool/clone_pre_20210127140000	dblab_pool@snapshot_20210127140000_pre
	dblab_pool/cls19p20l4rc73bc2v9g	dblab_pool/clone_pre_20210127133000@snapshot_20210127133008
	dblab_pool/cls19p20l4rc73bc2v9g	dblab_pool/clone_pre_20210127123000@snapshot_20210127133008
	dblab_pool/branch/main/20241121094823 dblab_pool@snapshot_20241121094523
	`
	expected := []string{"dblab_pool@snapshot_20210127133000_pre", "dblab_pool@snapshot_20210127123000_pre", "dblab_pool/branch/main/20241121094823",
		"dblab_pool@snapshot_20241121094523"}

	list := m.getBusySnapshotList(out)
	require.Equal(t, 4, len(list))
	assert.Contains(t, list, expected[0])
	assert.Contains(t, list, expected[1])
	assert.Contains(t, list, expected[2])
	assert.Contains(t, list, expected[3])
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
