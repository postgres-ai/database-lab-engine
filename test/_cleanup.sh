#!/bin/bash
set -euxo pipefail

ZFS_FILE="$(pwd)/zfs_file"

sudo docker rm -f dblab_pg_initdb || true
sudo zpool destroy test_pool || true
sudo umount /var/lib/dblab/data || true
sudo rm -f "${ZFS_FILE}"
