package logical

import (
	"testing"

	"github.com/moby/moby/api/types/mount"
	"github.com/stretchr/testify/assert"
)

func TestBuildAnalyzeCommand(t *testing.T) {
	testCases := []struct {
		conn     Connection
		jobs     int
		expected []string
	}{
		{
			conn:     Connection{Username: "postgres", DBName: "testdb"},
			jobs:     1,
			expected: []string{"vacuumdb", "--analyze", "--jobs", "1", "--username", "postgres", "--all"},
		},
		{
			conn:     Connection{Username: "john", DBName: "mydb"},
			jobs:     4,
			expected: []string{"vacuumdb", "--analyze", "--jobs", "4", "--username", "john", "--all"},
		},
		{
			conn:     Connection{Username: "admin"},
			jobs:     8,
			expected: []string{"vacuumdb", "--analyze", "--jobs", "8", "--username", "admin", "--all"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.conn.Username, func(t *testing.T) {
			result := buildAnalyzeCommand(tc.conn, tc.jobs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsAlreadyMounted(t *testing.T) {
	testCases := []struct {
		source         []mount.Mount
		dumpLocation   string
		expectedResult bool
	}{
		{
			source:         []mount.Mount{},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: false,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab/pool/dump/"}},
			dumpLocation:   "/var/lib/dblab/pool/dump/",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab/pool/dump"}},
			dumpLocation:   "/var/lib/dblab/pool/dump/",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab/pool/dump/"}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab/pool/dump"}},
			dumpLocation:   "/var/lib/dblab/new_pool/dump",
			expectedResult: false,
		},
		{
			source:         []mount.Mount{{Source: "/host/path/dump", Target: "/var/lib/dblab/pool/dump"}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/host/path/dump", Target: "/var/lib/dblab/pool/dump/"}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab", Target: "/var/lib/dblab/"}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/host/data", Target: "/var/lib/dblab"}},
			dumpLocation:   "/var/lib/dblab/pool/dump/",
			expectedResult: true,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab", Target: "/var/lib/dblab"}},
			dumpLocation:   "/var/lib/dblab_dumps/pool",
			expectedResult: false,
		},
		{
			source:         []mount.Mount{{Source: "/var/lib/dblab/pool/dump"}},
			dumpLocation:   "/var/lib/dblab/pool/dump/sub",
			expectedResult: false,
		},
		{
			source:         []mount.Mount{{Source: "/host/data", Target: "/var/lib/dblab", ReadOnly: true}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: false,
		},
		{
			source:         []mount.Mount{{Source: "/host/data", Target: "/var/lib/dblab/pool/dump", ReadOnly: true}},
			dumpLocation:   "/var/lib/dblab/pool/dump",
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, isAlreadyMounted(tc.source, tc.dumpLocation), tc.expectedResult)
	}
}
