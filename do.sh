#!/bin/bash

# 2019 © Postgres.ai

run() {
  set -x

  make dep
  make all

  set +x

  # Instead of editing default values here create a separate config file.
  # e.g. for staging environment create a file /deploy/configs/staging.sh
  # use `export` to define variables and run with `ENV=staging makerun.sh`.
  if [ -z ${ENV+x} ]; then
    echo "Using default variables"
    source ./config/envs/default.sh
  else
    source ./config/envs/${ENV}.sh
  fi

  # Read and set git status info if it wasn't done before.
  if [ "$INCLUDE_GIT_STATUS" == "true" ] && [ -z "$GIT_COMMIT_HASH" ]; then
    echo "Fetching git status..."
    export_git_status
  fi

  set -x

  ./bin/joe
}

export_git_status() {
  DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
  GIT_STATUS_RES=$(cd $DIR && git status --porcelain=v2 -b)
  EXIT_CODE=$?

  if [ $EXIT_CODE != 0 ]; then
    echo "git exit code: ${EXIT_CODE}"
    exit
  fi

  GIT_MODIFIED="false"
  GIT_BRANCH=""
  GIT_COMMIT_HASH=""
  while read -r line; do
    COLUMNS=($line)
    if [ ${#COLUMNS[@]} > 3 ]; then
      if [ ${COLUMNS[1]} == "branch.oid" ]; then
        GIT_COMMIT_HASH=${COLUMNS[2]}
        continue
      elif [ ${COLUMNS[1]} == "branch.head" ]; then
        GIT_BRANCH=${COLUMNS[2]}
        continue
      fi
    fi

    GIT_MODIFIED="true"
  done <<< "$GIT_STATUS_RES"

  export GIT_MODIFIED
  export GIT_BRANCH
  export GIT_COMMIT_HASH
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
