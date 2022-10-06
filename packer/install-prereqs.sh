#/!bin/bash

set -euxo pipefail

### Upgrade the existing software
sudo apt update -y
sudo apt upgrade -y
sudo apt full-upgrade

### Extend the list of apt repositories 
# yq repo
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys CC86BB64
sudo add-apt-repository ppa:rmescandon/yq

# Docker repo
wget --quiet -O - https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

# Postgres PGDG repo 
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo add-apt-repository "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main"

sudo apt-get update -y

### Installation steps – first, using apt, then non-apt steps, and, finally, get Docker container for DLE
sudo apt-get install -y \
  apt-transport-https \
  ca-certificates \
  gnupg-agent \
  python3-software-properties \
  software-properties-common \
  curl \
  gnupg2 \
  zfsutils-linux \
  docker-ce \
  docker-ce-cli \
  containerd.io \
  postgresql-client-14 \
  s3fs \
  yq \
  jq 	

# Install cfn-signal helper script 
sudo mkdir -p /opt/aws/bin
wget https://s3.amazonaws.com/cloudformation-examples/aws-cfn-bootstrap-py3-latest.tar.gz
sudo python3 -m easy_install --script-dir /opt/aws/bin aws-cfn-bootstrap-py3-latest.tar.gz
rm aws-cfn-bootstrap-py3-latest.tar.gz

# Install certbot
sudo snap install certbot --classic
sudo ln -s /snap/bin/certbot /usr/bin/certbot

# Install Envoy 
curl https://func-e.io/install.sh | sudo bash -s -- -b /usr/local/bin
sudo /usr/local/bin/func-e use 1.19.1 # https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.20.0#incompatible-behavior-changes

# Pull DLE image
image_version=$(echo ${dle_version} | sed 's/v*//')
sudo docker pull registry.gitlab.com/postgres-ai/database-lab/dblab-server:$image_version
sudo docker pull postgresai/ce-ui:1.1.0
sudo docker pull postgresai/extended-postgres:10
sudo docker pull postgresai/extended-postgres:11
sudo docker pull postgresai/extended-postgres:12
sudo docker pull postgresai/extended-postgres:13
sudo docker pull postgresai/extended-postgres:14

# upgrade ssm agent version
wget https://s3.us-east-1.amazonaws.com/amazon-ssm-us-east-1/amazon-ssm-agent/3.1.1575.0/amazon-ssm-agent-ubuntu-amd64.tar.gz
tar -xf amazon-ssm-agent-ubuntu-amd64.tar.gz
sudo bash snap-install.sh

