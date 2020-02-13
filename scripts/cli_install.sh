#!/bin/bash

# 2020 © Postgres.ai

mkdir -p ~/.dblab

curl --location --fail --output ~/.dblab/dblab \
  https://gitlab.com/postgres-ai/database-lab/-/jobs/artifacts/master/raw/bin/dblab-linux-amd64?job=build-binary-generic \
  && chmod a+x ~/.dblab/dblab

{
  rm -f /usr/local/bin/dblab 2> /dev/null \
    && mv ~/.dblab/dblab /usr/local/bin/dblab 2> /dev/null \
    && echo 'Done!'
} || {
  echo 'Database Lab client CLI is installed to "~/.dblab/dblab".'
  echo 'Add this path to $PATH or, alternatively, move the binary to the global space:'
  echo '    sudo mv ~/.dblab/dblab /usr/local/bin/dblab'
}

echo 'To start using Database Lab client CLI, run:'
echo '    dblab init'
