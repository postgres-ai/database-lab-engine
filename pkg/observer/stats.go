/*
2021 © Postgres.ai
*/

package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	pgStatStatementsType = "pg_stat_statements"
	pgStatUserTablesType = "pg_stat_user_tables"
	pgStatDatabaseType   = "pg_stat_database"
	pgStatBGWriterType   = "pg_stat_bgwriter"
	objectsSizeType      = "objects_size"
	artifactsSubDir      = "artifacts"
	summaryFilename      = "summary.json"

	// defaultArtifactFormat defines default format of collected artifacts.
	defaultArtifactFormat = "csv"
)

var availableArtifactTypes = map[string]struct{}{
	pgStatStatementsType: {},
	pgStatUserTablesType: {},
	pgStatDatabaseType:   {},
	pgStatBGWriterType:   {},
	objectsSizeType:      {},
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
	row := c.db.QueryRow(ctx, "select pg_database_size(current_database())")

	return row.Scan(dbSize)
}

// getMaxQueryTime gets maximum query duration.
func (c *ObservingClone) getMaxQueryTime(ctx context.Context, maxTime *float64) error {
	row := c.db.QueryRow(ctx, "select max(max_time) from pg_stat_statements")

	return row.Scan(&maxTime)
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

// dumpDatabaseStats dumps database stats.
func (c *ObservingClone) dumpDatabaseStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatDatabaseType)
	return c.dumpDBStats(ctx, "select * from pg_stat_database", BuildArtifactFilename(pgStatDatabaseType))
}

// dumpBGWriterStats stores bgWriter stats.
func (c *ObservingClone) dumpBGWriterStats(ctx context.Context) error {
	c.session.Artifacts = append(c.session.Artifacts, pgStatBGWriterType)
	return c.dumpDBStats(ctx, "select * from pg_stat_bgwriter", BuildArtifactFilename(pgStatBGWriterType))
}

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
	filePath := path.Join(c.currentArtifactsSessionPath(), artifactsSubDir, filename)

	log.Dbg("Dump data into file", filePath)

	if err := initStatFile(filePath); err != nil {
		return errors.Wrapf(err, "failed to init the stats file %v", filePath)
	}

	_, err := c.db.Exec(ctx, fmt.Sprintf(`COPY (%s) TO '%s' delimiter ',' csv header`, query, filePath))

	return err
}

func (c *ObservingClone) storeFileStats(data []byte, filename string) error {
	fullFilename := path.Join(c.currentArtifactsSessionPath(), filename)

	log.Dbg("Dump data into file", fullFilename)

	return ioutil.WriteFile(fullFilename, data, 0644)
}

func (c *ObservingClone) readFileStats(sessionID uint64, filename string) ([]byte, error) {
	fullFilename := path.Join(c.artifactsSessionPath(sessionID), filename)

	return ioutil.ReadFile(fullFilename)
}

// initStatFile touches a new file and adjusts file permissions.
func initStatFile(filename string) error {
	if err := tools.TouchFile(filename); err != nil {
		return err
	}

	if err := os.Chmod(filename, 0777); err != nil {
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
