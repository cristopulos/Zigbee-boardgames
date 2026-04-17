#!/usr/bin/env bash
set -euo pipefail

# setup-arch.sh — One-command setup for button-hub on a fresh Arch Linux machine.
# Run this as a regular user with passwordless sudo or be ready to enter
# your password when prompted.
#
# What it does:
#   - Installs system packages (mosquitto, git, curl, base-devel, etc.)
#   - Configures Mosquitto (localhost:1883, anonymous auth)
#   - Installs Node.js + pnpm (if missing)
#   - Installs Rust + wasm32 target + Trunk (if missing)
#   - Installs Go (if missing)
#   - Installs Zigbee2MQTT at /opt/zigbee2mqtt
#   - Creates systemd service for Zigbee2MQTT
#   - Builds the Rust workspace (release)
#   - Builds the Leptos dashboard
#   - Builds and tests the Go library
#   - Creates a .env file from .env.example

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "${BLUE}[STEP]${NC} $1"; }

echo "========================================"
echo " button-hub Setup Script (Arch Linux)"
echo "========================================"
echo ""

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------
if ! command -v sudo &>/dev/null; then
    log_error "sudo is required. Install it first: pacman -S sudo"
    exit 1
fi

if ! sudo -n true 2>/dev/null; then
    log_warn "You will be prompted for your sudo password during setup."
fi

if ! command -v pacman &>/dev/null; then
    log_error "This script requires pacman (Arch Linux). Use ./scripts/setup.sh for Debian/Ubuntu."
    exit 1
fi

# ---------------------------------------------------------------------------
# 1. System dependencies
# ---------------------------------------------------------------------------
log_step "1/9 — System packages"
log_info "Updating package database and installing dependencies..."
sudo pacman -Sy --noconfirm --needed mosquitto git curl base-devel pkgconf openssl >/dev/null 2>&1 && log_info "Packages installed" || {
    log_error "Failed to install packages. Check your internet connection and pacman mirrors."
    exit 1
}

# ---------------------------------------------------------------------------
# 2. Mosquitto
# ---------------------------------------------------------------------------
log_step "2/9 — Mosquitto MQTT broker"
MOSQUITTO_CONF="/etc/mosquitto/conf.d/local.conf"
if [[ -f "$MOSQUITTO_CONF" ]]; then
    log_warn "Mosquitto config already exists — skipping"
else
    sudo tee "$MOSQUITTO_CONF" > /dev/null <<'EOF'
listener 1883 localhost
allow_anonymous true
EOF
    log_info "Created $MOSQUITTO_CONF"
fi

sudo systemctl enable mosquitto &>/dev/null || true
sudo systemctl restart mosquitto
sleep 1
if systemctl is-active --quiet mosquitto; then
    log_info "Mosquitto is running on localhost:1883"
else
    log_error "Mosquitto failed to start. Check: sudo journalctl -u mosquitto"
    exit 1
fi

# ---------------------------------------------------------------------------
# 3. Node.js + pnpm
# ---------------------------------------------------------------------------
log_step "3/9 — Node.js and pnpm"
if command -v node &>/dev/null && [[ "$(node --version)" == v20* || "$(node --version)" == v2* ]]; then
    log_info "Node.js already installed: $(node --version)"
else
    log_info "Installing Node.js..."
    sudo pacman -S --noconfirm --needed nodejs npm >/dev/null 2>&1
    log_info "Node.js installed: $(node --version)"
fi

if command -v pnpm &>/dev/null; then
    log_info "pnpm already installed: $(pnpm --version)"
else
    log_info "Installing pnpm..."
    sudo npm install -g pnpm
    log_info "pnpm installed: $(pnpm --version)"
fi

# Ensure pnpm is available system-wide (needed by Zigbee2MQTT systemd service)
if [[ ! -f /usr/local/bin/pnpm ]]; then
    PNPM_PATH=""
    for path in "$HOME/.npm-global/bin/pnpm" "$HOME/.local/share/pnpm/pnpm"; do
        if [[ -f "$path" ]]; then
            PNPM_PATH="$path"
            break
        fi
    done
    if [[ -n "$PNPM_PATH" ]]; then
        sudo ln -sf "$PNPM_PATH" /usr/local/bin/pnpm
        log_info "Created /usr/local/bin/pnpm symlink"
    else
        log_warn "Could not find pnpm to create system symlink. Zigbee2MQTT service may fail."
    fi
fi

# ---------------------------------------------------------------------------
# 4. Rust
# ---------------------------------------------------------------------------
log_step "4/9 — Rust toolchain"
if command -v rustc &>/dev/null; then
    log_info "Rust already installed: $(rustc --version)"
else
    log_info "Installing Rust via rustup..."
    curl -fsSL https://sh.rustup.rs -o /tmp/rustup.sh
    sh /tmp/rustup.sh -y --no-modify-path >/dev/null
    rm /tmp/rustup.sh
    source "$HOME/.cargo/env"
    log_info "Rust installed: $(rustc --version)"
fi

# Ensure cargo is available for the rest of this script
[[ -f "$HOME/.cargo/env" ]] && source "$HOME/.cargo/env"

# WASM target for dashboard
if ! rustup target list --installed 2>/dev/null | grep -q wasm32-unknown-unknown; then
    log_info "Adding wasm32-unknown-unknown target..."
    rustup target add wasm32-unknown-unknown
fi

# Trunk for dashboard builds
if ! command -v trunk &>/dev/null; then
    log_info "Installing Trunk..."
    cargo install trunk --locked
    log_info "Trunk installed"
else
    log_info "Trunk already installed: $(trunk --version)"
fi

# ---------------------------------------------------------------------------
# 5. Go
# ---------------------------------------------------------------------------
log_step "5/9 — Go"
if command -v go &>/dev/null; then
    log_info "Go already installed: $(go version)"
else
    log_info "Installing Go..."
    GO_VERSION="1.22.5"
    GO_ARCH="linux-amd64"
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    if ! grep -q '/usr/local/go/bin' "$HOME/.bashrc" 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> "$HOME/.bashrc"
    fi
    export PATH=$PATH:/usr/local/go/bin
    log_info "Go installed: $(go version)"
fi

# ---------------------------------------------------------------------------
# 6. Zigbee2MQTT
# ---------------------------------------------------------------------------
log_step "6/9 — Zigbee2MQTT"
Z2M_DIR="/opt/zigbee2mqtt"
if [[ -d "$Z2M_DIR/.git" ]]; then
    log_info "Zigbee2MQTT already installed at $Z2M_DIR"
else
    log_info "Cloning Zigbee2MQTT to $Z2M_DIR..."
    sudo mkdir -p "$Z2M_DIR"
    sudo git clone --depth 1 https://github.com/Koenkk/zigbee2mqtt.git "$Z2M_DIR" >/dev/null 2>&1
    sudo chown -R "$(id -u):$(id -g)" "$Z2M_DIR"
    cd "$Z2M_DIR"
    log_info "Installing Zigbee2MQTT dependencies..."
    pnpm install >/dev/null 2>&1
    log_info "Building Zigbee2MQTT..."
    pnpm run build >/dev/null 2>&1
    log_info "Zigbee2MQTT installed"
fi

# Configuration
Z2M_CONFIG="$Z2M_DIR/data/configuration.yaml"
if [[ -f "$Z2M_CONFIG" ]]; then
    log_warn "Zigbee2MQTT config already exists — skipping"
else
    log_info "Creating default Zigbee2MQTT configuration..."
    SERIAL_PORT=""
    for pattern in /dev/ttyUSB* /dev/ttyACM*; do
        if ls "$pattern" 1> /dev/null 2>&1; then
            SERIAL_PORT=$(ls "$pattern" 2>/dev/null | head -1)
            break
        fi
    done

    if [[ -z "$SERIAL_PORT" ]]; then
        SERIAL_PORT="/dev/ttyUSB0"
        log_warn "No Zigbee dongle detected. Defaulting serial port to $SERIAL_PORT"
        log_warn "Update serial.port in $Z2M_CONFIG after plugging in your dongle."
    else
        log_info "Detected Zigbee dongle at $SERIAL_PORT"
    fi

    mkdir -p "$Z2M_DIR/data"
    cat > "$Z2M_CONFIG" <<EOF
homeassistant:
  enabled: false
mqtt:
  base_topic: zigbee2mqtt
  server: mqtt://localhost:1883
serial:
  port: $SERIAL_PORT
advanced:
  log_level: info
frontend:
  enabled: true
  port: 5001
EOF
    log_info "Created $Z2M_CONFIG"
fi

# Systemd service
Z2M_SERVICE="/etc/systemd/system/zigbee2mqtt.service"
if [[ -f "$Z2M_SERVICE" ]]; then
    log_warn "Zigbee2MQTT service already exists — skipping"
else
    log_info "Creating Zigbee2MQTT systemd service..."
    sudo tee "$Z2M_SERVICE" > /dev/null <<'EOF'
[Unit]
Description=Zigbee2MQTT
After=network.target

[Service]
ExecStart=/usr/bin/node /opt/zigbee2mqtt/index.js
WorkingDirectory=/opt/zigbee2mqtt
Restart=on-failure
# TODO: run as a dedicated zigbee2mqtt user instead of root for better security
User=root

[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl daemon-reload
    sudo systemctl enable zigbee2mqtt
    log_info "Created and enabled zigbee2mqtt.service"
fi

# ---------------------------------------------------------------------------
# 7. Build Rust workspace
# ---------------------------------------------------------------------------
log_step "7/9 — Build Rust workspace"
cd "$PROJECT_DIR"
cargo build --release
log_info "Rust workspace built successfully"

# ---------------------------------------------------------------------------
# 8. Build dashboard
# ---------------------------------------------------------------------------
log_step "8/9 — Build dashboard"
cd "$PROJECT_DIR/crates/dashboard"
if [[ ! -d "dist" ]] || [[ ! -f "dist/index.html" ]]; then
    trunk build --release
    log_info "Dashboard built"
else
    log_warn "Dashboard dist already exists — skipping. Run ./scripts/build-all.sh to rebuild."
fi
cd "$PROJECT_DIR"

# ---------------------------------------------------------------------------
# 9. Build Go library
# ---------------------------------------------------------------------------
log_step "9/9 — Build and test Go library"
GO_DIR="$PROJECT_DIR/go"
if [[ -d "$GO_DIR" ]]; then
    cd "$GO_DIR"
    go build ./...
    go test ./...
    cd "$PROJECT_DIR"
    log_info "Go library built and tested"
else
    log_warn "Go library not found at $GO_DIR — skipping"
fi

# ---------------------------------------------------------------------------
# 10. Environment file
# ---------------------------------------------------------------------------
if [[ ! -f "$PROJECT_DIR/.env" ]]; then
    cp "$PROJECT_DIR/.env.example" "$PROJECT_DIR/.env"
    log_info "Created $PROJECT_DIR/.env — edit it to customize settings"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "========================================"
echo -e "${GREEN}Setup complete!${NC}"
echo "========================================"
echo ""
echo -e "${BLUE}Prerequisites installed:${NC}"
echo "  • Mosquitto MQTT broker (localhost:1883)"
echo "  • Node.js $(node --version 2>/dev/null || echo 'N/A') + pnpm $(pnpm --version 2>/dev/null || echo 'N/A')"
echo "  • Rust $(rustc --version 2>/dev/null || echo 'N/A')"
echo "  • Go $(go version 2>/dev/null | awk '{print $3}' || echo 'N/A')"
echo "  • Zigbee2MQTT at $Z2M_DIR"
echo ""
echo -e "${BLUE}Project built:${NC}"
echo "  • Release binary: $PROJECT_DIR/target/release/button-hub"
echo "  • Dashboard: $PROJECT_DIR/crates/dashboard/dist/"
echo "  • Go library: $PROJECT_DIR/go/"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo ""
echo "  1. Plug in your Sonoff ZBDongle-P and verify the serial port:"
echo "     ls /dev/ttyUSB* /dev/ttyACM*"
echo ""
echo "  2. If the port differs from $Z2M_CONFIG,"
echo "     edit the file and update serial.port."
echo ""
echo "  3. Start Zigbee2MQTT:"
echo "     sudo systemctl start zigbee2mqtt"
echo "     sudo journalctl -u zigbee2mqtt -f"
echo ""
echo "  4. Start button-hub:"
echo "     $PROJECT_DIR/scripts/run-prod.sh"
echo ""
echo "  5. Open the dashboard:"
echo "     http://localhost:3000/dashboard/"
echo ""
echo "  6. Pair a button (press and hold pairing hole until LED blinks),"
echo "     then run a Go demo app:"
echo "     go run $PROJECT_DIR/go/cmd/demo --button <button_name>"
echo ""
