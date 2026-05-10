# AGENTS.md — button-hub

Compact instructions for AI agents working in this repo. Every line answers: "Would an agent likely miss this without help?"

## Repo Layout

- **Root contains system packages** (`libxi-dev.tar.gz`, etc.) — ignore them. The actual project lives entirely inside `button-hub/`.
- **Rust workspace** (7 members): `crates/core`, `crates/mqtt`, `crates/api`, `crates/dashboard`, `crates/client`, `apps/timer-switcher-rs`, `apps/initiative-tracker-rs`, and the root package `.` (binary entrypoint at `src/main.rs`).
- **Go library**: `go/` — deprecated, kept for backwards compatibility. New apps use `crates/client/`.
- **Rust apps**: `apps/timer-switcher-rs/` and `apps/initiative-tracker-rs/` — both use `crates/client/` for SSE connectivity and egui for UI.

## Developer Commands

### Rust (button-hub/)
```bash
cd button-hub

# Full verification pipeline (fmt → clippy → test → dev build → release build)
./scripts/test-all.sh

# Quick dev run (debug logs, prevents duplicate instances)
./scripts/run-dev.sh

# Production run (builds dashboard WASM first if missing)
./scripts/run-prod.sh

# Run with custom .env file
./scripts/run-with-env.sh .env.local

# Dashboard hot-reload dev server (port 9090, proxies API to :3000)
./scripts/run-dashboard-dev.sh

# Single test
cargo test --workspace -p <crate-name> <test-name>
```

### Rust Apps (timer-switcher-rs, initiative-tracker-rs)
```bash
# Run timer-switcher
cargo run -p timer-switcher

# Run initiative-tracker
cargo run -p initiative-tracker

# Tests for timer-switcher
cargo test -p timer-switcher

# Tests for initiative-tracker
cargo test -p initiative-tracker
```

## Build Order Constraints

1. **Dashboard WASM must be built before production runs.** `run-prod.sh` checks for `crates/dashboard/dist/index.html` and auto-builds if missing.
2. **Trunk dev server requires API to be running first** — it proxies `/api/`, `/health`, `/buttons`, `/events` to `localhost:3000`.
3. **All shell scripts use `set -euo pipefail`** and prevent duplicate `button-hub` instances via `pgrep`.

## Environment

- `.env` loaded via `dotenvy` in `src/main.rs`. `.env.example` is the reference.
- Key vars: `MQTT_BROKER_HOST` (default `localhost`), `MQTT_BROKER_PORT` (1883), `API_PORT` (3000), `MQTT_CLIENT_ID` (auto-generated as `button-hub-<PID>` if not set).
- **MQTT client ID must be unique per run** — the default PID-based ID prevents broker conflicts.

## Architecture Notes

### Rust crates
- `core`: Event types, ButtonRegistry, Hub (async event dispatch). `event.rs` has no feature gates; everything else requires `tokio` feature.
- `mqtt`: rumqttc client, parses Zigbee2MQTT JSON, sends events to Hub via mpsc channel.
- `api`: Axum REST API + SSE stream. `build_router()` creates all routes; `nest_service` (not `route_service`) serves dashboard static files at `/dashboard/`.
- `dashboard`: Leptos WASM app. Built with Trunk. `Trunk.toml` sets `public_url = "/dashboard/"`.

### Rust client library (`crates/client/`)
- Reuses event types from `button_core`.
- `Client::listen` (low-level): 30-second idle timeout, no auto-reconnect, SSE keep-alive comment lines (`: ` prefix) are handled.
- Auto-reconnect with exponential backoff is handled by the calling code or the app's main loop.
- **SSE client must not set HTTP timeout** — the stream is long-lived. Context cancellation handles shutdown.

### timer-switcher app (`apps/timer-switcher-rs/`)
- Architecture: `timer.rs` (logic, global state) → `main.rs` (async wiring) → `app.rs` (egui UI).
- Global `TIMER_STATE` (`Arc<RwLock<TimerState>>`) bridges tokio async tasks and egui's main thread.
- Button mapping: 1:1 direct map when `len(buttonIDs) == len(timers)`, otherwise cycle mode. In direct map, pressing an already-active button falls back to cycling.
- **1 Hz tick gating**: `TimerState::tick()` uses `Instant` to ensure only one tick per real second. Call `tick()` from the UI's `update()` loop.
- **100 ms repaint interval**: `ctx.request_repaint_after(100ms)` ensures button-press state changes are visible promptly.

### initiative-tracker app (`apps/initiative-tracker-rs/`)
- Architecture: `tracker.rs` (logic, global state) → `main.rs` (async wiring) → `app.rs` (egui UI).
- Global `TRACKER_STATE` (`Arc<RwLock<TrackerState>>`) bridges tokio async tasks and egui's main thread.
- 9 strategy cards defined in `INITIATIVE_DATA` (index 0 = Naalu). Naalu is disabled by default; use `--naalu` to include it.
- `--naalu` controls the `offset` field: `offset=0` shows indices 0-8, `offset=1` shows indices 1-8 (Naalu hidden).
- **100 ms repaint interval** (same as timer-switcher).

## egui App Gotchas

1. **Repaint polling**: egui doesn't redraw automatically between frames. Use `ctx.request_repaint_after()` with a short interval (e.g., 100ms) if you need to update the UI based on external state (button events, timers).
2. **Thread safety**: egui runs on the main thread; tokio tasks must communicate via `Arc<RwLock<T>>` or channels. Direct UI calls from async tasks will panic.
3. **`ViewportBuilder`**: Always set `with_resizable(true)` or the window may not be resizeable.
4. **Color32 premultiplication**: `egui::Color32::from_rgba_unmultiplied` stores values internally premultiplied. Use it only when you need to set alpha explicitly on an already-RGB-known color.

## Zigbee2MQTT Action Mapping

| Raw Input | Normalized To |
|-----------|---------------|
| `single` | `Single` |
| `double` | `Double` |
| `long_press`, `hold`, `long` | `LongPress` |
| anything else | `Unknown(...)` |

- SNZB-01P sends `"long"` for long presses (not `"long_press"` or `"hold"`).
- All action values serialize as plain strings in JSON (e.g. `"Unknown(shake)"`).

## Testing

- **Rust**: `cargo test --workspace`. Clippy runs with `-D warnings`.
- **Go library**: `go test ./...` in `go/`. 24 tests passing. (Deprecated — use `crates/client/` for new apps.)
- **Rust apps**: `cargo test -p timer-switcher`, `cargo test -p initiative-tracker`

## Deployment Notes

- Only `button-hub` needs `MQTT_BROKER_HOST` env var pointing to the remote MQTT broker.
- `timer-switcher` and `initiative-tracker` only need `--api http://<machine>:3000` — no MQTT access required.
- `Zigbee2MQTT` needs `pnpm` in system PATH for its systemd service.
