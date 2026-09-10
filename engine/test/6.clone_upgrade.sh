#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

# Environment variables for replacement rules
export POSTGRES_VERSION="${POSTGRES_VERSION:-16}"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9099}
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
export DLE_TEST_POOL_NAME="test_dblab_pool"

# The clone goes one major up. Both the upgrade image, which carries pg_upgrade and the binaries of
# the two clusters, and the image the upgraded clone runs on have to exist for that target.
TARGET_VERSION=$((POSTGRES_VERSION + 1))
# Ref-scoped exactly like IMAGE2TEST above: the upgrade image carries this branch's copy of
# pg_upgrade_clone.sh, and pulling the bare major would test whichever branch pushed last.
export PG_UPGRADE_IMAGE="${PG_UPGRADE_IMAGE:-registry.gitlab.com/postgres-ai/database-lab/pg-upgrade:${TARGET_VERSION}-${TAG}}"
TARGET_CLONE_IMAGE="registry.gitlab.com/postgres-ai/custom-images/extended-postgres:${TARGET_VERSION}"

# The majors the pipeline publishes a pg-upgrade image for; it has to match the parallel matrix of
# the build-image-*-pg-upgrade jobs in .gitlab-ci.yml, which is where CI passes it in from. A target
# on this list is one the job is expected to be able to run, so a missing image fails the job
# instead of skipping it: a skip that can be triggered by an unrelated registry problem is how a
# green pipeline ends up claiming coverage the test never produced.
PG_UPGRADE_MAJORS="${PG_UPGRADE_MAJORS:-17 18}"

# Set by the CI jobs whose version pair the pipeline itself guarantees. Those jobs are the coverage
# this test exists for, so a skip in one of them is a failure: it would otherwise leave a green
# pipeline that never ran an upgrade.
UPGRADE_TEST_REQUIRED="${UPGRADE_TEST_REQUIRED:-false}"

DIR=${0%/*}

# Every exit path says what the run did with the upgrade, so the trace can be checked for it.
report_upgrade_test(){
  echo "UPGRADE_TEST_RESULT: $1 source=${POSTGRES_VERSION} target=${TARGET_VERSION}"
}

skip_unless_required(){
  report_upgrade_test "$1"
  echo "SKIP: $2"

  if [[ "${UPGRADE_TEST_REQUIRED}" == "true" ]]; then
    echo "ERROR: this job is expected to upgrade PostgreSQL ${POSTGRES_VERSION} to ${TARGET_VERSION}"
    exit 1
  fi

  exit 0
}

### Step 0. Decide whether this version pair can be upgraded at all.

# The upgrade image is built per target major and its oldest source is 12. A POSTGRES_VERSION with
# no published target has nothing to upgrade into, so the test reports that and stops rather than
# failing the job. This runs before the prerequisites because it needs nothing but the version pair.
if [[ "${POSTGRES_VERSION}" -lt 12 ]]; then
  skip_unless_required "skipped-source-too-old" \
    "PostgreSQL ${POSTGRES_VERSION} is older than the upgrade image supports"
fi

if [[ " ${PG_UPGRADE_MAJORS} " != *" ${TARGET_VERSION} "* ]]; then
  skip_unless_required "skipped-target-not-published" \
    "no pg-upgrade image is published for PostgreSQL ${TARGET_VERSION} (published: ${PG_UPGRADE_MAJORS})"
fi

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"

# The pulls belong after the prerequisites, which install Docker. Both images are expected to exist
# for a published target, so a failure here is a failure of the job.
if ! sudo docker pull "${PG_UPGRADE_IMAGE}"; then
  report_upgrade_test "failed-upgrade-image-missing"
  echo "ERROR: the upgrade image for PostgreSQL ${TARGET_VERSION} must be published: ${PG_UPGRADE_IMAGE}"
  exit 1
fi

# The engine substitutes the major in the clone's own image tag, so that image has to be pullable.
# Pulling it here also keeps the engine from having to reach the registry mid-upgrade.
if ! sudo docker pull "${TARGET_CLONE_IMAGE}"; then
  report_upgrade_test "failed-clone-image-missing"
  echo "ERROR: the clone image for PostgreSQL ${TARGET_VERSION} must be published: ${TARGET_CLONE_IMAGE}"
  exit 1
fi

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
# 100,000 accounts, enough to tell a converted cluster from an empty one.
sudo docker exec dblab_pg_initdb pgbench -U postgres -i -s 1 test

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

# Edit the following options
yq eval -i '
  .global.debug = true |
  .platform.enableTelemetry = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .provision.pgUpgradeImage = strenv(PG_UPGRADE_IMAGE) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  del(.retrieval.jobs[] | select(. == "logicalDump")) |
  del(.retrieval.jobs[] | select(. == "logicalRestore")) |
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION)
' "${configDir}/server.yml"

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

### Step 3. Create a clone and put data in it

# Install Database Lab client CLI from job artifacts
sudo cp engine/bin/cli/dblab-linux-amd64 /usr/local/bin/dblab

dblab --version

# Initialize CLI configuration
dblab init \
  --environment-id=test \
  --url=http://localhost:${DLE_SERVER_PORT} \
  --token=secret_token \
  --insecure

dblab instance status

# The instance advertises the target it derived from pgUpgradeImage. Everything below assumes the
# engine and this script agree on it, and no request carries the version any more.
ADVERTISED_TARGET=$(dblab instance status | jq -r '.cloneUpgrade.targetVersion')
if [[ "${ADVERTISED_TARGET}" != "${TARGET_VERSION}" ]]; then
  echo "ERROR: the instance must advertise PostgreSQL ${TARGET_VERSION} as the upgrade target, got ${ADVERTISED_TARGET}" && exit 1
fi

if [[ "$(dblab instance status | jq -r '.cloneUpgrade.available')" != "true" ]]; then
  echo "ERROR: a configured upgrade image must make the upgrade available" && exit 1
fi

CLONE_ID="upgradeclone"

dblab clone create \
  --username dblab_user_1 \
  --password secret_password \
  --id ${CLONE_ID}

CLONE_PORT=$(dblab clone status ${CLONE_ID} | jq -r '.db.port')

psql_clone(){
  PGPASSWORD=secret_password psql \
    "host=localhost port=${CLONE_PORT} user=dblab_user_1 dbname=test" "$@"
}

# Written after the clone was created, so it exists only in the clone: pg_upgrade has to carry it
# over, and a clone re-created from its snapshot would lose it.
psql_clone -c 'create table upgrade_probe as select 42 as answer'

ACCOUNTS_BEFORE=$(psql_clone -tAc 'select count(*) from pgbench_accounts')
SHARED_BUFFERS_BEFORE=$(psql_clone -tAc "select current_setting('shared_buffers')")
# An extension setting that only ALTER SYSTEM knows about; it must survive the carry-over filter.
psql_clone -c "alter system set pg_stat_statements.track = 'all'"

### Step 4. A rejected upgrade must leave the clone usable

# The instance decides the target, and an explicit image of another major contradicts it. The
# request has to fail, and the clone has to stay alive and claimable afterwards - a clone that a
# rejected request left unusable could never be upgraded again.
if dblab clone upgrade --docker-image "postgresai/extended-postgres:99" ${CLONE_ID}; then
  echo "ERROR: an image of another major must be rejected" && exit 1
fi

REJECTED_STATUS=$(dblab clone status ${CLONE_ID} | jq -r '.status.code')
if [[ "${REJECTED_STATUS}" == "FATAL" ]]; then
  echo "ERROR: a rejected upgrade must not break the clone, got ${REJECTED_STATUS}" && exit 1
fi

psql_clone -tAc 'select answer from upgrade_probe'

### Step 5. An upgrade that fails once it has been accepted leaves the clone running

# The target image cannot be pulled, so the upgrade is abandoned before the clone is touched. This
# is the asynchronous failure path: the request is accepted, the clone ends up in WARNING, and it
# has to stay usable and claimable - the successful upgrade in the next step is what proves the
# latter, because a clone in WARNING has to be accepted for another attempt.
CLONE_DIR="${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/branch/main/${CLONE_ID}/r0"

if dblab clone upgrade \
  --docker-image "registry.gitlab.com/postgres-ai/database-lab/pg-upgrade:0-does-not-exist" \
  ${CLONE_ID}; then
  echo "ERROR: an upgrade to an unpullable image must not report success" && exit 1
fi

FAILED_STATUS=$(dblab clone status ${CLONE_ID})
echo "${FAILED_STATUS}"

FAILED_CODE=$(echo "${FAILED_STATUS}" | jq -r '.status.code')
if [[ "${FAILED_CODE}" != "WARNING" ]]; then
  echo "ERROR: a failed upgrade must leave the clone in WARNING, got ${FAILED_CODE}" && exit 1
fi

# The version has to be named: a clone with no recorded version used to render "PostgreSQL ."
FAILED_MESSAGE=$(echo "${FAILED_STATUS}" | jq -r '.status.message')
if [[ "${FAILED_MESSAGE}" != *"still running on PostgreSQL ${POSTGRES_VERSION}"* ]]; then
  echo "ERROR: the warning must name the version the clone runs, got \"${FAILED_MESSAGE}\"" && exit 1
fi

# A clone that is left running must carry no unfinished upgrade: the state file is what makes the
# next engine start stop the clone to recover an upgrade, and here there is nothing to recover.
if sudo test -e "${CLONE_DIR}/upgrade/state.json"; then
  echo "ERROR: an abandoned upgrade must not leave its state behind" && exit 1
fi

if sudo test -e "${CLONE_DIR}/data_new"; then
  echo "ERROR: an abandoned upgrade must not leave a new data directory behind" && exit 1
fi

psql_clone -tAc 'select answer from upgrade_probe'

### Step 6. Upgrade the clone

# A stale socket lock from a killed earlier run must not block the next one: the script removes
# whatever is in its socket directory before starting pg_upgrade.
sudo mkdir -p "${CLONE_DIR}/upgrade/logs"
sudo touch "${CLONE_DIR}/upgrade/logs/.s.PGSQL.50432.lock"

dblab clone upgrade ${CLONE_ID}

CLONE_STATUS=$(dblab clone status ${CLONE_ID})
echo "${CLONE_STATUS}"

STATUS_CODE=$(echo "${CLONE_STATUS}" | jq -r '.status.code')
if [[ "${STATUS_CODE}" != "OK" ]]; then
  echo "ERROR: the upgraded clone must be OK, got ${STATUS_CODE}" && exit 1
fi

REPORTED_VERSION=$(echo "${CLONE_STATUS}" | jq -r '.dbVersion')
if [[ "${REPORTED_VERSION}" != "${TARGET_VERSION}" ]]; then
  echo "ERROR: the clone must report PostgreSQL ${TARGET_VERSION}, got ${REPORTED_VERSION}" && exit 1
fi

### Step 7. The clone runs the new major and kept every row

SERVER_VERSION=$(psql_clone -tAc "select current_setting('server_version_num')::int / 10000")
if [[ "${SERVER_VERSION}" != "${TARGET_VERSION}" ]]; then
  echo "ERROR: the clone must run PostgreSQL ${TARGET_VERSION}, got ${SERVER_VERSION}" && exit 1
fi

PROBE_ANSWER=$(psql_clone -tAc 'select answer from upgrade_probe')
if [[ "${PROBE_ANSWER}" != "42" ]]; then
  echo "ERROR: data written to the clone must survive the upgrade, got \"${PROBE_ANSWER}\"" && exit 1
fi

ACCOUNTS_AFTER=$(psql_clone -tAc 'select count(*) from pgbench_accounts')
if [[ "${ACCOUNTS_AFTER}" != "${ACCOUNTS_BEFORE}" ]]; then
  echo "ERROR: pgbench_accounts must keep ${ACCOUNTS_BEFORE} rows, got ${ACCOUNTS_AFTER}" && exit 1
fi

# databaseConfigs travel with the clone: the upgraded cluster must run the same engine-managed
# settings as before, not initdb defaults.
sudo test -f "${CLONE_DIR}/data/postgresql.dblab.snapshot.conf" || \
  (echo "ERROR: postgresql.dblab.snapshot.conf must be carried over to the upgraded cluster" && exit 1)

SHARED_BUFFERS_AFTER=$(psql_clone -tAc "select current_setting('shared_buffers')")
if [[ "${SHARED_BUFFERS_AFTER}" != "${SHARED_BUFFERS_BEFORE}" ]]; then
  echo "ERROR: shared_buffers must stay ${SHARED_BUFFERS_BEFORE} after the upgrade, got ${SHARED_BUFFERS_AFTER}" && exit 1
fi

TRACK_AFTER=$(psql_clone -tAc "select current_setting('pg_stat_statements.track')")
if [[ "${TRACK_AFTER}" != "all" ]]; then
  echo "ERROR: extension settings must survive the upgrade, pg_stat_statements.track is \"${TRACK_AFTER}\"" && exit 1
fi

### Step 8. The upgraded clone survives an engine restart

sudo docker restart ${DLE_SERVER_NAME}

for i in {1..300}; do
  check_dle_readiness && break || echo "Database Lab Engine is not ready yet"
  sleep 1
done

check_dle_readiness || (echo "Database Lab Engine is not ready" && exit 1)

RESTARTED_STATUS=$(dblab clone status ${CLONE_ID} | jq -r '.status.code')
if [[ "${RESTARTED_STATUS}" != "OK" ]]; then
  echo "ERROR: the upgraded clone must survive a restart, got ${RESTARTED_STATUS}" && exit 1
fi

psql_clone -tAc 'select answer from upgrade_probe'

### Step 9. Resetting an upgraded clone returns it to the instance version

# This is the rollback the UI and the README promise: a reset re-provisions from the origin
# snapshot on the engine-wide image, so the clone comes back on its original major and loses
# everything written since it was created.
dblab clone reset ${CLONE_ID}

RESET_CODE=$(dblab clone status ${CLONE_ID} | jq -r '.status.code')
if [[ "${RESET_CODE}" != "OK" ]]; then
  echo "ERROR: a reset clone must be OK, got ${RESET_CODE}" && exit 1
fi

# The port is kept across a reset, but the connection is a new one.
CLONE_PORT=$(dblab clone status ${CLONE_ID} | jq -r '.db.port')

RESET_VERSION=$(psql_clone -tAc "select current_setting('server_version_num')::int / 10000")
if [[ "${RESET_VERSION}" != "${POSTGRES_VERSION}" ]]; then
  echo "ERROR: a reset clone must run PostgreSQL ${POSTGRES_VERSION}, got ${RESET_VERSION}" && exit 1
fi

# The probe table was written into the clone, never into the snapshot, so it has to be gone.
PROBE_EXISTS=$(psql_clone -tAc "select to_regclass('upgrade_probe') is not null")
if [[ "${PROBE_EXISTS}" != "f" ]]; then
  echo "ERROR: a reset clone must come back from its snapshot without the clone's own table" && exit 1
fi

ACCOUNTS_AFTER_RESET=$(psql_clone -tAc 'select count(*) from pgbench_accounts')
if [[ "${ACCOUNTS_AFTER_RESET}" != "${ACCOUNTS_BEFORE}" ]]; then
  echo "ERROR: the snapshot data must survive a reset, got ${ACCOUNTS_AFTER_RESET}" && exit 1
fi

### Step 10. Destroy clone
dblab clone destroy ${CLONE_ID}
dblab clone list

report_upgrade_test "executed"

## Stop DLE.
sudo docker stop ${DLE_SERVER_NAME}

### Finish. clean up
source "${DIR}/_cleanup.sh"
