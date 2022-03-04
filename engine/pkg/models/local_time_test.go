package models

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalTimeMarshalling(t *testing.T) {
	err := os.Setenv("TZ", "UTC")
	require.NoError(t, err)

	testCases := []struct {
		inputTime   time.Time
		marshalling string
	}{
		{
			inputTime:   time.Date(2006, 02, 1, 15, 04, 05, 0, time.UTC),
			marshalling: `"2006-02-01 15:04:05 +00:00"`,
		},
		{
			inputTime:   time.Time{},
			marshalling: `""`,
		},
	}

	for _, tc := range testCases {
		localTime := NewLocalTime(tc.inputTime)
		require.IsType(t, &LocalTime{}, localTime)

		marshalJSON, err := localTime.MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, tc.marshalling, string(marshalJSON))
	}
}

func TestLocalTimeUnMarshalling(t *testing.T) {
	err := os.Setenv("TZ", "UTC")
	require.NoError(t, err)

	testCases := []struct {
		unmarshalling []byte
		expectedTime  time.Time
	}{
		{
			unmarshalling: []byte(`"2006-02-01 15:04:05 +00:00"`),
			expectedTime:  time.Date(2006, 02, 1, 15, 04, 05, 0, time.Local),
		},
		{
			unmarshalling: []byte(`""`),
			expectedTime:  time.Time{},
		},
	}

	for _, tc := range testCases {
		localTime := NewLocalTime(time.Time{})
		require.IsType(t, &LocalTime{}, localTime)
		require.Equal(t, localTime.Time, time.Time{})

		err := localTime.UnmarshalJSON(tc.unmarshalling)
		require.NoError(t, err)
		require.Equal(t, localTime.Time, tc.expectedTime)
	}
}
