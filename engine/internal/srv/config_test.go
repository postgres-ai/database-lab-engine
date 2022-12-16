package srv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomOptions(t *testing.T) {
	testCases := []struct {
		customOptions  []interface{}
		expectedResult error
	}{
		{
			customOptions:  []interface{}{"--verbose"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--exclude-scheme=test_scheme"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{`--exclude-scheme="test_scheme"`},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--table=$(echo 'test')"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{"--table=test&table"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{5},
			expectedResult: errInvalidOptionType,
		},
	}

	for _, tc := range testCases {
		validationResult := validateCustomOptions(tc.customOptions)

		require.ErrorIs(t, validationResult, tc.expectedResult)
	}
}
