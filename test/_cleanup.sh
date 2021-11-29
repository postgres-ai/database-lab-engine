#!/bin/bash
set -euxo pipefail

DLE_TEST_MOUNT_DIR="/var/lib/test/dblab"
DLE_TEST_POOL_NAME="test_dblab_pool"
ZFS_FILE="$(pwd)/zfs_file"

# Stop and remove test Docker containers
sudo docker ps -aq --filter label="test_dblab_pool" | xargs --no-run-if-empty sudo docker rm -f
sudo docker ps -aq --filter label="dblab_test" | xargs --no-run-if-empty sudo docker rm -f

# Remove unused Docker images
sudo docker images -q | xargs --no-run-if-empty sudo docker rmi || true

# Clean up the data directory
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/*

# Remove dump directory
sudo umount ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump || true
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump || true

# Clean up the pool directory
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/* || true

# To start from the very beginning: destroy ZFS storage pool
sudo zpool destroy test_dblab_pool || true

# Remove ZFS FILE
sudo rm -f "${ZFS_FILE}"

# Remove CLI configuration
dblab config remove test || true

# Remove Database Lab client CLI
# sudo rm -f  /usr/local/bin/dblab || true

