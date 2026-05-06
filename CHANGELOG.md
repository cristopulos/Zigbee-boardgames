# Changelog

All notable changes to the button-hub project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Fixed

- **SSE goroutine leak** (`go/sse.go`): Replaced per-iteration goroutine spawn with a single persistent read goroutine. Previously, when `ctx.Done()` or the idle timer fired, the blocked read goroutine was leaked — one per timeout/reconnect. Over extended periods this caused goroutine accumulation and apps stopping responding. Now a single goroutine reads continuously and a separate goroutine closes `resp.Body` on context cancellation. Non-blocking sends on `lineChan` and `readErrChan` prevent leaks if the main loop returns while the read goroutine is trying to send. A `bodyCloseDone` channel ensures the body-close goroutine exits on normal return, preventing double-close with the defer and further goroutine accumulation.
- **SSE connection timeout** (`go/sse.go`): Added 30-second idle timeout to detect dead connections. Previously, if the network dropped or button-hub restarted, the SSE read would block indefinitely. Now uses a goroutine-based read with timer to unblock on timeout.
- **HTTP client timeout removed** (`go/sse.go`): Removed the 60-second `http.Client.Timeout` from `NewClient`. SSE streams are long-lived; the timeout was killing healthy connections. Use context cancellation for shutdown instead.

### Changed

- **Go client resilience**: `Listen` and `client.Listen` now use a single persistent read goroutine with a body-close goroutine on context cancellation. This replaces the previous per-iteration goroutine pattern which leaked on timeout/cancel. SSE keep-alive comment lines (`: ` prefix) are handled and 30-second read timeouts trigger automatic reconnection via exponential backoff.

## [0.1.0] - Initial release

- Zigbee button event bridging via Zigbee2MQTT + Mosquitto
- REST API on port 3000 with button registration
- Server-Sent Events (SSE) stream at `/api/events/stream`
- Leptos-based web dashboard
- Go client library in `go/`
- timer-switcher GUI application
