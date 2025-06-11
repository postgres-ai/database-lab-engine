#!/bin/bash
set -euxo pipefail

TAG="${TAG:-"master"}"
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"
export EXTENDED_IMAGE_TAG="-minor-update" # -0.5.3

# Environment variables for replacement rules
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
export DLE_TEST_POOL_NAME="test_dblab_pool"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9099}
export POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
export SOURCE_DBNAME="${SOURCE_DBNAME:-"test"}"
export SOURCE_USERNAME="${SOURCE_USERNAME:-"test_user"}"
export AWS_REGION="${AWS_REGION:-"us-east-2"}"
export RDS_DB_IDENTIFIER="${RDS_DB_IDENTIFIER:-"logical-rds-test1"}"
set +euxo pipefail # ---- do not display secrets
export AWS_ACCESS_KEY="${AWS_ACCESS_KEY:-""}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"
set -euxo pipefail # ----

DIR=${0%/*}

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"


### Step 2. Configure and launch the Database Lab Engine

configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"

# Copy the contents of configuration example 
mkdir -p "${configDir}"

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${TAG:-master}"/engine/configs/config.example.logical_rds_iam.yml \
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
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION) + env(EXTENDED_IMAGE_TAG) |
  .retrieval.spec.logicalDump.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump" |
  .retrieval.spec.logicalDump.options.source.connection.dbname = strenv(SOURCE_DBNAME) |
  .retrieval.spec.logicalDump.options.source.connection.username = strenv(SOURCE_USERNAME) |
  .retrieval.spec.logicalDump.options.source.rdsIam.awsRegion = strenv(AWS_REGION) |
  .retrieval.spec.logicalDump.options.source.rdsIam.dbInstanceIdentifier = strenv(RDS_DB_IDENTIFIER) |
  .retrieval.spec.logicalRestore.options.dumpLocation = env(DLE_TEST_MOUNT_DIR) + "/" + env(DLE_TEST_POOL_NAME) + "/dump"
' "${configDir}/server.yml"

# Edit the following options for PostgreSQL 9.6
if [ "${POSTGRES_VERSION}" = "9.6" ]; then
  yq eval -i '
  .databaseConfigs.configs.shared_preload_libraries = "pg_stat_statements, auto_explain" |
  .databaseConfigs.configs.log_directory = "log"
  ' "${configDir}/server.yml"
fi

# Download AWS RDS certificate
curl https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
  --output ~/.dblab/rds-combined-ca-bundle.pem

## Run Database Lab Engine
sudo docker run \
  --name ${DLE_SERVER_NAME} \
  --label dblab_control \
  --label dblab_test \
  --privileged \
  --publish ${DLE_SERVER_PORT}:${DLE_SERVER_PORT} \
  --volume "${configDir}":/home/dblab/configs \
  --volume "${metaDir}":/home/dblab/meta \
  --volume ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump:${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume ~/.dblab/rds-combined-ca-bundle.pem:/cert/rds-combined-ca-bundle.pem \
  --env AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY}" \
  --env AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
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
for i in {1..30}; do
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

### Finish. clean up
source "${DIR}/_cleanup.sh"
