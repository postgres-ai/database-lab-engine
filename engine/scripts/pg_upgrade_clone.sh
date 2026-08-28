#!/bin/bash
# 2026 © Postgres.ai
#
# Runs pg_upgrade --link against the data directory of a DBLab clone.
#
# Every parameter arrives via the environment; the engine validates them, this script does not.
# The only guards here are ":?" expansions that stop an empty variable from turning a path into
# a destructive one.
#
# Layout inside CLONE_DIR, where <data> is DATA_SUBDIR (the pool's dataSubDir, "data" by default):
#   <data>         the cluster being upgraded (old major)
#   <data>_new     created empty by the engine, initdb'd here, promoted to <data> by the engine
#   <data>_old     the previous <data> after the engine's swap
#
# CLONE_DIR belongs to the engine OS user and is NOT writable here, so this script never creates
# or renames anything inside it. The engine pre-creates <data>_new and performs the final swap;
# this script only writes inside <data>_new and UPGRADE_DIR, both chowned to the container user.
#
# UPGRADE_DIR is that working area:
#   stage          the stage reached so far, reported by the engine for diagnostics
#   state.json     written by the engine, never touched here
#   logs/          this script's log and pg_upgrade's own output directory
#
# On success this script also writes <data>_new/.dblab_upgrade_done. That marker is the engine's
# only proof that the conversion completed - see the comment at the transfer stage below.
#
# Exit codes report what this script believed happened. They are NOT a recovery contract:
#   0   success
#   10  failed before any file transfer
#   20  failed at or after file transfer
# The engine decides recovery from the data directory instead, because a directory cannot lie
# about its own state the way a reported code can. These codes and the stage file explain a
# failure; they never decide one.

set -euo pipefail
set -o errtrace

readonly EXIT_PRE_TRANSFER=10
readonly EXIT_POST_TRANSFER=20

readonly CLONE_DIR="${CLONE_DIR:?CLONE_DIR is required}"
readonly OLD_VERSION="${OLD_VERSION:?OLD_VERSION is required}"
readonly NEW_VERSION="${NEW_VERSION:?NEW_VERSION is required}"
readonly ENCODING="${ENCODING:?ENCODING is required}"
readonly LC_COLLATE_VALUE="${LC_COLLATE_VALUE:?LC_COLLATE_VALUE is required}"
readonly LC_CTYPE_VALUE="${LC_CTYPE_VALUE:?LC_CTYPE_VALUE is required}"
readonly LOCALE_PROVIDER="${LOCALE_PROVIDER:-}"
readonly ICU_LOCALE="${ICU_LOCALE:-}"
readonly DATA_CHECKSUMS="${DATA_CHECKSUMS:-off}"
readonly OLD_SERVER_OPTIONS="${OLD_SERVER_OPTIONS:-}"
readonly DATA_SUBDIR="${DATA_SUBDIR:-data}"
readonly UPGRADE_DIR="${UPGRADE_DIR:-${CLONE_DIR}/upgrade}"

readonly OLD_DATA="${CLONE_DIR}/${DATA_SUBDIR}"
readonly NEW_DATA="${CLONE_DIR}/${DATA_SUBDIR}_new"

# Written into NEW_DATA once pg_upgrade has returned 0, and the only proof the engine accepts that
# the conversion finished. Keep the name in step with upgradeDoneMarkerName in
# internal/provision/upgrade.go.
readonly DONE_MARKER_NAME=".dblab_upgrade_done"
readonly STAGE_FILE="${UPGRADE_DIR}/stage"
readonly LOG_DIR="${UPGRADE_DIR}/logs"
readonly LOG_FILE="${LOG_DIR}/pg_upgrade_clone.log"

readonly OLD_BIN="/usr/lib/postgresql/${OLD_VERSION}/bin"
readonly NEW_BIN="/usr/lib/postgresql/${NEW_VERSION}/bin"

log() {
  printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*" >> "${LOG_FILE}"
}

set_stage() {
  printf '%s' "$1" > "${STAGE_FILE}"
  log "stage: $1"
}

# shellcheck disable=SC2329  # invoked through the ERR trap.
# on_error maps the stage reached to the exit code the engine uses to decide whether the old
# cluster may be restarted in place. pg_upgrade --link hard-links the old cluster's files into
# the new one, so once "transfer" has begun the old cluster is no longer guaranteed consistent.
on_error() {
  local code=$?
  local stage

  stage=$(cat "${STAGE_FILE}" 2>/dev/null || echo unknown)
  log "failed at stage '${stage}' with code ${code}"

  case "${stage}" in
    prepare | initdb | check)
      # Cleaning up ${NEW_DATA} is the engine's job: removing the directory itself needs write
      # access to CLONE_DIR, which this container deliberately does not have.
      log "nothing was converted; the old cluster is intact"
      exit "${EXIT_PRE_TRANSFER}"
      ;;
    *)
      log "leaving the directory state as is for diagnostics; the clone must be re-provisioned"
      exit "${EXIT_POST_TRANSFER}"
      ;;
  esac
}

trap on_error ERR

mkdir -p "${LOG_DIR}"

log "upgrading ${CLONE_DIR} from PostgreSQL ${OLD_VERSION} to ${NEW_VERSION}"

# initdb the new cluster with the collation and checksum settings of the old one; pg_upgrade
# rejects any mismatch.
init_new_cluster() {
  set_stage "initdb"

  local args=(
    --pgdata "${NEW_DATA}"
    --encoding "${ENCODING}"
    --lc-collate "${LC_COLLATE_VALUE}"
    --lc-ctype "${LC_CTYPE_VALUE}"
  )

  if [[ -n "${LOCALE_PROVIDER}" ]]; then
    args+=(--locale-provider "${LOCALE_PROVIDER}")
  fi

  if [[ -n "${ICU_LOCALE}" ]]; then
    args+=(--icu-locale "${ICU_LOCALE}")
  fi

  if [[ "${DATA_CHECKSUMS}" == "on" ]]; then
    args+=(--data-checksums)
  elif [[ "${NEW_VERSION}" -ge 18 ]]; then
    # checksums are on by default since 18, and only that version knows the negative flag.
    args+=(--no-data-checksums)
  fi

  # ${NEW_DATA} already exists: the engine created it empty and handed it to this user. initdb
  # accepts an empty directory it owns.
  log "initdb ${args[*]}"
  "${NEW_BIN}/initdb" "${args[@]}" >> "${LOG_FILE}" 2>&1
}

# carry_conf copies a settings file into the new cluster, dropping the settings its binary does
# not know. A setting removed in the new major (wal_keep_segments, gone in 13) would otherwise
# make the server refuse to start, both during pg_upgrade and afterwards.
carry_conf() {
  local src="$1"
  local dst="$2"
  local tmp name line

  if [[ ! -f "${src}" ]]; then
    return 0
  fi

  tmp=$(mktemp)

  while IFS= read -r line || [[ -n "${line}" ]]; do
    if [[ -z "${line}" || "${line}" == \#* ]]; then
      printf '%s\n' "${line}" >> "${tmp}"
      continue
    fi

    name="${line%%=*}"
    name="${name//[[:space:]]/}"

    # A qualified name (auto_explain.log_min_duration) belongs to an extension "postgres -C"
    # cannot resolve without preloading it; the server accepts any such placeholder in a config
    # file, so it cannot be the setting that stops the cluster from starting.
    if [[ "${name}" == *.* ]] || "${NEW_BIN}/postgres" -D "${NEW_DATA}" -C "${name}" > /dev/null 2>&1; then
      printf '%s\n' "${line}" >> "${tmp}"
    else
      log "dropping unknown setting from ${src##*/}: ${name}"
    fi
  done < "${src}"

  mv "${tmp}" "${dst}"
  log "carried over ${src##*/}"
}

# ALTER SYSTEM settings.
copy_auto_conf() {
  carry_conf "${OLD_DATA}/postgresql.auto.conf" "${NEW_DATA}/postgresql.auto.conf"
}

# databaseConfigs from the engine config (shared_buffers, shared_preload_libraries, ...). The
# engine re-applies only its default files to the new data directory, so without this the
# upgraded clone would run on initdb defaults and lose every preload-dependent extension.
copy_snapshot_conf() {
  carry_conf "${OLD_DATA}/postgresql.dblab.snapshot.conf" "${NEW_DATA}/postgresql.dblab.snapshot.conf"
}

run_pg_upgrade() {
  local mode="$1"
  local args=(
    --old-datadir "${OLD_DATA}"
    --new-datadir "${NEW_DATA}"
    --old-bindir "${OLD_BIN}"
    --new-bindir "${NEW_BIN}"
    --link
  )

  if [[ -n "${OLD_SERVER_OPTIONS}" ]]; then
    args+=(--old-options "${OLD_SERVER_OPTIONS}")
  fi

  if [[ "${mode}" == "check" ]]; then
    args+=(--check)
  fi

  log "pg_upgrade ${args[*]}"
  "${NEW_BIN}/pg_upgrade" "${args[@]}" >> "${LOG_FILE}" 2>&1
}

init_new_cluster
# snapshot.conf first: once postgresql.auto.conf is in place, "postgres -C" parses it too, and an
# invalid value there would make every later setting look unknown.
copy_snapshot_conf
copy_auto_conf

# pg_upgrade writes pg_upgrade_output.d and its temporary sockets into the current directory.
cd "${LOG_DIR}"

# A killed run (docker kill, OOM, engine timeout) leaves the socket and lock file of the server
# pg_upgrade had started here, and the next run then fails at "check" with "lock file
# .s.PGSQL.50432.lock already exists". Nothing but this script's own container writes here, and
# the engine runs one per clone, so anything found at this point is stale.
rm -f "${LOG_DIR}"/.s.PGSQL.*

set_stage "check"
run_pg_upgrade check

set_stage "transfer"
run_pg_upgrade upgrade

# The one fact that proves the conversion ran to completion. It has to be written here, after
# pg_upgrade has returned 0, because nothing else in the directory says so: pg_upgrade renames
# global/pg_control away when linking STARTS, so the old cluster is already unstartable long
# before the transfer is finished. Without this marker a run killed mid-transfer looks exactly
# like a finished one, and the engine would promote a partially linked cluster and delete the
# only remaining copy of everything it had not linked yet.
#
# It goes inside NEW_DATA rather than UPGRADE_DIR so that it survives the engine's swap: the
# engine renames NEW_DATA onto DATA, and the marker travels with the cluster it describes.
#
# It is flushed before the script reports success. pg_upgrade fsyncs the cluster it built, so
# without this the one crash class the marker exists for - a hard interruption - could lose the
# marker while keeping the conversion, and the engine would then rebuild the clone from its
# snapshot and silently discard every write made to it since it was created.
touch "${NEW_DATA}/${DONE_MARKER_NAME}"
#
# sync -f flushes only the filesystem holding the marker. The bare sync fallback is deliberately
# last: inside a container it flushes every mounted filesystem on the host, which on a Database Lab
# host with large pools can stall unrelated work, so it is worth a log line saying why it happened.
sync -f "${NEW_DATA}/${DONE_MARKER_NAME}" || {
  log "sync -f is unavailable; flushing all filesystems instead"
  sync
}

set_stage "done"
log "conversion finished; the engine performs the swap"

exit 0
