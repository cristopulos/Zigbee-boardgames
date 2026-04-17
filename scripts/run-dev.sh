#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Prevent duplicate instances
if pgrep -x "button-hub" > /dev/null 2>&1; then
    echo "Error: button-hub is already running. Stop it first:"
    echo "  pkill -f button-hub"
    exit 1
fi

echo "=== Starting button-hub API (dev mode) ==="
echo "API: http://localhost:${API_PORT:-3000}/"
echo "Use ./scripts/run-dashboard-dev.sh for dashboard hot-reload"

RUST_LOG="${RUST_LOG:-debug}" cargo run --bin button-hub "$@"
