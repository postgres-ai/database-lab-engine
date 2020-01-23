package pglog

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPostgresLastActivity(t *testing.T) {
	testCases := []struct {
		stringLog    string
		timeActivity *time.Time
	}{
		{
			stringLog:    "2020-01-10 11:49:14.615 UTC [3100] postgres@test LOG:  duration: 9.893 ms  statement: SELECT 1;",
			timeActivity: pointer.ToTime(time.Date(2020, 1, 10, 11, 49, 14, 615000000, time.UTC)),
		},
		{
			stringLog:    "2020-01-11 13:10:58.503 NOVT [7819] postgres@template1 LOG:  duration: 0.077 ms  statement:",
			timeActivity: nil,
		},
		{
			stringLog:    "2020-01-11 12:10:56.867 UTC [6766] LOG:  database system is ready to accept connections",
			timeActivity: nil,
		},
		{
			stringLog:    "postgres@test LOG:  duration: 9.893 ms  statement: SELECT 1;",
			timeActivity: nil,
		},
	}

	for _, tc := range testCases {
		lastActivity, err := GetPostgresLastActivity(tc.stringLog)
		require.NoError(t, err)
		assert.Equal(t, tc.timeActivity, lastActivity)
	}
}

func TestGetPostgresLastActivityWhenFailedParseTime(t *testing.T) {
	testCases := []struct {
		stringLog   string
		errorString string
	}{
		{
			stringLog:   "2020-01-10 11:49:14 UTC [3100] postgres@test LOG:  duration: 9.893 ms  statement: SELECT 1;",
			errorString: `failed to parse the last activity time: parsing time "2020-01-10 11:49:14" as "2006-01-02 15:04:05.000": cannot parse "" as ".000"`,
		},
	}

	for _, tc := range testCases {
		lastActivity, err := GetPostgresLastActivity(tc.stringLog)
		require.Nil(t, lastActivity)
		assert.EqualError(t, err, tc.errorString)
	}
}
