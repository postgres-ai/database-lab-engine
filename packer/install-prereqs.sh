#/!bin/bash

set -euxo pipefail

sudo apt-get install -y \
  apt-transport-https \
  ca-certificates \
  gnupg-agent \
  software-properties-common \
  curl \
  postgresql-client  \
  postgresql-contrib

# ZFS
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

sudo add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"

# docker
sudo apt-get update && sudo apt-get install -y \
  docker-ce \
  docker-ce-cli \
  containerd.io \
  zfsutils-linux

