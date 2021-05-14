package observer

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskingField(t *testing.T) {
	testCases := []struct {
		logEntry       []string
		maskIndexes    []int
		expectedResult []string
	}{
		{
			logEntry:       []string{"explain select 5;", "select * from users where email = 'abc@example.com';"},
			maskIndexes:    []int{0},
			expectedResult: []string{"explain xxx;", "select * from users where email = 'abc@example.com';"},
		},
		{
			logEntry:       []string{"explain select 5;", "select * from users where email = 'abc@example.com';"},
			maskIndexes:    []int{1},
			expectedResult: []string{"explain select 5;", "select * from users where email = 'xxx@example.com';"},
		},
	}

	o := Observer{
		replacementRules: []ReplacementRule{
			{
				re:      regexp.MustCompile(`select (\d+)`),
				replace: "xxx",
			},
			{
				re:      regexp.MustCompile(`[a-z0-9._%+\-]+(@[a-z0-9.\-]+\.[a-z]{2,4})`),
				replace: "xxx$1",
			},
		},
	}

	for _, tc := range testCases {
		testLogEntry := make([]string, len(tc.logEntry))
		copy(testLogEntry, tc.logEntry)
		o.maskLogs(testLogEntry, tc.maskIndexes)
		assert.Equal(t, tc.expectedResult, testLogEntry)
	}
}
