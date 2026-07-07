/*
2026 © Postgres.ai
*/

package probe

import (
	"sort"
	"strings"
)

// requiredPreloadLib is injected by ResolvePreloadLibs whenever the source
// libs list omits it. DBLab clones are commonly used for query analysis
// (pg_stat_statements is the de-facto requirement for that workflow), so
// adding it silently is the more useful default than passing the source
// libs through unchanged.
const requiredPreloadLib = "pg_stat_statements"

// ResolvePreloadLibs deduplicates and alphabetically sorts source-side
// shared_preload_libraries, ensures pg_stat_statements is present, and
// returns the result as a comma-joined string suitable for writing into
// a Postgres configuration entry.
//
// Empty input names (after trimming) are dropped. The function performs no
// per-provider filtering — if the chosen DBLab Engine image does not bundle a library
// in the result, Postgres will fail to start in the clone container with
// "could not load library". The Simple-mode preview surfaces this list so
// users can spot mismatches before applying.
func ResolvePreloadLibs(sourceLibs []string) string {
	seen := make(map[string]struct{}, len(sourceLibs)+1)
	out := make([]string, 0, len(sourceLibs)+1)

	for _, lib := range sourceLibs {
		name := strings.TrimSpace(lib)
		if name == "" {
			continue
		}

		if _, dup := seen[name]; dup {
			continue
		}

		seen[name] = struct{}{}

		out = append(out, name)
	}

	if _, ok := seen[requiredPreloadLib]; !ok {
		out = append(out, requiredPreloadLib)
	}

	sort.Strings(out)

	return strings.Join(out, ",")
}
