#!/bin/bash
set -euxo pipefail


ZFS_FILE="$(pwd)/zfs_file"

# TODO: docker rm for all containers that are related to dblab
sudo docker ps -a --filter 'label=dblab_control' \
    | grep -v CONTAINER \
    | awk '{print $1}' \
    | sudo xargs --no-run-if-empty docker rm -f \
  || true
sudo zpool destroy test_pool || true
sudo rm -rf /var/lib/dblab/data/
sudo umount /var/lib/dblab/data || true
sudo rm -f "${ZFS_FILE}"
rm -f ~/.dblab/server_test.yml

dblab config remove test
