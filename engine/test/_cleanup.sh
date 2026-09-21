#!/bin/bash
set -euxo pipefail

DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
DLE_TEST_POOL_NAME="test_dblab_pool"
ZFS_FILE="$(pwd)/zfs_file"
META_DIR="$HOME/.dblab/engine/meta"

# Stop and remove test Docker containers
sudo docker ps -aq --filter label="test_dblab_pool" | xargs --no-run-if-empty sudo docker rm -f \
  || echo "Failed to remove test Docker containers, continuing..."
sudo docker ps -aq --filter label="dblab_clone=test_dblab_pool" | xargs --no-run-if-empty sudo docker rm -f \
  || echo "Failed to remove test Docker containers, continuing..."
sudo docker ps -aq --filter label="dblab_test" | xargs --no-run-if-empty sudo docker rm -f \
  || echo "Failed to remove dblab_test Docker containers, continuing..."

# Remove unused Docker images
sudo docker images --filter=reference='registry.gitlab.com/postgres-ai/database-lab/dblab-server:*' -q | xargs --no-run-if-empty sudo docker rmi \
  || echo "Docker image removal finished with errors but it is OK to ignore them."

# Clean up data directory
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/data/* \
  || echo "Data directory cleanup finished with errors but continuing..."

# Clean up branch directory
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/branch/* \
  || echo "Branch directory cleanup finished with errors but continuing..."

# Remove dump directory
sudo umount ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump \
  || echo "Unmounting dump directory finished with errors but it is OK to ignore them."
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/dump \
  || echo "Dump directory removal finished with errors but it is OK to ignore them."

# Clean up pool directory
sudo rm -rf ${DLE_TEST_MOUNT_DIR}/${DLE_TEST_POOL_NAME}/* \
  || echo "Cleaning up pool directory finished with errors but it is OK to ignore them."

# To start from the very beginning: destroy ZFS storage pool
sudo zpool destroy test_dblab_pool \
  || echo "Destroying ZFS storage pool finished with errors but it is OK to ignore them."

# Remove ZFS FILE
sudo rm -f "${ZFS_FILE}" \
  || echo "Failed to remove ZFS file, but continuing..."

# Remove CLI configuration
dblab config remove test \
  || echo "Removing CLI configuration finished with errors but it is OK to ignore them."

# The source database directories under ${DLE_TEST_DATA_DIR:-/var/tmp/dle_test} are intentionally
# left in place: 2.logical_generic.sh and 4.physical_basebackup.sh reuse them across jobs instead of
# reloading pgbench every run. They are bounded (one per script per PG major) and disk-backed.
# Because nothing removes them here, each of those scripts validates its own directory on start via
# _source_data.sh and discards one an interrupted run left unusable.

# Clean up engine meta directory, which every test mounts into the container. A sessions.json or
# pending.retrieval left by an earlier run is read back by the engine on start.
sudo rm -rf "${META_DIR}"/* \
  || echo "Cleaning up meta directory finished with errors but it is OK to ignore them."
