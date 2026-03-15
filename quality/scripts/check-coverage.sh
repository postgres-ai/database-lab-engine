#!/usr/bin/env bash
#
# Check test coverage for the engine and report packages below threshold.
#
# Usage:
#   ./quality/scripts/check-coverage.sh [threshold]
#
# Arguments:
#   threshold  minimum coverage percentage (default: 60)
#
# Exit codes:
#   0  all packages meet threshold
#   1  one or more packages below threshold

set -euo pipefail

THRESHOLD="${1:-60}"
ENGINE_DIR="$(cd "$(dirname "$0")/../../engine" && pwd)"

echo "Running tests with coverage (threshold: ${THRESHOLD}%)..."
cd "$ENGINE_DIR"

go test -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | tail -1

echo ""
echo "=== Coverage by Package ==="

FAILED=0
while IFS= read -r line; do
    pkg=$(echo "$line" | awk '{print $1}')
    cov=$(echo "$line" | awk '{print $3}' | sed 's/%//')

    if [ "$pkg" = "total:" ]; then
        echo ""
        echo "Total coverage: ${cov}%"
        continue
    fi

    if [ "$(echo "$cov < $THRESHOLD" | bc)" -eq 1 ]; then
        echo "  BELOW  ${cov}%  ${pkg}"
        FAILED=1
    fi
done < <(go tool cover -func=coverage.out | grep -E '(total:|^[a-z])')

echo ""

if [ "$FAILED" -eq 1 ]; then
    echo "WARNING: some packages are below ${THRESHOLD}% coverage"
    exit 1
fi

echo "All packages meet ${THRESHOLD}% coverage threshold"
