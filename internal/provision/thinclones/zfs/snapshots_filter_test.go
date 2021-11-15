/*
2021 © Postgres.ai
*/

package zfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCommand(t *testing.T) {
	testCases := []struct {
		filter          snapshotFilter
		expectedCommand string
	}{
		{
			filter: snapshotFilter{
				fields:  []string{"name", "creation"},
				sorting: []string{"-S creation"},
				dsType:  "snapshot",
				pool:    "test_pool",
			},
			expectedCommand: "zfs list -po name,creation -S creation -t snapshot -r test_pool",
		},
		{
			filter: snapshotFilter{
				fields:  []string{"name", "creation", "dblab:datastateat"},
				sorting: []string{"-S creation", "-S dblab:datastateat"},
				dsType:  "filesystem",
			},
			expectedCommand: "zfs list -po name,creation,dblab:datastateat -S creation -S dblab:datastateat -t filesystem",
		},
	}

	for _, tc := range testCases {
		listCmd := buildListCommand(tc.filter)
		assert.Equal(t, tc.expectedCommand, listCmd)
	}
}
