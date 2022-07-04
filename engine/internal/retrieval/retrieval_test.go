package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobGroup(t *testing.T) {
	testCases := []struct {
		jobName string
		group   jobGroup
	}{
		{
			jobName: "logicalDump",
			group:   refreshJobs,
		},
		{
			jobName: "logicalRestore",
			group:   refreshJobs,
		},
		{
			jobName: "physicalRestore",
			group:   refreshJobs,
		},
		{
			jobName: "logicalSnapshot",
			group:   snapshotJobs,
		},
		{
			jobName: "physicalSnapshot",
			group:   snapshotJobs,
		},
		{
			jobName: "unknownDump",
			group:   "",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.group, getJobGroup(tc.jobName))
	}
}
