#!/bin/bash

set -e

POOL_NAME="dblab_pool"
POOL_MNT="/var/lib/dblab/dblab_pool"
DISK_FILE="/zfs-disk"
DATASETS=(dataset_1 dataset_2 dataset_3)

echo "🔍 Checking if zfsutils-linux is installed..."
if ! command -v zfs >/dev/null 2>&1; then
  echo "📦 Installing zfsutils-linux..."
  sudo apt update
  sudo apt install -y zfsutils-linux
else
  echo "✅ ZFS already installed"
fi

if [ ! -f "$DISK_FILE" ]; then
  echo "🧱 Creating virtual ZFS disk at $DISK_FILE..."
  sudo truncate -s 5G "$DISK_FILE"
else
  echo "✅ ZFS disk file already exists"
fi

echo "🔗 Setting up loop device..."
sudo losetup -fP "$DISK_FILE"
LOOP=$(sudo losetup -j "$DISK_FILE" | cut -d: -f1)

echo "📂 Checking if pool '$POOL_NAME' exists..."
if ! zpool list | grep -q "$POOL_NAME"; then
  echo "🚀 Creating ZFS pool $POOL_NAME..."
  sudo zpool create -f \
    -O compression=on \
    -O atime=off \
    -O recordsize=128k \
    -O logbias=throughput \
    -m "$POOL_MNT" \
    "$POOL_NAME" \
    "$LOOP"
else
  echo "✅ ZFS pool '$POOL_NAME' already exists"
fi

echo "📦 Creating base datasets..."
for DATASET in "${DATASETS[@]}"; do
  if ! zfs list | grep -q "${POOL_NAME}/${DATASET}"; then
    echo "📁 Creating dataset ${POOL_NAME}/${DATASET}"
    sudo zfs create -o mountpoint="${POOL_MNT}/${DATASET}" "${POOL_NAME}/${DATASET}"
  else
    echo "⚠️ Dataset '${DATASET}' already exists"
  fi
done

echo "✅ ZFS setup complete."