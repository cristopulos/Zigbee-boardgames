# Changelog

All notable changes to the button-hub project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Fixed

- **SSE connection timeout** (`go/sse.go`): Added 30-second idle timeout to detect dead connections. Previously, if the network dropped or button-hub restarted, the SSE read would block indefinitely. Now uses a goroutine-based read with timer to unblock on timeout.

### Changed

- **Go client resilience**: `Listen` and `client.Listen` now properly handle SSE keep-alive comment lines (`: ` prefix) and return errors on 30-second read timeouts, enabling automatic reconnection via exponential backoff in `Listen`.

## [0.1.0] - Initial release

- Zigbee button event bridging via Zigbee2MQTT + Mosquitto
- REST API on port 3000 with button registration
- Server-Sent Events (SSE) stream at `/api/events/stream`
- Leptos-based web dashboard
- Go client library in `go/`
- timer-switcher GUI application
