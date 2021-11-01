#!/bin/bash
set -euxo pipefail

TAG="${TAG:-"master"}"
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
POSTGRES_VERSION="${POSTGRES_VERSION:-13}"
SOURCE_DBNAME="${SOURCE_DBNAME:-"test"}"
SOURCE_USERNAME="${SOURCE_USERNAME:-"test_user"}"
AWS_REGION="${AWS_REGION:-"us-east-2"}"
RDS_DB_IDENTIFIER="${RDS_DB_IDENTIFIER:-"logical-rds-test1"}"
set +euxo pipefail # ---- do not display secrets
AWS_ACCESS_KEY="${AWS_ACCESS_KEY:-""}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"
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

curl https://gitlab.com/postgres-ai/database-lab/-/raw/"${TAG}"/configs/config.example.logical_rds_iam.yml \
 --output "${configDir}/server.yml"

# Edit the following options
sed -ri "s/^(\s*)(debug:.*$)/\1debug: true/" "${configDir}/server.yml"
sed -ri '/^ *telemetry:/,/^ *[^:]*:/s/enabled: true/enabled: false/' "${configDir}/server.yml"
sed -ri "s/^(\s*)(dbname:.*$)/\1dbname: ${SOURCE_DBNAME}/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(username: test_user.*$)/\1username: \"${SOURCE_USERNAME}\"/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(awsRegion:.*$)/\1awsRegion: \"${AWS_REGION}\"/" "${configDir}/server.yml"
sed -ri "s/^(\s*)(dbInstanceIdentifier:.*$)/\1dbInstanceIdentifier: \"${RDS_DB_IDENTIFIER}\"/" "${configDir}/server.yml"
# replace postgres version
sed -ri "s/:13/:${POSTGRES_VERSION}/g"  "${configDir}/server.yml"

# Download AWS RDS certificate
curl https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
  --output ~/.dblab/rds-combined-ca-bundle.pem

## Run Database Lab Engine
sudo docker run \
  --name dblab_server \
  --label dblab_control \
  --privileged \
  --publish 2345:2345 \
  --volume "${configDir}":/home/dblab/configs:ro \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /var/lib/dblab/dblab_pool/dump:/var/lib/dblab/dblab_pool/dump \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume /var/lib/dblab:/var/lib/dblab/:rshared \
  --volume /sys/kernel/debug:/sys/kernel/debug:rw \
  --volume /lib/modules:/lib/modules:ro \
  --volume /proc:/host_proc:ro \
  --volume ~/.dblab/rds-combined-ca-bundle.pem:/cert/rds-combined-ca-bundle.pem \
  --env AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY}" \
  --env AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
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
  "host=localhost port=6000 user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

# Drop table
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c 'drop table pgbench_accounts'

PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

## Reset clone
dblab clone reset testclone

# Check the status of the clone
dblab clone status testclone

# Check the database objects (everything should be the same as when we started)
PGPASSWORD=secret_password psql \
  "host=localhost port=6000 user=dblab_user_1 dbname=${SOURCE_DBNAME}" -c '\dt+'

### Step 4. Destroy clone
dblab clone destroy testclone
dblab clone list

### Finish. clean up
source "${DIR}/_cleanup.sh"
