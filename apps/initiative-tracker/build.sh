#!/bin/bash
set -euo pipefail

echo "=== Initiative Tracker Build Script ==="

# Check for Fyne system dependencies
echo "Checking system dependencies..."

# Check for X11 or Wayland
if [[ -n "${DISPLAY:-}" ]]; then
    echo "  X11 detected (DISPLAY=$DISPLAY)"
elif [[ -n "${WAYLAND_DISPLAY:-}" ]]; then
    echo "  Wayland detected (WAYLAND_DISPLAY=$WAYLAND_DISPLAY)"
else
    echo "  WARNING: No X11 or Wayland display detected."
    echo "  GUI apps may not display without a display server."
    echo "  On Linux, install one of:"
    echo "    - X11: libx11, libxcb, libxkbcommon, libxcursor, libxrandr, libxi"
    echo "    - Wayland: libwayland-client, libwayland-egl, libxkbcommon"
    echo "  On Raspberry Pi: libegl1, libgles2, libinput"
    echo "  On Windows/macOS: native graphics drivers should work"
fi

# Check for OpenGL (required by Fyne)
if command -v glxinfo &> /dev/null; then
    if glxinfo 2>/dev/null | grep -q "OpenGL"; then
        echo "  OpenGL: available"
    else
        echo "  WARNING: OpenGL not detected via glxinfo"
    fi
else
    echo "  Note: glxinfo not found, skipping OpenGL check"
fi

# Build the binary
echo ""
echo "Building initiative-tracker..."
cd "$(dirname "$0")"
go build -o initiative-tracker .

echo "Build complete: ./initiative-tracker"
echo ""
echo "Usage:"
echo "  ./initiative-tracker                        # keyboard only"
echo "  ./initiative-tracker --button=<id>          # with remote button"
echo "  ./initiative-tracker --button=<id> --naalu  # with Naalu 0 token"
echo "  ./initiative-tracker --start=3              # start at initiative 3"
echo ""
echo "Keyboard controls:"
echo "  Space/Right/Up    - Next initiative"
echo "  Backspace/Left/Down - Previous initiative"
echo "  R                 - Reset to start"
echo "  Escape            - Quit"
echo ""
echo "Mouse:"
echo "  Click any card    - Toggle enabled/disabled"