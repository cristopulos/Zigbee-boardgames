#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "=== Installing prerequisites ==="
rustup target add wasm32-unknown-unknown
cargo install trunk --locked 2>/dev/null || true

echo "=== Building dashboard (WASM) ==="
cd crates/dashboard
trunk build --release
cd ../..

echo "=== Building Rust workspace ==="
cargo build --release

echo "=== Build complete ==="
echo "Run ./scripts/run-prod.sh to start the application."
