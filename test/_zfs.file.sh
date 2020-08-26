#!/bin/bash
set -euxo pipefail

ZFS_FILE="$(pwd)/zfs_file"

truncate --size 1GB "${ZFS_FILE}"

sudo zpool create -f \
  -O compression=on \
  -O atime=off \
  -O recordsize=8k \
  -O logbias=throughput \
  -m /var/lib/dblab/data \
  test_pool \
  "${ZFS_FILE}"

sudo chmod 0750 /var/lib/dblab/data

zfs list
