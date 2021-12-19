#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

# Environment variables for replacement rules
export SOURCE_DBNAME="${SOURCE_DBNAME:-test}"
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
  TMP_DATA_DIR="/tmp/dle_test/logical_generic"
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
    --env POSTGRES_DB="${SOURCE_DBNAME}" \
    --env POSTGRES_HOST_AUTH_METHOD=md5 \
    --volume "${TMP_DATA_DIR}"/postgresql/"${POSTGRES_VERSION}"/test:/var/lib/postgresql/pgdata \
    --detach \
    postgres:"${POSTGRES_VERSION}-alpine"

  check_database_readiness(){
    sudo docker exec postgres"${POSTGRES_VERSION}" psql -d "${SOURCE_DBNAME}" -U postgres -c 'select' > /dev/null 2>&1
    return $?
  }

  for i in {1..300}; do
    check_database_readiness && break || echo "test database is not ready yet"
    sleep 1
  done

  check_database_readiness || (echo "test database is not ready" && exit 1)

  # Generate data in the test database using pgbench
  # 1,000,000 accounts, ~0.14 GiB of data.
  sudo docker exec postgres"${POSTGRES_VERSION}" pgbench -U postgres -i -s 10 "${SOURCE_DBNAME}"

  # Database info
  sudo docker exec postgres"${POSTGRES_VERSION}" psql -U postgres -c "\l+ ${SOURCE_DBNAME}"
fi

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"


### Step 2. Configure and launch the Database Lab Engine

configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"

# Copy the contents of configuration example 
mkdir -p "${configDir}"

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${CI_COMMIT_BRANCH:-master}"/configs/config.example.logical_generic.yml \
 --output "${configDir}/server.yml"

# Edit the following options
yq eval -i '
  .global.debug = true |
  .global.telemetry.enabled = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .retrieval.spec.logicalDump.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump" |
  .retrieval.spec.logicalDump.options.source.connection.dbname = strenv(SOURCE_DBNAME) |
  .retrieval.spec.logicalDump.options.source.connection.host = strenv(SOURCE_HOST) |
  .retrieval.spec.logicalDump.options.source.connection.port = env(SOURCE_PORT) |
  .retrieval.spec.logicalDump.options.source.connection.username = strenv(SOURCE_USERNAME) |
  .retrieval.spec.logicalDump.options.source.connection.password = strenv(SOURCE_PASSWORD) |
  .retrieval.spec.logicalRestore.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump" |
  .databaseContainer.dockerImage = "postgresai/extended-postgres:" + strenv(POSTGRES_VERSION)
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
  --volume ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump:${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume "${configDir}":/home/dblab/configs:ro \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /sys/kernel/debug:/sys/kernel/debug:rw \
  --volume /lib/modules:/lib/modules:ro \
  --volume /proc:/host_proc:ro \
  --env DOCKER_API_VERSION=1.39 \
  --detach \
  "${IMAGE2TEST}"

# Check the Database Lab Engine logs
sudo docker logs ${DLE_SERVER_NAME} -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &

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
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

# Drop table
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c 'drop table pgbench_accounts'

PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

### Step 4. Destroy clone
dblab clone destroy testclone
dblab clone list

## Stop DLE.
sudo docker stop ${DLE_SERVER_NAME}

### Finish. clean up
source "${DIR}/_cleanup.sh"
