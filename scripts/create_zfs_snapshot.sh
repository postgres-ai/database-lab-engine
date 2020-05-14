#!/bin/bash
# 2019 © Postgres.ai

set -euxo pipefail

# Script for manual creation of ZFS snapshot from PG replica instance.
# Default values provided for Ubuntu FS layout.

# Utils.
pre="_pre"
ntries=1000
now=$(date +%Y%m%d%H%M%S)

# Storage configuration.
# Name of the ZFS pool which contains PGDATA.
zfs_pool=${ZFS_POOL:-"dblab_pool"}
# Default PGDATA directory.
pgdata_dir=${PGDATA_DIR:-"/var/lib/dblab/data"}
# Subdirectory relative to a ZFS pool in which PGDATA is located with ending "/".
# For example, if your ZFS pool configured in `/var/lib/dblab/data` and PGDATA located in `/var/lib/dblab/data/subdir` use `export PGDATA_SUBDIR="/subdir/"`.
pgdata_subdir=${PGDATA_SUBDIR:-""}

# Clone configuration.
# Mount directory for DB Lab clones.
mount_dir=${MOUNT_DIR:-"/var/lib/dblab/clones"}
# Name of a clone which will be created and used for PGDATA manipulation.
clone_name="clone${pre}_${now}"
# Full name of the clone for ZFS commands.
clone_full_name="${zfs_pool}/${clone_name}"
# Clone mount directory.
clone_dir="${mount_dir}/${clone_name}"
# Directory of PGDATA in the clone mount.
clone_pgdata_dir="${clone_dir}${pgdata_subdir}"
# Directory of PGDATA in the promote container.
container_pgdata_dir="/var/lib/postgresql/pgdata"

# Postgres configuration.
# Port on which Postgres will be started using clone's PGDATA.
clone_port=${CLONE_PORT:-5432}
pg_sock_dir=${PG_SOCK_DIR:-"/var/run/postgresql"}
pg_username=${PGUSERNAME:-"postgres"}
# Set password with PGPASSWORD env.
pg_db=${PGDB:-"postgres"}
sudo_cmd=${SUDO_CMD:-""} # Use `sudo -u postgres` for default environment

# Snapshot.
# Name of resulting snapshot after PGDATA manipulation.
snapshot_name="snapshot_${now}"

# TODO: decide: do we need to stop the shadow Postgres instance?
# OR: we can tell the shadow Postgres: select pg_start_backup('database-lab-snapshot');
# .. and in the very end: select pg_stop_backup();

# config
pgdata_dir="/var/lib/dblab/demo/data"

# If you have a running sync instance, uncomment this line before getting a snapshot.
# sudo docker stop sync-instance
sudo chown -R postgres ${pgdata_dir}

# Start snapshot creation.
sudo zfs snapshot ${zfs_pool}@${snapshot_name}${pre}
sudo zfs clone ${zfs_pool}@${snapshot_name}${pre} ${clone_full_name} -o mountpoint=${clone_dir}

# Destroy zfs clone, snapshot and clone directory.
destroy_zfs_clone() {
  sudo zfs destroy -r ${clone_full_name}
  sudo zfs destroy -r ${zfs_pool}@${snapshot_name}${pre}
  sudo rm -rf ${clone_dir}

  start_sync_instance
}

# Start a sync instance after getting a snapshot.
start_sync_instance() {
  # Remember to resume a sync instance.
  # sudo docker start sync-instance
  echo >&2 "The sync instance can be started now."
}

cd /tmp # To avoid errors about lack of permissions.

pg_ver=$(${sudo_cmd} cat ${clone_pgdata_dir}/PG_VERSION | cut -f1 -d".")

${sudo_cmd} bash -f - <<SH
set -ex

rm -rf ${clone_pgdata_dir}/postmaster.pid # Questionable -- it's better to have snapshot created with Postgres being down

# We do not want to deal with postgresql.conf symlink (if any)
cat ${clone_pgdata_dir}/postgresql.conf > ${clone_pgdata_dir}/postgresql_real.conf
chmod 600 ${clone_pgdata_dir}/postgresql_real.conf
rm ${clone_pgdata_dir}/postgresql.conf
mv ${clone_pgdata_dir}/postgresql_real.conf ${clone_pgdata_dir}/postgresql.conf

### ADJUST CONFIGS ###
### postgresql.conf
# TODO: why do we use absolute paths here?
sed -i 's/^\\(.*data_directory\\)/# \\1/' ${clone_pgdata_dir}/postgresql.conf
sed -i 's/^\\(.*hba_file\\)/# \\1/' ${clone_pgdata_dir}/postgresql.conf
sed -i 's/^\\(.*external_pid_file\\)/# \\1/' ${clone_pgdata_dir}/postgresql.conf
sed -i 's/^\\(.*ident_file\\)/# \\1/' ${clone_pgdata_dir}/postgresql.conf
sed -i 's/^\\(.*archive_command\\)/# \\1/' ${clone_pgdata_dir}/postgresql.conf

# Turn off the replication.
sed -i '/restore_command/s/^#*/#/' ${clone_pgdata_dir}/postgresql.conf
sed -i '/recovery_target_timeline/s/^#*/#/' ${clone_pgdata_dir}/postgresql.conf

# TODO: Improve security aspects.
echo "listen_addresses = '*'" >> ${clone_pgdata_dir}/postgresql.conf
echo "unix_socket_directories = '${pg_sock_dir}'" >> ${clone_pgdata_dir}/postgresql.conf

echo "log_destination = 'stderr'" >> ${clone_pgdata_dir}/postgresql.conf
echo "log_connections = on" >> ${clone_pgdata_dir}/postgresql.conf

# detect idle clones
echo "log_min_duration_statement = 0" >> ${clone_pgdata_dir}/postgresql.conf
echo "log_statement = 'none'" >> ${clone_pgdata_dir}/postgresql.conf
echo "log_timezone = 'Etc/UTC'" >> ${clone_pgdata_dir}/postgresql.conf
echo "log_line_prefix = '%m [%p]: [%l-1] db=%d,user=%u (%a,%h) '" >> ${clone_pgdata_dir}/postgresql.conf

### Replication mode
if [[ "${pg_ver}" -ge 12 ]]; then
  ## Use signal files
  echo ">=12"
  touch ${clone_pgdata_dir}/standby.signal
else
  # Use recovery.conf
  echo "<12"
  echo "standby_mode = 'on'" > ${clone_pgdata_dir}/recovery.conf # overriding
  echo "primary_conninfo = ''" >> ${clone_pgdata_dir}/recovery.conf
  echo "restore_command = ''" >> ${clone_pgdata_dir}/recovery.conf
fi;

### pg_hba.conf
echo "local all all trust" > ${clone_pgdata_dir}/pg_hba.conf
echo "host all all 0.0.0.0/0 md5" >> ${clone_pgdata_dir}/pg_hba.conf

### pg_ident.conf
echo "" > ${clone_pgdata_dir}/pg_ident.conf
SH

echo >&2 "Run container"

# Make sure that PGDATA has correct permissions.
user_owner=$(sudo ls -ld ${clone_pgdata_dir}/PG_VERSION | awk '{print $3}')
group_owner=$(sudo ls -ld ${clone_pgdata_dir}/PG_VERSION | awk '{print $4}')
sudo chown -R ${user_owner}:${group_owner} ${clone_pgdata_dir}

# If needed, use a custom Docker image (export PROMOTING_DOCKER_IMAGE="custom-image") to run the promotion container.
docker_image=${PROMOTING_DOCKER_IMAGE:-postgres:${pg_ver}-alpine}
container_name="dblab_promote"

sudo docker run \
  --name ${container_name} \
  --label dblab_control \
  --restart on-failure \
  --volume ${clone_pgdata_dir}:${container_pgdata_dir} \
  --env PGDATA=${container_pgdata_dir} \
  --user postgres \
  --detach \
  ${docker_image}

# Now we are going to wait until we can connect to the server.
# If it was a replica, it may take a while..
# During that period, we will have "FATAL:  the database system is starting up".
# Alternatively, we could use pg_ctl's "-w" option above (instead of manual checking).

failed=true
for i in {1..1000}; do
  if [[ $(sudo docker exec ${container_name} psql -p ${clone_port} -U ${pg_username} -d ${pg_db} -h ${pg_sock_dir} -XAtc 'select 1') == "1" ]]; then
    failed=false
    break
  fi

  sleep 2
done

if $failed; then
  echo >&2 "Failed to start Postgres (in standby mode)"
  sudo docker rm --force ${container_name}
  destroy_zfs_clone
  exit 1
fi

should_be_promoted=$(sudo docker exec ${container_name} psql -p ${clone_port} -U ${pg_username} -d ${pg_db} -h ${pg_sock_dir} -XAtc 'select pg_is_in_recovery()')

# Save data state timestamp.
#   - if we had a replica, we can use `pg_last_xact_replay_timestamp()`,
#   - if it is a master initially, the DB state timestamp must be provided by user in unix time format.
#   - otherwise use the current datetime by default
if [[ ! -z ${DATA_STATE_AT+x} ]]; then
  # For testing, use:
  #    DATA_STATE_AT=$(TZ=UTC date '+%Y%m%d%H%M%S')
  data_state_at="${DATA_STATE_AT}"
elif [[ $should_be_promoted == "t" ]]; then
  data_state_at=$(sudo docker exec ${container_name} psql \
    --set ON_ERROR_STOP=on \
    -p ${clone_port} \
    -U ${pg_username} \
    -d ${pg_db} \
    -h ${pg_sock_dir} \
    -XAt \
    -c "select to_char(coalesce(pg_last_xact_replay_timestamp(), NOW()) at time zone 'UTC', 'YYYYMMDDHH24MISS')")

  if [[ -z $data_state_at ]]; then
    echo >&2 "Failed to get data_state_at: pg_data should be promoted, but pg_last_xact_replay_timestamp() returns the empty result."
    echo >&2 "Check if pg_data is correct, either explicitly define DATA_STATE_AT via an environment variable."
    sudo docker rm --force ${container_name}
    destroy_zfs_clone
    exit 1
  fi
else
  echo >&2 -e "\e[31mAs DATA_STATE_AT is not defined, the current datetime will be used.\e[0m"
  data_state_at=$(TZ=UTC date '+%Y%m%d%H%M%S')
fi

# Promote to the master. Again, it may take a while.
if [[ $should_be_promoted == "t" ]]; then
  echo >&2 "Promote"
  sudo docker exec ${container_name} pg_ctl -D ${container_pgdata_dir} -W promote
fi

failed=true
for i in {1..1000}; do
  if [[ $(sudo docker exec ${container_name} psql -p ${clone_port} -U ${pg_username} -d ${pg_db} -h ${pg_sock_dir} -XAtc 'select pg_is_in_recovery()') == "f" ]]; then
    failed=false
    break
  fi

  sleep 2
done

if $failed; then
  echo >&2 "Failed to promote Postgres to master"
  sudo docker rm --force ${container_name}
  destroy_zfs_clone
  exit 1
fi

################################################################################
# We're about to finalize everything and create a snapshot that will be used
# for thin cloning.
#
# If needed, put any data transformations here (e.g., remove personal data).
# For better speed, do it in several parallel jobs (depending on resources).
# All thin clones will have transformed state.
# TODO: friendly interface to inject transformations.
################################################################################

# Finally, stop Postgres and create the base snapshot ready to be used for thin provisioning
sudo docker stop ${container_name}

${sudo_cmd} rm -rf ${clone_pgdata_dir}/pg_log

sudo zfs snapshot ${clone_full_name}@${snapshot_name}
sudo zfs set dblab:datastateat="${data_state_at}" ${clone_full_name}@${snapshot_name}

# Snapshot "datastore/postgresql/db_state_1_pre@db_state_1" is ready and can be used for thin provisioning

${sudo_cmd} rm -rf /tmp/trigger_${clone_port}

sudo docker rm ${container_name}

# Restart the sync instance.
start_sync_instance

# Return to previous working directory.
cd -
