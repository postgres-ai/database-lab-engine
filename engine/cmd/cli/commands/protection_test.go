package commands

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestParseDurationMinutes(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		hasError bool
	}{
		{input: "0", expected: 0},
		{input: "30", expected: 30},
		{input: "120", expected: 120},
		{input: "0m", expected: 0},
		{input: "30m", expected: 30},
		{input: "90m", expected: 90},
		{input: "30M", expected: 30},
		{input: "0h", expected: 0},
		{input: "1h", expected: 60},
		{input: "23h", expected: 1380},
		{input: "2H", expected: 120},
		{input: "0d", expected: 0},
		{input: "1d", expected: 1440},
		{input: "7d", expected: 10080},
		{input: "7D", expected: 10080},
		{input: "365d", expected: 525600},
		{input: "abc", hasError: true},
		{input: "10x", hasError: true},
		{input: "m", hasError: true},
		{input: "h", hasError: true},
		{input: "d", hasError: true},
		{input: "-1", hasError: true},
		{input: "1.5h", hasError: true},
		{input: " 30", hasError: true},
		{input: "30 ", hasError: true},
		{input: "366d", hasError: true},
		{input: "8761h", hasError: true},
		{input: "4294967295d", hasError: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDurationMinutes(tt.input)
			if tt.hasError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func newProtectedContext(value string, isSet bool) *cli.Context {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("protected", "", "")

	if isSet {
		_ = fs.Set("protected", value)
	}

	return cli.NewContext(&cli.App{}, fs, nil)
}

func uintPtr(v uint) *uint { return &v }

func boolPtr(v bool) *bool { return &v }

func TestParseProtectedFlag(t *testing.T) {
	tests := []struct {
		name              string
		value             string
		isSet             bool
		expectedProtected *bool
		expectedDuration  *uint
		hasError          bool
	}{
		{name: "not set", value: "", isSet: false, expectedProtected: nil, expectedDuration: nil},
		{name: "empty string", value: "", isSet: true, expectedProtected: boolPtr(true), expectedDuration: nil},
		{name: "true", value: "true", isSet: true, expectedProtected: boolPtr(true), expectedDuration: nil},
		{name: "TRUE", value: "TRUE", isSet: true, expectedProtected: boolPtr(true), expectedDuration: nil},
		{name: "false", value: "false", isSet: true, expectedProtected: boolPtr(false), expectedDuration: nil},
		{name: "FALSE", value: "FALSE", isSet: true, expectedProtected: boolPtr(false), expectedDuration: nil},
		{name: "zero minutes", value: "0", isSet: true, expectedProtected: boolPtr(true), expectedDuration: uintPtr(0)},
		{name: "plain minutes", value: "30", isSet: true, expectedProtected: boolPtr(true), expectedDuration: uintPtr(30)},
		{name: "minutes suffix", value: "30m", isSet: true, expectedProtected: boolPtr(true), expectedDuration: uintPtr(30)},
		{name: "hours suffix", value: "2h", isSet: true, expectedProtected: boolPtr(true), expectedDuration: uintPtr(120)},
		{name: "days suffix", value: "7d", isSet: true, expectedProtected: boolPtr(true), expectedDuration: uintPtr(10080)},
		{name: "invalid value", value: "abc", isSet: true, hasError: true},
		{name: "overflow value", value: "366d", isSet: true, hasError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cliCtx := newProtectedContext(tt.value, tt.isSet)

			protected, duration, err := ParseProtectedFlag(cliCtx)
			if tt.hasError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectedProtected == nil {
				assert.Nil(t, protected)
			} else {
				require.NotNil(t, protected)
				assert.Equal(t, *tt.expectedProtected, *protected)
			}

			if tt.expectedDuration == nil {
				assert.Nil(t, duration)
			} else {
				require.NotNil(t, duration)
				assert.Equal(t, *tt.expectedDuration, *duration)
			}
		})
	}
}
