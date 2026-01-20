# Prometheus Metrics

DBLab Engine exposes Prometheus metrics via the `/metrics` endpoint. These metrics can be used to monitor the health and performance of the DBLab instance.

## Endpoint

```
GET /metrics
```

The endpoint is publicly accessible (no authentication required) and returns metrics in Prometheus text format.

## Available Metrics

### Instance Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_instance_info` | Gauge | `instance_id`, `version`, `edition` | Information about the DBLab instance (always 1) |
| `dblab_instance_uptime_seconds` | Gauge | - | Time in seconds since the DBLab instance started |
| `dblab_instance_status` | Gauge | `status_code` | Status of the DBLab instance (1=active for status) |
| `dblab_retrieval_status` | Gauge | `mode`, `status` | Status of data retrieval (1=active for status) |

### Disk/Pool Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_disk_total_bytes` | Gauge | `pool` | Total disk space in bytes |
| `dblab_disk_free_bytes` | Gauge | `pool` | Free disk space in bytes |
| `dblab_disk_used_bytes` | Gauge | `pool` | Used disk space in bytes |
| `dblab_disk_used_by_snapshots_bytes` | Gauge | `pool` | Disk space used by snapshots in bytes |
| `dblab_disk_used_by_clones_bytes` | Gauge | `pool` | Disk space used by clones in bytes |
| `dblab_disk_data_size_bytes` | Gauge | `pool` | Size of the data directory in bytes |
| `dblab_disk_compress_ratio` | Gauge | `pool` | Compression ratio of the filesystem (ZFS) |
| `dblab_pool_status` | Gauge | `pool`, `mode`, `status` | Status of the pool (1=active for status) |

### Clone Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_clones_total` | Gauge | - | Total number of clones |
| `dblab_clone_info` | Gauge | `clone_id`, `branch`, `snapshot_id`, `pool`, `status`, `protected` | Information about a clone (always 1) |
| `dblab_clone_age_seconds` | Gauge | `clone_id` | Age of the clone in seconds since creation |
| `dblab_clone_max_age_seconds` | Gauge | - | Maximum age of any clone in seconds |
| `dblab_clone_diff_size_bytes` | Gauge | `clone_id` | Extra disk space used by the clone (diff from snapshot) |
| `dblab_clone_logical_size_bytes` | Gauge | `clone_id` | Logical size of the clone data |
| `dblab_clone_cpu_usage_percent` | Gauge | `clone_id` | CPU usage percentage of the clone container |
| `dblab_clone_memory_usage_bytes` | Gauge | `clone_id` | Memory usage in bytes of the clone container |
| `dblab_clone_memory_limit_bytes` | Gauge | `clone_id` | Memory limit in bytes of the clone container |

### Snapshot Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_snapshots_total` | Gauge | - | Total number of snapshots |
| `dblab_snapshot_info` | Gauge | `snapshot_id`, `pool`, `branch` | Information about a snapshot (always 1) |
| `dblab_snapshot_age_seconds` | Gauge | `snapshot_id` | Age of the snapshot in seconds since creation |
| `dblab_snapshot_max_age_seconds` | Gauge | - | Maximum age of any snapshot in seconds |
| `dblab_snapshot_physical_size_bytes` | Gauge | `snapshot_id` | Physical disk space used by the snapshot |
| `dblab_snapshot_logical_size_bytes` | Gauge | `snapshot_id` | Logical size of the snapshot data |
| `dblab_snapshot_data_lag_seconds` | Gauge | `snapshot_id` | Time difference between snapshot data state and now |
| `dblab_snapshot_max_data_lag_seconds` | Gauge | - | Maximum data lag of any snapshot in seconds |
| `dblab_snapshot_num_clones` | Gauge | `snapshot_id` | Number of clones using this snapshot |

### Branch Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_branches_total` | Gauge | - | Total number of branches |
| `dblab_branch_info` | Gauge | `branch_name`, `pool`, `snapshot_id` | Information about a branch (always 1) |

### Dataset Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_datasets_total` | Gauge | `pool` | Total number of datasets (slots) in the pool |
| `dblab_datasets_available` | Gauge | `pool` | Number of available (non-busy) dataset slots for reuse |
| `dblab_dataset_info` | Gauge | `pool`, `dataset_name` | Information about a dataset (1=busy, 0=available) |

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'dblab'
    static_configs:
      - targets: ['<dblab-host>:<dblab-port>']
    metrics_path: /metrics
```

## Example Queries

### Free Disk Space Percentage

```promql
100 * dblab_disk_free_bytes / dblab_disk_total_bytes
```

### Number of Active Clones

```promql
dblab_clones_total
```

### Maximum Clone Age in Hours

```promql
dblab_clone_max_age_seconds / 3600
```

### Data Freshness (lag from current time)

```promql
dblab_snapshot_max_data_lag_seconds / 60
```

### Memory Usage per Clone

```promql
dblab_clone_memory_usage_bytes{clone_id="my-clone"}
```

### CPU Usage per Clone

```promql
dblab_clone_cpu_usage_percent{clone_id="my-clone"}
```

## Alerting Examples

### Low Disk Space Alert

```yaml
- alert: DBLabLowDiskSpace
  expr: (dblab_disk_free_bytes / dblab_disk_total_bytes) * 100 < 20
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "DBLab low disk space"
    description: "DBLab pool {{ $labels.pool }} has less than 20% free disk space"
```

### Stale Snapshot Alert

```yaml
- alert: DBLabStaleSnapshot
  expr: dblab_snapshot_max_data_lag_seconds > 86400
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "DBLab snapshot data is stale"
    description: "DBLab snapshot data is more than 24 hours old"
```

### High Clone Count Alert

```yaml
- alert: DBLabHighCloneCount
  expr: dblab_clones_total > 50
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "DBLab has many clones"
    description: "DBLab has {{ $value }} clones running"
```
