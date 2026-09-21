# Manual e2e: logical dump from RDS with IAM authentication

Verifies that the engine can dump a live RDS instance using IAM token authentication, restore it
into a pool, and serve clones from it.

This runs by hand. It needs a real RDS instance and AWS credentials, so it cannot run in CI.

## Prerequisites

- An Ubuntu host with a disk, Docker, and ZFS. `engine/test/_prerequisites.ubuntu.sh` installs
  Docker, ZFS, `psql`, `jq`, and `yq`; `engine/test/_zfs.file.sh` creates a file-backed pool for
  throwaway runs.
- An RDS PostgreSQL instance with IAM database authentication enabled, holding a pgbench dataset
  (the checks below drop `pgbench_accounts`).
- A database user granted `rds_iam`, and an AWS access key whose IAM policy allows
  `rds-db:connect` against that user.
- The `dblab` CLI on `PATH`.

## Variables

```bash
export TAG="${TAG:-master}"                       # dblab-server image tag to test
export EXTENDED_IMAGE_TAG="-minor-update"         # set to "" for POSTGRES_VERSION=18
export POSTGRES_VERSION=13
export DLE_TEST_MOUNT_DIR=/var/lib/test/dblab_mount
export DLE_TEST_POOL_NAME=test_dblab_pool
export DLE_SERVER_PORT=12345
export DLE_PORT_POOL_FROM=9000
export DLE_PORT_POOL_TO=9099

export SOURCE_DBNAME=test
export SOURCE_USERNAME=test_user
export AWS_REGION=us-east-2
export RDS_DB_IDENTIFIER=logical-rds-test1
export AWS_ACCESS_KEY=...
export AWS_SECRET_ACCESS_KEY=...
```

## 1. Configure

```bash
configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"
mkdir -p "${configDir}" "${metaDir}"

curl "https://gitlab.com/postgres-ai/database-lab/-/raw/${TAG}/engine/configs/config.example.logical_rds_iam.yml" \
  --output "${configDir}/server.yml"

yq eval -i '
  .global.debug = true |
  .platform.enableTelemetry = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION) + env(EXTENDED_IMAGE_TAG) |
  .retrieval.spec.logicalDump.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump" |
  .retrieval.spec.logicalDump.options.source.connection.dbname = strenv(SOURCE_DBNAME) |
  .retrieval.spec.logicalDump.options.source.connection.username = strenv(SOURCE_USERNAME) |
  .retrieval.spec.logicalDump.options.source.rdsIam.awsRegion = strenv(AWS_REGION) |
  .retrieval.spec.logicalDump.options.source.rdsIam.dbInstanceIdentifier = strenv(RDS_DB_IDENTIFIER) |
  .retrieval.spec.logicalRestore.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump"
' "${configDir}/server.yml"
```

The IAM path presents a TLS certificate that the engine must be able to verify, so fetch the RDS CA
bundle and mount it at the path the example config references:

```bash
curl https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
  --output ~/.dblab/rds-combined-ca-bundle.pem
```

## 2. Start the engine

The AWS credentials go to the engine as environment variables, not into the config file — the engine
mints a short-lived IAM token per connection.

```bash
sudo docker run \
  --name dblab_server_test \
  --label dblab_control --label dblab_test \
  --privileged \
  --publish ${DLE_SERVER_PORT}:${DLE_SERVER_PORT} \
  --volume "${configDir}":/home/dblab/configs \
  --volume "${metaDir}":/home/dblab/meta \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume ~/.dblab/rds-combined-ca-bundle.pem:/cert/rds-combined-ca-bundle.pem \
  --env AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY}" \
  --env AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
  --env DBLAB_VERIFICATION_TOKEN=secret_token \
  --detach \
  "registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
```

Wait for `.retrieving.status` to reach `finished`:

```bash
curl --silent --header 'Verification-Token: secret_token' \
  "http://localhost:${DLE_SERVER_PORT}/status" | jq -r .retrieving.status
```

A failure here is almost always one of: the user lacks `rds_iam`, the IAM policy does not cover
`rds-db:connect` for that user, or the CA bundle is not mounted where the config expects it.

## 3. Clone and verify

```bash
dblab init --environment-id=test --url="http://localhost:${DLE_SERVER_PORT}" --token=secret_token --insecure
dblab instance status
dblab clone create --username dblab_user_1 --password secret_password --id testclone
```

Check the clone came up from a clean shutdown — the newest `.csv` under
`${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/branch/main/testclone/r0/data/log` must **not** contain
`database system was not properly shut down; automatic recovery in progress`.

Then confirm the dump carried real data over and that reset restores it:

```bash
PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" \
  -c '\dt+'
PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" \
  -c 'drop table pgbench_accounts'

dblab clone reset testclone
dblab clone status testclone

PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" \
  -c '\dt+'                      # pgbench_accounts must be back

dblab clone destroy testclone
```

## 4. Shut down and check the main data directory

```bash
sudo docker stop dblab_server_test
```

The newest `.csv` under `${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/log` must end with both
`received fast shutdown request` and `database system is shut down`.

## 5. Clean up

```bash
bash engine/test/_cleanup.sh
```
