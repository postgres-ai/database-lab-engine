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
  zfsutils-linux \

# Install Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update && sudo apt-get install -y \
  docker-ce \
  docker-ce-cli \
  containerd.io

#sudo docker pull  postgresai/dblab-server:$dle_version

#install postgres client
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ `lsb_release -cs`-pgdg main" |sudo tee /etc/apt/sources.list.d/pgdg.list
sudo apt-get update && sudo apt-get install -y postgresql-client-13

#install certbot
sudo snap install certbot --classic
sudo ln -s /snap/bin/certbot /usr/bin/certbot

#install envoy
curl https://func-e.io/install.sh | sudo bash -s -- -b /usr/local/bin
sudo /usr/local/bin/func-e use 1.19.1 # https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.20.0#incompatible-behavior-changes


#install s3fs
sudo apt install s3fs
