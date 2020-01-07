/*
2019 © Postgres.ai
*/

package util

import (
	"testing"
	"time"
)

func TestSecondsAgo(t *testing.T) {
	ts1 := time.Date(2010, time.December, 10, 10, 10, 10, 0, time.UTC)
	if SecondsAgo(ts1) == 0 {
		t.FailNow()
	}

	ts2 := time.Date(292277020000, time.January, 1, 1, 0, 0, 0, time.UTC)
	if SecondsAgo(ts2) != 0 {
		t.FailNow()
	}
}

func TestDurationToString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "10µs",
			expected: "0.010 ms",
		},
		{
			input:    "1m",
			expected: "1.000 min",
		},
		{
			input:    "1h25m10s",
			expected: "85.167 min",
		},
	}

	for i, test := range tests {
		d, err := time.ParseDuration(test.input)
		if err != nil {
			t.Errorf("Incorrect input in %d test case.", i)
			t.FailNow()
		}

		actual := DurationToString(d)
		if actual != test.expected {
			t.Errorf("(%d) got different result than expected: \n%s\n",
				i, diff(test.expected, actual))
			t.FailNow()
		}
	}
}

func TestMillisecondsToString(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{
			input:    100.0,
			expected: "100.000 ms",
		},
		{
			input:    1000.0,
			expected: "1.000 s",
		},
		{
			input:    10000.0,
			expected: "10.000 s",
		},
	}

	for i, test := range tests {
		actual := MillisecondsToString(test.input)
		if actual != test.expected {
			t.Errorf("(%d) got different result than expected: \n%s\n",
				i, diff(test.expected, actual))
			t.FailNow()
		}
	}
}

func TestFormatTime(t *testing.T) {
	ts1 := time.Date(2019, time.December, 10, 23, 0, 10, 0, time.UTC)
	actual := FormatTime(ts1)
	expected := "2019-12-10 23:00:10 UTC"
	if actual != expected {
		t.Errorf("Got different result than expected: \n%s\n", diff(expected, actual))
		t.FailNow()
	}
}
