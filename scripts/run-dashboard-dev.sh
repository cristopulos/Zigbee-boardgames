#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../crates/dashboard"

echo "=== Starting Trunk dev server ==="
echo "Dashboard: http://localhost:9090/dashboard/"
echo "Make sure the API is running: ./scripts/run-dev.sh"

trunk serve --port 9090 "$@"
