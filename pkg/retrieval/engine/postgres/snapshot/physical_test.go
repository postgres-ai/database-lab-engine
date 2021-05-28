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

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
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

func TestParsingWalLine(t *testing.T) {
	log.DEBUG = false

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
	}

	for _, tc := range testCases {
		dsa := parseWALLine(tc.line)
		assert.EqualValues(t, tc.expectedDataStateAt, dsa)
	}
}
