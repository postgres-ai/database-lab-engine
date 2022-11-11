/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

const dblalbutils = {
  getArtifactDescription(artifactType: string) {
    const descriptions = {
      log: 'database logs',
      pg_stat_statements:
        'tracking planning and execution statistics of all ' +
        'SQL statements executed during migration run',
      pg_stat_kcache:
        'statistics about real reads and writes done by the filesystem layer',
      pg_stat_all_indexes: 'statistics about accesses to specific index',
      pg_stat_all_tables: 'statistics about accesses to specific table',
      pg_stat_database: 'database-wide statistics',
      pg_statio_all_indexes: 'statistics about I/O on specific index',
      pg_statio_all_tables: 'statistics about I/O on specific table',
      pg_statio_all_sequences: 'statistics about I/O on specific sequence',
      pg_stat_bgwriter:
        "statistics about the background writer process's activity",
      pg_stat_user_functions: 'statistics about executions of each function',
      pg_stat_slru:
        'contain one row for each tracked SLRU cache, showing ' +
        'statistics about access to cached pages',
      pg_current_wal_lsn: 'calculate number of bytes WAL written',
      pg_wal_lsn_diff: 'calculate number of bytes WAL written',
      pg_stat_activity:
        'dynamic view, no pont compare start and stop snapshots',
      pg_stat_progress_create_index:
        'Dynamic view, no pont compare start and stop snapshots',
      pg_stat_progress_cluster:
        'dynamic view, no pont compare start and stop snapshots',
      pg_stat_progress_basebackup:
        'dynamic view, no pont compare start and stop snapshots',
      pg_stat_progress_analyze:
        'dynamic view, no pont compare start and stop snapshots',
      pg_stat_user_tables:
        'statistics about accesses to each non-system table ' +
        'in the current database',
    }

    return descriptions[artifactType as keyof typeof descriptions]
  },
}

export default dblalbutils
