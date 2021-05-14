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
		logTime      string
		logMessage   string
		timeActivity *time.Time
	}{
		{
			logTime:      "2020-01-10 11:49:14.615 UTC",
			logMessage:   "duration: 9.893 ms  statement: SELECT 1;",
			timeActivity: pointer.ToTime(time.Date(2020, 1, 10, 11, 49, 14, 615000000, time.UTC)),
		},
		{
			logTime:      "2020-01-11 13:10:58.503 UTC",
			logMessage:   "duration: 0.077 ms  statement:",
			timeActivity: pointer.ToTime(time.Date(2020, 1, 11, 13, 10, 58, 503000000, time.UTC)),
		},
		{
			logTime:      "2020-01-11 12:10:56.867 UTC",
			logMessage:   "database system is ready to accept connections",
			timeActivity: nil,
		},
		{
			logTime:      "",
			logMessage:   "duration: 9.893 ms  statement: SELECT 1;",
			timeActivity: nil,
		},
		{
			logTime:      "2021-03-24 15:33:56.135 UTC",
			logMessage:   "duration: 0.544 ms  execute lrupsc_28_0: EXPLAIN (FORMAT TEXT) select 1",
			timeActivity: pointer.ToTime(time.Date(2021, 3, 24, 15, 33, 56, 135000000, time.UTC)),
		},
	}

	for _, tc := range testCases {
		lastActivity, err := ParsePostgresLastActivity(tc.logTime, tc.logMessage)
		require.NoError(t, err)
		assert.Equal(t, tc.timeActivity, lastActivity)
	}
}

func TestGetPostgresLastActivityWhenFailedParseTime(t *testing.T) {
	testCases := []struct {
		logTime     string
		logMessage  string
		errorString string
	}{
		{
			logTime:     "2020-01-10 11:49:14",
			logMessage:  "duration: 9.893 ms  statement: SELECT 1;",
			errorString: `failed to parse the last activity time: parsing time "2020-01-10 11:49:14" as "2006-01-02 15:04:05.000 UTC": cannot parse "" as ".000"`,
		},
	}

	for _, tc := range testCases {
		lastActivity, err := ParsePostgresLastActivity(tc.logTime, tc.logMessage)
		require.Nil(t, lastActivity)
		assert.EqualError(t, err, tc.errorString)
	}
}
