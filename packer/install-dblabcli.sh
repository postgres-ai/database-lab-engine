#/!bin/bash

set -x
mkdir ~/.dblab
curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/scripts/cli_install.sh | bash 
sudo mv ~/.dblab/dblab /usr/local/bin/dblab
echo $dle_version > ~/.dblab/dle_version
sudo curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.logical_generic.yml --output ~/.dblab/config.example.logical_generic.yml
sudo curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.logical_rds_iam.yml --output ~/.dblab/config.example.logical_rds_iam.yml
sudo curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.physical_generic.yml --output ~/.dblab/config.example.physical_generic.yml
sudo curl https://gitlab.com/postgres-ai/database-lab/-/raw/$dle_version/configs/config.example.physical_walg.yml --output  ~/.dblab/config.example.physical_walg.yml

