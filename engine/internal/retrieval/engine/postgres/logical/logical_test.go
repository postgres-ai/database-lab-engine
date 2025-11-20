package logical

import (
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/assert"
)

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
	}

	for _, tc := range testCases {
		assert.Equal(t, isAlreadyMounted(tc.source, tc.dumpLocation), tc.expectedResult)
	}
}
