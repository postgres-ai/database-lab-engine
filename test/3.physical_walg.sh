#!/bin/bash
set -euxo pipefail

TAG="${TAG:-"master"}"
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

# Environment variables for replacement rules
export POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
export WALG_BACKUP_NAME="${WALG_BACKUP_NAME:-"LATEST"}"
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab"
export DLE_TEST_POOL_NAME="test_dblab_pool"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9100}
## AWS
set +euxo pipefail # ---- do not display secrets
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-""}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"
export WALG_S3_PREFIX="${WALG_S3_PREFIX:-""}"
## GS
export WALG_GS_PREFIX="${WALG_GS_PREFIX:-""}"
export GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_APPLICATION_CREDENTIALS:-""}"
# check variables
[ -z "${WALG_S3_PREFIX}" ] && [ -z "${WALG_GS_PREFIX}" ] && echo "Variables not specified" && exit 1
set -euxo pipefail # ----

DIR=${0%/*}


### Step 1: Prepare a machine with two disks, Docker and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

### Step 2. Configure and launch the Database Lab Engine

configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"

# Copy the contents of configuration example
mkdir -p "${configDir}"

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${CI_COMMIT_BRANCH:-master}"/configs/config.example.physical_walg.yml \
 --output "${configDir}/server.yml"

# Edit the following options
yq eval -i '
  .global.debug = true |
  .global.telemetry.enabled = false |
  .localUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .databaseContainer.dockerImage = "postgresai/extended-postgres:" + strenv(POSTGRES_VERSION) |
  .retrieval.spec.physicalRestore.options.walg.backupName = strenv(WALG_BACKUP_NAME) |
  .retrieval.spec.physicalRestore.options.sync.configs.shared_buffers = "512MB" |
  .retrieval.spec.physicalSnapshot.options.skipStartSnapshot = true
' "${configDir}/server.yml"

# logerrors is not supported in PostgreSQL 9.6
if [ "${POSTGRES_VERSION}" = "9.6" ]; then
  yq eval -i '.databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain"' "${configDir}/server.yml"
fi

set +euxo pipefail # ---- do not display secrets
if [ -n "${WALG_S3_PREFIX}" ] ; then
  yq eval -i '
  del(.retrieval.spec.physicalRestore.options.envs.WALG_GS_PREFIX) |
  del(.retrieval.spec.physicalRestore.options.envs.GOOGLE_APPLICATION_CREDENTIALS) |
  .retrieval.spec.physicalRestore.options.envs.AWS_ACCESS_KEY_ID = strenv(AWS_ACCESS_KEY_ID) |
  .retrieval.spec.physicalRestore.options.envs.AWS_SECRET_ACCESS_KEY = strenv(AWS_SECRET_ACCESS_KEY) |
  .retrieval.spec.physicalRestore.options.envs.WALG_S3_PREFIX = strenv(WALG_S3_PREFIX) |
  .retrieval.spec.physicalSnapshot.options.envs.AWS_ACCESS_KEY_ID = strenv(AWS_ACCESS_KEY_ID) |
  .retrieval.spec.physicalSnapshot.options.envs.AWS_SECRET_ACCESS_KEY = strenv(AWS_SECRET_ACCESS_KEY) |
  .retrieval.spec.physicalSnapshot.options.envs.WALG_S3_PREFIX = strenv(WALG_S3_PREFIX)
' "${configDir}/server.yml"

elif [ -n "${WALG_GS_PREFIX}" ] ; then
  yq eval -i '
  .retrieval.spec.physicalRestore.options.envs.WALG_GS_PREFIX = strenv(WALG_GS_PREFIX) |
  .retrieval.spec.physicalRestore.options.envs.GOOGLE_APPLICATION_CREDENTIALS = strenv(GOOGLE_APPLICATION_CREDENTIALS) |
  .retrieval.spec.physicalSnapshot.options.envs.WALG_GS_PREFIX = strenv(WALG_GS_PREFIX) |
  .retrieval.spec.physicalSnapshot.options.envs.GOOGLE_APPLICATION_CREDENTIALS = strenv(GOOGLE_APPLICATION_CREDENTIALS)
' "${configDir}/server.yml"
fi
set -euxo pipefail # ----

## Launch Database Lab server
sudo docker run \
  --name ${DLE_SERVER_NAME} \
  --label dblab_control \
  --label dblab_test \
  --privileged \
  --publish ${DLE_SERVER_PORT}:${DLE_SERVER_PORT} \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume "${configDir}":/home/dblab/configs:ro \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /sys/kernel/debug:/sys/kernel/debug:rw \
  --volume /lib/modules:/lib/modules:ro \
  --volume /proc:/host_proc:ro \
  --volume /tmp:/tmp:ro \
  --env DOCKER_API_VERSION=1.39 \
  --detach \
  "${IMAGE2TEST}"

cleanup_service_containers() {
  sudo docker ps -aq --filter label="dblab_engine_name=${DLE_SERVER_NAME}" | xargs --no-run-if-empty sudo docker rm -f
}

trap cleanup_service_containers EXIT

# Check the Database Lab Engine logs
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &

check_dle_readiness(){
  curl http://localhost:${DLE_SERVER_PORT} > /dev/null 2>&1
  return $?
}

### Waiting for the Database Lab Engine initialization.
for i in {1..30}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 10
done

check_dle_readiness || (echo "Database Lab Engine is not ready" && exit 1)

# Test increasing default configuration parameters from pg_controldata. If the Database Lab Engine will start successfully, the test is passed.
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo -e '\nmax_connections = 300' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo 'max_prepared_transactions = 5' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo 'max_locks_per_transaction = 100' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo 'max_worker_processes = 12' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo 'track_commit_timestamp = on' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
sudo docker exec ${DLE_SERVER_NAME} bash -c "echo 'max_wal_senders = 15' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"

# Enable snapshotting on start to make a new snapshot
sed -ri "s/^(\s*)(skipStartSnapshot:.*$)/\1skipStartSnapshot: false/" "${configDir}/server.yml"

sudo docker restart ${DLE_SERVER_NAME}

# Check the Database Lab Engine logs
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &

### Waiting for the Database Lab Engine initialization.
for i in {1..30}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 10
done

check_dle_readiness || (echo "Database Lab Engine is not ready" && exit 1)

### Step 3. Start cloning

# Install Database Lab client CLI
curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/scripts/cli_install.sh | bash
sudo mv ~/.dblab/dblab /usr/local/bin/dblab

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
dblab clone create \
  --username dblab_user_1 \
  --password secret_password \
  --id testclone

PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" -c 'show max_wal_senders'

# Connect to a clone and check the available table
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" -c '\dt+'

# Create table
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" -c 'create table test_table()'

PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" -c '\dt+'

### Step 4. Destroy clone
dblab clone destroy testclone
dblab clone list

## Stop DLE.
sudo docker stop ${DLE_SERVER_NAME}

## Stop control containers.
cleanup_service_containers

### Finish. clean up
source "${DIR}/_cleanup.sh"
