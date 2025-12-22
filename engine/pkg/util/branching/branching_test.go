package branching

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBranch(t *testing.T) {
	assert.Equal(t, "main", DefaultBranch, "default branch must be main")
}

func TestParsingBranchNameFromSnapshot(t *testing.T) {
	const poolName = "pool/pg17"

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "pool/pg17@snapshot_20250407101616",
			expected: "",
		},
		{
			input:    "pool/pg17/branch/dev@20250407101828",
			expected: "dev",
		},
		{
			input:    "pool/pg17/branch/main/cvpqe8gn9i6s73b49e3g/r0@20250407102140",
			expected: "main",
		},
	}

	for _, tc := range testCases {
		branchName := ParseBranchNameFromSnapshot(tc.input, poolName)

		assert.Equal(t, tc.expected, branchName)
	}
}
