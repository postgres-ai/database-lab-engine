/*
2020 © Postgres.ai
*/

package observer

import (
	"fmt"
)

const queryLocks = `with lock_data as (
  select
    a.datname,
    l.relation::regclass,
    l.transactionid,
    l.mode,
    l.locktype,
    l.granted,
    a.usename,
    a.query,
    a.query_start,
    a.state,
    a.wait_event_type,
    a.wait_event,
    a.xact_start,
    clock_timestamp() - a.xact_start as xact_duration,
    a.query_start,
    clock_timestamp() - a.query_start as query_duration,
    a.state_change,
    clock_timestamp() - a.state_change as state_changed_ago,
    a.pid
  from pg_stat_activity a
  join pg_locks l on l.pid = a.pid
  where l.mode = 'AccessExclusiveLock' and l.locktype = 'relation' and a.application_name <> '%s'
)
select row_to_json(lock_data)
from lock_data
where query_duration > interval '%d second'
order by query_duration desc;`

func buildLocksMetricQuery(exclusionApplicationName string, maxLockDurationSeconds uint64) string {
	return fmt.Sprintf(queryLocks, exclusionApplicationName, maxLockDurationSeconds)
}
