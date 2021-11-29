#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

# Environment variables for replacement rules
export SOURCE_HOST="${SOURCE_HOST:-172.17.0.1}"
export SOURCE_PORT="${SOURCE_PORT:-7432}"
export SOURCE_USERNAME="${SOURCE_USERNAME:-postgres}"
export SOURCE_PASSWORD="${SOURCE_PASSWORD:-secretpassword}"
export POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab"
export DLE_TEST_POOL_NAME="test_dblab_pool"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9100}

DIR=${0%/*}

if [[ "${SOURCE_HOST}" = "172.17.0.1" ]]; then
### Step 0. Create source database
  TMP_DATA_DIR="/tmp/dle_test/physical_basebackup"
  cleanup_testdata_dir() {
    sudo rm -rf "${TMP_DATA_DIR}"/postgresql/"${POSTGRES_VERSION}"/test || true
  }

  trap cleanup_testdata_dir EXIT

  cleanup_testdata_dir
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
    --volume "${TMP_DATA_DIR}"/postgresql/"${POSTGRES_VERSION}"/test:/var/lib/postgresql/pgdata \
    --detach \
    postgres:"${POSTGRES_VERSION}-alpine"

  for i in {1..300}; do
    sudo docker exec postgres"${POSTGRES_VERSION}" psql -d test -U postgres -c 'select' > /dev/null 2>&1  && break || echo "test database is not ready yet"
    sleep 1
  done

  # add "host replication" to pg_hba.conf
  sudo docker exec postgres"${POSTGRES_VERSION}" bash -c 'echo "host replication all 0.0.0.0/0 md5" >> $PGDATA/pg_hba.conf'
  # reload conf
  sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c 'select pg_reload_conf()'

  # Change wal_level and max_wal_senders parameters for PostgreSQL 9.6
  if [[ "${POSTGRES_VERSION}" = "9.6" ]]; then
    sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c 'ALTER SYSTEM SET wal_level = replica'
    sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c 'ALTER SYSTEM SET max_wal_senders = 10'
    sudo docker restart postgres"${POSTGRES_VERSION}"
    for i in {1..300}; do
      sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c 'select' > /dev/null 2>&1  && break || echo "test database is not ready yet"
      sleep 1
    done
  fi

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

# Copy the contents of configuration example 
mkdir -p "${configDir}"

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${CI_COMMIT_BRANCH:-master}"/configs/config.example.physical_generic.yml \
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
  .retrieval.spec.physicalRestore.options.envs.PGUSER = strenv(SOURCE_USERNAME) |
  .retrieval.spec.physicalRestore.options.envs.PGPASSWORD = strenv(SOURCE_PASSWORD) |
  .retrieval.spec.physicalRestore.options.envs.PGHOST = strenv(SOURCE_HOST) |
  .retrieval.spec.physicalRestore.options.envs.PGPORT = env(SOURCE_PORT) |
  .retrieval.spec.physicalSnapshot.options.envs.PGUSER = strenv(SOURCE_USERNAME) |
  .retrieval.spec.physicalSnapshot.options.envs.PGPASSWORD = strenv(SOURCE_PASSWORD) |
  .retrieval.spec.physicalSnapshot.options.envs.PGHOST = strenv(SOURCE_HOST) |
  .retrieval.spec.physicalSnapshot.options.envs.PGPORT = env(SOURCE_PORT) |
  .retrieval.spec.physicalRestore.options.customTool.command = "pg_basebackup -X stream -D " + strenv(DLE_TEST_MOUNT_DIR) + "/" + strenv(DLE_TEST_POOL_NAME) + "/data"
' "${configDir}/server.yml"

# logerrors is not supported in PostgreSQL 9.6
if [ "${POSTGRES_VERSION}" = "9.6" ]; then
  yq eval -i '.databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain"' "${configDir}/server.yml"
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
  --volume "${configDir}":/home/dblab/configs:ro \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /sys/kernel/debug:/sys/kernel/debug:rw \
  --volume /lib/modules:/lib/modules:ro \
  --volume /proc:/host_proc:ro \
  --env DOCKER_API_VERSION=1.39 \
  --detach \
  "${IMAGE2TEST}"

cleanup_service_containers() {
  sudo docker ps -aq --filter label="dblab_engine_name=${DLE_SERVER_NAME}" | xargs --no-run-if-empty sudo docker rm -f
}

trap cleanup_service_containers EXIT

# Check the Database Lab Engine logs
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &

### Waiting for the Database Lab Engine initialization.
for i in {1..30}; do
  curl http://localhost:${DLE_SERVER_PORT} > /dev/null 2>&1 && break || echo "dblab is not ready yet"
  sleep 10
done

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

## Stop control containers.
cleanup_service_containers

### Finish. clean up
source "${DIR}/_cleanup.sh"
