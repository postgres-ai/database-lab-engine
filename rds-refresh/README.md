# RDS/Aurora Refresh for DBLab

Refresh DBLab from RDS/Aurora snapshots without touching production.

## Why?

DBLab logical mode runs `pg_dump` against your database. On large databases, this:
- **Holds xmin horizon for hours** → bloat accumulation
- **Creates load on production**
- **Requires direct network access** to production

This tool dumps from a **temporary snapshot clone** instead. Production is never touched.

```
Production RDS ──snapshot──► Snapshot ──restore──► Temp Clone ──pg_dump──► DBLab
                (automated)                         (deleted)
```

## Quick Start

```bash
# 1. Configure
cat > config.yaml << 'EOF'
source:
  type: rds                    # or "aurora-cluster"
  identifier: my-prod-db
  dbName: postgres
  username: postgres
  password: ${DB_PASSWORD}

clone:
  instanceClass: db.t3.medium
  securityGroups: [sg-xxx]     # must allow DBLab inbound

dblab:
  apiEndpoint: https://dblab:2345
  token: ${DBLAB_TOKEN}

aws:
  region: us-east-1
EOF

# 2. Test
docker run --rm \
  -v $PWD/config.yaml:/config.yaml \
  -e DB_PASSWORD -e DBLAB_TOKEN -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
  postgresai/rds-refresh -config /config.yaml -dry-run

# 3. Run
docker run --rm \
  -v $PWD/config.yaml:/config.yaml \
  -e DB_PASSWORD -e DBLAB_TOKEN -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
  postgresai/rds-refresh -config /config.yaml
```

## Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `source.type` | ✓ | `rds` or `aurora-cluster` |
| `source.identifier` | ✓ | RDS/Aurora identifier |
| `source.dbName` | ✓ | Database name |
| `source.username` | ✓ | Database user |
| `source.password` | ✓ | Password (use `${ENV_VAR}`) |
| `clone.instanceClass` | ✓ | Clone instance type |
| `clone.securityGroups` | | SGs allowing DBLab access |
| `clone.subnetGroup` | | DB subnet group |
| `dblab.apiEndpoint` | ✓ | DBLab API URL |
| `dblab.token` | ✓ | DBLab verification token |
| `dblab.timeout` | | Max refresh wait (default: 4h) |
| `aws.region` | ✓ | AWS region |

Full example: [config.example.yaml](config.example.yaml)

## Scheduling

```bash
# Cron (weekly, Sunday 2 AM)
0 2 * * 0 docker run --rm -v /etc/dblab/config.yaml:/config.yaml \
  --env-file /etc/dblab/env postgresai/rds-refresh -config /config.yaml
```

<details>
<summary>Kubernetes CronJob</summary>

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: dblab-refresh
spec:
  schedule: "0 2 * * 0"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: dblab-refresh  # IRSA
          containers:
          - name: refresh
            image: postgresai/rds-refresh
            args: ["-config", "/config/config.yaml"]
            envFrom:
            - secretRef:
                name: dblab-refresh-secrets
            volumeMounts:
            - name: config
              mountPath: /config
          volumes:
          - name: config
            configMap:
              name: dblab-refresh-config
          restartPolicy: Never
```
</details>

<details>
<summary>ECS Scheduled Task</summary>

```bash
aws events put-rule --name dblab-refresh --schedule-expression "cron(0 2 ? * SUN *)"
aws events put-targets --rule dblab-refresh --targets '[{
  "Id": "1",
  "Arn": "arn:aws:ecs:REGION:ACCOUNT:cluster/CLUSTER",
  "RoleArn": "arn:aws:iam::ACCOUNT:role/ecsEventsRole",
  "EcsParameters": {
    "TaskDefinitionArn": "arn:aws:ecs:REGION:ACCOUNT:task-definition/dblab-refresh",
    "TaskCount": 1, "LaunchType": "FARGATE"
  }
}]'
```
</details>

## IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["rds:DescribeDBSnapshots", "rds:DescribeDBClusterSnapshots",
                 "rds:DescribeDBInstances", "rds:DescribeDBClusters"],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["rds:RestoreDBInstanceFromDBSnapshot", "rds:RestoreDBClusterFromSnapshot",
                 "rds:CreateDBInstance", "rds:DeleteDBInstance", "rds:DeleteDBCluster",
                 "rds:AddTagsToResource", "rds:ModifyDBInstance", "rds:ModifyDBCluster"],
      "Resource": ["arn:aws:rds:*:ACCOUNT:db:dblab-refresh-*",
                   "arn:aws:rds:*:ACCOUNT:cluster:dblab-refresh-*",
                   "arn:aws:rds:*:ACCOUNT:snapshot:*",
                   "arn:aws:rds:*:ACCOUNT:cluster-snapshot:*",
                   "arn:aws:rds:*:ACCOUNT:subgrp:*", "arn:aws:rds:*:ACCOUNT:pg:*"]
    }
  ]
}
```

## Network

Clone must be reachable from DBLab on port 5432. Same VPC or peered.

## DBLab Setup

Must run in **logical mode**. Tool updates config via API (no SSH needed).

```yaml
retrieval:
  refresh:
    timetable: ""  # disable built-in scheduler
  jobs: [logicalDump, logicalRestore, logicalSnapshot]
  spec:
    logicalDump:
      options:
        source:
          connection:
            host: placeholder  # updated by rds-refresh
            port: 5432
```

## How It Works

1. Check DBLab health
2. Find latest RDS snapshot
3. Create temp clone (`dblab-refresh-YYYYMMDD-HHMMSS`)
4. Wait for clone (~15 min)
5. Update DBLab config via API
6. Trigger refresh, wait for completion
7. Delete clone (always, even on error)

## Troubleshooting

| Error | Fix |
|-------|-----|
| No snapshots | Enable automated backups on RDS |
| Clone not accessible | Check security group allows 5432 from DBLab |
| Config update failed | Verify DBLab endpoint and token |
| Timeout | Increase `dblab.timeout`, check DBLab logs |

## Cost

Clone cost only while running (~2-5 hours):
- db.t3.medium: ~$0.35
- db.r5.large: ~$1.20

## License

Apache 2.0 — [Postgres.ai](https://postgres.ai)
