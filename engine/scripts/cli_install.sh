#!/bin/bash
################################################
# Welcome to DBLab 🖖
# This script downloads DBLab CLI (`dblab`).
#
# To install it on macOS/Linux/Windows:
#      curl -sSL dblab.sh | bash
#
# ⭐️ Contribute to DBLab: https://dblab.dev
# 📚 DBLab Docs: https://docs.dblab.dev
# 💻 CLI reference: https://cli-docs.dblab.dev/
# 👨‍💻 API reference: https://api.dblab.dev
################################################

cli_version=${DBLAB_CLI_VERSION:-"master"}
cli_version=${cli_version#v}

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
 arm64*|aarch64*) arch="arm64" ;;
  *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac

echo "Detected OS: $os, architecture: $arch"

url="https://storage.googleapis.com/database-lab-cli/${cli_version}/dblab-${os}-${arch}"

curl --fail -Ss --output ~/.dblab/dblab $url \
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
  ~/.dblab/dblab --version
  echo "::::::::::::::::::::::::::::::::::::::::"
  echo "Installed to:"
  {
    rm -f /usr/local/bin/dblab 2> /dev/null \
      && mv ~/.dblab/dblab /usr/local/bin/dblab 2> /dev/null \
      && echo '    /usr/local/bin/dblab'
  } || {
    echo '    ~/.dblab/dblab'
    echo 'Add it to $PATH:'
    echo '    export PATH=$PATH:~/.dblab/dblab'
    echo 'or move:'
    echo '    sudo mv ~/.dblab/dblab /usr/local/bin/dblab'
  }

  echo "::::::::::::::::::::::::::::::::::::::::"
  echo 'To start, run:'
  echo '    dblab init'
  echo
else
  >&2 echo "dblab setup failure – cannot download binaries from $url"
fi
