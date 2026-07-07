/*
2026 © Postgres.ai
*/

package probe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTuningParamNames_NoDuplicates(t *testing.T) {
	seen := make(map[string]struct{}, len(tuningParamNames))

	for _, name := range tuningParamNames {
		_, dup := seen[name]
		require.Falsef(t, dup, "duplicate tuning param name in whitelist: %q", name)
		seen[name] = struct{}{}
	}
}

func TestTuningParamNames_AlphabeticalOrder(t *testing.T) {
	// keeping the list alphabetical makes diffs and merge conflicts boring.
	sorted := make([]string, len(tuningParamNames))
	copy(sorted, tuningParamNames)
	sort.Strings(sorted)

	require.Equal(t, sorted, tuningParamNames, "tuningParamNames must stay alphabetically sorted")
}

func TestTuningQuery_UsesAnyOnNames(t *testing.T) {
	// guard against accidental regression to the prior regex-based query in tools/db/pg.go;
	// the simplified-install plan calls for an explicit whitelist via name = any($1).
	require.Contains(t, tuningQuery, "name = any($1)")
	require.NotContains(t, tuningQuery, "~")
}
