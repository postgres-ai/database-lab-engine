#!/bin/bash
set -euxo pipefail

ZFS_FILE="$(pwd)/zfs_file"

# TODO: Remove all docker containers related to the Database Lab.
sudo docker ps -a --filter 'label=dblab_control' \
    | grep -v CONTAINER \
    | awk '{print $1}' \
    | sudo xargs --no-run-if-empty docker rm -f \
  || true
sudo zpool destroy test_pool || true
sudo rm -rf /var/lib/dblab/data/
sudo umount /var/lib/dblab/data || true
sudo rm -f "${ZFS_FILE}"
sudo rm -f ~/.dblab/server_test.yml
sudo rm -rf /var/lib/dblab/db.dump || true
sudo rm -rf /var/lib/dblab/rds_db.dump || true
dblab config remove test || true
