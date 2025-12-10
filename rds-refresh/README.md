# DBLab RDS/Aurora Refresh

A standalone tool that automates DBLab Engine full refresh using temporary RDS or Aurora clones created from snapshots.

## Overview

This tool provides a hassle-free way to keep your DBLab Engine data synchronized with your production RDS/Aurora database:

1. **Creates a temporary clone** from the latest RDS/Aurora snapshot
2. **Triggers DBLab full refresh** to sync data from the clone
3. **Deletes the temporary clone** after refresh completes

This approach avoids impacting your production database during the data sync process.

## Quick Start

### Build

```bash
# Clone this repository
git clone https://github.com/postgres-ai/rds-refresh.git
cd rds-refresh

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
# Dry run (validates configuration)
./rds-refresh -config config.yaml -dry-run

# Full refresh
./rds-refresh -config config.yaml
```

## Deployment Options

### Option 1: AWS Lambda (Recommended)

Deploy as a serverless function with automatic scheduling via EventBridge.

#### Prerequisites

- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html)
- AWS credentials configured
- Go 1.21+

#### Deploy

```bash
# Build and deploy
sam build
sam deploy --guided
```

During guided deployment, you'll be prompted for:

| Parameter | Description | Example |
|-----------|-------------|---------|
| `RDSSourceType` | `rds` or `aurora-cluster` | `rds` |
| `RDSSourceIdentifier` | Source DB identifier | `production-db` |
| `RDSCloneInstanceClass` | Clone instance size | `db.t3.medium` |
| `DBLabAPIEndpoint` | DBLab API URL | `https://dblab.example.com:2345` |
| `DBLabToken` | DBLab verification token | `your-secret-token` |
| `ScheduleExpression` | Refresh schedule | `rate(7 days)` |

#### Manual Invocation

```bash
# Dry run
aws lambda invoke --function-name dblab-rds-refresh \
  --cli-binary-format raw-in-base64-out \
  --payload '{"dryRun": true}' \
  response.json && cat response.json

# Full refresh
aws lambda invoke --function-name dblab-rds-refresh \
  --cli-binary-format raw-in-base64-out \
  --payload '{"dryRun": false}' \
  response.json && cat response.json
```

### Option 2: CLI with Cron

```bash
# Build
make build

# Install
sudo mv rds-refresh /usr/local/bin/

# Create config
sudo mkdir -p /etc/dblab
sudo cp config.example.yaml /etc/dblab/rds-refresh.yaml
sudo vim /etc/dblab/rds-refresh.yaml

# Add to crontab (every Sunday at 2 AM)
echo "0 2 * * 0 /usr/local/bin/rds-refresh -config /etc/dblab/rds-refresh.yaml >> /var/log/rds-refresh.log 2>&1" | crontab -
```

### Option 3: Docker

```bash
# Build
docker build -t rds-refresh .

# Run
docker run \
  -v /path/to/config.yaml:/config.yaml \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e DBLAB_TOKEN \
  rds-refresh -config /config.yaml
```

### Option 4: Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: dblab-rds-refresh
spec:
  schedule: "0 2 * * 0"  # Every Sunday at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: dblab-rds-refresh  # with IRSA
          containers:
          - name: rds-refresh
            image: your-registry/rds-refresh:latest
            args: ["-config", "/config/config.yaml"]
            volumeMounts:
            - name: config
              mountPath: /config
            env:
            - name: DBLAB_TOKEN
              valueFrom:
                secretKeyRef:
                  name: dblab-secrets
                  key: token
          volumes:
          - name: config
            configMap:
              name: rds-refresh-config
          restartPolicy: OnFailure
```

## Configuration

See [config.example.yaml](config.example.yaml) for a fully documented example.

### Environment Variables

When running as Lambda, configuration is loaded from environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `RDS_SOURCE_IDENTIFIER` | Yes | Source RDS instance or Aurora cluster ID |
| `RDS_CLONE_INSTANCE_CLASS` | Yes | Instance class for clone (e.g., `db.t3.medium`) |
| `DBLAB_API_ENDPOINT` | Yes | DBLab Engine API endpoint |
| `DBLAB_TOKEN` | Yes | DBLab verification token |
| `AWS_REGION` | Yes | AWS region |
| `RDS_SOURCE_TYPE` | No | `rds` or `aurora-cluster` (default: `rds`) |
| `RDS_SNAPSHOT_IDENTIFIER` | No | Specific snapshot ID (default: latest) |
| `RDS_CLONE_SUBNET_GROUP` | No | DB subnet group name |
| `RDS_CLONE_SECURITY_GROUPS` | No | JSON array of security group IDs |
| `RDS_CLONE_PUBLIC` | No | `true` to make clone publicly accessible |
| `RDS_CLONE_ENABLE_IAM_AUTH` | No | `true` to enable IAM authentication |
| `RDS_CLONE_STORAGE_TYPE` | No | Storage type (gp2, gp3, io1, etc.) |
| `DBLAB_INSECURE` | No | `true` to skip TLS verification |

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

## DBLab Engine Configuration

Configure DBLab Engine to connect to the temporary clone. The clone will be named `dblab-refresh-YYYYMMDD-HHMMSS`.

Example DBLab retrieval configuration:

```yaml
retrieval:
  refresh:
    timetable: ""  # Disable built-in scheduler (managed externally)
    skipStartRefresh: true

  jobs:
    - logicalDump
    - logicalRestore
    - logicalSnapshot

  spec:
    logicalDump:
      options:
        source:
          type: rdsIam
          connection:
            dbname: mydb
            username: dblab_user
          rdsIam:
            awsRegion: us-east-1
            dbInstanceIdentifier: dblab-refresh-current  # Will be the temp clone
```

## Troubleshooting

### Common Issues

**Clone creation fails with "DBSubnetGroup not found"**
- Ensure the subnet group exists and is in the correct VPC

**Clone not accessible from DBLab**
- Verify security groups allow inbound connections from DBLab
- Check if `publiclyAccessible` setting matches your network topology

**DBLab refresh timeout**
- Increase `dblab.timeout` in configuration
- Check DBLab Engine logs for issues

**AWS credentials not found**
- Ensure AWS credentials are configured (env vars, IAM role, or credentials file)

### Debug Mode

```bash
# Enable verbose AWS SDK logging
export AWS_SDK_LOAD_CONFIG=1
./rds-refresh -config config.yaml 2>&1 | tee refresh.log
```

## Cost Considerations

- **Clone runtime**: You pay for the clone instance while it exists
- **Storage**: Clones don't duplicate storage (snapshot-based)
- **Lambda**: Minimal cost (typically < $0.10/month for weekly refreshes)

**Cost optimization tips**:
- Use a smaller instance class than production
- Use `gp3` storage type for better price/performance
- Schedule refreshes during off-peak hours

## License

Apache 2.0

## Links

- [DBLab Engine Documentation](https://postgres.ai/docs/database-lab-engine)
- [Postgres.ai](https://postgres.ai)
