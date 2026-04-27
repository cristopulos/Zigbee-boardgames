# button-hub

A Rust application that bridges Zigbee2MQTT button events to async callbacks and a REST API.

## Architecture

```
[SNZB-01P ×N] --Zigbee--> [ZBDongle-P] --USB--> [Zigbee2MQTT]
                                                   --MQTT--> [Mosquitto :1883]
                                                                --rumqttc--> [button-hub]
                                                                                 --> [ButtonRegistry: callbacks]
                                                                                 --> [REST API :3000]
                                                                                 --> [Dashboard /dashboard/]
                                                                                 --> [Go apps via SSE]
```

## Prerequisites

- Rust stable (install via [rustup](https://rustup.rs/))
- Mosquitto MQTT broker
- Zigbee2MQTT + Node.js 20
- Sonoff ZBDongle-P (or ZBDongle-E)
- Sonoff SNZB-01P buttons

## Setup

### Fresh machine (automated)

Run the setup script for your distribution. It installs all prerequisites, configures Mosquitto, installs Zigbee2MQTT, and builds everything:

**Debian / Ubuntu:**
```bash
./scripts/setup.sh
```

**Arch Linux:**
```bash
./scripts/setup-arch.sh
```

You will be prompted for your sudo password.

### Development machine (build only)

If prerequisites are already installed, just build the project:

```bash
./scripts/dev-setup.sh
```

### Manual setup

If you prefer manual control, follow the steps below.

## First-time hardware setup

1. Plug the ZBDongle-P into a USB port.
2. Find the serial port:
   ```bash
   ls /dev/ttyUSB* /dev/ttyACM*
   ```
3. Update `/opt/zigbee2mqtt/data/configuration.yaml` with the correct `serial.port`.
4. Start the Zigbee2MQTT service:
   ```bash
   sudo systemctl start zigbee2mqtt
   ```
5. Press and hold the pairing hole on the SNZB-01P button until the LED blinks.
6. Watch the logs for the new friendly name:
   ```bash
   journalctl -u zigbee2mqtt -f
   ```
7. Register the button in `src/main.rs` (for Rust callbacks) or use the Go library / `POST /buttons` API (for external apps).

## Running

### Quick start (production)
```bash
./scripts/run-prod.sh
```

### Development
```bash
# Terminal 1: API server
./scripts/run-dev.sh

# Terminal 2: dashboard hot-reload dev server
./scripts/run-dashboard-dev.sh
```

### With a custom environment file
```bash
./scripts/run-with-env.sh .env.local
```

### Available scripts

| Script | Purpose |
|--------|---------|
| `./scripts/setup.sh` | Full setup on a fresh Debian/Ubuntu machine (installs all deps) |
| `./scripts/setup-arch.sh` | Full setup on a fresh Arch Linux machine (installs all deps) |
| `./scripts/dev-setup.sh` | Build-only setup for development (assumes deps installed) |
| `./scripts/build-all.sh` | Build dashboard WASM + release binary |
| `./scripts/run-prod.sh` | Run production release |
| `./scripts/run-dev.sh` | Run API in dev mode (`RUST_LOG=debug`) |
| `./scripts/run-dashboard-dev.sh` | Trunk dev server on port 9090 |
| `./scripts/run-with-env.sh <file>` | Run with a specific `.env` file |
| `./scripts/test-all.sh` | fmt + clippy + tests + dev/release builds |
| `./scripts/test-setup.sh` | Validate setup scripts (syntax, permissions, idempotency) |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MQTT_BROKER_HOST` | `localhost` | MQTT broker hostname |
| `MQTT_BROKER_PORT` | `1883` | MQTT broker port |
| `MQTT_CLIENT_ID` | `button-hub-<PID>` | MQTT client identifier |
| `API_PORT` | `3000` | REST API listening port |
| `RUST_LOG` | `info` | Tracing log filter |

## REST API Reference

| Method | Path | Description | Example Response |
|--------|------|-------------|------------------|
| `GET` | `/health` | Service health check | `{"status":"ok","timestamp":"2024-01-01T12:00:00Z"}` |
| `GET` | `/buttons` | List registered button IDs | `{"buttons":["button_1","button_2"]}` |
| `POST` | `/buttons` | Register a new button ID | `{"registered":true,"button_id":"kitchen_button"}` |
| `DELETE` | `/buttons/:button_id` | Unregister a button ID | `{"unregistered":true,"button_id":"kitchen_button"}` or `404` |
| `GET` | `/events?limit=N` | Get latest events (default 20, max 100) | `{"events":[...],"count":2}` |
| `GET` | `/events/:button_id` | Get last event for a specific button | `{"button_id":"button_1","action":"Single",...}` or `404` |
| `GET` | `/events/:button_id/history?limit=N` | Get event history for a specific button | `{"events":[...],"count":5}` |
| `GET` | `/api/events/stream` | SSE stream of live button events | `data: {"button_id":"btn1","action":"Single",...}` |

## Dashboard

A Leptos-based WASM dashboard is available at `http://localhost:3000/dashboard/`.

It displays:
- **Connection status** — API and MQTT health indicators
- **Button cards** — Last action, battery level, and online status for each registered button
- **Live event timeline** — Real-time event log via Server-Sent Events

### Building the dashboard

```bash
# One-time: install Trunk and the WASM target
rustup target add wasm32-unknown-unknown
cargo install trunk --locked

# Build dashboard for production
cd crates/dashboard
trunk build --release
cd ../..

# Run the full application
cargo run --release
```

### Dashboard development server

```bash
cd crates/dashboard
# In a separate terminal, start the API server first:
#   cargo run --release
# Then start the Trunk dev server (proxies API calls to :3000):
trunk serve --port 9090
```

Open `http://localhost:9090/dashboard/` in your browser.

## Supported Actions

The following action strings are recognized (case-sensitive in serialized JSON):

| Raw Input | Normalized To |
|-----------|---------------|
| `single` | `Single` |
| `double` | `Double` |
| `long_press`, `hold`, `long` | `LongPress` |
| Any other string | `Unknown(...)` |

## Adding a new button

### Rust (inside button-hub)

```rust
registry.register(Button::new("my_button", |event| async move {
    tracing::info!("My button pressed: {:?} (battery: {:?}%)", event.action, event.battery);
}));
```

### Go (external app)

A Go client library lives in `go/`.

```bash
go get github.com/cristopulos/button-hub/go
```

```go
package main

import (
    "context"
    gobutton "github.com/cristopulos/button-hub/go"
)

func main() {
    ctx := context.Background()
    _ = gobutton.Listen(ctx, "http://localhost:3000", "kitchen_button", func(e gobutton.Event) {
        println("Button pressed!", e.Action)
    })
}
```

`gobutton.Listen` automatically:
1. Registers the button via `POST /buttons`
2. Opens an SSE stream to `/api/events/stream`
3. Calls your handler for every matching event
4. Reconnects with exponential backoff on disconnect
5. Unregisters the button when the context is cancelled

#### Reference CLI demo

```bash
go run ./go/cmd/demo --api http://localhost:3000 --button kitchen_button
```

## timer-switcher

A Fyne-based GUI app that displays configurable timers and cycles through them based on button presses or keyboard input. Source lives in `apps/timer-switcher/`.

### Connection behavior

timer-switcher connects to button-hub via Server-Sent Events (SSE) and automatically reconnects if the connection is lost. If button-hub restarts or the network drops, it will retry with exponential backoff (1s → 2s → 4s ... up to 30s) until the connection is restored. Timer state persists locally regardless of connection status.

### Features

- Displays N configurable timers (minimum 2) in a horizontal row
- One timer is active at a time, highlighted with a cyan accent
- Active timer counts up in `HH:MM:SS` format
- Timer text scales dynamically with window resize
- Click any timer card to switch to it

### Building

```bash
cd apps/timer-switcher
./build.sh
# or: go build -o timer-switcher .
```

### Usage

```bash
# Single button — cycles through 3 timers
./timer-switcher --button kitchen_button

# Multiple buttons with custom names
./timer-switcher --button btn1,btn2,btn3 --timers "Player 1,Player 2,Player 3"

# With debug logging
./timer-switcher --button snzb-01p-01 --debug
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--button` | (required) | Comma-separated button IDs (e.g. `snzb-01p-01` or `btn1,btn2,btn3`) |
| `--timers` | `Timer 1,Timer 2,Timer 3` | Comma-separated timer names (minimum 2) |
| `--api` | `http://localhost:3000` | button-hub API base URL |
| `--debug` | `false` | Enable verbose event logging |

### Controls

**Keyboard:**

| Key | Action |
|-----|--------|
| `SPACE` | Switch to the next timer (cycles) |
| `ENTER` | Reset the active timer to 00:00:00 |
| `P` | Pause/resume the active timer |
| `ESC` | Quit |

**Mouse:**

| Action | Effect |
|--------|--------|
| Click timer card | Switch to that timer |

**Remote button:**

| Action | Effect |
|--------|--------|
| Single press | Switch to next timer (cycle mode) or activate mapped timer (1:1 mode). If the mapped timer is already active, falls back to cycling. |
| Double press | Pause/resume the active timer |

**Paused state:** When paused, the active timer's time displays in amber and stops incrementing.

## Tests

### button-hub (Rust)
```bash
cd button-hub
cargo test
```
**31 tests passing**

### timer-switcher (Go)
```bash
cd apps/timer-switcher
go test ./...
```
**23 tests passing**

## Event JSON Schema

```json
{
  "button_id": "button_1",
  "action": "Single",
  "battery": 85,
  "timestamp": "2024-01-15T09:30:00Z"
}
```

All action values serialize as plain strings (e.g., `Single`, `LongPress`, `Unknown(shake)`).

### Field descriptions

- `button_id` — Friendly name of the Zigbee button (1-64 characters)
- `action` — Button action: `Single`, `Double`, `LongPress`, or `Unknown(...)` for unrecognized actions
- `battery` — Optional battery percentage reported by the device
- `timestamp` — UTC timestamp when the event was received
