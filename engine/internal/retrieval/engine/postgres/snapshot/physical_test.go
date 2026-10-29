/*
2021 © Postgres.ai
*/

package snapshot

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/dbmarker"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

func TestInitParamsExtraction(t *testing.T) {
	testCases := []struct {
		controlData         string
		expectedDataStateAt string
	}{
		{
			controlData: `
pg_control version number:            1201
Latest checkpoint location:           67E/8966A4A0
Time of latest checkpoint:            Fri 12 Feb 2021 01:15:09 PM MSK
Minimum recovery ending location:     0/0
`,
			expectedDataStateAt: "20210212131509",
		},
		{
			controlData: `
pg_control version number:            1201
Latest checkpoint location:           67E/8966A4A0
Time of latest checkpoint:            Mon Feb 15 08:51:38 2021
Minimum recovery ending location:     0/0
`,
			expectedDataStateAt: "20210215085138",
		},
	}

	for _, tc := range testCases {
		dsa, err := getCheckPointTimestamp(context.Background(), bytes.NewBufferString(tc.controlData))

		require.Nil(t, err)
		assert.EqualValues(t, tc.expectedDataStateAt, dsa)
	}
}

func TestWalDir(t *testing.T) {
	log.SetDebug(false)

	assert.EqualValues(t, "/tmp/pg_wal", walDir("/tmp"))
	assert.EqualValues(t, "/data/pg_wal", walDir("/data"))
}

func TestWalCommand(t *testing.T) {
	log.SetDebug(false)

	testCases := []struct {
		pgVersion          float64
		walName            string
		expectedWalCommand string
	}{
		{
			pgVersion:          10,
			walName:            "000000010000000000000002",
			expectedWalCommand: "/usr/lib/postgresql/10/bin/pg_waldump 000000010000000000000002 -r Transaction | tail -1",
		},
		{
			pgVersion:          11,
			walName:            "000000010000000000000002",
			expectedWalCommand: "/usr/lib/postgresql/11/bin/pg_waldump 000000010000000000000002 -r Transaction | tail -1",
		},
		{
			pgVersion:          12,
			walName:            "000000010000000000000002",
			expectedWalCommand: "/usr/lib/postgresql/12/bin/pg_waldump 000000010000000000000002 -r Transaction | tail -1",
		},
		{
			pgVersion:          13,
			walName:            "000000010000000000000002",
			expectedWalCommand: "/usr/lib/postgresql/13/bin/pg_waldump 000000010000000000000002 -r Transaction | tail -1",
		},
	}

	for _, tc := range testCases {
		resultCommand := walCommand(tc.pgVersion, tc.walName)
		assert.EqualValues(t, tc.expectedWalCommand, resultCommand)
	}
}

func TestParsingWalLine(t *testing.T) {
	log.SetDebug(false)

	testCases := []struct {
		line                string
		expectedDataStateAt string
	}{
		{
			line:                "",
			expectedDataStateAt: "",
		},
		{
			line:                "COMMIT",
			expectedDataStateAt: "",
		},
		{
			line:                `Transaction len (rec/tot):     34/    34, tx:      62566, lsn: C8/3E013E78, prev C8/3E013E40, desc: COMMIT 2021-05-23 02:50:59.993820 UTC`,
			expectedDataStateAt: "20210523025059",
		},
		{
			line:                "rmgr: Transaction len (rec/tot):     82/    82, tx:      62559, lsn: C8/370012E0, prev C8/37001290, desc: COMMIT 2021-05-23 01:17:21.531705 UTC; inval msgs: catcache 11 catcache 10",
			expectedDataStateAt: "20210523011721",
		},
		{
			line:                "rmgr: Transaction len (rec/tot):    130/   130, tx:        557, lsn: 0/020005E0, prev 0/02000598, desc: COMMIT 2021-06-02 11:00:43.735108 UTC; rels: base/12994/16384; inval msgs: catcache 54 catcache 50 catcache 49 relcache 16384",
			expectedDataStateAt: "20210602110043",
		},
	}

	for _, tc := range testCases {
		dsa := parseWALLine(tc.line)
		assert.EqualValues(t, tc.expectedDataStateAt, dsa)
	}
}

func TestRecoveryConfig(t *testing.T) {
	testCases := []struct {
		fileConfig     map[string]string
		userConfig     map[string]string
		recoveryConfig map[string]string
	}{
		{
			fileConfig: map[string]string{
				"restore_command":          "cp /mnt/wal/%f %p",
				"standby_mode":             "true",
				"recovery_target_timeline": "latest",
			},
			recoveryConfig: map[string]string{
				"restore_command":          "cp /mnt/wal/%f %p",
				"recovery_target":          "immediate",
				"recovery_target_action":   "promote",
				"recovery_target_timeline": "latest",
			},
		},
		{
			fileConfig: map[string]string{
				"restore_command":          "cp /mnt/wal/%f %p",
				"standby_mode":             "true",
				"recovery_target_timeline": "latest",
			},
			userConfig: map[string]string{
				"restore_command":          "wal-g wal-fetch %f %p",
				"recovery_target":          "immediate",
				"recovery_target_action":   "promote",
				"recovery_target_timeline": "latest",
			},
			recoveryConfig: map[string]string{
				"restore_command":          "wal-g wal-fetch %f %p",
				"recovery_target":          "immediate",
				"recovery_target_action":   "promote",
				"recovery_target_timeline": "latest",
			},
		},
	}

	for _, tc := range testCases {
		recoveryConfig := buildRecoveryConfig(tc.fileConfig, tc.userConfig)
		assert.EqualValues(t, tc.recoveryConfig, recoveryConfig)
	}
}

func TestBuildRecoveryConfig_noRestoreCommand(t *testing.T) {
	fileConfig := map[string]string{"standby_mode": "true", "recovery_target_timeline": "latest"}

	result := buildRecoveryConfig(fileConfig, nil)

	assert.Equal(t, map[string]string{"standby_mode": "true", "recovery_target_timeline": "latest"}, result)
}

func TestBuildRecoveryConfig_emptyFileConfig(t *testing.T) {
	result := buildRecoveryConfig(map[string]string{}, nil)
	assert.Equal(t, map[string]string{}, result)
}

func TestBuildRecoveryConfig_userConfigOverridesFileConfig(t *testing.T) {
	fileConfig := map[string]string{"restore_command": "cp /mnt/wal/%f %p", "standby_mode": "true"}
	userConfig := map[string]string{"restore_command": "custom_cmd %f %p"}

	result := buildRecoveryConfig(fileConfig, userConfig)

	assert.Equal(t, userConfig, result)
}

func TestParseWALLine_invalidTimestamp(t *testing.T) {
	log.SetDebug(false)

	testCases := []struct {
		name string
		line string
	}{
		{name: "no commit token", line: "rmgr: Heap len (rec/tot): 54/54, tx: 100"},
		{name: "commit with garbage timestamp", line: "desc: COMMIT not-a-date"},
		{name: "commit with empty trailing", line: "desc: COMMIT "},
		{name: "multiple commits takes last", line: "COMMIT bad COMMIT also-bad"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, parseWALLine(tc.line))
		})
	}
}

func TestParseWALLine_semicolonSeparator(t *testing.T) {
	log.SetDebug(false)

	line := "desc: COMMIT 2023-11-15 10:30:45.123456 UTC; rels: base/12994/16384"
	assert.Equal(t, "20231115103045", parseWALLine(line))
}

func TestWalCommand_outputFormat(t *testing.T) {
	testCases := []struct {
		name      string
		pgVersion float64
		walPath   string
		contains  string
	}{
		{name: "pg 10 uses pg_waldump", pgVersion: 10, walPath: "/wal/000000010000000000000001", contains: "pg_waldump"},
		{name: "pg 14 uses pg_waldump", pgVersion: 14, walPath: "/wal/000000010000000000000001", contains: "pg_waldump"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := walCommand(tc.pgVersion, tc.walPath)
			assert.Contains(t, cmd, tc.contains)
			assert.Contains(t, cmd, tc.walPath)
			assert.Contains(t, cmd, "-r Transaction | tail -1")
		})
	}
}

func TestGetCheckPointTimestamp_missingLabel(t *testing.T) {
	controlData := `
pg_control version number:            1201
Latest checkpoint location:           67E/8966A4A0
Minimum recovery ending location:     0/0
`
	_, err := getCheckPointTimestamp(context.Background(), bytes.NewBufferString(controlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint timestamp not found")
}

func TestGetCheckPointTimestamp_emptyReader(t *testing.T) {
	_, err := getCheckPointTimestamp(context.Background(), bytes.NewBufferString(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint timestamp not found")
}

func TestGetCheckPointTimestamp_cancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	controlData := `
pg_control version number:            1201
Latest checkpoint location:           67E/8966A4A0
Time of latest checkpoint:            Fri 12 Feb 2021 01:15:09 PM MSK
`
	_, err := getCheckPointTimestamp(ctx, bytes.NewBufferString(controlData))
	require.Error(t, err)
}

func TestGetCheckPointTimestamp_invalidTimestamp(t *testing.T) {
	controlData := `Time of latest checkpoint:            not-a-valid-date`

	_, err := getCheckPointTimestamp(context.Background(), bytes.NewBufferString(controlData))
	require.Error(t, err)
}

func TestGetCheckPointTimestamp_utcConversion(t *testing.T) {
	controlData := `Time of latest checkpoint:            Thu 01 Jul 2021 12:00:00 PM UTC`

	dsa, err := getCheckPointTimestamp(context.Background(), bytes.NewBufferString(controlData))
	require.NoError(t, err)
	assert.Equal(t, "20210701120000", dsa)
}

func TestSkipSnapshotErr(t *testing.T) {
	msg := "snapshot already contains latest data"
	err := newSkipSnapshotErr(msg)

	assert.Equal(t, msg, err.Error())
	assert.Implements(t, (*error)(nil), err)
}

func TestSkipSnapshotErr_emptyMessage(t *testing.T) {
	err := newSkipSnapshotErr("")
	assert.Equal(t, "", err.Error())
}

func TestPhysicalInitial_UpdateDataStateAtWritesPoolServedAfterReload(t *testing.T) {
	startupPool := &resources.Pool{Name: "dblab_pool", Mode: zfs.PoolMode, DataSubDir: "data"}
	reloadedPool := &resources.Pool{Name: "dblab_pool", Mode: zfs.PoolMode, DataSubDir: "data", DSA: time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC)}

	fsm := zfs.NewFSManager(nil, zfs.Config{Pool: startupPool})
	job := &PhysicalInitial{fsPool: startupPool, cloneManager: fsm, dbMark: &dbmarker.Config{DataStateAt: "20261027212755"}}

	_, err := pool.BuildFromExistingManager(fsm, pool.ManagerConfig{Pool: reloadedPool})
	require.NoError(t, err)

	job.updateDataStateAt()

	assert.Equal(t, time.Date(2026, 10, 27, 21, 27, 55, 0, time.UTC), fsm.Pool().DSA)
}
