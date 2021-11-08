#!/bin/bash
set -euxo pipefail

CI_COMMIT_REF_SLUG="${CI_COMMIT_REF_SLUG:-master}" # use master branch if the CI_COMMIT_REF_SLUG is not defined

TAG=${TAG:-${CI_COMMIT_REF_SLUG}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
SOURCE_HOST="${SOURCE_HOST:-172.17.0.1}"
SOURCE_PORT="${SOURCE_PORT:-7432}"
SOURCE_USERNAME="${SOURCE_USERNAME:-postgres}"
SOURCE_PASSWORD="${SOURCE_PASSWORD:-secretpassword}"
POSTGRES_VERSION="${POSTGRES_VERSION:-13}"

DIR=${0%/*}

if [[ "${SOURCE_HOST}" = "172.17.0.1" ]]; then
### Step 0. Create source database
  sudo rm -rf "${HOME}"/postgresql/"${POSTGRES_VERSION}"/test-"${TAG}" || true
  sudo docker run \
    --name postgres"${POSTGRES_VERSION}" \
    --label pgdb \
    --privileged \
    --publish 172.17.0.1:"${SOURCE_PORT}":5432 \
    --env PGDATA=/var/lib/postgresql/pgdata \
    --env POSTGRES_USER="${SOURCE_USERNAME}" \
    --env POSTGRES_PASSWORD="${SOURCE_PASSWORD}" \
    --env POSTGRES_DB=test \
    --env POSTGRES_HOST_AUTH_METHOD=md5 \
    --volume "${HOME}"/postgresql/"${POSTGRES_VERSION}"/test-"${TAG}":/var/lib/postgresql/pgdata \
    --detach \
    postgres:"${POSTGRES_VERSION}"

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

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${TAG}"/configs/config.example.physical_generic.yml \
 --output "${configDir}/server.yml"

# Edit the following options
sed -ri "s/^(\s*)(debug:.*$)/\1debug: true/" "${configDir}/server.yml"
sed -ri '/^ *telemetry:/,/^ *[^:]*:/s/enabled: true/enabled: false/' "${configDir}/server.yml"
sed -ri "s/^(\s*)(PGUSER:.*$)/\1PGUSER: ${SOURCE_USERNAME}/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(PGPASSWORD:.*$)/\1PGPASSWORD: ${SOURCE_PASSWORD}/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(PGHOST:.*$)/\1PGHOST: ${SOURCE_HOST}/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(PGPORT:.*$)/\1PGPORT: ${SOURCE_PORT}/" "${configDir}/server.yml"
# replace postgres version
sed -ri "s/:13/:${POSTGRES_VERSION}/g"  "${configDir}/server.yml"

# logerrors is not supported in PostgreSQL 9.6
if [ "${POSTGRES_VERSION}" = "9.6" ]; then
  sed -ri 's/^(\s*)(shared_preload_libraries:.*$)/\1shared_preload_libraries: "pg_stat_statements, auto_explain"/' "${configDir}/server.yml"
fi

## Launch Database Lab server
sudo docker run \
  --name dblab_server \
  --label dblab_control \
  --privileged \
  --publish 2345:2345 \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume /var/lib/dblab:/var/lib/dblab/:rshared \
  --volume "${configDir}":/home/dblab/configs:ro \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /sys/kernel/debug:/sys/kernel/debug:rw \
  --volume /lib/modules:/lib/modules:ro \
  --volume /proc:/host_proc:ro \
  --env DOCKER_API_VERSION=1.39 \
  --detach \
  "${IMAGE2TEST}"

# Check the Database Lab Engine logs
sudo docker logs dblab_server -f 2>&1 | awk '{print "[CONTAINER dblab_server]: "$0}' &

### Waiting for the Database Lab Engine initialization.
for i in {1..30}; do
  curl http://localhost:2345 > /dev/null 2>&1 && break || echo "dblab is not ready yet"
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
  --url=http://localhost:2345 \
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
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c '\dt+'

# Drop table
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c 'drop table pgbench_accounts'

PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c '\dt+'


### Step 4. Check clone and sync instance durability on DLE restart.

## Restart DLE.
sudo docker restart dblab_server

### Waiting for the Database Lab Engine to start.
for i in {1..300}; do
  curl http://localhost:2345 > /dev/null 2>&1 && break || echo "dblab is not ready yet"
  sleep 1
done

## Reset clone.
dblab clone reset testclone

# Check the status of the clone.
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c '\dt+'

# Check the sync instance.
if [[ $(sudo docker ps --format "{{.Names}}" --filter name=^/dblab_sync) ]]; then
  for i in {1..30}; do
    # check the postgres of the sync instance
    sudo docker exec "$(sudo docker ps --format "{{.Names}}" --filter name=^/dblab_sync)" psql -U postgres -c 'select' > /dev/null 2>&1  && break || echo "postgres of the sync instance is not ready yet"
    sleep 2
  done
    # list containers
    sudo docker ps
else
  # clean up and exit with error
  source "${DIR}/_cleanup.sh"
    if [[ "${SOURCE_HOST}" = "172.17.0.1" ]]; then sudo rm -rf "$HOME"/postgresql/"${POSTGRES_VERSION}"/test-"${TAG}" || true; fi
  echo "sync instance is not running" && exit 1
fi


### Step 5. Destroy clone
dblab clone destroy testclone
dblab clone list

### Finish. clean up
source "${DIR}/_cleanup.sh"
# clean up postgres test directory
if [[ "${SOURCE_HOST}" = "172.17.0.1" ]]; then
  sudo rm -rf "$HOME"/postgresql/"${POSTGRES_VERSION}"/test-"${TAG}" || true
fi
