#!/bin/bash

#--------------------------------------------------------------------------
# Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
# All Rights Reserved
# Unauthorized copying of this file, via any medium is strictly prohibited
# Proprietary and confidential
#--------------------------------------------------------------------------

set -x

install_gettext() {
  if [[ "$OSTYPE" == "darwin"* ]]; then # macOS.
    brew install gettext
    brew link --force gettext
  else  # Linux
    if [[ "$(command -v apk)" != "" ]]; then
      apk add gettext
    elif [[ "$(command -v apt-get)" != "" ]]; then
      apt-get update
      apt-get install gettext-base
    else
      echo "Unsupported OS!"
      exit 1
    fi
  fi
}

subs_envs() {
  rm -f $2

  if [[ "$(command -v envsubst)" == "" ]]; then
    install_gettext
  fi

  # Import additional envs from file specified in second arg.
  if [ ! -z ${ENV+x} ]; then
    source "./ui/packages/platform/deploy/configs/${ENV}.sh"
  fi

  cat $1 | envsubst > $2
}

is_command_defined() {
    type $1 2>/dev/null | grep -q 'is a function'
}

# Parse command and arguments.
COMMAND=$1
shift
ARGUMENTS=${@}

# Run command.
is_command_defined $COMMAND
if [ $? -eq 0 ]; then
  $COMMAND $ARGUMENTS
else
  echo "Command not found"
fi
