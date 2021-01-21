#!/bin/bash
set -euxo pipefail

ZFS_FILE="$(pwd)/zfs_file"

truncate --size 1GB "${ZFS_FILE}"

sudo zpool create -f \
  -O compression=on \
  -O atime=off \
  -O recordsize=8k \
  -O logbias=throughput \
  -m /var/lib/dle/test \
  test_pool \
  "${ZFS_FILE}"

sudo mkdir -p /var/lib/dle/test/data
sudo chmod 0755 /var/lib/dle/test/data

zfs list
