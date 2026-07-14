# Prometheus metrics

DBLab Engine exposes Prometheus metrics via the `/metrics` endpoint. These metrics can be used to monitor the health and performance of the DBLab instance.

## Endpoint

```
GET /metrics
```

The endpoint is publicly accessible (no authentication required) and returns metrics in Prometheus text format.

## Available metrics

### Instance metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_instance_info` | Gauge | `instance_id`, `version`, `edition` | Information about the DBLab instance (always 1) |
| `dblab_instance_uptime_seconds` | Gauge | - | Time in seconds since the DBLab instance started |
| `dblab_instance_status_code` | Gauge | - | Status code of the DBLab instance (0=OK, 1=Warning, 2=Bad) |
| `dblab_retrieval_status` | Gauge | `mode`, `status` | Status of data retrieval (1=active for status) |

### Disk/pool metrics

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

### Clone metrics (aggregate)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_clones_total` | Gauge | - | Total number of clones |
| `dblab_clones_by_status` | Gauge | `status` | Number of clones by status |
| `dblab_clone_max_age_seconds` | Gauge | - | Maximum age of any clone in seconds |
| `dblab_clone_total_diff_size_bytes` | Gauge | - | Total extra disk space used by all clones (sum of diffs from snapshots) |
| `dblab_clone_total_logical_size_bytes` | Gauge | - | Total logical size of all clone data |
| `dblab_clone_total_cpu_usage_percent` | Gauge | - | Total CPU usage percentage across all clone containers |
| `dblab_clone_avg_cpu_usage_percent` | Gauge | - | Average CPU usage percentage across all clone containers with valid data |
| `dblab_clone_total_memory_usage_bytes` | Gauge | - | Total memory usage in bytes across all clone containers |
| `dblab_clone_total_memory_limit_bytes` | Gauge | - | Total memory limit in bytes across all clone containers |
| `dblab_clone_protected_count` | Gauge | - | Number of protected clones |

### Snapshot metrics (aggregate)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_snapshots_total` | Gauge | - | Total number of snapshots |
| `dblab_snapshots_by_pool` | Gauge | `pool` | Number of snapshots by pool |
| `dblab_snapshot_max_age_seconds` | Gauge | - | Maximum age of any snapshot in seconds |
| `dblab_snapshot_total_physical_size_bytes` | Gauge | - | Total physical disk space used by all snapshots |
| `dblab_snapshot_total_logical_size_bytes` | Gauge | - | Total logical size of all snapshot data |
| `dblab_snapshot_max_data_lag_seconds` | Gauge | - | Maximum data lag of any snapshot in seconds |
| `dblab_snapshot_total_num_clones` | Gauge | - | Total number of clones across all snapshots |

### Branch metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_branches_total` | Gauge | - | Total number of branches |

### Dataset metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_datasets_total` | Gauge | `pool` | Total number of datasets (slots) in the pool |
| `dblab_datasets_available` | Gauge | `pool` | Number of available (non-busy) dataset slots for reuse |

### Sync instance metrics (physical mode)

These metrics are only available when DBLab is running in physical mode with a sync instance enabled. They track the WAL replay status of the sync instance.

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_sync_status` | Gauge | `status` | Status of the sync instance (1=active for status code) |
| `dblab_sync_wal_lag_seconds` | Gauge | - | WAL replay lag in seconds for the sync instance |
| `dblab_sync_uptime_seconds` | Gauge | - | Uptime of the sync instance in seconds |
| `dblab_sync_last_replayed_timestamp` | Gauge | - | Unix timestamp of the last replayed transaction |

### Observability metrics

These metrics help monitor the health of the metrics collection system itself.

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `dblab_scrape_success_timestamp` | Gauge | - | Unix timestamp of last successful metrics collection |
| `dblab_scrape_duration_seconds` | Gauge | - | Duration of last metrics collection in seconds |
| `dblab_scrape_errors_total` | Counter | - | Total number of errors during metrics collection |

## Prometheus configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'dblab'
    static_configs:
      - targets: ['<dblab-host>:<dblab-port>']
    metrics_path: /metrics
```

## Example queries

### Free disk space percentage

```promql
100 * dblab_disk_free_bytes / dblab_disk_total_bytes
```

### Number of active clones

```promql
dblab_clones_total
```

### Maximum clone age in hours

```promql
dblab_clone_max_age_seconds / 3600
```

### Data freshness (lag from current time)

```promql
dblab_snapshot_max_data_lag_seconds / 60
```

### Total memory usage across all clones

```promql
dblab_clone_total_memory_usage_bytes
```

### Average CPU usage across all clones

```promql
dblab_clone_avg_cpu_usage_percent
```

### Clones by status

```promql
dblab_clones_by_status
```

### Metrics collection health

```promql
time() - dblab_scrape_success_timestamp
```

### WAL replay lag (physical mode)

```promql
dblab_sync_wal_lag_seconds
```

### Time since last replayed transaction

```promql
time() - dblab_sync_last_replayed_timestamp
```

## Alerting examples

### Low disk space alert

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

### Stale snapshot alert

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

### High clone count alert

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

### Metrics collection stale alert

```yaml
- alert: DBLabMetricsStale
  expr: time() - dblab_scrape_success_timestamp > 300
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "DBLab metrics collection is stale"
    description: "DBLab metrics have not been updated for more than 5 minutes"
```

### High WAL replay lag alert (physical mode)

```yaml
- alert: DBLabHighWALLag
  expr: dblab_sync_wal_lag_seconds > 3600
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "DBLab sync instance has high WAL lag"
    description: "DBLab sync instance WAL replay is {{ $value | humanizeDuration }} behind"
```

### Sync instance down alert (physical mode)

```yaml
- alert: DBLabSyncDown
  expr: dblab_sync_status{status="down"} == 1 or dblab_sync_status{status="error"} == 1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "DBLab sync instance is down"
    description: "DBLab sync instance is not healthy"
```

## OpenTelemetry integration

DBLab metrics can be exported to OpenTelemetry-compatible backends using the OpenTelemetry Collector. This allows you to send metrics to Grafana Cloud, Datadog, New Relic, and other observability platforms.

### Quick start

1. Install the OpenTelemetry Collector:
   ```bash
   # Using Docker
   docker pull otel/opentelemetry-collector-contrib:latest
   ```

2. Copy the example configuration:
   ```bash
   cp engine/configs/otel-collector.example.yml otel-collector.yml
   ```

3. Edit `otel-collector.yml` to configure your backend:
   ```yaml
   exporters:
     otlp:
       endpoint: "your-otlp-endpoint:4317"
       headers:
         Authorization: "Bearer <your-token>"
   ```

4. Run the collector:
   ```bash
   docker run -v $(pwd)/otel-collector.yml:/etc/otelcol/config.yaml \
     -p 4317:4317 -p 8889:8889 \
     otel/opentelemetry-collector-contrib:latest
   ```

### Architecture

```
┌─────────────┐     scrape      ┌──────────────────┐      OTLP       ┌─────────────┐
│   DBLab     │ ──────────────► │  OTel Collector  │ ──────────────► │  Backend    │
│  /metrics   │    :2345        │                  │    :4317        │ (Grafana,   │
└─────────────┘                 └──────────────────┘                 │  Datadog)   │
                                                                     └─────────────┘
```

### Supported backends

The OTel Collector can export to:
- **Grafana Cloud** - Use OTLP exporter with Grafana Cloud endpoint
- **Datadog** - Use the datadog exporter
- **New Relic** - Use OTLP exporter with New Relic endpoint
- **Prometheus Remote Write** - Use prometheusremotewrite exporter
- **AWS CloudWatch** - Use awsemf exporter
- **Any OTLP-compatible backend**

See `engine/configs/otel-collector.example.yml` for a complete configuration example.
