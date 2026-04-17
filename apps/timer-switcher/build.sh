#!/usr/bin/env bash
# build.sh — Build timer-switcher.
# System dependencies must be installed first:
#   sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev \
#     libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
#     libxcursor-dev libvulkan-dev libxrandr-dev libxinerama-dev libxi-dev \
#     libxxf86vm-dev

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$SCRIPT_DIR"
go build -o timer-switcher .
echo "Built: $SCRIPT_DIR/timer-switcher"
