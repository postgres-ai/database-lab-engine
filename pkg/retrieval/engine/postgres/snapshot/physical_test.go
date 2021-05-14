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
