#!/usr/bin/env bash
set -euo pipefail

# dev-setup.sh — Build-only script for development.
# Assumes prerequisites (mosquitto, node, pnpm, rust, go, zigbee2mqtt) are already installed.
# Run this after pulling code changes or on a machine where setup.sh was already used.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "========================================"
echo " button-hub Dev Setup"
echo "========================================"
echo ""

cd "$PROJECT_DIR"

# Ensure .env exists
if [[ ! -f ".env" ]]; then
    if [[ -f ".env.example" ]]; then
        cp .env.example .env
        log_info "Created .env from .env.example"
    fi
fi

# Build Rust workspace
log_info "Building Rust workspace (release)..."
cargo build --release

# Build dashboard if needed
if [[ ! -d "crates/dashboard/dist" ]] || [[ ! -f "crates/dashboard/dist/index.html" ]]; then
    log_info "Building dashboard..."
    cd crates/dashboard
    trunk build --release
    cd "$PROJECT_DIR"
else
    log_warn "Dashboard dist exists — skipping. Use ./scripts/build-all.sh to force rebuild."
fi

# Build and test Go library
log_info "Building and testing Go library..."
GO_DIR="$PROJECT_DIR/go"
if [[ -d "$GO_DIR" ]]; then
    cd "$GO_DIR"
    go build ./...
    go test ./...
    cd "$PROJECT_DIR"
else
    log_warn "Go library not found at $GO_DIR — skipping"
fi

log_info "Dev setup complete."
echo ""
echo "Run the project:"
echo "  ./scripts/run-prod.sh       # production"
echo "  ./scripts/run-dev.sh        # development (debug logs)"
echo ""
