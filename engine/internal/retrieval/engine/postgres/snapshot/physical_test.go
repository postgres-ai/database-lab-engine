/*
2021 © Postgres.ai
*/

package snapshot

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	testCases := []struct {
		pgVersion      float64
		cloneDir       string
		expectedWalDir string
	}{
		{
			pgVersion:      9.6,
			cloneDir:       "/tmp",
			expectedWalDir: "/tmp/pg_xlog",
		},
		{
			pgVersion:      10,
			cloneDir:       "/tmp",
			expectedWalDir: "/tmp/pg_wal",
		},
		{
			pgVersion:      11,
			cloneDir:       "/tmp",
			expectedWalDir: "/tmp/pg_wal",
		},
		{
			pgVersion:      12,
			cloneDir:       "/tmp",
			expectedWalDir: "/tmp/pg_wal",
		},
		{
			pgVersion:      13,
			cloneDir:       "/tmp",
			expectedWalDir: "/tmp/pg_wal",
		},
	}

	for _, tc := range testCases {
		resultDir := walDir(tc.cloneDir, tc.pgVersion)
		assert.EqualValues(t, tc.expectedWalDir, resultDir)
	}
}

func TestWalCommand(t *testing.T) {
	log.SetDebug(false)

	testCases := []struct {
		pgVersion          float64
		walName            string
		expectedWalCommand string
	}{
		{
			pgVersion:          9.6,
			walName:            "000000010000000000000002",
			expectedWalCommand: "/usr/lib/postgresql/9.6/bin/pg_xlogdump 000000010000000000000002 -r Transaction | tail -1",
		},
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
