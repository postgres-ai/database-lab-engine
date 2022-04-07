#/!bin/bash

set -x
sudo su - ubuntu
mkdir ~/.dblab
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/scripts/cli_install.sh | bash 
sudo mv ~/.dblab/dblab /usr/local/bin/dblab
echo $dle_version > ~/.dblab/dle_version

# Copy base templates
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.logical_generic.yml --output ~/.dblab/config.example.logical_generic.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.logical_rds_iam.yml --output ~/.dblab/config.example.logical_rds_iam.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.physical_generic.yml --output ~/.dblab/config.example.physical_generic.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.physical_walg.yml --output  ~/.dblab/config.example.physical_walg.yml

# Adjust DLE config
dle_config_path="/home/ubuntu/.dblab/engine/configs"
dle_meta_path="/home/ubuntu/.dblab/engine/meta"
postgres_conf_path="/home/ubuntu/.dblab/postgres_conf"

mkdir -p $dle_config_path
mkdir -p $dle_meta_path
mkdir -p $postgres_conf_path

curl https://gitlab.com/postgres-ai/database-lab/-/raw/${dle_version}/configs/config.example.logical_generic.yml --output $dle_config_path/server.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/${dle_version}/configs/standard/postgres/control/pg_hba.conf \
  --output $postgres_conf_path/pg_hba.conf
curl https://gitlab.com/postgres-ai/database-lab/-/raw/${dle_version}/configs/standard/postgres/control/postgresql.conf --output $postgres_conf_path/postgresql.conf
