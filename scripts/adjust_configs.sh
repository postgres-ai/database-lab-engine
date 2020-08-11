#!/bin/bash
# 2020 © Postgres.ai

echo "Run adjust script"

set -euxo pipefail

# Script for manual creation of ZFS snapshot from PG replica instance.
# Default values provided for Ubuntu FS layout.

# Utils.
pre="_pre"

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
clone_name=${CLONE_NAME:-""}
# Full name of the clone for ZFS commands.
clone_full_name="${zfs_pool}/${clone_name}"
# Clone mount directory.
clone_dir="${mount_dir}/${clone_name}"
# Directory of PGDATA in the clone mount.
clone_pgdata_dir="${clone_dir}${pgdata_subdir}"

pg_sock_dir=${PG_SOCK_DIR:-"/var/run/postgresql"}
pg_username=${PGUSERNAME:-"postgres"}

sudo_cmd=${SUDO_CMD:-""} # Use `sudo -u postgres` for default environment
pg_ver=${PG_VER:-"11"}

${sudo_cmd} bash -f - <<SH
set -ex

rm -rf ${clone_pgdata_dir}/postmaster.pid # Questionable -- it's better to have snapshot created with Postgres being down


# We do not want to deal with postgresql.conf symlink (if any)
echo "" > ${clone_pgdata_dir}/postgresql.conf
#cat ${clone_pgdata_dir}/postgresql.conf > ${clone_pgdata_dir}/postgresql_real.conf
#chmod 600 ${clone_pgdata_dir}/postgresql_real.conf
#rm ${clone_pgdata_dir}/postgresql.conf
#mv ${clone_pgdata_dir}/postgresql_real.conf ${clone_pgdata_dir}/postgresql.conf

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
echo "ssl = off" >> ${clone_pgdata_dir}/postgresql.conf

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

echo "Run container"
