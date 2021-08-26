/*
2021 © Postgres.ai
*/

package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/retrieval/engine/postgres/tools"
)

const (
	pgStatStatementsType   = "pg_stat_statements"
	pgStatUserTablesType   = "pg_stat_user_tables"
	pgStatDatabaseType     = "pg_stat_database"
	pgStatBGWriterType     = "pg_stat_bgwriter"
	pgStatKCache           = "pg_stat_kcache"
	pgStatAllIndexesType   = "pg_stat_all_indexes"
	pgStatAllTablesType    = "pg_stat_all_tables"
	pgStatIOIndexesType    = "pg_statio_all_indexes"
	pgStatIOTablesType     = "pg_statio_all_tables"
	pgStatIOSequencesType  = "pg_statio_all_sequences"
	pgStatUserFunctionType = "pg_stat_user_functions"
	pgStatSLRUType         = "pg_stat_slru"
	objectsSizeType        = "objects_size"
	logErrorsType          = "log_errors"
	artifactsSubDir        = "artifacts"
	summaryFilename        = "summary.json"

	// defaultArtifactFormat defines default format of collected artifacts.
	defaultArtifactFormat = "json"
)

var availableArtifactTypes = map[string]struct{}{
	pgStatStatementsType:   {},
	pgStatUserTablesType:   {},
	pgStatDatabaseType:     {},
	pgStatBGWriterType:     {},
	pgStatKCache:           {},
	pgStatAllIndexesType:   {},
	pgStatAllTablesType:    {},
	pgStatIOIndexesType:    {},
	pgStatIOTablesType:     {},
	pgStatIOSequencesType:  {},
	pgStatUserFunctionType: {},
	pgStatSLRUType:         {},
	objectsSizeType:        {},
	logErrorsType:          {},
}

func (c *ObservingClone) storeSummary() error {
	log.Dbg("Store observation summary for SessionID: ", c.session.SessionID)

	if c.session.Result == nil {
		return errors.New("session result is empty")
	}

	summary := SummaryArtifact{
		SessionID: c.session.SessionID,
		CloneID:   c.cloneID,
		Duration: Duration{
			Total:            (time.Duration(c.session.Result.Summary.TotalDuration) * time.Second).String(),
			StartedAt:        c.session.StartedAt,
			FinishedAt:       c.session.FinishedAt,
			MaxQueryDuration: (time.Duration(c.session.state.MaxDBQueryTimeMS) * time.Millisecond).String(),
		},
		DBSize: DBSize{
			Total:       humanize.BigIBytes(big.NewInt(c.session.state.CurrentDBSize)),
			Diff:        humanize.BigIBytes(big.NewInt(c.session.state.CurrentDBSize - c.session.state.InitialDBSize)),
			ObjectsStat: c.session.state.ObjectStat,
		},
		Locks: Locks{
			TotalInterval:   int(c.session.Result.Summary.TotalIntervals),
			WarningInterval: int(c.session.Result.Summary.WarningIntervals),
		},
		LogErrors:     c.session.state.LogErrors,
		ArtifactTypes: c.session.Artifacts,
	}

	summaryData, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	return c.storeFileStats(summaryData, summaryFilename)
}

// getDBSize gets the size of database.
func (c *ObservingClone) getDBSize(ctx context.Context, dbSize *int64) error {
	row := c.superUserDB.QueryRow(ctx, "select pg_database_size(current_database())")

	return row.Scan(dbSize)
}

// getMaxQueryTime gets maximum query duration.
func (c *ObservingClone) getMaxQueryTime(ctx context.Context, maxTime *float64) error {
	row := c.superUserDB.QueryRow(ctx, "select max(max_time) from pg_stat_statements")

	return row.Scan(&maxTime)
}

// countLogErrors counts log errors.
func (c *ObservingClone) countLogErrors(ctx context.Context, logErrors *LogErrors) error {
	row := c.superUserDB.QueryRow(ctx, `select coalesce(sum(count), 0), coalesce(string_agg(distinct message, ','), '') 
	from pg_log_errors_stats() 
	where type in ('ERROR', 'FATAL') and database = current_database()`)

	return row.Scan(&logErrors.Count, &logErrors.Message)
}

// dumpStatementsStats dumps stats statements.
func (c *ObservingClone) dumpStatementsStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatStatementsType)
	return c.dumpDBStats(ctx, "select * from pg_stat_statements(true)", BuildArtifactFilename(pgStatStatementsType))
}

const relationsQuery = `
select
	(extract(epoch from now()) * 1e9)::int8 as epoch_ns,
	quote_ident(schemaname) as tag_schema,
	quote_ident(relname) as tag_table_name,
	quote_ident(schemaname)||'.'||quote_ident(relname) as tag_table_full_name,
	pg_relation_size(relid) as table_size_b,
	pg_total_relation_size(relid) as total_relation_size_b,
	pg_relation_size((select reltoastrelid from pg_class where oid = ut.relid)) as toast_size_b,
	extract(epoch from now() - greatest(last_vacuum, last_autovacuum)) as seconds_since_last_vacuum,
	extract(epoch from now() - greatest(last_analyze, last_autoanalyze)) as seconds_since_last_analyze,
	seq_scan,
	seq_tup_read,
	idx_scan,
	idx_tup_fetch,
	n_tup_ins,
	n_tup_upd,
	n_tup_del,
	n_tup_hot_upd,
	n_dead_tup,
	vacuum_count,
	autovacuum_count,
	analyze_count,
	autoanalyze_count
from
pg_stat_user_tables ut
where
-- leaving out fully locked tables as pg_relation_size also wants a lock and would wait
not exists (select 1 from pg_locks where relation = relid and mode = 'AccessExclusiveLock' and granted)`

// dumpRelationsSize dumps statistics of database relations.
func (c *ObservingClone) dumpRelationsSize(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatUserTablesType)
	return c.dumpDBStats(ctx, relationsQuery, BuildArtifactFilename(pgStatUserTablesType))
}

// dumpDatabaseStats stores database-wide statistics.
func (c *ObservingClone) dumpDatabaseStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatDatabaseType)
	return c.dumpDBStats(ctx, "select * from pg_stat_database where datname = current_database()", BuildArtifactFilename(pgStatDatabaseType))
}

// dumpDatabaseStats dumps log errors stats.
func (c *ObservingClone) dumpDatabaseErrors(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, logErrorsType)
	return c.dumpDBStats(ctx, "select * from pg_log_errors_stats()", BuildArtifactFilename(logErrorsType))
}

// dumpBGWriterStats stores statistics about the background writer process's activity.
func (c *ObservingClone) dumpBGWriterStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatBGWriterType)
	return c.dumpDBStats(ctx, "select * from pg_stat_bgwriter", BuildArtifactFilename(pgStatBGWriterType))
}

// dumpKCacheStats stores statistics about real reads and writes done by the filesystem layer.
func (c *ObservingClone) dumpKCacheStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatKCache)
	return c.dumpDBStats(ctx, "select * from pg_stat_kcache", BuildArtifactFilename(pgStatKCache))
}

// dumpIndexStats stores statistics about accesses to specific index.
func (c *ObservingClone) dumpIndexStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatAllIndexesType)
	return c.dumpDBStats(ctx, "select * from pg_stat_all_indexes where idx_scan > 0", BuildArtifactFilename(pgStatAllIndexesType))
}

// dumpAllTablesStats stores statistics about accesses to specific table.
func (c *ObservingClone) dumpAllTablesStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatAllTablesType)

	return c.dumpDBStats(ctx, `select * from pg_stat_all_tables 
where seq_scan > 0 or idx_scan > 0 
or n_tup_ins > 0 or n_tup_upd > 0 or n_tup_del > 0 
or vacuum_count > 0 or analyze_count > 0`, BuildArtifactFilename(pgStatAllTablesType))
}

// dumpIOIndexesStats stores statistics about I/O on specific index.
func (c *ObservingClone) dumpIOIndexesStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatIOIndexesType)

	return c.dumpDBStats(ctx, `select * from pg_statio_all_indexes where idx_blks_read > 0 or idx_blks_hit > 0`,
		BuildArtifactFilename(pgStatIOIndexesType))
}

// dumpIOTablesStats stores statistics about I/O on specific table.
func (c *ObservingClone) dumpIOTablesStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatIOTablesType)

	return c.dumpDBStats(ctx, `select * from pg_statio_all_tables 
where heap_blks_read > 0 or heap_blks_hit > 0 
or idx_blks_read > 0 or idx_blks_hit > 0 
or toast_blks_read > 0 or toast_blks_hit > 0 
or tidx_blks_read > 0 or tidx_blks_hit > 0`, BuildArtifactFilename(pgStatIOTablesType))
}

// dumpIOSequencesStats stores statistics about I/O on specific sequence.
func (c *ObservingClone) dumpIOSequencesStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatIOSequencesType)

	return c.dumpDBStats(ctx, `select * from pg_statio_all_sequences 
where blks_read > 0 or blks_hit > 0`, BuildArtifactFilename(pgStatIOSequencesType))
}

// dumpUserFunctionsStats stores statistics about executions of each function.
func (c *ObservingClone) dumpUserFunctionsStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatUserFunctionType)
	return c.dumpDBStats(ctx, `select * from pg_stat_user_functions where calls > 0`, BuildArtifactFilename(pgStatUserFunctionType))
}

// TODO: uncomment for Postgres 13+
// dumpSLRUStats stores one row for each tracked SLRU cache, showing statistics about access to cached pages.
// func (c *ObservingClone) dumpSLRUStats(ctx context.Context) error {
//   c.session.Artifacts = append(c.session.Artifacts, pgStatSLRUType)
//   return c.dumpDBStats(ctx, `select * from pg_stat_slru`, BuildArtifactFilename(pgStatSLRUType))
// }

// dumpObjectsSize dumps objects size.
func (c *ObservingClone) dumpObjectsSize(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, objectsSizeType)
	return c.dumpDBStats(ctx, fmt.Sprintf(objectsSizeDump, objSizeContent), BuildArtifactFilename(objectsSizeType))
}

//nolint:lll
const (
	objSizeContent = `
with data as (
  select
    c.oid,
    (select spcname from pg_tablespace where oid = reltablespace) as tblspace,
    nspname as schema_name,
    relname as table_name,
    c.reltuples as row_estimate,
    pg_total_relation_size(c.oid) as total_bytes,
    pg_indexes_size(c.oid) as index_bytes,
    pg_total_relation_size(reltoastrelid) as toast_bytes,
    pg_total_relation_size(c.oid) - pg_indexes_size(c.oid) - coalesce(pg_total_relation_size(reltoastrelid), 0) as table_bytes
  from pg_class c
       left join pg_namespace n on n.oid = c.relnamespace
  where relkind = 'r' and nspname <> 'information_schema'
  order by c.relpages desc
), tables as (
  select
        coalesce(nullif(schema_name, 'public') || '.', '') || table_name || coalesce(' [' || tblspace || ']', '') as "table",
        row_estimate as row_estimate,
        total_bytes as total_size_bytes,
        round(
              100 * total_bytes::numeric / nullif(sum(total_bytes) over (partition by (schema_name is null), left(table_name, 3) = '***'), 0),
              2
          ) as "total_size_percent",
        table_bytes as table_size_bytes,
        round(
              100 * table_bytes::numeric / nullif(sum(table_bytes) over (partition by (schema_name is null), left(table_name, 3) = '***'), 0),
              2
          ) as "table_size_percent",
        index_bytes as indexes_size_bytes,
        round(
              100 * index_bytes::numeric / nullif(sum(index_bytes) over (partition by (schema_name is null), left(table_name, 3) = '***'), 0),
              2
          ) as "index_size_percent",
        toast_bytes as toast_size_bytes,
        round(
              100 * toast_bytes::numeric / nullif(sum(toast_bytes) over (partition by (schema_name is null), left(table_name, 3) = '***'), 0),
              2
          ) as "toast_size_percent"
  from data
  order by oid is null desc, total_bytes desc nulls last
)`

	objectsSizeDump = `%s select * from tables`

	objectsSizeSummary = `
%s, total_data as (
  select
    sum(1) as count,
    sum("row_estimate") as "row_estimate_sum",
    sum("total_size_bytes") as "total_size_bytes_sum",
    sum("table_size_bytes") as "table_size_bytes_sum",
    sum("indexes_size_bytes") as "indexes_size_bytes_sum",
    sum("toast_size_bytes") as "toast_size_bytes_sum"
  from tables
)
select * from total_data`
)

// getObjectsSizeStats gets statistics of objects size.
func (c *ObservingClone) getObjectsSizeStats(ctx context.Context, stat *ObjectsStat) error {
	return c.db.QueryRow(ctx, fmt.Sprintf(objectsSizeSummary, objSizeContent)).Scan(
		&stat.Count,
		&stat.RowEstimateSum,
		&stat.TotalSizeBytesSum,
		&stat.TableSizeBytesSum,
		&stat.IndexesSizeBytesSum,
		&stat.ToastSizeBytesSum)
}

// dumpDBStats stores collected statistics.
func (c *ObservingClone) dumpDBStats(ctx context.Context, query, filename string) error {
	dstFilePath := path.Join(c.currentArtifactsSessionPath(), artifactsSubDir, filename)

	if err := initStatFile(dstFilePath); err != nil {
		return errors.Wrapf(err, "cannot init the stat file %s", dstFilePath)
	}

	// Backslash characters (\) can be used in the COPY data to quote data characters
	// that might otherwise be taken as row or column delimiters.
	// In particular, the following characters must be preceded by a backslash if they appear as part of a column value:
	// backslash itself, newline, carriage return, and the current delimiter character.
	// https://www.postgresql.org/docs/current/sql-copy.html
	//
	// It will lead to producing an invalid JSON with double escaped quotes.
	// To work around this issue, we can pipe the output of the COPY command to `sed` to revert the double escaping of quotes.
	exportQuery := fmt.Sprintf(
		`COPY (select coalesce(json_agg(row_to_json(t)), '[]'::json) from (%s) as t) TO PROGRAM $$sed 's/\\\\/\\/g' > %s$$`,
		query, dstFilePath)
	if _, err := c.db.Exec(ctx, exportQuery); err != nil {
		return errors.Wrap(err, "failed to export data")
	}

	return nil
}

func (c *ObservingClone) storeFileStats(data []byte, filename string) error {
	fullFilename := path.Join(c.currentArtifactsSessionPath(), filename)

	log.Dbg("Dump data into file", fullFilename)

	return os.WriteFile(fullFilename, data, 0644)
}

func (c *ObservingClone) readFileStats(sessionID uint64, filename string) ([]byte, error) {
	fullFilename := path.Join(c.artifactsSessionPath(sessionID), filename)

	return os.ReadFile(fullFilename)
}

// initStatFile touches a new file and adjusts file permissions.
func initStatFile(filename string) error {
	if err := tools.TouchFile(filename); err != nil {
		return err
	}

	if err := os.Chmod(filename, 0666); err != nil {
		return err
	}

	return nil
}

// IsAvailableArtifactType checks if artifact type is available.
func IsAvailableArtifactType(artifactType string) bool {
	_, ok := availableArtifactTypes[artifactType]

	return ok
}

// BuildArtifactPath generates a full path to the artifact file.
func (c *ObservingClone) BuildArtifactPath(sessionID uint64, artifactType string) string {
	fullFilename := path.Join(c.artifactsSessionPath(sessionID), artifactsSubDir, BuildArtifactFilename(artifactType))

	return fullFilename
}

// BuildArtifactFilename builds an artifact filename.
func BuildArtifactFilename(artifactType string) string {
	return fmt.Sprintf("%s.%s", artifactType, defaultArtifactFormat)
}
