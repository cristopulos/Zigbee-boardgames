#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Prevent duplicate instances
if pgrep -x "button-hub" > /dev/null 2>&1; then
    echo "Error: button-hub is already running. Stop it first:"
    echo "  pkill -f button-hub"
    exit 1
fi

# Ensure dashboard dist exists
if [ ! -d "crates/dashboard/dist" ] || [ ! -f "crates/dashboard/dist/index.html" ]; then
    echo "Dashboard dist not found. Building first..."
    ./scripts/build-all.sh
fi

# Ensure release binary exists
if [ ! -f "target/release/button-hub" ]; then
    echo "Release binary not found. Building..."
    cargo build --release
fi

echo "=== Starting button-hub (production) ==="
echo "Dashboard: http://localhost:${API_PORT:-3000}/dashboard/"
RUST_LOG="${RUST_LOG:-info}" ./target/release/button-hub "$@"
