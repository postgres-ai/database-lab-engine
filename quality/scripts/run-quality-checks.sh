#!/usr/bin/env bash
#
# Run all quality checks locally before pushing.
#
# Usage:
#   ./quality/scripts/run-quality-checks.sh
#
# This script runs:
#   1. Go formatting check
#   2. Go linting
#   3. Unit tests with race detection
#   4. Coverage threshold check
#   5. Cyclomatic complexity check
#   6. Vulnerability scan (if govulncheck is installed)

set -euo pipefail

ENGINE_DIR="$(cd "$(dirname "$0")/../../engine" && pwd)"
QUALITY_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0

run_check() {
    local name="$1"
    shift
    echo ""
    echo "━━━ ${name} ━━━"
    if "$@"; then
        echo "  ✓ ${name} passed"
        PASS=$((PASS + 1))
    else
        echo "  ✗ ${name} FAILED"
        FAIL=$((FAIL + 1))
    fi
}

cd "$ENGINE_DIR"

# 1. formatting
run_check "Go formatting" bash -c '
    UNFORMATTED=$(gofmt -l . 2>/dev/null | head -20)
    if [ -n "$UNFORMATTED" ]; then
        echo "Unformatted files:"
        echo "$UNFORMATTED"
        exit 1
    fi
'

# 2. linting
run_check "Go linting" make run-lint

# 3. unit tests with race detection
run_check "Unit tests (race)" make test

# 4. coverage
run_check "Coverage threshold" bash "$QUALITY_DIR/scripts/check-coverage.sh" 60

# 5. cyclomatic complexity
run_check "Cyclomatic complexity" bash -c '
    if command -v gocyclo &>/dev/null; then
        COMPLEX=$(gocyclo -over 30 . 2>/dev/null || true)
        if [ -n "$COMPLEX" ]; then
            echo "Functions exceeding complexity 30:"
            echo "$COMPLEX"
            exit 1
        fi
    else
        echo "gocyclo not installed, skipping (install: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)"
    fi
'

# 6. vulnerability scan
run_check "Vulnerability scan" bash -c '
    if command -v govulncheck &>/dev/null; then
        govulncheck ./...
    else
        echo "govulncheck not installed, skipping (install: go install golang.org/x/vuln/cmd/govulncheck@latest)"
    fi
'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "Fix failures before pushing."
    exit 1
fi

echo ""
echo "All quality checks passed."
