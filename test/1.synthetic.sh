#!/bin/bash
set -euxo pipefail

DIR=${0%/*}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:master"

### Step 1: Prepare a machine with two disks, Docker and ZFS

source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

### Step 2. Prepare database data directory

sudo docker run \
  --name dblab_pg_initdb \
  --label dblab_control \
  --env PGDATA=/var/lib/postgresql/pgdata \
  --env POSTGRES_HOST_AUTH_METHOD=trust \
  --volume /var/lib/dblab/data:/var/lib/postgresql/pgdata \
  --detach \
  postgres:12-alpine

while true; do
  sudo docker exec -it dblab_pg_initdb psql -U postgres -c 'select' && break
  sleep 1
done

sudo docker exec -it dblab_pg_initdb psql -U postgres -c 'create database test'

# 1,000,000 accounts, ~0.14 GiB of data.
sudo docker exec -it dblab_pg_initdb pgbench -U postgres -i -s 10 test

sudo docker stop dblab_pg_initdb
sudo docker rm dblab_pg_initdb

### Step ?. Configure and launch the Database Lab server
mkdir -p ~/.dblab
cp ./configs/config.example.physical_generic.yml ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(port: 2345$)/\1port: 12345/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(debug:.*$)/\1debug: true/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(pool:.*$)/\1pool: "test_pool"/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(pool:.*$)/\1pool: "test_pool"/' ~/.dblab/server_test.yml

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

### Step ?. Setup Database Lab client CLI
