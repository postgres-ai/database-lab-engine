#!/bin/bash
set -euxo pipefail

# Install dependencies
sudo apt-get update && sudo apt-get install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg-agent \
  software-properties-common

# Install Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

sudo add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"

sudo apt-get install -y \
  docker-ce \
  docker-ce-cli \
  containerd.io

# Install ZFS
sudo apt-get install -y \
  zfsutils-linux

# Install psql
sudo apt-get install -y \
  postgresql-client
