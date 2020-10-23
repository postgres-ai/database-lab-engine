#!/bin/bash
set -euxo pipefail

DIR=${0%/*}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:v2-0"
POSTGRES_VERSION="${POSTGRES_VERSION:-10}"
### Step 1. Prepare a machine with two disks, Docker and ZFS.

source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

### Step 2. Prepare database data directory.
sudo docker run \
  --name dblab_pg_initdb \
  --label dblab_control \
  --env PGDATA=/var/lib/dblab/data \
  --env POSTGRES_HOST_AUTH_METHOD=trust \
  --volume /var/lib/dblab:/var/lib/dblab \
  --detach \
  postgres:${POSTGRES_VERSION}-alpine

for i in {1..300}; do
  sudo docker exec -it dblab_pg_initdb psql -U postgres -c 'select' > /dev/null 2>&1  && break || echo "test database is not ready yet"
  sleep 1
done

sleep 10
sudo docker exec -it dblab_pg_initdb psql -U postgres -c 'create database test'

# 1,000,000 accounts, ~0.14 GiB of data.
sudo docker exec -it dblab_pg_initdb pgbench -U postgres -i -s 10 test

sudo docker stop dblab_pg_initdb
sudo docker rm dblab_pg_initdb

### Step 3. Configure and launch the Database Lab server.
mkdir -p ~/.dblab
cp ./configs/config.example.physical_generic.yml ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(host:.*$)/\1host: ""/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(port: 2345$)/\1port: 12345/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(debug:.*$)/\1debug: true/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(pool:.*$)/\1pool: "test_pool"/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(- physicalRestore$)/\1/' ~/.dblab/server_test.yml

# replace postgres version
sed -ri "s/:12/:${POSTGRES_VERSION}/g"  ~/.dblab/server_test.yml

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

### Waiting for the Database Lab Engine initialization.
for i in {1..30}; do
  curl http://localhost:12345 > /dev/null 2>&1 && break || echo "dblab is not ready yet"
  sleep 10
done

### Step 4. Setup Database Lab CLI.
curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/scripts/cli_install.sh | bash
dblab --version
dblab init --url http://localhost:12345 --token secret_token --environment-id test
dblab instance status

### Step 5. Create a clone and connect to it.
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
