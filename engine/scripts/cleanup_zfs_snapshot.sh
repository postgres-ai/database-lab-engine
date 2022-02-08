#!/bin/bash
# 2019 © Postgres.ai

### !!! DEPRECATED !!! ###
### In version 2.0, snapshotting policy, as well as
### snapshot retention are now automated and defined in
### the main configuration file. See: 
### https://postgres.ai/docs/database-lab/config-reference#section-retrieval-data-retrieval

# Name of the ZFS pool which contains PGDATA.
zfs_pool=${ZFS_POOL:-"dblab_pool"}

# Maximum number of ZFS snapshots.
snapshot_limit=24

# Destroy snapshots.
sudo zfs list -t snapshot -r ${zfs_pool} -H -o name | grep -v clone | head -n -${snapshot_limit} | xargs -n1 --no-run-if-empty sudo zfs destroy -R 2>&1  | logger --stderr --tag "cleanup_zfs_snapshot"
