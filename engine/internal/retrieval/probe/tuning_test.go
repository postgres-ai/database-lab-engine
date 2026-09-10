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

func TestTuningParamNames_ExcludesSeparatelyWrittenKeys(t *testing.T) {
	// shared_buffers and shared_preload_libraries are written by the clients
	// from their own proposal fields, into the same databaseConfigs.configs map
	// the tuning params land in. The CLI and the UI resolve a collision in
	// opposite orders, so adding either name here would make them disagree.
	for _, name := range []string{"shared_buffers", "shared_preload_libraries"} {
		require.NotContainsf(t, tuningParamNames, name,
			"%q is written from a dedicated proposal field; adding it to the whitelist diverges the CLI and UI", name)
	}
}

func TestTuningQuery_UsesAnyOnNames(t *testing.T) {
	// guard against accidental regression to the prior regex-based query in tools/db/pg.go;
	// the simplified-install plan calls for an explicit whitelist via name = any($1).
	require.Contains(t, tuningQuery, "name = any($1)")
	require.NotContains(t, tuningQuery, "~")
}
