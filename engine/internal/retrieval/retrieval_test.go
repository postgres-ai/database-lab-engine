package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
)

func TestParallelJobSpecs(t *testing.T) {
	testCases := []map[string]config.JobSpec{
		{
			"logicalRestore": {},
		},
		{
			"physicalRestore": {},
		},
		{
			"logicalDump": {},
		},
		{
			"logicalDump":    {},
			"logicalRestore": {},
		},
	}

	for _, tc := range testCases {
		r := Retrieval{
			jobSpecs: tc,
		}

		err := r.validate()
		assert.Nil(t, err)
	}

}

func TestInvalidParallelJobSpecs(t *testing.T) {
	testCases := []map[string]config.JobSpec{
		{
			"physicalRestore": {},
			"logicalRestore":  {},
		},
	}

	for _, tc := range testCases {
		r := Retrieval{
			jobSpecs: tc,
		}

		err := r.validate()
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
		r := Retrieval{
			jobSpecs: tc.spec,
		}

		hasPhysicalJob := r.hasPhysicalJob()
		assert.Equal(t, tc.hasPhysical, hasPhysicalJob)
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
		r := Retrieval{
			jobSpecs: tc.spec,
		}

		hasLogicalJob := r.hasLogicalJob()
		assert.Equal(t, tc.hasLogical, hasLogicalJob)
	}
}
