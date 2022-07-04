package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
)

func TestParallelJobSpecs(t *testing.T) {
	testCases := []config.Config{
		{
			Jobs: []string{"logicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"logicalRestore": {},
			},
		},
		{
			Jobs: []string{"physicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"physicalRestore": {},
			},
		},
		{
			Jobs: []string{"logicalDump"},
			JobsSpec: map[string]config.JobSpec{
				"logicalDump": {},
			},
		},
		{
			Jobs: []string{"logicalDump", "logicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"logicalDump":    {},
				"logicalRestore": {},
			},
		},
	}

	for _, tc := range testCases {
		err := validateStructure(&tc)
		assert.Nil(t, err)
	}

}

func TestInvalidParallelJobSpecs(t *testing.T) {
	testCases := []config.Config{
		{
			Jobs: []string{"logicalRestore", "physicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"physicalRestore": {},
				"logicalRestore":  {},
			},
		},
	}

	for _, tc := range testCases {
		err := validateStructure(&tc)
		assert.Error(t, err)
	}
}

func TestPhysicalJobs(t *testing.T) {
	testCases := []struct {
		spec        map[string]config.JobSpec
		hasPhysical bool
	}{
		{
			spec:        map[string]config.JobSpec{"physicalSnapshot": {}},
			hasPhysical: true,
		},
		{
			spec:        map[string]config.JobSpec{"physicalRestore": {}},
			hasPhysical: true,
		},
		{
			spec: map[string]config.JobSpec{
				"physicalSnapshot": {},
				"physicalRestore":  {},
			},
			hasPhysical: true,
		},
		{
			spec:        map[string]config.JobSpec{},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalDump": {}},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalRestore": {}},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalSnapshot": {}},
			hasPhysical: false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.hasPhysical, hasPhysicalJob(tc.spec))
	}
}

func TestLogicalJobs(t *testing.T) {
	testCases := []struct {
		spec       map[string]config.JobSpec
		hasLogical bool
	}{
		{
			spec:       map[string]config.JobSpec{"logicalSnapshot": {}},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{"logicalRestore": {}},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{"logicalDump": {}},
			hasLogical: true,
		},
		{
			spec: map[string]config.JobSpec{
				"logicalDump":     {},
				"logicalRestore":  {},
				"logicalSnapshot": {},
			},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{},
			hasLogical: false,
		},
		{
			spec:       map[string]config.JobSpec{"physicalRestore": {}},
			hasLogical: false,
		},
		{
			spec:       map[string]config.JobSpec{"physicalSnapshot": {}},
			hasLogical: false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.hasLogical, hasLogicalJob(tc.spec))
	}
}
