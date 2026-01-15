# DBLab Prometheus Metrics

DBLab Engine exports metrics in Prometheus format via the `/metrics` endpoint. This endpoint does not require authentication.

## Endpoint

```
GET http://<dblab-host>:<port>/metrics
```

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'dblab'
    static_configs:
      - targets: ['<dblab-host>:2345']
    scrape_interval: 30s
```

## Available Metrics

### Engine Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dblab_engine_info` | Gauge | Engine information with labels: version, edition, instance_id |
| `dblab_engine_uptime_seconds` | Gauge | Time since Database Lab Engine started in seconds |

### Retrieval Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dblab_retrieval_mode` | Gauge | Current retrieval mode (1=physical, 2=logical, 0=unknown) |
| `dblab_retrieval_status` | Gauge | Current retrieval status with label: status |
| `dblab_retrieval_last_refresh_timestamp_seconds` | Gauge | Unix timestamp of last data refresh |
| `dblab_retrieval_next_refresh_timestamp_seconds` | Gauge | Unix timestamp of next scheduled data refresh |
| `dblab_retrieval_data_freshness_seconds` | Gauge | Time since last data refresh in seconds |
| `dblab_retrieval_alerts_total` | Gauge | Number of retrieval alerts with labels: type, level |

### Synchronization Metrics (Physical Mode)

| Metric | Type | Description |
|--------|------|-------------|
| `dblab_sync_replication_lag_seconds` | Gauge | Replication lag in seconds |
| `dblab_sync_replication_uptime_seconds` | Gauge | Replication uptime in seconds |

### Pool Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dblab_pool_status` | Gauge | pool, mode | Pool status (1=active, 2=refreshing, 3=empty) |
| `dblab_pool_data_state_at_timestamp_seconds` | Gauge | pool | Unix timestamp of the pool data state |
| `dblab_pool_size_bytes` | Gauge | pool | Total pool size in bytes |
| `dblab_pool_free_bytes` | Gauge | pool | Free space in pool in bytes |
| `dblab_pool_used_bytes` | Gauge | pool | Used space in pool in bytes |
| `dblab_pool_data_size_bytes` | Gauge | pool | Logical data size in bytes |
| `dblab_pool_used_by_snapshots_bytes` | Gauge | pool | Space used by snapshots in bytes |
| `dblab_pool_used_by_clones_bytes` | Gauge | pool | Space used by clones in bytes |
| `dblab_pool_compress_ratio` | Gauge | pool | Compression ratio of the pool |
| `dblab_pool_clones_total` | Gauge | pool | Number of clones in the pool |

### Clone Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dblab_clones_total` | Gauge | - | Total number of clones |
| `dblab_clones_by_status` | Gauge | status | Number of clones by status |
| `dblab_clones_expected_cloning_time_seconds` | Gauge | - | Expected time to create a clone in seconds |
| `dblab_clones_protected_total` | Gauge | - | Number of protected clones |
| `dblab_clone_diff_size_bytes` | Gauge | clone_id, branch | Clone diff size in bytes |
| `dblab_clone_logical_size_bytes` | Gauge | clone_id, branch | Clone logical size in bytes |
| `dblab_clone_cloning_time_seconds` | Gauge | clone_id, branch | Time taken to create clone in seconds |

### Snapshot Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dblab_snapshots_total` | Gauge | pool, branch, type | Total number of snapshots (type: auto/user) |
| `dblab_snapshot_physical_size_bytes` | Gauge | snapshot_id, pool, branch | Snapshot physical size in bytes |
| `dblab_snapshot_logical_size_bytes` | Gauge | snapshot_id, pool, branch | Snapshot logical size in bytes |
| `dblab_snapshot_clone_count` | Gauge | snapshot_id, pool | Number of clones using this snapshot |

### Branch Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dblab_branches_total` | Gauge | Total number of branches in use |

### Resource/Slot Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dblab_slots_busy_total` | Gauge | Number of busy slots preventing full refresh in logical mode |

## Example Grafana Queries

### Monitor disk usage

```promql
100 - (dblab_pool_free_bytes / dblab_pool_size_bytes * 100)
```

### Monitor replication lag (physical mode)

```promql
dblab_sync_replication_lag_seconds
```

### Data freshness (logical mode)

```promql
dblab_retrieval_data_freshness_seconds / 3600
```

### Clone count over time

```promql
dblab_clones_total
```

### Alert on high disk usage

```promql
(1 - dblab_pool_free_bytes / dblab_pool_size_bytes) > 0.85
```

### Alert on replication lag

```promql
dblab_sync_replication_lag_seconds > 300
```

## Sample Output

```
# HELP dblab_engine_info Database Lab Engine information
# TYPE dblab_engine_info gauge
dblab_engine_info{edition="standard",instance_id="my-instance",version="3.5.0"} 1

# HELP dblab_engine_uptime_seconds Time since Database Lab Engine started in seconds
# TYPE dblab_engine_uptime_seconds gauge
dblab_engine_uptime_seconds 86400

# HELP dblab_retrieval_mode Current retrieval mode (1 for physical, 2 for logical, 0 for unknown)
# TYPE dblab_retrieval_mode gauge
dblab_retrieval_mode 1

# HELP dblab_sync_replication_lag_seconds Replication lag in seconds (physical mode)
# TYPE dblab_sync_replication_lag_seconds gauge
dblab_sync_replication_lag_seconds 5

# HELP dblab_pool_size_bytes Total pool size in bytes
# TYPE dblab_pool_size_bytes gauge
dblab_pool_size_bytes{pool="dblab_pool"} 107374182400

# HELP dblab_pool_free_bytes Free space in pool in bytes
# TYPE dblab_pool_free_bytes gauge
dblab_pool_free_bytes{pool="dblab_pool"} 53687091200

# HELP dblab_clones_total Total number of clones
# TYPE dblab_clones_total gauge
dblab_clones_total 3
```
