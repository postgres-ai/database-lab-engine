#/!bin/bash

set -e

curl https://gitlab.com/postgres-ai/database-lab/-/raw/2.3.1/scripts/cli_install.sh | bash 
sudo mv ~/.dblab/dblab /usr/local/bin/dblab
curl https://gitlab.com/postgres-ai/database-lab/-/raw/2.3.1/configs/config.example.logical_generic.yml --output ~/.dblab/config.example.logical_generic.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/2.3.1/configs/config.example.logical_rds_iam.yml --output ~/.dblab/config.example.logical_rds_iam.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/2.3.1/configs/config.example.physical_generic.yml --output  ~/.dblab/config.example.physical_generic.yml
curl https://gitlab.com/postgres-ai/database-lab/-/raw/2.3.1/configs/config.example.physical_walg.yml --output  ~/.dblab/config.example.physical_walg.yml

