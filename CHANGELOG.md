# Changelog

All notable changes to the button-hub project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Rust client library** (`crates/client/`): New `button-client` crate providing async Rust client for button-hub. Replaces the Go `go/` library for new Rust applications. Features: SSE stream handling via `tokio` + `reqwest`, 30-second idle timeout, automatic reconnection with exponential backoff. Reuses event types from `button_core`.
- **Rust timer-switcher** (`apps/timer-switcher-rs/`): New Rust/egui app replacing the Go/Fyne version. Uses `crates/client/` for SSE connectivity. Features: configurable timers, 1:1 or cycle button mapping, 1 Hz tick gating, 100 ms repaint interval.
- **Rust initiative-tracker** (`apps/initiative-tracker-rs/`): New Rust/egui app replacing the Go/Fyne version. Uses `crates/client/` for SSE connectivity. Features: TI4 strategy card cycling, keyboard and button controls, `--naalu` flag for Naalu initiative.

### Changed

- **Go timer-switcher removed**: `apps/timer-switcher/` (Go/Fyne) deleted. Use `apps/timer-switcher-rs/` (Rust/egui) instead.
- **Go initiative-tracker removed**: `apps/initiative-tracker/` (Go/Fyne) deleted. Use `apps/initiative-tracker-rs/` (Rust/egui) instead.
- **Go client resilience**: `Listen` and `client.Listen` now use a single persistent read goroutine with a body-close goroutine on context cancellation. This replaces the previous per-iteration goroutine pattern which leaked on timeout/cancel. SSE keep-alive comment lines (`: ` prefix) are handled and 30-second read timeouts trigger automatic reconnection via exponential backoff.

### Fixed

- **SSE goroutine leak** (`go/sse.go`): Replaced per-iteration goroutine spawn with a single persistent read goroutine. Previously, when `ctx.Done()` or the idle timer fired, the blocked read goroutine was leaked — one per timeout/reconnect. Over extended periods this caused goroutine accumulation and apps stopping responding. Now a single goroutine reads continuously and a separate goroutine closes `resp.Body` on context cancellation. Non-blocking sends on `lineChan` and `readErrChan` prevent leaks if the main loop returns while the read goroutine is trying to send. A `bodyCloseDone` channel ensures the body-close goroutine exits on normal return, preventing double-close with the defer and further goroutine accumulation.
- **SSE connection timeout** (`go/sse.go`): Added 30-second idle timeout to detect dead connections. Previously, if the network dropped or button-hub restarted, the SSE read would block indefinitely. Now uses a goroutine-based read with timer to unblock on timeout.
- **HTTP client timeout removed** (`go/sse.go`): Removed the 60-second `http.Client.Timeout` from `NewClient`. SSE streams are long-lived; the timeout was killing healthy connections. Use context cancellation for shutdown instead.

## [0.1.0] - Initial release

- Zigbee button event bridging via Zigbee2MQTT + Mosquitto
- REST API on port 3000 with button registration
- Server-Sent Events (SSE) stream at `/api/events/stream`
- Leptos-based web dashboard
- Go client library in `go/`
- timer-switcher GUI application
