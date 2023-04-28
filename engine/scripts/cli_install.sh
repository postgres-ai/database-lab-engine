#!/bin/bash

cli_version=${DBLAB_CLI_VERSION:-"master"}

mkdir -p ~/.dblab

# Detect OS
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  cygwin_nt*|mingw*|msys_nt*|nt*|win*) os="windows" ;;
  darwin*) os="darwin" ;;
  linux*) os="linux" ;;
  freebsd*) os="freebsd" ;;
  *) echo "Unsupported OS: $os"; exit 1 ;;
esac

# Detect architecture  
arch=$(uname -m)
case "$arch" in
 x86_64*) arch="amd64" ;;
 arm64*) arch="arm64" ;;
  *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac

echo "Detected OS: $os, architecture: $arch"

curl --fail -Ss --output ~/.dblab/dblab \
  https://storage.googleapis.com/database-lab-cli/${cli_version}/dblab-${os}-${arch} \
  && chmod a+x ~/.dblab/dblab

if [ $? -eq 0 ]; then
  echo '
     888 888      888          888      
     888 888      888          888      
     888 888      888          888      
 .d88888 88888b.  888  8888b.  88888b.  
d88" 888 888 "88b 888     "88b 888 "88b 
888  888 888  888 888 .d888888 888  888 
Y88b 888 888 d88P 888 888  888 888 d88P 
 "Y88888 88888P"  888 "Y888888 88888P"
'

  echo "::::::::::::::::::::::::::::::::::::::::"

  echo 'SUCCESS! DLE CLI ("dblab") downloaded to:'

  {
    rm -f /usr/local/bin/dblab 2> /dev/null \
      && mv ~/.dblab/dblab /usr/local/bin/dblab 2> /dev/null \
      && echo 'Done!'
  } || {
    echo '    ~/.dblab/dblab'
    echo 'Add it to $PATH or move manually:'
    echo '    sudo mv ~/.dblab/dblab /usr/local/bin/dblab'
  }

  echo "::::::::::::::::::::::::::::::::::::::::"
  echo 'To start, run:'
  echo '    dblab init'
  echo
else
  >&2 echo "dblab setup failure – cannot download binaries"
fi
