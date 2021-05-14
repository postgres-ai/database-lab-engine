package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/config"
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
