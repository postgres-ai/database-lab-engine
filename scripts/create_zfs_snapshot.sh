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
zfs_pool=${ZFS_POOL:-"datastore/postgresql"}
# Sudirectory in which PGDATA is located.
pgdata_subdir=${PGDATA_SUBDIR:-"/9.6/main"}

# Clone configuration.
# Mount directory for DB Lab clones.
mount_dir=${MOUNT_DIR:-"/var/lib/postgresql/dblab/clones"}
# Name of a clone which will be created and used for PGDATA manipulation.
clone_name="clone${pre}_${now}"
# Full name of the clone for ZFS commands.
clone_full_name="${zfs_pool}/${clone_name}"
# Clone mount directory.
clone_dir="${mount_dir}/${clone_name}"
# Directory of PGDATA in the clone mount.
clone_pgdata_dir="${clone_dir}${pgdata_subdir}"

# Postgres configuration.
# Port on which Postgres will be started using clone's PGDATA.
clone_port=${CLONE_PORT:-6999}
# Here we assume that PGDATA is in a subdirectory "./9.6/main" -- the Ubuntu way is here, and PG version is 9.6
pg_bin_dir=${PG_BIN_DIR:-"/usr/lib/postgresql/9.6/bin"}
pg_username=${PGUSERNAME:-"postgres"}
# Set password with PGPASSWORD env.
pg_db=${PGDB:-"postgres"}

# Snapshot.
# Name of resulting snapshot after PGDATA manipulation.
snapshot_name="snapshot_${now}"

# TODO: decide: do we need to stop the shadow Postgres instance?
# OR: we can tell the shadow Postgres: select pg_start_backup('database-lab-snapshot');
# .. and in the very end: select pg_stop_backup();

sudo zfs snapshot -r ${zfs_pool}@${snapshot_name}${pre}
sudo zfs clone ${zfs_pool}@${snapshot_name}${pre} ${clone_full_name} -o mountpoint=${clone_dir}

cd /tmp # To avoid errors about lack of permissions.

sudo -u postgres sh -f - <<SH
rm -rf ${clone_pgdata_dir}/postmaster.pid # Questionable -- it's better to have snapshot created with Postgres being down

# We do not want to deal with postgresql.conf symlink (if any)
cat ${clone_pgdata_dir}/postgresql.conf > ${clone_pgdata_dir}/postgresql_real.conf
chmod 600 ${clone_pgdata_dir}/postgresql_real.conf
rm ${clone_pgdata_dir}/postgresql.conf
mv ${clone_pgdata_dir}/postgresql_real.conf ${clone_pgdata_dir}/postgresql.conf

### ADJUST CONFIGS ###
### postgresql.conf
echo "external_pid_file='${clone_pgdata_dir}/postmaster.pid'" >>  ${clone_pgdata_dir}/postgresql.conf
echo "data_directory='${clone_pgdata_dir}'" >> ${clone_pgdata_dir}/postgresql.conf
echo "log_directory='${clone_pgdata_dir}/log'" >> ${clone_pgdata_dir}/postgresql.conf
# TODO: improve secirity aspects
echo "listen_addresses = '*'" >> ${clone_pgdata_dir}/postgresql.conf
# TODO: adjust log settings, memory setting

### recovery.conf
#echo "standby_mode = 'on'" > ${clone_pgdata_dir}/recovery.conf # overriding
#echo "primary_conninfo = ''" >> ${clone_pgdata_dir}/recovery.conf
#echo "restore_command = ''" >> ${clone_pgdata_dir}/recovery.conf

touch ${clone_pgdata_dir}/standby.signal

### pg_hba.conf
echo "host all all 127.0.0.1/32 trust" > ${clone_pgdata_dir}/pg_hba.conf

### pg_ident.conf
echo "" > ${clone_pgdata_dir}/pg_ident.conf
SH

sudo -u postgres ${pg_bin_dir}/pg_ctl \
  -D "${clone_pgdata_dir}" \
  -o "-p ${clone_port} -c 'shared_buffers=4096'" \
  -W \
  start

# Now we are going to wait until we can connect to the server.
# If it was a replica, it may take a while..
# During that period, we will have "FATAL:  the database system is starting up".
# Alternatively, we could use pg_ctl's "-w" option above (instead of manual checking).

failed=true
for i in {1..1000}; do
  if [[ $(${pg_bin_dir}/psql -p $clone_port -U ${pg_username} -d ${pg_db} -h localhost -XAtc 'select pg_is_in_recovery()') == "t" ]]; then
    failed=false
    break
  fi

  sleep 2
done

if $failed; then
  >&2 echo "Failed to start Postgres (in standby mode)"
  exit 1
fi

# Save data state timestamp.
#   - if we had a replica, we can use `pg_last_xact_replay_timestamp()`,
#   - if it is a master initially, the DB state timestamp must be provided by user in unix time format.
if [[ ! -z ${DATA_STATE_AT+x} ]]; then
  # For testing, use:
  #    DATA_STATE_AT=$(TZ=UTC date '+%Y%m%d%H%M%S')
  data_state_at="${DATA_STATE_AT}"
else
  data_state_at=$(${pg_bin_dir}/psql -p ${clone_port} -U ${pg_username} -d ${pg_db} -h localhost -XAtc 'select extract(epoch from pg_last_xact_replay_timestamp())')
fi

# Promote to the master. Again, it may take a while.
sudo -u postgres ${pg_bin_dir}/pg_ctl -D ${clone_pgdata_dir} -W promote

failed=true
for i in {1..1000}; do
  if [[ $(${pg_bin_dir}/psql -p ${clone_port} -U ${pg_username} -d ${pg_db} -h localhost -XAtc 'select pg_is_in_recovery()') == "f" ]]; then
    failed=false
    break
  fi

  sleep 2
done

if $failed; then
  >&2 echo "Failed to promote Postgres to master"
  exit 1
fi

# Finally, stop Postgres and create the base snapshot ready to be used for thin provisioning
sudo -u postgres ${pg_bin_dir}/pg_ctl -D ${clone_pgdata_dir} -w stop
# todo: check that it's stopped, similiraly as above

# Finally, we don't wan't to want 'trust', we need to use password always.
# Note, that this line overrides the whole pg_hba.conf
sudo -u postgres sh -c "echo \"host all postgres 127.0.0.1/32 trust\" > ${clone_pgdata_dir}/pg_hba.conf"
sudo -u postgres sh -c "echo \"host all all all md5\" >> ${clone_pgdata_dir}/pg_hba.conf"

sudo zfs snapshot -r ${clone_full_name}@${snapshot_name}
sudo zfs set dblab:datastateat="${data_state_at}" ${clone_full_name}@${snapshot_name}

# Snapshot "datastore/postgresql/db_state_1_pre@db_state_1" is ready and can be used for thin provisioning

sudo -u postgres rm -rf /tmp/trigger_${clone_port}

# Return to previous working directory.
cd -
