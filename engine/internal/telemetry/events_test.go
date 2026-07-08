/*
2026 © Postgres.ai
*/

package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventPayloadMarshaling(t *testing.T) {
	testCases := []struct {
		name    string
		payload interface{}
		want    string
	}{
		{name: "snapshot updated", payload: SnapshotUpdated{ID: "pool@snap", Protected: true}, want: `{"id":"pool@snap","protected":true}`},
		{name: "snapshot destroyed", payload: SnapshotDestroyed{ID: "pool@snap"}, want: `{"id":"pool@snap"}`},
		{name: "branch updated", payload: BranchUpdated{Name: "dev", Protected: false}, want: `{"name":"dev","protected":false}`},
		{name: "branch destroyed", payload: BranchDestroyed{Name: "dev"}, want: `{"name":"dev"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := json.Marshal(tc.payload)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(out))
		})
	}
}
