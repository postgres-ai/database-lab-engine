#/!bin/bash

set -euxo pipefail

sudo apt update -y
sudo apt upgrade -y
sudo apt full-upgrade

sudo apt-get update && sudo apt-get install -y \
  apt-transport-https \
  ca-certificates \
  gnupg-agent \
  python3-software-properties \
  software-properties-common \
  curl \
  gnupg2 \
  zfsutils-linux

#install docker 
#sudo apt-get remove -y  docker docker-engine docker.io containerd runc
#sudo rm -rf /etc/systemd/system/docker.s*

#curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

#sudo add-apt-repository \
#  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
#  $(lsb_release -cs) \
#  stable"

#sudo apt-get update && sudo apt-get install -y \
#  docker-ce \
#  docker-ce-cli \
#  containerd.io || echo "issue with docker install"

#sudo docker pull  postgresai/dblab-server:2.3-latest

#install postgres client
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ `lsb_release -cs`-pgdg main" |sudo tee /etc/apt/sources.list.d/pgdg.list
sudo apt-get update && sudo apt-get install -y postgresql-client-13

#install envoy
curl -sL 'https://getenvoy.io/gpg' | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://dl.bintray.com/tetrate/getenvoy-deb $(lsb_release -cs) stable"
sudo apt update && sudo apt-get install -y  getenvoy-envoy

