#!/bin/bash
# 2019 © Postgres.ai

# Use this script with caution! Note that this script is going to destroy all ZFS snapshots
# on the machine, except:
#   - the last $1 snapshots (the first parameter in the CLI call),
#   - those snapthots that are used now,
#   - snapshots having "clone" or "NAME" in the name (however, they are supposed to be
#     dependant; so with old 'main' snapshots, dependant ones will be deleted as well,
#     thanks to '-R' option when calling 'zfs destroy').
#
# Example of use:
#    bash ./scripts/delete_old_zfs_snapshots.sh 5

set -euxo pipefail

n=$1

if [[ -z "$n" ]]; then
  echo "Specify the number of snapshots to keep (Example: 'bash ./scripts/delete_old_zfs_snapshots.sh 5' to delete all but last 5 snapshots)." 
else
  sudo zfs list -t snapshot -o name \
    | grep -v clone \
    | grep -v NAME \
    | head -n -$n \
    | xargs -n1 --no-run-if-empty sudo zfs destroy -R
fi

## An example of crontab entry, setting up auto-deletion of all unused ZFS snapshots except the last 3 ones.
##     0 6 * * * sudo zfs list -t snapshot -o name | grep -v clone | grep -v NAME | head -n -3 | xargs -n1 --no-run-if-empty sudo zfs destroy -R 2>&1 | logger --stderr --tag "cleanup_zfs_snapshot"
