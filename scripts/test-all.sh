#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "=== cargo fmt check ==="
cargo fmt -- --check

echo "=== cargo clippy ==="
cargo clippy --workspace -- -D warnings

echo "=== cargo test ==="
cargo test --workspace

echo "=== cargo build (dev) ==="
cargo build --workspace

echo "=== cargo build (release) ==="
cargo build --workspace --release

echo ""
echo "=== All checks passed ==="
