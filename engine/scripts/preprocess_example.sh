#!/bin/bash
# Use this as a template for any additional data transformations
# during automated snapshot preparation. Some examples:
#   - rename the database to avoid confusions when working
#     with clones (this example)
#   - personal data (PII) removal / anonymization
#   - dynamic data masking

# Now (version 2.0 of DLE), to run an SQL query, you need to start
# Postgres container and then remove it. This is something that is
# to be automated in future versions.

set -eo pipefail # todo: add a trap function for cleanup (remove container, etc)

PGVERSION="13"

# Cleanup might be needed if the previous attempt has failed
docker rm -f dblab_preprocess_tmp || echo 'nothing to clean up'

# Determine the latest clone
clone=$(zfs list -S dblab:datastateat -t all | grep dblab_pool | grep clone_pre | tail -1 | awk '{print $5}')
clone="${clone##*/}"

# This example for PGDATA objtaned by WAL-G's backup-fetch and GCS
# -- adjust as needed
docker run \
  --name dblab_preprocess_tmp \
  --label dblab_control \
  --detach \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume /var/lib/dblab/clones/$clone:/var/lib/dblab:rshared \
  --volume /etc/wal-g.d/gcs.json:/tmp/sa.json \
  --env PGDATA=/var/lib/dblab/postgresql/data \
  --env WALG_GS_PREFIX="gs://___bucket___/___folder___" \
  --env GOOGLE_APPLICATION_CREDENTIALS="/tmp/sa.json" \
  --env WALG_DOWNLOAD_CONCURRENCY=16 \
  postgresai/sync-instance:$PGVERSION

# Start Postgres, explicitly waiting
docker exec dblab_preprocess_tmp \
  bash -c "chown -R postgres:postgres /var/lib/dblab"
docker exec dblab_preprocess_tmp \
  bash -c "su - postgres -c \"/usr/lib/postgresql/11/bin/pg_ctl --wait --timeout=1800  -D /var/lib/dblab/postgresql/data/  start\""

# Execute any SQL – in this case, rename the target DB
docker exec dblab_preprocess_tmp \
  psql -U gitlab-superuser postgres \
    -c 'ALTER DATABASE example_production RENAME TO example_dblab'

# Stop Postgres, explicitly waiting
docker exec dblab_preprocess_tmp \
  bash -c "su - postgres -c \"/usr/lib/postgresql/11/bin/pg_ctl --wait --timeout=1800  -D /var/lib/dblab/postgresql/data/  stop\""

# Clean up
docker stop dblab_preprocess_tmp
docker rm -f dblab_preprocess_tmp
