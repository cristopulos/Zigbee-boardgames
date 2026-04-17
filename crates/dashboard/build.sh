#!/bin/bash
set -e
cd "$(dirname "$0")"
cargo install trunk --locked 2>/dev/null || true
rustup target add wasm32-unknown-unknown
trunk build --release
