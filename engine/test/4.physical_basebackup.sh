#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"
export EXTENDED_IMAGE_TAG="-minor-update" # -0.5.3

# Environment variables for replacement rules
export SOURCE_HOST="${SOURCE_HOST:-172.17.0.1}"
export SOURCE_PORT="${SOURCE_PORT:-7432}"
export SOURCE_USERNAME="${SOURCE_USERNAME:-postgres}"
export SOURCE_PASSWORD="${SOURCE_PASSWORD:-secretpassword}"
export POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
export DLE_TEST_POOL_NAME="test_dblab_pool"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9099}

if [ "${POSTGRES_VERSION}" = "18" ]; then
  EXTENDED_IMAGE_TAG=""
fi

DIR=${0%/*}

if [[ "${SOURCE_HOST}" = "172.17.0.1" ]]; then
### Step 0. Create source database
  # A fresh source cluster per run, on disk. _cleanup.sh removes the whole data tree before and after
  # every job, so nothing in it outlives the job. Not under /tmp: on the dle-test runners it is a
  # tmpfs sized at half of RAM.
  SOURCE_DATA_DIR="${DLE_TEST_DATA_DIR:-/var/lib/dle_test}/physical_basebackup/postgresql/${POSTGRES_VERSION}"
  sudo rm -rf "${SOURCE_DATA_DIR}"

  sudo docker rm postgres"${POSTGRES_VERSION}" || true

  sudo docker run \
    --name postgres"${POSTGRES_VERSION}" \
    --label pgdb \
    --label dblab_test \
    --privileged \
    --publish 172.17.0.1:"${SOURCE_PORT}":5432 \
    --env PGDATA=/var/lib/postgresql/pgdata \
    --env POSTGRES_USER="${SOURCE_USERNAME}" \
    --env POSTGRES_PASSWORD="${SOURCE_PASSWORD}" \
    --env POSTGRES_DB=test \
    --env POSTGRES_HOST_AUTH_METHOD=md5 \
    --volume "${SOURCE_DATA_DIR}":/var/lib/postgresql/pgdata \
    --detach \
    postgres:"${POSTGRES_VERSION}-alpine"

  check_database_readiness(){
    sudo docker exec postgres"${POSTGRES_VERSION}" psql -d test -U postgres -c 'select' > /dev/null 2>&1
    return $?
  }

  for i in {1..300}; do
    check_database_readiness && break || echo "test database is not ready yet"
    sleep 1
  done

  # Without this the loop above just falls through and the pg_hba.conf edit below runs against a
  # dead server, which reports the failure far from its cause.
  if ! check_database_readiness; then
    echo "ERROR: the source database is not ready" >&2
    exit 1
  fi

  # add "host replication" to pg_hba.conf
  sudo docker exec postgres"${POSTGRES_VERSION}" bash -c 'echo "host replication all 0.0.0.0/0 md5" >> $PGDATA/pg_hba.conf'
  # reload conf
  sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c 'select pg_reload_conf()'

  # Generate data in the test database using pgbench
  # 1,000,000 accounts, ~0.14 GiB of data.
  sudo docker exec postgres"${POSTGRES_VERSION}" pgbench -U postgres -i -s 10 test

  # Database info
  sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c "\l+ test"
fi

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"


### Step 2. Configure and launch the Database Lab Engine

configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"
logsDir="$HOME/.dblab/engine/logs"

# Copy the contents of configuration example 
mkdir -p "${configDir}"
mkdir -p "${logsDir}"

# Use CI_COMMIT_REF_NAME to get the original branch name, as CI_COMMIT_REF_SLUG replaces "/" with "-".
# Fallback to TAG (which is CI_COMMIT_REF_SLUG) or "master".
BRANCH_FOR_URL="${CI_COMMIT_REF_NAME:-${TAG:-master}}"
ENCODED_BRANCH_FOR_URL=$(echo "${BRANCH_FOR_URL}" | sed 's|/|%2F|g')
curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${ENCODED_BRANCH_FOR_URL}"/engine/configs/config.example.physical_generic.yml \
 --output "${configDir}/server.yml"

# Edit the following options
yq eval -i '
  .global.debug = true |
  .platform.enableTelemetry = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION) + strenv(EXTENDED_IMAGE_TAG) |
  .retrieval.spec.physicalRestore.options.envs.PGUSER = strenv(SOURCE_USERNAME) |
  .retrieval.spec.physicalRestore.options.envs.PGPASSWORD = strenv(SOURCE_PASSWORD) |
  .retrieval.spec.physicalRestore.options.envs.PGHOST = strenv(SOURCE_HOST) |
  .retrieval.spec.physicalRestore.options.envs.PGPORT = env(SOURCE_PORT) |
  .retrieval.spec.physicalRestore.options.customTool.command = "pg_basebackup -X stream -D " + strenv(DLE_TEST_MOUNT_DIR) + "/" + strenv(DLE_TEST_POOL_NAME) + "/data"
' "${configDir}/server.yml"

# Edit the following options for PostgreSQL 15
if [ "${POSTGRES_VERSION}" = "15" ]; then
  yq eval -i '
  .databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain, logerrors, pg_stat_kcache"
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
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume "${configDir}":/home/dblab/configs \
  --volume "${metaDir}":/home/dblab/meta \
  --volume "${logsDir}":/home/dblab/logs \
  --env DBLAB_VERIFICATION_TOKEN=secret_token \
  --detach \
  "${IMAGE2TEST}"

LOG_FOLLOWER_PID=""

cleanup_on_exit() {
  sudo docker ps -aq --filter label="dblab_engine_name=${DLE_SERVER_NAME}" | xargs --no-run-if-empty sudo docker rm -f
  if [[ -n "${LOG_FOLLOWER_PID}" ]]; then
    kill "${LOG_FOLLOWER_PID}" 2>/dev/null || true
  fi
}

trap cleanup_on_exit EXIT

# Check the Database Lab Engine logs. The follower inherits the job's output, which the runner
# waits on, so it has to die with the script or a failing run sits at the job timeout instead of
# failing.
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &
LOG_FOLLOWER_PID=$!

check_dle_readiness(){
  if [[ $(curl --silent --header 'Verification-Token: secret_token' --header 'Content-Type: application/json' http://localhost:${DLE_SERVER_PORT}/status | jq -r .retrieving.status) ==  "finished" ]] ; then
      return 0
  fi
  return 1
}

### Waiting for the Database Lab Engine initialization (7 minutes).
for i in {1..42}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 10
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


## Create a clone
CLONE_ID="testclone"

dblab clone create \
  --username dblab_user_1 \
  --password secret_password \
  --id ${CLONE_ID}

### Check that database system was properly shut down (clone data dir)
BRANCH_MAIN="main"
REVISION_0="r0"
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

### Step 4. Destroy clone
dblab clone destroy testclone
dblab clone list

## Stop DLE.
sudo docker stop ${DLE_SERVER_NAME}

### Check that database system was properly shut down (main data dir)
LOG_DIR="${DLE_TEST_MOUNT_DIR}"/"${DLE_TEST_POOL_NAME}"/data/log
LOG_FILE_CSV=$(sudo ls -t "$LOG_DIR" | grep .csv | head -n 1)
if sudo test -d "$LOG_DIR"
then
  if [[ $(sudo tail -n 10 "$LOG_DIR"/"$LOG_FILE_CSV" | grep -c 'received fast shutdown request\|database system is shut down') = 2 ]]
  then
      echo "INFO: database system was properly shut down - OK"
  else
      echo "ERROR: database system was not properly shut down" && exit 1
  fi
else
  echo "ERROR: the log directory \"$LOG_DIR\" does not exist" && exit 1
fi

## Stop control containers and the log follower.
cleanup_on_exit

### Finish. clean up
source "${DIR}/_cleanup.sh"
