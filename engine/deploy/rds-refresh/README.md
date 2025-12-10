# DBLab RDS/Aurora Refresh Component

Automates DBLab Engine full refresh using temporary RDS or Aurora clones created from snapshots.

## Overview

This component provides a hassle-free way to keep your DBLab Engine data synchronized with your production RDS/Aurora database. It:

1. **Creates a temporary clone** from the latest RDS/Aurora snapshot
2. **Triggers DBLab full refresh** to sync data from the clone
3. **Deletes the temporary clone** after refresh completes

This approach avoids impacting your production database during the data sync process.

## Deployment Options

### Option 1: AWS Lambda (Recommended)

Deploy as a serverless function with automatic scheduling via EventBridge.

#### Prerequisites

- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html)
- AWS credentials configured
- Go 1.21+ (for building)

#### Quick Start

```bash
# Clone the repository
git clone https://gitlab.com/postgres-ai/database-lab.git
cd database-lab/engine/deploy/rds-refresh

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
# Dry run (validates configuration)
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

### Option 2: CLI Binary

Run as a standalone binary via cron or systemd timer.

#### Build

```bash
cd engine
go build -o rds-refresh ./cmd/rds-refresh
```

#### Usage

```bash
# Dry run
./rds-refresh -config config.yaml -dry-run

# Full refresh
./rds-refresh -config config.yaml
```

#### Example Configuration

```yaml
# config.yaml
source:
  type: rds                    # or aurora-cluster
  identifier: production-db    # RDS instance or Aurora cluster ID
  # snapshotIdentifier: ""     # optional: specific snapshot (default: latest)

clone:
  instanceClass: db.t3.medium  # smaller than prod for cost savings
  subnetGroup: default-vpc     # same VPC as DBLab Engine
  securityGroups:
    - sg-12345678              # must allow DBLab to connect
  publiclyAccessible: false
  enableIAMAuth: true          # recommended for secure access
  # parameterGroup: ""         # optional: custom parameter group
  # storageType: gp3           # optional: storage type

dblab:
  apiEndpoint: https://dblab.example.com:2345
  token: ${DBLAB_TOKEN}        # environment variable expansion
  pollInterval: 30s
  timeout: 4h

aws:
  region: us-east-1
```

#### Cron Example

```bash
# Run every Sunday at 2 AM
0 2 * * 0 /usr/local/bin/rds-refresh -config /etc/dblab/rds-refresh.yaml >> /var/log/rds-refresh.log 2>&1
```

### Option 3: Docker Container

```bash
# Build (from repository root)
docker build -t dblab-rds-refresh -f engine/deploy/rds-refresh/Dockerfile .

# Run
docker run -v /path/to/config.yaml:/config.yaml \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e DBLAB_TOKEN \
  dblab-rds-refresh -config /config.yaml
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
            image: postgresai/rds-refresh:latest
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

## AWS IAM Permissions

### Minimal IAM Policy

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

### For IAM Database Authentication

If using RDS IAM authentication (recommended), the DBLab Engine also needs:

```json
{
  "Sid": "RDSIAMConnect",
  "Effect": "Allow",
  "Action": "rds-db:connect",
  "Resource": "arn:aws:rds-db:*:ACCOUNT_ID:dbuser:*/dblab_user"
}
```

## DBLab Engine Configuration

Configure DBLab Engine to connect to the temporary clone using RDS IAM authentication:

```yaml
# server.yml (DBLab Engine config)
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
        dockerImage: "postgresai/extended-postgres:17"
        dumpLocation: "/var/lib/dblab/dblab_pool/dump"

        source:
          type: rdsIam
          connection:
            dbname: mydb
            username: dblab_user
          rdsIam:
            awsRegion: us-east-1
            # This will be updated by rds-refresh or pre-configured
            dbInstanceIdentifier: dblab-refresh-current
            sslRootCert: "/cert/rds-combined-ca-bundle.pem"

        parallelJobs: 4
        customOptions:
          - "--exclude-schema=rdsdms"
```

## Security Best Practices

1. **Use IAM Database Authentication** - Avoid storing database passwords
2. **Use Secrets Manager** - Store the DBLab token in AWS Secrets Manager
3. **VPC Configuration** - Run clones in a private subnet accessible only to DBLab
4. **Minimal Permissions** - Use the minimal IAM policy above
5. **Encryption** - Ensure clones inherit encryption from snapshots

## Monitoring

### CloudWatch Metrics (Lambda)

The Lambda function emits standard metrics:
- `Invocations` - Number of refresh attempts
- `Errors` - Failed refreshes
- `Duration` - Execution time

### Custom CloudWatch Dashboard

```bash
# View recent logs
aws logs tail /aws/lambda/dblab-rds-refresh --follow
```

### Alerting

Set up CloudWatch Alarms for:
- Lambda errors > 0
- Lambda duration > threshold
- (Optional) Custom metrics on refresh success/failure

## Troubleshooting

### Common Issues

**Clone creation fails with "DBSubnetGroup not found"**
- Ensure the subnet group exists and is in the same VPC

**Clone creation fails with "VPCSecurityGroupNotFound"**
- Verify security group IDs are correct

**DBLab refresh timeout**
- Increase `dblab.timeout` in configuration
- Check DBLab Engine logs for issues

**Clone not accessible from DBLab**
- Verify security groups allow connection from DBLab
- Check if publiclyAccessible setting is correct

### Debug Mode

```bash
# CLI: Enable verbose logging
./rds-refresh -config config.yaml 2>&1 | tee refresh.log

# Lambda: Check CloudWatch logs
aws logs tail /aws/lambda/dblab-rds-refresh --since 1h
```

## Cost Considerations

- **Clone runtime**: You pay for the clone instance while it exists
- **Storage**: Clones don't duplicate storage (snapshot-based)
- **Lambda**: Minimal cost (typically < $0.10/month for weekly refreshes)

**Cost optimization tips**:
- Use a smaller instance class than production
- Use `gp3` storage type for better price/performance
- Schedule refreshes during off-peak hours

## Contributing

See the main [Database Lab Engine contributing guide](../../CONTRIBUTING.md).

## License

Apache 2.0 - see [LICENSE](../../LICENSE).
