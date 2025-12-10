# DBLab RDS/Aurora Refresh

A standalone tool that automates DBLab Engine full refresh using temporary RDS or Aurora clones created from snapshots.

## Overview

This tool provides a hassle-free way to keep your DBLab Engine data synchronized with your production RDS/Aurora database:

1. **Creates a temporary clone** from the latest RDS/Aurora snapshot
2. **Updates DBLab configuration** with the new clone's endpoint
3. **Triggers DBLab full refresh** to sync data from the clone
4. **Deletes the temporary clone** after refresh completes

This approach avoids impacting your production database during the data sync process.

## Quick Start

### Build

```bash
# Clone the repository
git clone https://github.com/postgres-ai/database-lab-engine.git
cd database-lab-engine/rds-refresh

# Build
make build

# Or build directly
go build -o rds-refresh .
```

### Configure

```bash
# Copy example config
cp config.example.yaml config.yaml

# Edit with your settings
vim config.yaml
```

### Run

```bash
# Dry run (validates configuration without creating resources)
./rds-refresh -config config.yaml -dry-run

# Full refresh
./rds-refresh -config config.yaml
```

## Deployment Options

The refresh process can take 1-4 hours depending on database size, so this tool is designed for long-running execution environments.

### Option 1: Docker (Recommended)

```bash
# Build image
make docker-build

# Run
docker run \
  -v /path/to/config.yaml:/config.yaml \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e DB_PASSWORD \
  -e DBLAB_TOKEN \
  postgresai/rds-refresh -config /config.yaml
```

### Option 2: ECS Task

Create an ECS Task Definition for scheduled execution:

```json
{
  "family": "dblab-rds-refresh",
  "networkMode": "awsvpc",
  "containerDefinitions": [
    {
      "name": "rds-refresh",
      "image": "postgresai/rds-refresh:latest",
      "command": ["-config", "/config/config.yaml"],
      "mountPoints": [
        {
          "sourceVolume": "config",
          "containerPath": "/config"
        }
      ],
      "secrets": [
        {
          "name": "DB_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:db-password"
        },
        {
          "name": "DBLAB_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:dblab-token"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/dblab-rds-refresh",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ],
  "taskRoleArn": "arn:aws:iam::123456789:role/dblab-rds-refresh-task",
  "executionRoleArn": "arn:aws:iam::123456789:role/ecsTaskExecutionRole",
  "volumes": [
    {
      "name": "config",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-12345678"
      }
    }
  ]
}
```

Schedule with EventBridge:
```bash
aws events put-rule \
  --name dblab-rds-refresh-weekly \
  --schedule-expression "rate(7 days)"

aws events put-targets \
  --rule dblab-rds-refresh-weekly \
  --targets "Id"="1","Arn"="arn:aws:ecs:us-east-1:123456789:cluster/my-cluster","RoleArn"="arn:aws:iam::123456789:role/ecsEventsRole","EcsParameters"="{\"taskDefinitionArn\": \"arn:aws:ecs:us-east-1:123456789:task-definition/dblab-rds-refresh:1\",\"taskCount\": 1}"
```

### Option 3: Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: dblab-rds-refresh
spec:
  schedule: "0 2 * * 0"  # Every Sunday at 2 AM
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          serviceAccountName: dblab-rds-refresh  # with IRSA for AWS access
          containers:
          - name: rds-refresh
            image: postgresai/rds-refresh:latest
            args: ["-config", "/config/config.yaml"]
            volumeMounts:
            - name: config
              mountPath: /config
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
          volumes:
          - name: config
            configMap:
              name: rds-refresh-config
          restartPolicy: Never
```

### Option 4: CLI with Cron

```bash
# Build and install
make build
sudo mv rds-refresh /usr/local/bin/

# Create config directory
sudo mkdir -p /etc/dblab
sudo cp config.example.yaml /etc/dblab/rds-refresh.yaml
sudo chmod 600 /etc/dblab/rds-refresh.yaml
sudo vim /etc/dblab/rds-refresh.yaml

# Add to crontab (every Sunday at 2 AM)
echo "0 2 * * 0 /usr/local/bin/rds-refresh -config /etc/dblab/rds-refresh.yaml >> /var/log/rds-refresh.log 2>&1" | crontab -
```

## Configuration

See [config.example.yaml](config.example.yaml) for a fully documented example.

### Key Configuration Fields

```yaml
source:
  type: rds                    # "rds" or "aurora-cluster"
  identifier: production-db    # RDS instance or Aurora cluster ID
  dbName: myapp               # Database name for DBLab to connect to
  username: postgres          # Database username
  password: ${DB_PASSWORD}    # Use env var expansion for secrets

clone:
  instanceClass: db.t3.medium # Can be smaller than production
  subnetGroup: my-subnet      # Must be accessible from DBLab
  securityGroups:
    - sg-12345678             # Must allow DBLab inbound access

dblab:
  apiEndpoint: https://dblab.example.com:2345
  token: ${DBLAB_TOKEN}
  pollInterval: 30s           # Status check frequency
  timeout: 4h                 # Max wait for refresh completion

aws:
  region: us-east-1
```

### Environment Variables

The configuration file supports environment variable expansion using `${VAR_NAME}` syntax. This is useful for secrets:

```yaml
source:
  password: ${DB_PASSWORD}
dblab:
  token: ${DBLAB_TOKEN}
```

## AWS IAM Permissions

The tool requires the following IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "RDSReadSnapshots",
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
      "Sid": "RDSCreateClone",
      "Effect": "Allow",
      "Action": [
        "rds:RestoreDBInstanceFromDBSnapshot",
        "rds:RestoreDBClusterFromSnapshot",
        "rds:CreateDBInstance",
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
    },
    {
      "Sid": "RDSDeleteClone",
      "Effect": "Allow",
      "Action": [
        "rds:DeleteDBInstance",
        "rds:DeleteDBCluster"
      ],
      "Resource": [
        "arn:aws:rds:*:ACCOUNT_ID:db:dblab-refresh-*",
        "arn:aws:rds:*:ACCOUNT_ID:cluster:dblab-refresh-*"
      ]
    }
  ]
}
```

Replace `ACCOUNT_ID` with your AWS account ID.

## DBLab Engine Requirements

The tool dynamically updates the DBLab Engine's source configuration before triggering refresh. Your DBLab Engine must:

1. **Support config updates via API** - The `/admin/config` endpoint must be available
2. **Run in logical mode** - Using pg_dump/pg_restore for data retrieval
3. **Be accessible** - The API endpoint must be reachable from where this tool runs

Example DBLab configuration:

```yaml
retrieval:
  refresh:
    timetable: ""  # Disable built-in scheduler (managed by this tool)
    skipStartRefresh: true

  jobs:
    - logicalDump
    - logicalRestore
    - logicalSnapshot

  spec:
    logicalDump:
      options:
        source:
          type: local  # Will be updated dynamically
          connection:
            dbname: mydb
            username: postgres
            # host and port will be updated by rds-refresh
```

## Workflow

The tool executes the following steps:

1. **Health check** - Verifies DBLab Engine is healthy and not already refreshing
2. **Source validation** - Gets source RDS/Aurora database info
3. **Snapshot discovery** - Finds the latest automated snapshot
4. **Clone creation** - Creates a temporary RDS instance/cluster from the snapshot
5. **Wait for clone** - Polls until clone is available (10-30 minutes typical)
6. **Config update** - Updates DBLab's source configuration with the clone endpoint
7. **Trigger refresh** - Initiates DBLab full refresh
8. **Wait for completion** - Polls until refresh completes (1-4 hours typical)
9. **Cleanup** - Deletes the temporary clone

If any step fails, the clone is automatically deleted (cleanup runs in defer).

## Troubleshooting

### Common Issues

**Clone creation fails with "DBSubnetGroup not found"**
- Ensure the subnet group exists and is in the correct VPC
- Verify the subnet group name in your configuration

**Clone not accessible from DBLab**
- Verify security groups allow inbound connections from DBLab on port 5432
- Check if `publiclyAccessible` setting matches your network topology
- Ensure the clone and DBLab are in the same VPC or have network connectivity

**DBLab config update fails**
- Verify the DBLab API endpoint is correct
- Check that the verification token is valid
- Ensure DBLab supports the `/admin/config` endpoint

**DBLab refresh timeout**
- Increase `dblab.timeout` in configuration (default is 4 hours)
- Check DBLab Engine logs for issues during refresh
- Consider the database size - larger databases take longer

**AWS credentials not found**
- For ECS/Kubernetes: Use IAM Roles for Service Accounts (IRSA) or ECS Task Roles
- For CLI: Configure AWS credentials via environment variables or credentials file
- Verify IAM permissions are correctly attached

### Debug Mode

```bash
# Enable verbose output
./rds-refresh -config config.yaml 2>&1 | tee refresh.log

# Check AWS credential chain
aws sts get-caller-identity
```

## Cost Considerations

- **Clone runtime**: You pay for the clone instance while it exists (typically 2-5 hours)
- **Storage**: Clones don't duplicate storage initially (snapshot-based, copy-on-write)
- **Data transfer**: Minimal if DBLab is in the same region

**Cost optimization tips**:
- Use a smaller instance class than production (e.g., `db.t3.medium`)
- Use `gp3` storage type for better price/performance
- Schedule refreshes during off-peak hours
- The tool automatically deletes clones after completion

## License

Apache 2.0

## Links

- [DBLab Engine Documentation](https://postgres.ai/docs/database-lab-engine)
- [Postgres.ai](https://postgres.ai)
