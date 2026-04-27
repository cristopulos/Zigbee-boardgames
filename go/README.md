# gobutton — Go client for button-hub

This is a tiny Go library that lets your Go applications react to Zigbee button presses via the `button-hub` REST API.

## Install

```bash
go get github.com/cristopulos/button-hub/go
```

## One-line usage

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

`Listen` does three things automatically:
1. **Registers** the button with button-hub (`POST /buttons`)
2. **Opens** an SSE stream to `/api/events/stream`
3. **Reconnects** with exponential backoff if the stream drops

When the context is cancelled it also unregisters the button.

## CLI demo

A reference CLI lives in `cmd/demo`:

```bash
go run ./go/cmd/demo --api http://localhost:3000 --button kitchen_button
```

## Data model

```go
type Event struct {
    ButtonID  string     `json:"button_id"`
    Action    ActionType `json:"action"`   // Single, Double, LongPress, or Unknown(...)
    Battery   *uint8     `json:"battery"`
    Timestamp string     `json:"timestamp"`
}
```

All action values serialize as plain strings (e.g., `"Single"`, `"LongPress"`).

## Advanced: manual SSE client

If you want more control, use the lower-level `Client`:

```go
client := gobutton.NewClient("http://localhost:3000")
err := client.Listen(ctx, "kitchen_button", func(e gobutton.Event) {
    // handle event
})
```

`client.Listen` blocks until the context is cancelled or the stream breaks. It does **not** auto-reconnect — wrap it in your own loop if you need that.

### Connection resilience

Both `Listen` and `client.Listen` implement connection monitoring:

- **30-second idle timeout** — If no SSE data is received for 30 seconds (e.g., network drop, server restart), the stream returns an error. This prevents hung connections from blocking indefinitely.
- **Automatic reconnection** (Listen only) — On disconnect, `Listen` retries with exponential backoff (1s → 2s → 4s ... up to 30s max) until the context is cancelled.
- **Keep-alive support** — The SSE handler properly ignores comment/ping lines (`:` prefix) sent by servers as keep-alive heartbeats.
