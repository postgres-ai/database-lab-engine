#!/bin/bash
set -euxo pipefail

ZFS_FILE="$(pwd)/zfs_file"

# Stop and remove all Docker containers
sudo docker ps -aq | xargs --no-run-if-empty sudo docker rm -f

# Remove all Docker images
sudo docker images -q | xargs --no-run-if-empty sudo docker rmi

# Clean up the data directory
sudo rm -rf /var/lib/dblab/dblab_pool/data/*

# Remove dump directory
sudo umount /var/lib/dblab/dblab_pool/dump || true
sudo rm -rf /var/lib/dblab/dblab_pool/dump || true

# Clean up the pool directory
sudo rm -rf /var/lib/dblab/dblab_pool/*

# To start from the very beginning: destroy ZFS storage pool
sudo zpool destroy dblab_pool || true

# Remove ZFS FILE
sudo rm -f "${ZFS_FILE}"

# Remove CLI configuration
dblab config remove test || true

# Remove Database Lab client CLI
sudo rm -f  /usr/local/bin/dblab || true

