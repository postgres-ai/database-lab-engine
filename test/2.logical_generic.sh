#!/bin/bash
set -euxo pipefail

DIR=${0%/*}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:master"
SOURCE_DBNAME="${SOURCE_DBNAME:-test}"
SOURCE_HOST="${SOURCE_HOST:-undefined}"
SOURCE_HOST="${SOURCE_HOST:-undefined}"
SOURCE_USERNAME="${SOURCE_USERNAME:-undefined}"
SOURCE_PASSWORD="${SOURCE_PASSWORD:-undefined}"

### Step 1: Prepare a machine with two disks, Docker and ZFS

source "${DIR}/_prerequisites.ubuntu.sh"
source "${DIR}/_zfs.file.sh"

### Step 2. Prepare database data directory

### Step ?. Configure and launch the Database Lab server
mkdir -p ~/.dblab
cp ./configs/config.example.logical_generic.yml ~/.dblab/server_test.yml
sed -ri 's/^(\s\s)(port:.*$)/\1port: 12345/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(debug:.*$)/\1debug: true/' ~/.dblab/server_test.yml
sed -ri 's/^(\s*)(pool:.*$)/\1pool: "test_pool"/' ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(host: 34\.56\.78\.90$)/\1host: \"${SOURCE_HOST}\"/" ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(dbname:.*$)/\1dbname: \"${SOURCE_DBNAME}\"/" ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(username: postgres$)/\1username: \"${SOURCE_USERNAME}\"/" ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(password:.*$)/\1password: \"${SOURCE_PASSWORD}\"/" ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(parallelJobs:.*$)/\1parallelJobs: 1/" ~/.dblab/server_test.yml
sed -ri "s/^(\s*)(forceInit:.*$)/\1forceInit: true/" ~/.dblab/server_test.yml

# TMP: turn off "initialize" completely

sudo docker run \
  --detach \
  --name dblab_test \
  --label dblab_control \
  --privileged \
  --publish 12345:12345 \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume /var/lib/dblab/data:/var/lib/dblab/data:rshared \
  --volume ~/.dblab/server_test.yml:/home/dblab/configs/config.yml \
  "${IMAGE2TEST}"

### Step ?. Setup Database Lab client CLI
