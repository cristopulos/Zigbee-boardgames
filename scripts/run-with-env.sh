#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="${1:-.env}"

if [ ! -f "$ENV_FILE" ]; then
    echo "Error: Environment file '$ENV_FILE' not found."
    echo "Usage: $0 [env-file]"
    exit 1
fi

echo "=== Loading environment from $ENV_FILE ==="
set -a
source "$ENV_FILE"
set +a

echo "=== Starting button-hub ==="
echo "MQTT_BROKER_HOST=$MQTT_BROKER_HOST"
echo "MQTT_BROKER_PORT=$MQTT_BROKER_PORT"
echo "API_PORT=$API_PORT"

# Ensure dashboard is built for prod-like run
if [ ! -d "crates/dashboard/dist" ] || [ ! -f "crates/dashboard/dist/index.html" ]; then
    echo "Building dashboard..."
    cd crates/dashboard
    trunk build --release
    cd ../..
fi

if [ ! -f "target/release/button-hub" ]; then
    cargo build --release
fi

./target/release/button-hub
