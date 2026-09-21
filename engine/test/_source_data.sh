#!/bin/bash

# Validates the reusable source-database directory before a test binds it as PGDATA, and exposes
# discard_source_data for the readiness check that follows.
#
# 2.logical_generic.sh and 4.physical_basebackup.sh keep their source cluster across jobs rather than
# reloading pgbench every run, and _cleanup.sh deliberately leaves it in place, so a directory an
# interrupted run left unusable is inherited by every later job for that major on the same runner.
#
# The postgres entrypoint decides whether to initialize from a non-empty PG_VERSION, and initdb
# refuses to run against a non-empty PGDATA, so it never repairs such a directory by itself. The
# check below covers the two states visible before the server starts: a missing PG_VERSION, left by
# an initdb interrupted before it wrote one, and a differing one, left by another major. A run
# killed after that file was written - during the CREATE DATABASE that follows initdb, say - leaves
# a cluster that starts but is incomplete. Only the readiness check downstream sees that, and it
# calls discard_source_data so the next job rebuilds instead of inheriting the same failure.
#
# Expects SOURCE_DATA_DIR and POSTGRES_VERSION.

discard_source_data() {
  echo "Discarding source data dir ${SOURCE_DATA_DIR}: $1"
  sudo rm -rf "${SOURCE_DATA_DIR}"
}

if [ -d "${SOURCE_DATA_DIR}" ]; then
  source_data_major=$(sudo cat "${SOURCE_DATA_DIR}/PG_VERSION" 2>/dev/null || true)

  if [ "${source_data_major}" != "${POSTGRES_VERSION}" ]; then
    discard_source_data "PG_VERSION is '${source_data_major:-missing}', expected '${POSTGRES_VERSION}'"
  fi
fi
