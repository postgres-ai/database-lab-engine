#!/bin/bash
set -euxo pipefail

TAG="${TAG:-"master"}"
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
POSTGRES_VERSION="${POSTGRES_VERSION:-13}"

WALG_BACKUP_NAME="${WALG_BACKUP_NAME:-"LATEST"}"
# AWS
set +euxo pipefail # ---- do not display secrets
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-""}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"
WALG_S3_PREFIX="${WALG_S3_PREFIX:-""}"
# GS
WALG_GS_PREFIX="${WALG_GS_PREFIX:-""}"
GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_APPLICATION_CREDENTIALS:-""}"
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

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${TAG}"/configs/config.example.physical_walg.yml \
 --output "${configDir}/server.yml"

# Edit the following options
sed -ri "s/^(\s*)(debug:.*$)/\1debug: true/" "${configDir}/server.yml"
sed -ri '/^ *telemetry:/,/^ *[^:]*:/s/enabled: true/enabled: false/' "${configDir}/server.yml"
# set WAL-G envs
sed -ri "s/^(\s*)(backupName:.*$)/\1backupName: ${WALG_BACKUP_NAME}/" "${configDir}/server.yml"
set +euxo pipefail # ---- do not display secrets
if [ -n "${WALG_S3_PREFIX}" ] ; then
sed -ri "s/^(\s*)(WALG_GS_PREFIX:.*$)/\1AWS_ACCESS_KEY_ID: \"${AWS_ACCESS_KEY_ID}\" \n          AWS_SECRET_ACCESS_KEY: \"${AWS_SECRET_ACCESS_KEY}\"\n          WALG_S3_PREFIX: \"${WALG_S3_PREFIX}\"/" "${configDir}/server.yml"
sed -i "/GOOGLE_APPLICATION_CREDENTIALS/d" "${configDir}/server.yml"
elif [ -n "${WALG_GS_PREFIX}" ] ; then
sed -ri "s/^(\s*)(WALG_GS_PREFIX:.*$)/\1WALG_GS_PREFIX: \"${WALG_GS_PREFIX}\"/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(GOOGLE_APPLICATION_CREDENTIALS:.*$)/\1GOOGLE_APPLICATION_CREDENTIALS: \"${GOOGLE_APPLICATION_CREDENTIALS}\"/" "${configDir}/server.yml"
fi
set -euxo pipefail # ----
# replace postgres version
sed -ri "s/:13/:${POSTGRES_VERSION}/g"  "${configDir}/server.yml"
# reduce shared_buffers (optional)
sed -ri "s/^(\s*)(shared_buffers:.*$)/\1shared_buffers: 512MB/" "${configDir}/server.yml"
# skip snapshotting on start to replace some Postgres parameters after PGDATA receiving
sed -ri "s/^(\s*)(skipStartSnapshot:.*$)/\1skipStartSnapshot: true/" "${configDir}/server.yml"

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

# Test increasing default configuration parameters from pg_controldata. If the Database Lab Engine will start successfully, the test is passed.
sudo docker exec -it dblab_server bash -c 'echo -e "\nmax_connections = 300" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'
sudo docker exec -it dblab_server bash -c 'echo "max_prepared_transactions = 5" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'
sudo docker exec -it dblab_server bash -c 'echo "max_locks_per_transaction = 100" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'
sudo docker exec -it dblab_server bash -c 'echo "max_worker_processes = 12" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'
sudo docker exec -it dblab_server bash -c 'echo "track_commit_timestamp = on" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'
sudo docker exec -it dblab_server bash -c 'echo "max_wal_senders = 15" >> /var/lib/dblab/dblab_pool/data/postgresql.dblab.postgresql.conf'

# Enable snapshotting on start to make a new snapshot
sed -ri "s/^(\s*)(skipStartSnapshot:.*$)/\1skipStartSnapshot: false/" "${configDir}/server.yml"

sudo docker restart dblab_server

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

PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=test" -c 'show max_wal_senders'

# Connect to a clone and check the available table
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=postgres" -c '\dt+'

# Create table
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=postgres" -c 'create table test_table()'

PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=postgres" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=postgres" -c '\dt+'

### Step 4. Destroy clone
dblab clone destroy testclone
dblab clone list

### Finish. clean up
source "${DIR}/_cleanup.sh"
