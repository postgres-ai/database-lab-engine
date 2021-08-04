#!/bin/bash

# 2020 © Postgres.ai

cli_version=${DBLAB_CLI_VERSION:-"latest"}

mkdir -p ~/.dblab

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*|mingw*|msys_nt*|nt*|win*) os="windows" ;;
  esac
  echo "$os"
}

curl --location --fail --output ~/.dblab/dblab \
  https://storage.googleapis.com/database-lab-cli/${cli_version}/dblab-$(uname_os)-amd64 \
  && chmod a+x ~/.dblab/dblab

{
  rm -f /usr/local/bin/dblab 2> /dev/null \
    && mv ~/.dblab/dblab /usr/local/bin/dblab 2> /dev/null \
    && echo 'Done!'
} || {
  echo 'Downloaded to:'
  echo '    ~/.dblab/dblab'
  echo 'Add it to $PATH or move the binary manually:'
  echo '    sudo mv ~/.dblab/dblab /usr/local/bin/dblab'
}

echo 'To start, run:'
echo '    dblab init'
