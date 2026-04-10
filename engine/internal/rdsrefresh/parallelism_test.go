/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInstanceSize(t *testing.T) {
	testCases := []struct {
		instanceClass string
		expectedSize  string
		expectErr     bool
	}{
		{instanceClass: "db.m5.xlarge", expectedSize: "xlarge"},
		{instanceClass: "db.t3.medium", expectedSize: "medium"},
		{instanceClass: "db.r6g.2xlarge", expectedSize: "2xlarge"},
		{instanceClass: "db.m5.metal", expectedSize: "metal"},
		{instanceClass: "db.t3.micro", expectedSize: "micro"},
		{instanceClass: "db.r6g.16xlarge", expectedSize: "16xlarge"},
		{instanceClass: "m5.xlarge", expectErr: true},
		{instanceClass: "db.m5", expectErr: true},
		{instanceClass: "db.", expectErr: true},
		{instanceClass: "", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.instanceClass, func(t *testing.T) {
			size, err := extractInstanceSize(tc.instanceClass)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedSize, size)
		})
	}
}

func TestResolveRDSInstanceVCPUs(t *testing.T) {
	testCases := []struct {
		instanceClass string
		expectedVCPUs int
		expectErr     bool
	}{
		{instanceClass: "db.t3.micro", expectedVCPUs: 1},
		{instanceClass: "db.t3.small", expectedVCPUs: 1},
		{instanceClass: "db.t3.medium", expectedVCPUs: 2},
		{instanceClass: "db.m5.large", expectedVCPUs: 2},
		{instanceClass: "db.m5.xlarge", expectedVCPUs: 4},
		{instanceClass: "db.r6g.2xlarge", expectedVCPUs: 8},
		{instanceClass: "db.r6g.4xlarge", expectedVCPUs: 16},
		{instanceClass: "db.r6g.8xlarge", expectedVCPUs: 32},
		{instanceClass: "db.r6g.16xlarge", expectedVCPUs: 64},
		{instanceClass: "db.m5.24xlarge", expectedVCPUs: 96},
		{instanceClass: "db.m5.metal", expectedVCPUs: 96},
		{instanceClass: "db.m5.5xlarge", expectedVCPUs: 20},
		{instanceClass: "invalid", expectErr: true},
		{instanceClass: "db.m5", expectErr: true},
		{instanceClass: "db.m5.unknown", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.instanceClass, func(t *testing.T) {
			vcpus, err := resolveRDSInstanceVCPUs(tc.instanceClass)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedVCPUs, vcpus)
		})
	}
}

func TestParseXlargeMultiplier(t *testing.T) {
	testCases := []struct {
		size          string
		expectedVCPUs int
		expectErr     bool
	}{
		{size: "2xlarge", expectedVCPUs: 8},
		{size: "4xlarge", expectedVCPUs: 16},
		{size: "5xlarge", expectedVCPUs: 20},
		{size: "xlarge", expectErr: true},
		{size: "large", expectErr: true},
		{size: "abcxlarge", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.size, func(t *testing.T) {
			vcpus, err := parseXlargeMultiplier(tc.size)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedVCPUs, vcpus)
		})
	}
}

func TestResolveLocalVCPUs(t *testing.T) {
	vcpus := resolveLocalVCPUs()

	assert.Equal(t, runtime.NumCPU(), vcpus)
	assert.GreaterOrEqual(t, vcpus, minParallelJobs)
}

func TestResolveParallelism(t *testing.T) {
	t.Run("resolves both dump and restore jobs", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{InstanceClass: "db.m5.xlarge"}}

		result, err := ResolveParallelism(cfg)

		require.NoError(t, err)
		assert.Equal(t, 4, result.DumpJobs)
		assert.Equal(t, runtime.NumCPU(), result.RestoreJobs)
	})

	t.Run("returns error for invalid instance class", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{InstanceClass: "invalid"}}

		_, err := ResolveParallelism(cfg)

		require.Error(t, err)
	})
}
