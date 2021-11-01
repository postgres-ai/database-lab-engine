#!/bin/bash
set -euxo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
POSTGRES_VERSION="${POSTGRES_VERSION:-13}"

DIR=${0%/*}

### Step 1. Prepare a machine with disk, Docker, and ZFS
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"


### Step 2. Configure and launch the Database Lab Engine

## Prepare database data directory.
sudo docker run \
  --name dblab_pg_initdb \
  --label dblab_sync \
  --env PGDATA=/var/lib/postgresql/pgdata \
  --env POSTGRES_HOST_AUTH_METHOD=trust \
  --volume /var/lib/dblab/dblab_pool/data:/var/lib/postgresql/pgdata \
  --detach \
  postgres:"${POSTGRES_VERSION}"-alpine

for i in {1..300}; do
  sudo docker exec dblab_pg_initdb psql -U postgres -c 'select' > /dev/null 2>&1  && break || echo "test database is not ready yet"
  sleep 1
done

# Restart container explicitly after initdb to make sure that the server will not receive a shutdown request and queries will not be interrupted.
sudo docker restart dblab_pg_initdb

for i in {1..300}; do
  sudo docker exec dblab_pg_initdb psql -U postgres -c 'select' > /dev/null 2>&1  && break || echo "test database is not ready yet"
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

# Copy the contents of configuration example
mkdir -p "${configDir}"

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${TAG}"/configs/config.example.logical_generic.yml \
 --output "${configDir}/server.yml"

# Edit the following options
sed -ri 's/^(\s*)(debug:.*$)/\1debug: true/' "${configDir}/server.yml"
sed -ri '/^ *telemetry:/,/^ *[^:]*:/s/enabled: true/enabled: false/' "${configDir}/server.yml"
sed -ri 's/^(\s*)(- logicalDump$)/\1/' "${configDir}/server.yml"
sed -ri 's/^(\s*)(- logicalRestore$)/\1/' "${configDir}/server.yml"
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
  --volume /var/lib/dblab/dblab_pool/dump:/var/lib/dblab/dblab_pool/dump \
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
for i in {1..300}; do
  curl http://localhost:2345 > /dev/null 2>&1 && break || echo "dblab is not ready yet"
  sleep 1
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


### Step 4. Check clone durability on DLE restart.

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


### Step 5. Destroy clone
dblab clone destroy testclone
dblab clone list

## Stop DLE.
sudo docker stop dblab_server

### Finish. clean up
source "${DIR}/_cleanup.sh"
