# RDS/Aurora Refresh for DBLab

Automate DBLab Engine data refresh from RDS/Aurora snapshots without impacting production.

## The Problem

DBLab Engine in logical mode needs to connect to a source PostgreSQL database to perform `pg_dump`. Connecting directly to production RDS/Aurora during refresh:
- Creates load on production
- Requires opening network access from DBLab to production
- Can take hours, blocking production resources

## The Solution

This tool creates a **temporary clone** from an RDS/Aurora snapshot, points DBLab at the clone for refresh, then deletes the clone. Production is never touched.

```
┌─────────────┐     snapshot      ┌─────────────┐
│ Production  │ ───────────────►  │  Snapshot   │
│ RDS/Aurora  │   (automated)     │             │
└─────────────┘                   └──────┬──────┘
                                         │
                                         │ restore
                                         ▼
┌─────────────┐    pg_dump        ┌─────────────┐
│   DBLab     │ ◄──────────────── │  Temp Clone │
│   Engine    │                   │  (deleted)  │
└─────────────┘                   └─────────────┘
```

## Prerequisites

- **DBLab Engine** running in logical mode with API access enabled
- **AWS credentials** with RDS permissions (see [IAM Permissions](#iam-permissions))
- **Network access** from the temp clone to DBLab Engine (same VPC or peered)
- **RDS automated snapshots** enabled on your source database

## Quick Start

### 1. Get the tool

**Option A: Docker (recommended)**
```bash
docker pull postgresai/rds-refresh:latest
```

**Option B: Build from source**
```bash
git clone https://github.com/postgres-ai/database-lab-engine.git
cd database-lab-engine/rds-refresh
make build
```

### 2. Create configuration

```bash
cat > config.yaml << 'EOF'
source:
  type: rds                      # or "aurora-cluster"
  identifier: my-production-db   # your RDS instance ID
  dbName: postgres               # database to dump
  username: postgres
  password: ${DB_PASSWORD}       # from environment variable

clone:
  instanceClass: db.t3.medium    # can be smaller than prod
  subnetGroup: default           # must allow DBLab access
  securityGroups:
    - sg-xxxxxxxxx               # must allow inbound from DBLab

dblab:
  apiEndpoint: https://dblab.example.com:2345
  token: ${DBLAB_TOKEN}
  timeout: 4h

aws:
  region: us-east-1
EOF
```

### 3. Run

**Dry run first** (validates config, finds snapshot, no changes made):
```bash
# Docker
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml \
  -e DB_PASSWORD -e DBLAB_TOKEN \
  -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
  postgresai/rds-refresh -config /config.yaml -dry-run

# Binary
export DB_PASSWORD="..." DBLAB_TOKEN="..."
./rds-refresh -config config.yaml -dry-run
```

**Full refresh:**
```bash
# Docker
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml \
  -e DB_PASSWORD -e DBLAB_TOKEN \
  -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
  postgresai/rds-refresh -config /config.yaml

# Binary
./rds-refresh -config config.yaml
```

## Configuration Reference

| Field | Required | Description |
|-------|----------|-------------|
| `source.type` | Yes | `rds` or `aurora-cluster` |
| `source.identifier` | Yes | RDS instance or Aurora cluster identifier |
| `source.dbName` | Yes | Database name for DBLab to connect to |
| `source.username` | Yes | Database username |
| `source.password` | Yes | Database password (use `${ENV_VAR}` syntax) |
| `clone.instanceClass` | Yes | Instance class for temp clone (e.g., `db.t3.medium`) |
| `clone.subnetGroup` | No | DB subnet group (uses source's if not set) |
| `clone.securityGroups` | No | Security group IDs for the clone |
| `clone.publiclyAccessible` | No | Make clone publicly accessible (default: false) |
| `dblab.apiEndpoint` | Yes | DBLab Engine API URL |
| `dblab.token` | Yes | DBLab verification token |
| `dblab.timeout` | No | Max wait for refresh (default: 4h) |
| `dblab.pollInterval` | No | Status check interval (default: 30s) |
| `dblab.insecure` | No | Skip TLS verification (default: false) |
| `aws.region` | Yes | AWS region |

See [config.example.yaml](config.example.yaml) for all options with comments.

## Scheduling

The refresh takes 1-4 hours. Schedule it during off-peak hours.

### Docker with cron

```bash
# /etc/cron.d/dblab-refresh
0 2 * * 0 root docker run --rm \
  -v /etc/dblab/config.yaml:/config.yaml \
  -e DB_PASSWORD -e DBLAB_TOKEN \
  -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
  postgresai/rds-refresh -config /config.yaml \
  >> /var/log/dblab-refresh.log 2>&1
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: dblab-rds-refresh
spec:
  schedule: "0 2 * * 0"  # Sundays at 2 AM
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          serviceAccountName: dblab-rds-refresh  # IRSA for AWS
          containers:
          - name: rds-refresh
            image: postgresai/rds-refresh:latest
            args: ["-config", "/config/config.yaml"]
            env:
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: dblab-secrets
                  key: db-password
            - name: DBLAB_TOKEN
              valueFrom:
                secretKeyRef:
                  name: dblab-secrets
                  key: token
            volumeMounts:
            - name: config
              mountPath: /config
          volumes:
          - name: config
            configMap:
              name: rds-refresh-config
          restartPolicy: Never
```

### AWS ECS Scheduled Task

```bash
# Create EventBridge rule
aws events put-rule \
  --name dblab-refresh-weekly \
  --schedule-expression "cron(0 2 ? * SUN *)"

# Target ECS task (create task definition first)
aws events put-targets \
  --rule dblab-refresh-weekly \
  --targets '[{
    "Id": "1",
    "Arn": "arn:aws:ecs:us-east-1:ACCOUNT:cluster/CLUSTER",
    "RoleArn": "arn:aws:iam::ACCOUNT:role/ecsEventsRole",
    "EcsParameters": {
      "TaskDefinitionArn": "arn:aws:ecs:us-east-1:ACCOUNT:task-definition/dblab-rds-refresh",
      "TaskCount": 1,
      "LaunchType": "FARGATE",
      "NetworkConfiguration": {
        "awsvpcConfiguration": {
          "Subnets": ["subnet-xxx"],
          "SecurityGroups": ["sg-xxx"],
          "AssignPublicIp": "DISABLED"
        }
      }
    }
  }]'
```

## IAM Permissions

Minimal IAM policy for the tool:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBSnapshots",
        "rds:DescribeDBClusterSnapshots",
        "rds:DescribeDBInstances",
        "rds:DescribeDBClusters"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "rds:RestoreDBInstanceFromDBSnapshot",
        "rds:RestoreDBClusterFromSnapshot",
        "rds:CreateDBInstance",
        "rds:DeleteDBInstance",
        "rds:DeleteDBCluster",
        "rds:AddTagsToResource",
        "rds:ModifyDBInstance",
        "rds:ModifyDBCluster"
      ],
      "Resource": [
        "arn:aws:rds:*:ACCOUNT_ID:db:dblab-refresh-*",
        "arn:aws:rds:*:ACCOUNT_ID:cluster:dblab-refresh-*",
        "arn:aws:rds:*:ACCOUNT_ID:snapshot:*",
        "arn:aws:rds:*:ACCOUNT_ID:cluster-snapshot:*",
        "arn:aws:rds:*:ACCOUNT_ID:subgrp:*",
        "arn:aws:rds:*:ACCOUNT_ID:pg:*",
        "arn:aws:rds:*:ACCOUNT_ID:og:*"
      ]
    }
  ]
}
```

## Network Requirements

The temporary clone needs to be accessible from DBLab Engine:

```
┌─────────────────────────────────────────────────────┐
│                       VPC                           │
│  ┌──────────────┐          ┌──────────────┐        │
│  │  DBLab       │ ──5432─► │  Temp Clone  │        │
│  │  Engine      │          │  (RDS)       │        │
│  └──────────────┘          └──────────────┘        │
│                                                     │
│  Security Group: Allow inbound 5432 from DBLab SG  │
└─────────────────────────────────────────────────────┘
```

- Clone and DBLab should be in the same VPC (or peered VPCs)
- Security group must allow PostgreSQL port (5432) from DBLab
- If using `publiclyAccessible: true`, ensure DBLab can reach public endpoint

## DBLab Engine Setup

Your DBLab must be running in **logical mode**. The tool updates DBLab's source config via API before triggering refresh.

Minimal DBLab config:

```yaml
server:
  port: 2345

retrieval:
  refresh:
    timetable: ""           # Disable built-in scheduler
    skipStartRefresh: true  # Don't refresh on startup

  jobs:
    - logicalDump
    - logicalRestore
    - logicalSnapshot

  spec:
    logicalDump:
      options:
        source:
          connection:
            host: placeholder    # Updated by rds-refresh
            port: 5432
            dbname: postgres
            username: postgres
            password: placeholder
```

## Workflow Details

1. **Health Check** - Verify DBLab is reachable and not mid-refresh
2. **Find Snapshot** - Get latest automated snapshot (or specified one)
3. **Create Clone** - Restore snapshot to new RDS instance (`dblab-refresh-YYYYMMDD-HHMMSS`)
4. **Wait for Clone** - Poll until instance is available (10-30 min)
5. **Update DBLab** - PATCH `/admin/config` with clone's endpoint
6. **Trigger Refresh** - POST `/admin/full-refresh`
7. **Wait for Refresh** - Poll status until complete (1-4 hours)
8. **Delete Clone** - Remove temporary instance (always runs, even on failure)

## Troubleshooting

**"No snapshots found"**
- Ensure automated backups are enabled on your RDS instance
- Check the `source.identifier` matches exactly

**"Clone not accessible"**
- Verify security group allows inbound 5432
- Check subnet group has proper routing to DBLab
- Try `publiclyAccessible: true` for testing

**"Config update failed"**
- Verify DBLab API endpoint and token
- Check DBLab is running in logical mode
- Ensure `/admin/config` endpoint is enabled

**"Refresh timeout"**
- Increase `dblab.timeout` (default 4h)
- Check DBLab logs for pg_dump/pg_restore errors
- Large databases take longer - consider partial dumps

**"AWS credentials error"**
- For ECS: attach IAM role to task definition
- For Kubernetes: use IRSA (IAM Roles for Service Accounts)
- For local: set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`

## Cost

You pay for the clone instance only while it exists:
- **db.t3.medium**: ~$0.068/hour → ~$0.34 for 5-hour refresh
- **db.r5.large**: ~$0.24/hour → ~$1.20 for 5-hour refresh

Storage is not duplicated (snapshot-based restore uses copy-on-write).

## License

Apache 2.0 - [Postgres.ai](https://postgres.ai)
