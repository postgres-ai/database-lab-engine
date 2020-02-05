#!/bin/bash

# 2020 © Postgres.ai

mkdir -p ~/.dblab

curl -L -o ~/.dblab/dblab \
  https://gitlab.com/postgres-ai/database-lab/-/jobs/artifacts/74-docker-ci/raw/bin/dblab-linux-amd64?job=build-binary-generic && \
  chmod +x ~/.dblab/dblab

{
  rm -f /usr/local/bin/dblab 2> /dev/null &&
  ln -s ~/.dblab/dblab /usr/local/bin/dblab 2> /dev/null &&
  echo 'Done!' &&
  echo 'Run "dblab init" to configure Database Lab CLI'
} || {
  echo 'We installed Database Lab CLI to ~/.dblab/dblab.'
  echo 'You can add it to $PATH or run specified command to finish installation:'
  echo "sudo ln -s ~/.dblab/dblab /usr/local/bin/dblab"
  echo 'Then run "dblab init" to configure Database Lab CLI'
}
