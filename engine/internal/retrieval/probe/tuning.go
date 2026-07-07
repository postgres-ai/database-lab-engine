/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// tuningParamNames is the explicit set of pg_settings names the probe
// collects from a source. The list is intentionally narrow — only knobs
// where copying the source value to the clone meaningfully improves query
// plan parity. Settings not in this list (e.g. effective_io_concurrency,
// log_*) are out of scope.
//
// Minimum pg version each GUC requires is noted alongside. Settings missing
// on older versions are simply absent from the result map (no error).
var tuningParamNames = []string{
	"default_statistics_target",        // pg7.4+
	"effective_cache_size",             // pg7.x+
	"jit",                              // pg11+
	"jit_provider",                     // pg11+
	"maintenance_work_mem",             // pg7.x+
	"max_parallel_maintenance_workers", // pg11+
	"max_parallel_workers",             // pg10+
	"max_parallel_workers_per_gather",  // pg9.6+
	"random_page_cost",                 // pg7.4+
	"work_mem",                         // pg7.x+
}

const tuningQuery = `select name, setting from pg_settings where name = any($1)`

// CollectTuningParams queries the source's pg_settings for the listed
// parameter names and returns a map of name → current setting. Parameters
// that do not exist on the source's Postgres version are silently omitted —
// the caller may compare result.Len() against len(tuningParamNames) if it
// needs to surface version-driven gaps.
func CollectTuningParams(ctx context.Context, conn *pgx.Conn) (map[string]string, error) {
	rows, err := conn.Query(ctx, tuningQuery, tuningParamNames)
	if err != nil {
		return nil, fmt.Errorf("query pg_settings: %w", err)
	}

	defer rows.Close()

	result := make(map[string]string, len(tuningParamNames))

	for rows.Next() {
		var name, setting string

		if err := rows.Scan(&name, &setting); err != nil {
			return nil, fmt.Errorf("scan pg_settings row: %w", err)
		}

		result[name] = setting
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pg_settings rows: %w", err)
	}

	return result, nil
}
