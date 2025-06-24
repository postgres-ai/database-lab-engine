#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

# Environment variables for replacement rules
export POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9099}
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
export DLE_TEST_POOL_NAME="test_dblab_pool"

DIR=${0%/*}

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"


### Step 2. Configure and launch the Database Lab Engine

## Prepare database data directory.
sudo docker rm dblab_pg_initdb || true

sudo docker run \
  --name dblab_pg_initdb \
  --label dblab_sync \
  --label dblab_test \
  --env PGDATA=/var/lib/postgresql/pgdata \
  --env POSTGRES_HOST_AUTH_METHOD=trust \
  --volume ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data:/var/lib/postgresql/pgdata \
  --detach \
  postgres:"${POSTGRES_VERSION}"-alpine

check_database_readiness(){
  sudo docker exec dblab_pg_initdb psql -U postgres -c 'select' > /dev/null 2>&1
  return $?
}

for i in {1..300}; do
  check_database_readiness && break || echo "test database is not ready yet"
  sleep 1
done

# Restart container explicitly after initdb to make sure that the server will not receive a shutdown request and queries will not be interrupted.
sudo docker restart dblab_pg_initdb

for i in {1..300}; do
  check_database_readiness && break || echo "test database is not ready yet"
  sleep 1
done

# Create the test database
sudo docker exec dblab_pg_initdb psql -U postgres -c 'create database test'

# Generate data in the test database using pgbench
# 1,000,000 accounts, ~0.14 GiB of data.
sudo docker exec dblab_pg_initdb pgbench -U postgres -i -s 10 test

# Stop and remove the container
sudo docker stop dblab_pg_initdb
sudo docker rm dblab_pg_initdb

configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"
logsDir="$HOME/.dblab/engine/logs"

# Copy the contents of configuration example
mkdir -p "${configDir}"
mkdir -p "${metaDir}"
mkdir -p "${logsDir}"

# Use CI_COMMIT_REF_NAME to get the original branch name, as CI_COMMIT_REF_SLUG replaces "/" with "-".
# Fallback to TAG (which is CI_COMMIT_REF_SLUG) or "master".
BRANCH_FOR_URL="${CI_COMMIT_REF_NAME:-${TAG:-master}}"
ENCODED_BRANCH_FOR_URL=$(echo "${BRANCH_FOR_URL}" | sed 's|/|%2F|g')
curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${ENCODED_BRANCH_FOR_URL}"/engine/configs/config.example.logical_generic.yml \
 --output "${configDir}/server.yml"

# TODO: replace the dockerImage tag back to 'postgresai/extended-postgres' after releasing a new version with custom port and unix socket dir.
# Edit the following options
yq eval -i '
  .global.debug = true |
  .platform.enableTelemetry = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  del(.retrieval.jobs[] | select(. == "logicalDump")) |
  del(.retrieval.jobs[] | select(. == "logicalRestore")) |
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION)
' "${configDir}/server.yml"

# Edit the following options for PostgreSQL 9.6
if [ "${POSTGRES_VERSION}" = "9.6" ]; then
  yq eval -i '
  .databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain" |
  .databaseConfigs.configs.log_directory = "log"
  ' "${configDir}/server.yml"
fi

# Edit the following options for PostgreSQL 15
if [ "${POSTGRES_VERSION}" = "15" ]; then
  yq eval -i '
  .databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain, logerrors, pg_stat_kcache" |
  .databaseConfigs.configs.log_directory = "log"
  ' "${configDir}/server.yml"
fi

## Launch Database Lab server
sudo docker run \
  --name ${DLE_SERVER_NAME} \
  --label dblab_control \
  --label dblab_test \
  --privileged \
  --publish ${DLE_SERVER_PORT}:${DLE_SERVER_PORT} \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump:${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume "${configDir}":/home/dblab/configs \
  --volume "${metaDir}":/home/dblab/meta \
  --volume "${logsDir}":/home/dblab/logs \
  --env DOCKER_API_VERSION=1.39 \
  --detach \
  "${IMAGE2TEST}"

# Check the Database Lab Engine logs
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER ${DLE_SERVER_PORT}]: "$0}' &

check_dle_readiness(){
  if [[ $(curl --silent --header 'Verification-Token: secret_token' --header 'Content-Type: application/json' http://localhost:${DLE_SERVER_PORT}/status | jq -r .retrieving.status) ==  "finished" ]] ; then
      return 0
  fi
  return 1
}

### Waiting for the Database Lab Engine initialization.
for i in {1..300}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 1
done

check_dle_readiness || (echo "Database Lab Engine is not ready" && exit 1)

### Step 3. Start cloning

# Install Database Lab client CLI from job artifacts
sudo cp engine/bin/cli/dblab-linux-amd64 /usr/local/bin/dblab

dblab --version

# Initialize CLI configuration
dblab init \
  --environment-id=test \
  --url=http://localhost:${DLE_SERVER_PORT} \
  --token=secret_token \
  --insecure

# Check the configuration by fetching the status of the instance:
dblab instance status

# Check the snapshot list
if [[ $(dblab snapshot list | jq length) -eq 0 ]] ; then
  echo "No snapshot found" && exit 1
fi

## Create a clone
CLONE_ID="testclone"

dblab clone create \
  --username dblab_user_1 \
  --password secret_password \
  --id ${CLONE_ID}

### Check that database system was properly shut down (clone data dir)
BRANCH_MAIN="main"
REVISION_0="r0"
# /var/lib/test/dblab_mount/test_dblab_pool/branch/main/testclone/r0
CLONE_LOG_DIR="${DLE_TEST_MOUNT_DIR}"/"${DLE_TEST_POOL_NAME}"/branch/"${BRANCH_MAIN}"/"${CLONE_ID}"/"${REVISION_0}"/data/log
LOG_FILE_CSV=$(sudo ls -t "$CLONE_LOG_DIR" | grep .csv | head -n 1)
if sudo test -d "$CLONE_LOG_DIR"
then
  if sudo grep -q 'database system was not properly shut down; automatic recovery in progress' "$CLONE_LOG_DIR"/"$LOG_FILE_CSV"
  then
      echo "ERROR: database system was not properly shut down" && exit 1
  else
      echo "INFO: database system was properly shut down - OK"
  fi
else
  echo "ERROR: the log directory \"$CLONE_LOG_DIR\" does not exist" && exit 1
fi

# Connect to a clone and check the available table
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c '\dt+'

# Drop table
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c 'drop table pgbench_accounts'

PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c '\dt+'


### Step 4. Check clone durability on DLE restart.

## Restart DLE.
sudo docker restart ${DLE_SERVER_NAME}

### Waiting for the Database Lab Engine initialization.
for i in {1..300}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 1
done

check_dle_readiness || (echo "Database Lab Engine is not ready" && exit 1)

## Reset clone.
dblab clone reset testclone

# Check the status of the clone.
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c '\dt+'


### Step 5. Destroy clone
dblab clone destroy testclone
dblab clone list

### Data branching.
dblab branch || (echo "Failed when data branching is not initialized" && exit 1)
dblab branch 001-branch || (echo "Failed to create a data branch" && exit 1)
dblab branch

dblab clone create \
  --username john \
  --password secret_test_123 \
  --branch 001-branch \
  --id branchclone001 || (echo "Failed to create a clone on branch" && exit 1)

dblab commit --clone-id branchclone001 --message branchclone001 || (echo "Failed to create a snapshot" && exit 1)

dblab clone create \
  --username alice \
  --password secret_password_123 \
  --branch 001-branch \
  --id branchclone002 || (echo "Failed to create a clone on branch" && exit 1)

dblab commit --clone-id branchclone002 -m branchclone002 || (echo "Failed to create a snapshot" && exit 1)

dblab log 001-branch || (echo "Failed to show branch history" && exit 1)

dblab clone destroy branchclone001 || (echo "Failed to destroy clone" && exit 1)
dblab clone destroy branchclone002 || (echo "Failed to destroy clone" && exit 1)

sudo docker wait branchclone001 branchclone002 || echo "Clones have been removed"

dblab clone list
dblab snapshot list

dblab switch main

dblab clone create \
  --username alice \
  --password secret_password_123 \
  --branch 001-branch \
  --id branchclone003 || (echo "Failed to create a clone on branch" && exit 1)

dblab commit --clone-id branchclone003 --message branchclone001 || (echo "Failed to create a snapshot" && exit 1)

dblab snapshot delete "$(dblab snapshot list | jq -r .[0].id)" || (echo "Failed to delete a snapshot" && exit 1)

dblab clone destroy branchclone003 || (echo "Failed to destroy clone" && exit 1)

dblab branch --delete 001-branch || (echo "Failed to delete data branch" && exit 1)

dblab branch

## Stop DLE.
sudo docker stop ${DLE_SERVER_NAME}

### Finish. clean up
source "${DIR}/_cleanup.sh"
