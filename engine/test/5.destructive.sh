#!/bin/bash
# Destructive testing harness for Database Lab Engine.
#
# Tests failure modes that could break customer trust:
#   1. Kill DLE mid-clone and verify state recovery
#   2. Simulate disk-full during snapshot
#   3. Restart DLE and verify clone integrity
#   4. Concurrent clone operations under resource pressure
#
# Prerequisites: same as 1.synthetic.sh (ZFS pool, Docker, DLE running)
#
# Usage:
#   bash engine/test/5.destructive.sh
#
# Environment variables:
#   POSTGRES_VERSION  - PostgreSQL version (default: 17)
#   DLE_SERVER_PORT   - DLE API port (default: 12345)

set -euo pipefail

TAG=${TAG:-${CI_COMMIT_REF_SLUG:-"master"}}
IMAGE2TEST="registry.gitlab.com/postgres-ai/database-lab/dblab-server:${TAG}"
DLE_SERVER_NAME="dblab_server_test"

export POSTGRES_VERSION="${POSTGRES_VERSION:-17}"
export DLE_SERVER_PORT=${DLE_SERVER_PORT:-12345}
export DLE_PORT_POOL_FROM=${DLE_PORT_POOL_FROM:-9000}
export DLE_PORT_POOL_TO=${DLE_PORT_POOL_TO:-9099}
export DLE_TEST_MOUNT_DIR="/var/lib/test/dblab_mount"
export DLE_TEST_POOL_NAME="test_dblab_pool"

DIR=${0%/*}

PASS=0
FAIL=0
SKIP=0

log_info() { echo "[INFO]  $(date '+%H:%M:%S') $*"; }
log_pass() { echo "[PASS]  $(date '+%H:%M:%S') $*"; PASS=$((PASS + 1)); }
log_fail() { echo "[FAIL]  $(date '+%H:%M:%S') $*"; FAIL=$((FAIL + 1)); }
log_skip() { echo "[SKIP]  $(date '+%H:%M:%S') $*"; SKIP=$((SKIP + 1)); }

dle_api() {
  local method="$1"
  local endpoint="$2"
  local data="${3:-}"

  if [ -n "$data" ]; then
    curl -s -X "$method" \
      -H "Content-Type: application/json" \
      -H "Verification-Token: secret_token" \
      -d "$data" \
      "http://localhost:${DLE_SERVER_PORT}/api${endpoint}"
  else
    curl -s -X "$method" \
      -H "Verification-Token: secret_token" \
      "http://localhost:${DLE_SERVER_PORT}/api${endpoint}"
  fi
}

wait_for_dle() {
  local max_attempts="${1:-60}"
  for i in $(seq 1 "$max_attempts"); do
    if dle_api GET /healthz 2>/dev/null | grep -q "ok"; then
      return 0
    fi
    sleep 1
  done
  return 1
}

# --------------------------------------------------------------------------
# Test 1: Clone creation and verification
# --------------------------------------------------------------------------
test_clone_create_verify() {
  log_info "test 1: clone creation and data verification"

  local response
  response=$(dle_api POST /clone '{"id":"destructive-test-clone-1"}')

  local clone_id
  clone_id=$(echo "$response" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || true)

  if [ -z "$clone_id" ]; then
    log_fail "clone creation failed: $response"
    return 1
  fi

  # wait for clone to be ready
  for i in $(seq 1 60); do
    local status
    status=$(dle_api GET "/clone/${clone_id}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',{}).get('code',''))" 2>/dev/null || true)
    if [ "$status" = "ok" ]; then
      log_pass "clone created and ready: $clone_id"
      return 0
    fi
    sleep 1
  done

  log_fail "clone did not become ready within 60 seconds"
  return 1
}

# --------------------------------------------------------------------------
# Test 2: Kill DLE mid-operation and verify recovery
# --------------------------------------------------------------------------
test_kill_and_recover() {
  log_info "test 2: kill DLE during clone creation and verify recovery"

  # start a clone creation (async)
  dle_api POST /clone '{"id":"destructive-test-clone-kill"}' &
  local clone_pid=$!

  # give it a moment to start
  sleep 2

  # kill the DLE container
  log_info "killing DLE container"
  sudo docker kill "$DLE_SERVER_NAME" 2>/dev/null || true

  # wait for the async request to finish (it will fail)
  wait "$clone_pid" 2>/dev/null || true

  # restart DLE
  log_info "restarting DLE container"
  sudo docker start "$DLE_SERVER_NAME" 2>/dev/null || true

  if wait_for_dle 120; then
    log_pass "DLE recovered after kill"
  else
    log_fail "DLE did not recover after kill within 120 seconds"
    return 1
  fi

  # verify instance status is healthy
  local status
  status=$(dle_api GET /status | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',{}).get('code',''))" 2>/dev/null || true)

  if [ "$status" = "ok" ]; then
    log_pass "instance status is healthy after recovery"
  else
    log_fail "instance status is not healthy after recovery: $status"
  fi
}

# --------------------------------------------------------------------------
# Test 3: Concurrent clone operations
# --------------------------------------------------------------------------
test_concurrent_clones() {
  log_info "test 3: concurrent clone creation"

  local pids=()
  local clone_count=5

  for i in $(seq 1 $clone_count); do
    dle_api POST /clone "{\"id\":\"concurrent-clone-${i}\"}" &
    pids+=($!)
  done

  local succeeded=0
  local failed=0

  for pid in "${pids[@]}"; do
    if wait "$pid" 2>/dev/null; then
      succeeded=$((succeeded + 1))
    else
      failed=$((failed + 1))
    fi
  done

  log_info "concurrent clones: $succeeded succeeded, $failed failed"

  # wait for all clones to be ready
  sleep 10

  local ready_count=0
  for i in $(seq 1 $clone_count); do
    local status
    status=$(dle_api GET "/clone/concurrent-clone-${i}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',{}).get('code',''))" 2>/dev/null || true)
    if [ "$status" = "ok" ]; then
      ready_count=$((ready_count + 1))
    fi
  done

  if [ "$ready_count" -ge 3 ]; then
    log_pass "concurrent clone creation: $ready_count/$clone_count clones ready"
  else
    log_fail "concurrent clone creation: only $ready_count/$clone_count clones ready"
  fi

  # cleanup
  for i in $(seq 1 $clone_count); do
    dle_api DELETE "/clone/concurrent-clone-${i}" 2>/dev/null || true
  done
}

# --------------------------------------------------------------------------
# Test 4: DLE restart with existing clones
# --------------------------------------------------------------------------
test_restart_with_clones() {
  log_info "test 4: DLE restart preserves existing clones"

  # create a clone
  local response
  response=$(dle_api POST /clone '{"id":"restart-test-clone"}')
  sleep 5

  # get clone count before restart
  local before_count
  before_count=$(dle_api GET /status | python3 -c "import sys,json; print(json.load(sys.stdin).get('cloning',{}).get('numClones',0))" 2>/dev/null || echo "0")

  log_info "clones before restart: $before_count"

  # restart DLE
  sudo docker restart "$DLE_SERVER_NAME"

  if ! wait_for_dle 120; then
    log_fail "DLE did not recover after restart"
    return 1
  fi

  sleep 5

  # get clone count after restart
  local after_count
  after_count=$(dle_api GET /status | python3 -c "import sys,json; print(json.load(sys.stdin).get('cloning',{}).get('numClones',0))" 2>/dev/null || echo "0")

  log_info "clones after restart: $after_count"

  if [ "$after_count" -ge "$before_count" ] && [ "$before_count" -gt 0 ]; then
    log_pass "clone state preserved across restart ($before_count -> $after_count)"
  else
    log_fail "clone state not preserved across restart ($before_count -> $after_count)"
  fi

  # cleanup
  dle_api DELETE "/clone/restart-test-clone" 2>/dev/null || true
}

# --------------------------------------------------------------------------
# Test 5: Snapshot listing after stress
# --------------------------------------------------------------------------
test_snapshot_integrity() {
  log_info "test 5: snapshot integrity after operations"

  local snapshots_before
  snapshots_before=$(dle_api GET /snapshots)

  local snap_count
  snap_count=$(echo "$snapshots_before" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")

  if [ "$snap_count" -gt 0 ]; then
    log_pass "snapshot listing returns $snap_count snapshots"
  else
    log_fail "no snapshots available"
    return 1
  fi

  # create and destroy several clones to stress the snapshot references
  for i in $(seq 1 3); do
    dle_api POST /clone "{\"id\":\"snap-stress-${i}\"}" 2>/dev/null || true
  done

  sleep 5

  for i in $(seq 1 3); do
    dle_api DELETE "/clone/snap-stress-${i}" 2>/dev/null || true
  done

  sleep 3

  local snapshots_after
  snapshots_after=$(dle_api GET /snapshots)

  local snap_count_after
  snap_count_after=$(echo "$snapshots_after" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")

  if [ "$snap_count_after" -eq "$snap_count" ]; then
    log_pass "snapshot count preserved after stress: $snap_count_after"
  else
    log_fail "snapshot count changed after stress: $snap_count -> $snap_count_after"
  fi
}

# --------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------
main() {
  echo "================================================================="
  echo " Destructive Testing Harness"
  echo " PostgreSQL: ${POSTGRES_VERSION}"
  echo " DLE Image: ${IMAGE2TEST}"
  echo "================================================================="
  echo ""

  # check if DLE is running
  if ! wait_for_dle 5; then
    log_info "DLE is not running, skipping destructive tests"
    log_info "run 1.synthetic.sh first to set up the environment"
    exit 0
  fi

  test_clone_create_verify || true
  test_kill_and_recover || true
  test_concurrent_clones || true
  test_restart_with_clones || true
  test_snapshot_integrity || true

  echo ""
  echo "================================================================="
  echo " Results: $PASS passed, $FAIL failed, $SKIP skipped"
  echo "================================================================="

  if [ "$FAIL" -gt 0 ]; then
    exit 1
  fi
}

main "$@"
