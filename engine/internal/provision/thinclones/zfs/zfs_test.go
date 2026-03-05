package zfs

import (
	"encoding/base64"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

type runnerMock struct {
	cmdOutput string
	err       error
}

func (r runnerMock) Run(string, ...bool) (string, error) {
	return r.cmdOutput, r.err
}

type multiRunnerMock struct {
	outputs map[string]string
	errs    map[string]error
}

func (r multiRunnerMock) Run(cmd string, _ ...bool) (string, error) {
	if e, ok := r.errs[cmd]; ok {
		return "", e
	}

	if o, ok := r.outputs[cmd]; ok {
		return o, nil
	}

	return "", nil
}

type recordingRunnerMock struct {
	props   map[string]string
	cmds    []string
	outputs map[string]string
}

func newRecordingRunner() *recordingRunnerMock {
	return &recordingRunnerMock{props: make(map[string]string), outputs: make(map[string]string)}
}

func (r *recordingRunnerMock) Run(cmd string, _ ...bool) (string, error) {
	r.cmds = append(r.cmds, cmd)

	// Handle: zfs set prop="value" snapshot  (setProperty uses %q for the value).
	if after, ok := strings.CutPrefix(cmd, "zfs set "); ok {
		prop, rest, _ := strings.Cut(after, "=")
		closingQuote := strings.LastIndex(rest, `"`)

		if closingQuote > 0 {
			val, _ := strconv.Unquote(rest[:closingQuote+1])
			snap := strings.TrimSpace(rest[closingQuote+1:])
			r.props[snap+":"+prop] = val
		}

		return "", nil
	}

	// Handle: zfs get -H -o value prop snapshot
	if after, ok := strings.CutPrefix(cmd, "zfs get -H -o value "); ok {
		prop, snap, _ := strings.Cut(after, " ")
		key := snap + ":" + prop

		if v, ok := r.props[key]; ok {
			return v + "\n", nil
		}

		return "-\n", nil
	}

	if o, ok := r.outputs[cmd]; ok {
		return o, nil
	}

	return "", nil
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
datastore/branch/main/cls19p20l4rc73bc2v9g/r0
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
			},
		},
		{
			caseName: "multiple clones",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
datastore/branch/main/cls19p20l4rc73bc2v9g/r0
datastore/branch/main/cls184a0l4rc73bc2v90/r0
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
				"cls184a0l4rc73bc2v90",
			},
		},
		{
			caseName: "clone duplicate",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
datastore/branch/main/cls19p20l4rc73bc2v9g/r0
datastore/branch/main/cls19p20l4rc73bc2v9g/r1
`,
			cloneNames: []string{
				"cls19p20l4rc73bc2v9g",
			},
		},
		{
			caseName: "different pool",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
dblab_pool/branch/main/cls19p20l4rc73bc2v9g/r0
datastore/branch/main/cls184a0l4rc73bc2v90/r0
`,
			cloneNames: []string{
				"cls184a0l4rc73bc2v90",
			},
		},
		{
			caseName: "no matched clone",
			cmdOutput: `datastore/branch/main/clone_pre_20200831030000
dblab_pool/branch/main/cls19p20l4rc73bc2v9g/r0
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

func TestCleanupEmptyDatasets(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedDestroyed []string
	}{
		{
			name: "datasets with children should not be removed",
			input: `test_pool	-
test_pool/branch	-
test_pool/branch/main	-
test_pool/branch/main/clone001	-
test_pool/branch/main/clone001/r0	test_pool@snapshot001`,
			expectedDestroyed: []string{},
		},
		{
			name: "empty branch datasets without children should not be removed",
			input: `test_pool	-
test_pool/branch	-
test_pool/branch/main	-
test_pool/branch/branch1	-
test_pool/branch/branch2	-`,
			expectedDestroyed: []string{},
		},
		{
			name: "mixed case - some with children, some without",
			input: `test_pool	-
test_pool/branch	-
test_pool/branch/main	-
test_pool/branch/main/clone001	-
test_pool/branch/main/clone001/r0	test_pool@snapshot001
test_pool/branch/main/clone002	-
test_pool/branch/main/clone002/r0	test_pool@snapshot002
test_pool/branch/branch	-
test_pool/temporary	-`,
			expectedDestroyed: []string{},
		},

		{
			name:              "empty input",
			input:             ``,
			expectedDestroyed: []string{},
		},
		{
			name:              "only whitespace",
			input:             `   `,
			expectedDestroyed: []string{},
		},
		{
			name: "malformed lines should be skipped",
			input: `test_pool	-
test_pool/branch
invalid line without tabs
test_pool/orphaned	-
	-
test_pool/valid	test_pool@snap1`,
			expectedDestroyed: []string{},
		},
		{
			name: "original example",
			input: `test_dblab_pool	-
test_dblab_pool/branch	-
test_dblab_pool/branch/main	-
test_dblab_pool/branch/main/clone_pre_20250923095219	-
test_dblab_pool/branch/main/clone_pre_20250923095219/r0	test_dblab_pool@snapshot_20250923095219_pre
test_dblab_pool/branch/main/clone_pre_20250923095500	-
test_dblab_pool/branch/main/clone_pre_20250923095500/r0	test_dblab_pool@snapshot_20250923095500_pre
test_dblab_pool/branch/main/clone_pre_20250923100000	-
test_dblab_pool/branch/main/clone_pre_20250923100000/r0	test_dblab_pool@snapshot_20250923100000_pre`,
			expectedDestroyed: []string{},
		},
		{
			name: "should skip branch datasets and only process clones",
			input: `test_pool	-
test_pool/branch	-
test_pool/branch/main	-
test_pool/branch/main/clone001	-
test_pool/branch/main/clone002	-
test_pool/branch/feature	-
test_pool/branch/feature/orphaned_clone	-
test_pool/other	-
test_pool/other/dataset	-`,
			expectedDestroyed: []string{
				"test_pool/branch/main/clone001",
				"test_pool/branch/main/clone002",
				"test_pool/branch/feature/orphaned_clone",
				// Note: test_pool/other/dataset is NOT removed (not under /branch/)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsManager := NewFSManager(runnerMock{}, Config{Pool: &resources.Pool{Name: "testPool"}})

			destroyedDatasets := fsManager.getEmptyDatasets(tt.input)

			sort.Strings(destroyedDatasets)
			sort.Strings(tt.expectedDestroyed)

			if len(destroyedDatasets) != len(tt.expectedDestroyed) {
				t.Errorf("destroyed count mismatch: got %d, want %d\nDestroyed: %v\nExpected: %v",
					len(destroyedDatasets), len(tt.expectedDestroyed),
					destroyedDatasets, tt.expectedDestroyed)
				return
			}

			for i := range destroyedDatasets {
				if destroyedDatasets[i] != tt.expectedDestroyed[i] {
					t.Errorf("destroyed dataset mismatch at index %d: got %s, want %s",
						i, destroyedDatasets[i], tt.expectedDestroyed[i])
				}
			}
		})
	}
}

func TestListAllBranches(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected []models.BranchEntity
	}{
		{
			name:     "empty output",
			output:   "",
			expected: []models.BranchEntity{},
		},
		{
			name:     "tab-delimited output with space-containing branch name",
			output:   "my branch\tpool/branch/main@snap1",
			expected: []models.BranchEntity{{Name: "my branch", Dataset: "pool", SnapshotID: "pool/branch/main@snap1"}},
		},
		{
			name:     "multiple branches from comma-separated field",
			output:   "br1,br2\tpool@snap1",
			expected: []models.BranchEntity{{Name: "br1", Dataset: "pool", SnapshotID: "pool@snap1"}, {Name: "br2", Dataset: "pool", SnapshotID: "pool@snap1"}},
		},
		{
			name:     "line with wrong column count is skipped",
			output:   "single_field_no_tab\nbranch1\tpool@snap2",
			expected: []models.BranchEntity{{Name: "branch1", Dataset: "pool", SnapshotID: "pool@snap2"}},
		},
		{
			name:   "multiple lines with tabs",
			output: "main\tpool@snap1\nfeature\tpool@snap2",
			expected: []models.BranchEntity{
				{Name: "main", Dataset: "pool", SnapshotID: "pool@snap1"},
				{Name: "feature", Dataset: "pool", SnapshotID: "pool@snap2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Manager{runner: runnerMock{cmdOutput: tc.output}, config: Config{Pool: resources.NewPool("pool")}}

			branches, err := m.ListAllBranches(nil)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, branches)
		})
	}
}

func TestListBranches(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected map[string]string
	}{
		{
			name:     "empty output",
			output:   "",
			expected: map[string]string{},
		},
		{
			name:     "normal two-column tab-delimited output",
			output:   "main\tpool@snap1\nfeature\tpool@snap2",
			expected: map[string]string{"main": "pool@snap1", "feature": "pool@snap2"},
		},
		{
			name:     "snapshot name containing spaces is preserved",
			output:   "main\tpool@snap with spaces",
			expected: map[string]string{"main": "pool@snap with spaces"},
		},
		{
			name:     "single-field line without tab is skipped",
			output:   "no_tab_here\nmain\tpool@snap1",
			expected: map[string]string{"main": "pool@snap1"},
		},
		{
			name:     "comma-separated branches map to same snapshot",
			output:   "br1,br2\tpool@snap1",
			expected: map[string]string{"br1": "pool@snap1", "br2": "pool@snap1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Manager{runner: runnerMock{cmdOutput: tc.output}, config: Config{Pool: resources.NewPool("pool")}}

			branches, err := m.listBranches()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, branches)
		})
	}
}

func TestGetRepo(t *testing.T) {
	msg := base64.StdEncoding.EncodeToString([]byte("initial commit"))

	testCases := []struct {
		name          string
		output        string
		wantSnapshots int
		wantBranches  map[string]string
	}{
		{
			name:          "empty output",
			output:        "",
			wantSnapshots: 0,
			wantBranches:  map[string]string{},
		},
		{
			name:          "all 8 fields populated",
			output:        "pool@snap1\tparent1\tchild1\tmain\troot1\t20250101\t" + msg + "\t-",
			wantSnapshots: 1,
			wantBranches:  map[string]string{"main": "pool@snap1"},
		},
		{
			name:          "line with fewer than 8 fields is skipped",
			output:        "pool@snap1\tparent1\tchild1\tmain\troot1\t20250101\t" + msg,
			wantSnapshots: 0,
			wantBranches:  map[string]string{},
		},
		{
			name:          "base64 commit message decodes correctly",
			output:        "pool@snap1\t-\t-\tmain\t-\t20250101\t" + msg + "\t-",
			wantSnapshots: 1,
			wantBranches:  map[string]string{"main": "pool@snap1"},
		},
		{
			name: "multiple snapshots with branches",
			output: "pool@snap1\t-\tpool@snap2\tmain\troot1\t20250101\t" + msg + "\t-\n" +
				"pool@snap2\tpool@snap1\t-\tfeature\troot1\t20250102\t" + msg + "\t-",
			wantSnapshots: 2,
			wantBranches:  map[string]string{"main": "pool@snap1", "feature": "pool@snap2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Manager{runner: runnerMock{cmdOutput: tc.output}, config: Config{Pool: resources.NewPool("pool")}}

			repo, err := m.getRepo(cmdCfg{pool: "pool"})
			require.NoError(t, err)
			assert.Len(t, repo.Snapshots, tc.wantSnapshots)
			assert.Equal(t, tc.wantBranches, repo.Branches)

			if tc.wantSnapshots > 0 {
				for _, snap := range repo.Snapshots {
					if snap.Message != "" && snap.Message != "-" {
						assert.Equal(t, "initial commit", snap.Message)
					}
				}
			}
		})
	}
}

func TestGetSnapshotProperties(t *testing.T) {
	msg := base64.StdEncoding.EncodeToString([]byte("test message"))

	testCases := []struct {
		name     string
		output   string
		expected thinclones.SnapshotProperties
	}{
		{
			name:   "well-formed 8-field output",
			output: "pool@snap1\tparent1\tchild1\tmain\troot1\t20250101\t" + msg + "\tclone1",
			expected: thinclones.SnapshotProperties{
				Name: "pool@snap1", Parent: "parent1", Child: "child1", Branch: "main",
				Root: "root1", DataStateAt: "20250101", Message: "test message", Clones: "clone1",
			},
		},
		{
			name:   "field containing spaces is preserved",
			output: "pool@snap1\tparent with spaces\tchild1\tmain\troot1\t20250101\t" + msg + "\tclone1",
			expected: thinclones.SnapshotProperties{
				Name: "pool@snap1", Parent: "parent with spaces", Child: "child1", Branch: "main",
				Root: "root1", DataStateAt: "20250101", Message: "test message", Clones: "clone1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Manager{runner: runnerMock{cmdOutput: tc.output}, config: Config{Pool: resources.NewPool("pool")}}

			props, err := m.GetSnapshotProperties("pool@snap1")
			require.NoError(t, err)
			assert.Equal(t, tc.expected, props)
		})
	}
}

func TestGetSnapshotPropertiesMalformed(t *testing.T) {
	msg := base64.StdEncoding.EncodeToString([]byte("test message"))

	m := Manager{
		runner: runnerMock{cmdOutput: "pool@snap1\tparent1\tchild1\tmain\troot1\t20250101\t" + msg},
		config: Config{Pool: resources.NewPool("pool")},
	}

	_, err := m.GetSnapshotProperties("pool@snap1")
	assert.Error(t, err)
}

func TestVerifyBranchMetadata_MultiBranchSnapshot(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	// snap2 is a fork point: head of "feature" but NOT head of "main" (snap3 is).
	m.snapshots = []resources.Snapshot{
		{ID: "pool@snap3", Branch: "main"},
		{ID: "pool@snap2", Branch: "main,feature"},
		{ID: "pool@snap1", Branch: "main"},
	}

	err := m.VerifyBranchMetadata()
	require.NoError(t, err)

	// snap3 must be the head of "main".
	mainTag := runner.props["pool@snap3:dle:branch"]
	assert.Contains(t, mainTag, "main")

	// snap2 must be the head of "feature" but must NOT carry "main".
	featureTag := runner.props["pool@snap2:dle:branch"]
	assert.Contains(t, featureTag, "feature")
	assert.NotContains(t, featureTag, "main")
}

func TestSplitBranches(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{input: "", expected: []string{"main"}},
		{input: "-", expected: []string{"main"}},
		{input: "main", expected: []string{"main"}},
		{input: "feature", expected: []string{"feature"}},
		{input: "main,feature", expected: []string{"main", "feature"}},
		{input: "main, feature", expected: []string{"main", "feature"}},
		{input: "a,b,c", expected: []string{"a", "b", "c"}},
		{input: ",", expected: []string{"main"}},
		{input: "-,-", expected: []string{"main"}},
		{input: "main,,feature", expected: []string{"main", "feature"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, splitBranches(tc.input))
		})
	}
}

func TestGetBranchHeadSnapshots(t *testing.T) {
	t.Run("pool-level branch heads are protected directly", func(t *testing.T) {
		listCmd := `zfs list -H -t snapshot -o dle:branch,name -r testpool | grep -v "^-" | cat`

		m := NewFSManager(multiRunnerMock{
			outputs: map[string]string{
				listCmd:                               "main\ttestpool@snapshot_20250101_pre\n",
				"zfs get -H -o value origin testpool": "-\n",
			},
		}, Config{Pool: resources.NewPool("testpool")})

		protected, err := m.getBranchHeadSnapshots()
		require.NoError(t, err)
		assert.Contains(t, protected, "testpool@snapshot_20250101_pre")
	})

	t.Run("clone-level branch heads protect both snapshot and origin", func(t *testing.T) {
		listCmd := `zfs list -H -t snapshot -o dle:branch,name -r testpool | grep -v "^-" | cat`
		cloneDataset := "testpool/branch/main/clone_pre_20250101/r0"

		m := NewFSManager(multiRunnerMock{
			outputs: map[string]string{
				listCmd: "feature\t" + cloneDataset + "@snap_20250101\n",
				"zfs get -H -o value origin " + cloneDataset: "testpool@snapshot_20250101_pre\n",
			},
		}, Config{Pool: resources.NewPool("testpool")})

		protected, err := m.getBranchHeadSnapshots()
		require.NoError(t, err)
		assert.Contains(t, protected, cloneDataset+"@snap_20250101")
		assert.Contains(t, protected, "testpool@snapshot_20250101_pre")
	})

	t.Run("list branches error returns error", func(t *testing.T) {
		listCmd := `zfs list -H -t snapshot -o dle:branch,name -r testpool | grep -v "^-" | cat`

		m := NewFSManager(multiRunnerMock{
			errs: map[string]error{listCmd: errors.New("zfs error")},
		}, Config{Pool: resources.NewPool("testpool")})

		protected, err := m.getBranchHeadSnapshots()
		assert.Error(t, err)
		assert.Nil(t, protected)
	})
}

func TestReadRootProperties(t *testing.T) {
	t.Run("parses single and multiple roots", func(t *testing.T) {
		listCmd := "zfs list -H -t snapshot -o name,dle:root -r pool"
		m := NewFSManager(multiRunnerMock{
			outputs: map[string]string{
				listCmd: "pool@snap1\t-\npool@snap2\tfeature\npool@snap3\t-\npool@snap4\tbr1,br2\n",
			},
		}, Config{Pool: resources.NewPool("pool")})

		roots, err := m.readRootProperties()
		require.NoError(t, err)
		assert.Empty(t, roots["pool@snap1"])
		assert.Equal(t, []string{"feature"}, roots["pool@snap2"])
		assert.Empty(t, roots["pool@snap3"])
		assert.Equal(t, []string{"br1", "br2"}, roots["pool@snap4"])
	})

	t.Run("returns empty map when no roots set", func(t *testing.T) {
		listCmd := "zfs list -H -t snapshot -o name,dle:root -r pool"
		m := NewFSManager(multiRunnerMock{
			outputs: map[string]string{listCmd: "pool@snap1\t-\npool@snap2\t-\n"},
		}, Config{Pool: resources.NewPool("pool")})

		roots, err := m.readRootProperties()
		require.NoError(t, err)
		assert.Empty(t, roots)
	})

	t.Run("returns error on runner failure", func(t *testing.T) {
		listCmd := "zfs list -H -t snapshot -o name,dle:root -r pool"
		m := NewFSManager(multiRunnerMock{
			errs: map[string]error{listCmd: errors.New("zfs error")},
		}, Config{Pool: resources.NewPool("pool")})

		roots, err := m.readRootProperties()
		assert.Error(t, err)
		assert.Nil(t, roots)
	})
}

func TestVerifyBranchMetadata_CrossBranchLinks(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	// main has 3 snapshots, "feature" was forked from snap2.
	// snap2 has dle:root=feature, feature has one snapshot snap_f1.
	m.snapshots = []resources.Snapshot{
		{ID: "pool@snap3", Branch: "main"},
		{ID: "pool@snap_f1", Branch: "feature"},
		{ID: "pool@snap2", Branch: "main"},
		{ID: "pool@snap1", Branch: "main"},
	}

	listCmd := "zfs list -H -t snapshot -o name,dle:root -r pool"
	runner.outputs[listCmd] = "pool@snap1\t-\npool@snap_f1\t-\npool@snap2\tfeature\npool@snap3\t-\n"

	err := m.VerifyBranchMetadata()
	require.NoError(t, err)

	// within-branch: main chain snap1 -> snap2 -> snap3
	assert.Equal(t, "pool@snap1", runner.props["pool@snap2:dle:parent"])
	assert.Equal(t, "pool@snap2", runner.props["pool@snap3:dle:parent"])
	assert.Contains(t, runner.props["pool@snap1:dle:child"], "pool@snap2")
	assert.Contains(t, runner.props["pool@snap2:dle:child"], "pool@snap3")

	// cross-branch: snap2 is parent of snap_f1 (oldest on feature branch)
	assert.Equal(t, "pool@snap2", runner.props["pool@snap_f1:dle:parent"])
	assert.Contains(t, runner.props["pool@snap2:dle:child"], "pool@snap_f1")
}

func TestVerifyBranchMetadata_ForkPointWithBothSidesPopulated(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	// snap2 is a fork point tagged "main,feature".
	// main: snap1 → snap2 → snap3 (head)
	// feature: snap2 → snap_f1 → snap_f2 (head)
	// Both branches have snapshots on either side of the fork point.
	m.snapshots = []resources.Snapshot{
		{ID: "pool@snap3", Branch: "main"},
		{ID: "pool@snap_f2", Branch: "feature"},
		{ID: "pool@snap_f1", Branch: "feature"},
		{ID: "pool@snap2", Branch: "main,feature"},
		{ID: "pool@snap1", Branch: "main"},
	}

	err := m.VerifyBranchMetadata()
	require.NoError(t, err)

	// main chain: snap1 → snap2 → snap3.
	assert.Equal(t, "-", runner.props["pool@snap1:dle:parent"])
	assert.Equal(t, "pool@snap1", runner.props["pool@snap2:dle:parent"])
	assert.Equal(t, "pool@snap2", runner.props["pool@snap3:dle:parent"])
	assert.Contains(t, runner.props["pool@snap1:dle:child"], "pool@snap2")
	assert.Contains(t, runner.props["pool@snap2:dle:child"], "pool@snap3")

	// feature chain: snap2 → snap_f1 → snap_f2.
	assert.Equal(t, "pool@snap2", runner.props["pool@snap_f1:dle:parent"])
	assert.Equal(t, "pool@snap_f1", runner.props["pool@snap_f2:dle:parent"])
	assert.Contains(t, runner.props["pool@snap2:dle:child"], "pool@snap_f1")
	assert.Contains(t, runner.props["pool@snap_f1:dle:child"], "pool@snap_f2")

	// branch heads: snap3=main, snap_f2=feature.
	assert.Contains(t, runner.props["pool@snap3:dle:branch"], "main")
	assert.Contains(t, runner.props["pool@snap_f2:dle:branch"], "feature")

	// fork point snap2 must NOT remain head of either branch.
	branchTag := runner.props["pool@snap2:dle:branch"]
	assert.NotContains(t, branchTag, "main")
	assert.NotContains(t, branchTag, "feature")
}

func TestVerifyBranchMetadata_DuplicateBranchTagsNoDuplicateChildren(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	// Degraded state: consecutive snapshots carry stale multi-branch tags.
	m.snapshots = []resources.Snapshot{
		{ID: "pool@snap4", Branch: "main"},
		{ID: "pool@snap3", Branch: "main,feature"},
		{ID: "pool@snap2", Branch: "main,feature"},
		{ID: "pool@snap1", Branch: "main"},
	}

	err := m.VerifyBranchMetadata()
	require.NoError(t, err)

	// snap2 is parent of both snap3 (main+feature) — child list must not contain duplicates.
	childVal := runner.props["pool@snap2:dle:child"]
	assert.Equal(t, "pool@snap3", childVal, "snap2 child must list snap3 exactly once")
}

func TestVerifyBranchMetadata_SkipsUnchangedProperties(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	m.snapshots = []resources.Snapshot{
		{ID: "pool@snap2", Branch: "main"},
		{ID: "pool@snap1", Branch: "main"},
	}

	// First run: sets all properties.
	err := m.VerifyBranchMetadata()
	require.NoError(t, err)

	assert.Equal(t, "pool@snap1", runner.props["pool@snap2:dle:parent"])
	assert.Equal(t, "pool@snap2", runner.props["pool@snap1:dle:child"])

	// Pre-populate bulk read output with already-correct values so the second run can diff.
	pcCmd := "zfs list -H -t snapshot -o name,dle:parent,dle:child -r pool"
	runner.outputs[pcCmd] = "pool@snap2\tpool@snap1\t-\npool@snap1\t-\tpool@snap2\n"
	runner.cmds = nil

	err = m.VerifyBranchMetadata()
	require.NoError(t, err)

	// No parent/child set commands should have been issued on the second run
	// because the existing values match the computed ones.
	assert.Zero(t, countSetCmds(runner.cmds), "expected no parent/child set commands on re-run with unchanged data, got: %v", runner.cmds)
}

func countSetCmds(cmds []string) int {
	count := 0

	for _, c := range cmds {
		if strings.HasPrefix(c, "zfs set dle:parent") || strings.HasPrefix(c, "zfs set dle:child") {
			count++
		}
	}

	return count
}
