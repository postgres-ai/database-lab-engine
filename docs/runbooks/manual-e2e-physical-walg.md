# Manual e2e: physical restore from WAL-G

Verifies that the engine can build a pool from a WAL-G base backup, that it tolerates a clone whose
`postgresql.conf` raises parameters constrained by `pg_controldata`, and that both the clone and the
main data directory shut down cleanly.

This runs by hand. CI cannot: the `dle-test` jobs use the shell executor, where GitLab `services:`
are unavailable, and the procedure needs a WAL-G archive that already holds a base backup — nothing
in this repository produces one.

## Prerequisites

- An Ubuntu host with two disks, Docker, and ZFS. `engine/test/_prerequisites.ubuntu.sh` installs
  Docker, ZFS, `psql`, `jq`, and `yq`; `engine/test/_zfs.file.sh` creates a file-backed pool for
  throwaway runs.
- A WAL-G archive holding at least one base backup, in S3 or GCS, plus credentials that can read it.
- The `dblab` CLI on `PATH`.

## Variables

```bash
export TAG="${TAG:-master}"                       # dblab-server image tag to test
export POSTGRES_VERSION=13
export WALG_BACKUP_NAME=LATEST
export DLE_TEST_MOUNT_DIR=/var/lib/test/dblab_mount
export DLE_TEST_POOL_NAME=test_dblab_pool
export DLE_SERVER_PORT=12345
export DLE_PORT_POOL_FROM=9000
export DLE_PORT_POOL_TO=9099
```

S3:

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export WALG_S3_PREFIX=s3://bucket/path
```

GCS:

```bash
export WALG_GS_PREFIX=gs://bucket/path
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
```

## 1. Configure

```bash
configDir="$HOME/.dblab/engine/configs"
metaDir="$HOME/.dblab/engine/meta"
mkdir -p "${configDir}" "${metaDir}"

curl "https://gitlab.com/postgres-ai/database-lab/-/raw/${TAG}/engine/configs/config.example.physical_walg.yml" \
  --output "${configDir}/server.yml"

yq eval -i '
  .global.debug = true |
  .platform.enableTelemetry = false |
  .embeddedUI.enabled = false |
  .server.port = env(DLE_SERVER_PORT) |
  .poolManager.mountDir = env(DLE_TEST_MOUNT_DIR) |
  .provision.portPool.from = env(DLE_PORT_POOL_FROM) |
  .provision.portPool.to = env(DLE_PORT_POOL_TO) |
  .databaseContainer.dockerImage = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres:" + strenv(POSTGRES_VERSION) |
  .retrieval.spec.physicalRestore.options.walg.backupName = strenv(WALG_BACKUP_NAME) |
  .retrieval.spec.physicalRestore.options.sync.configs.shared_buffers = "512MB" |
  .retrieval.spec.physicalSnapshot.options.skipStartSnapshot = true
' "${configDir}/server.yml"
```

Then write the storage credentials into `retrieval.spec.physicalRestore.options.envs` **and**
`retrieval.spec.physicalSnapshot.options.envs`. For S3 that is `AWS_ACCESS_KEY_ID`,
`AWS_SECRET_ACCESS_KEY`, and `WALG_S3_PREFIX`; for GCS, `WALG_GS_PREFIX` and
`GOOGLE_APPLICATION_CREDENTIALS`, and delete the S3 keys the example ships with. Both stanzas need
them — restore and snapshot run as separate containers.

## 2. Start the engine

```bash
sudo docker run \
  --name dblab_server_test \
  --label dblab_control --label dblab_test \
  --privileged \
  --publish ${DLE_SERVER_PORT}:${DLE_SERVER_PORT} \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --volume ${DLE_TEST_MOUNT_DIR}:${DLE_TEST_MOUNT_DIR}/:rshared \
  --volume "${configDir}":/home/dblab/configs \
  --volume "${metaDir}":/home/dblab/meta \
  --volume /tmp:/tmp:ro \
  --env DBLAB_VERIFICATION_TOKEN=secret_token \
  --detach \
  "registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
```

Wait for `.retrieving.status` to reach `finished`:

```bash
curl --silent --header 'Verification-Token: secret_token' \
  "http://localhost:${DLE_SERVER_PORT}/status" | jq -r .retrieving.status
```

## 3. Raise the `pg_controldata`-constrained parameters

This is the part of the check worth keeping. Append to the pool's config, then restart with
snapshotting on and confirm the engine still reaches `finished` — a restore that ignores these
values fails to start the clone later.

```bash
for setting in \
  'max_connections = 300' \
  'max_prepared_transactions = 5' \
  'max_locks_per_transaction = 100' \
  'max_worker_processes = 12' \
  'track_commit_timestamp = on' \
  'max_wal_senders = 15'
do
  sudo docker exec dblab_server_test bash -c \
    "echo '${setting}' >> ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/postgresql.dblab.postgresql.conf"
done

sed -ri 's/^(\s*)(skipStartSnapshot:.*$)/\1skipStartSnapshot: false/' "${configDir}/server.yml"
sudo docker restart dblab_server_test
```

Wait for `finished` again.

## 4. Clone and verify

```bash
dblab init --environment-id=test --url="http://localhost:${DLE_SERVER_PORT}" --token=secret_token --insecure
dblab instance status
dblab clone create --username dblab_user_1 --password secret_password --id testclone
```

Check the clone came up from a clean shutdown — the newest `.csv` under
`${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/branch/main/testclone/r0/data/log` must **not** contain
`database system was not properly shut down; automatic recovery in progress`.

Confirm the raised parameter survived, then exercise reset:

```bash
PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=test" \
  -c 'show max_wal_senders'      # expect 15

PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" \
  -c 'create table test_table()'
dblab clone reset testclone
PGPASSWORD=secret_password psql "host=localhost port=${DLE_PORT_POOL_FROM} user=dblab_user_1 dbname=postgres" \
  -c '\dt+'                      # test_table must be gone

dblab clone destroy testclone
```

## 5. Shut down and check the main data directory

```bash
sudo docker stop dblab_server_test
```

The newest `.csv` under `${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/log` must end with both
`received fast shutdown request` and `database system is shut down`.

## 6. Clean up

```bash
sudo docker ps -aq --filter label="dblab_engine_name=dblab_server_test" | xargs --no-run-if-empty sudo docker rm -f
bash engine/test/_cleanup.sh
```
