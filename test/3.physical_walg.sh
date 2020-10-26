#!/bin/bash
set -euxo pipefail

DIR=${0%/*}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:v2-0"
POSTGRES_VERSION="${POSTGRES_VERSION:-10}"
### Step 1: Prepare a machine with two disks, Docker and ZFS

source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

### Step 2. Prepare database data directory

### Step ?. Configure and launch the Database Lab server
mkdir -p ~/.dblab
cp ./configs/config.example.physical_walg.yml ~/.dblab/server_test.yml
sed -ri 's/^(\s\s)(port:.*$)/\1port: 12345/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(debug:.*$)/\1debug: true/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(pool:.*$)/\1pool: "test_pool"/' ~/.dblab/server_test.yml
# set AWS config
sed -ri 's/^(\s*)(credentialsFile:.*$)/\1#credentialsFile/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(WALG_GS_PREFIX:.*$)/\1WALE_S3_PREFIX: "s3:\/\/dblab-test-database-backup" \n          AWS_ACCESS_KEY_ID: "A"\n          AWS_SECRET_ACCESS_KEY: "T\/2nOe"/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(storage:.*$)/\1storage: "s3"/' ~/.dblab/server_test.yml
# replace postgres version 
sed -ri 's/:12/:11/g'  ~/.dblab/server_test.yml

### Step ? Run dblab
sudo docker run \
  --detach \
  --name dblab_test \
  --label dblab_control \
  --privileged \
  --publish 12345:12345 \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume /var/lib/dblab:/var/lib/dblab:rshared \
  --volume ~/.dblab/server_test.yml:/home/dblab/configs/config.yml \
  "${IMAGE2TEST}"

sudo docker logs -f dblab_test 2>&1 | awk '{print "[CONTAINER dblab_test]: "$0}' &

### Waiting fori dblab initialization
for i in {1..30}; do
  curl http://localhost:12345 > /dev/null 2>&1 && break || echo "dblab is not ready yet"
  sleep 10
done

### Step ?. Setup Dnd init atabase Lab client CLI
curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/scripts/cli_install.sh | bash
dblab --version
dblab init --url http://localhost:12345 --token secret_token --environment-id test
dblab instance status

### Step ?. Create clone and connect to it
dblab clone create --username testuser --password testuser --id testclone
dblab clone list
export PGPASSWORD=testuser
psql "host=localhost port=6000 user=testuser dbname=test" -c '\l'


### Step 6. Reset clone
psql "host=localhost port=6000 user=testuser dbname=test" -c 'create database reset_database';
psql "host=localhost port=6000 user=testuser dbname=test" -c '\l'
dblab clone reset testclone
dblab clone status testclone
psql "host=localhost port=6000 user=testuser dbname=test" -c '\l'
dblab clone destroy testclone

### Step 7. Destroy clone
dblab clone create --username testuser --password testuser --id testclone2
dblab clone list
dblab clone destroy testclone2
dblab clone list
