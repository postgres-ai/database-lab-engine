#!/bin/bash
set -euxo pipefail

DIR=${0%/*}
source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

sudo docker run \
  --name dblab_pg_initdb \
  --label dblab_sync \
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
