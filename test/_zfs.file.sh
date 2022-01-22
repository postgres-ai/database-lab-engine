#!/bin/bash
set -euxo pipefail

DLE_TEST_MOUNT_DIR="/var/lib/test/dblab"
DLE_TEST_POOL_NAME="test_dblab_pool"
ZFS_FILE="$(pwd)/zfs_file"

# If previous run was interrupted without cleanup,
# test_dblab_pool and $ZFS_FILE are still here. Cleanup.
sudo zpool destroy test_dblab_pool || true
sudo rm -f "${ZFS_FILE}"

truncate --size 1GB "${ZFS_FILE}"

sudo zpool create -f \
  -O compression=on \
  -O atime=off \
  -O recordsize=128k \
  -O logbias=throughput \
  -m ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME} \
  test_dblab_pool \
  "${ZFS_FILE}"

sudo zfs list
