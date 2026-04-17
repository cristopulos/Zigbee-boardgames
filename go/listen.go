package gobutton

// Package gobutton provides a high-level Listen function that handles
// button registration, SSE streaming with automatic reconnection, and cleanup.

import (
	"context"
	"fmt"
	"time"
)

// Listen registers the button, opens the SSE stream, and invokes handler
// for every matching event. It reconnects automatically with exponential
// backoff until the context is cancelled.
// On graceful shutdown it also unregisters the button.
func Listen(ctx context.Context, apiURL, buttonID string, handler func(Event)) error {
	if err := Register(ctx, apiURL, buttonID); err != nil {
		fmt.Printf("[gobutton] register warning for %s: %v\n", buttonID, err)
	} else {
		fmt.Printf("[gobutton] registered %s\n", buttonID)
	}

	client := NewClient(apiURL)
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			_ = Unregister(context.Background(), apiURL, buttonID)
			return ctx.Err()
		default:
		}

		err := client.Listen(ctx, buttonID, handler)
		if err != nil {
			fmt.Printf("[gobutton] sse error for %s: %v (retry in %v)\n", buttonID, err, backoff)
		}

		select {
		case <-ctx.Done():
			_ = Unregister(context.Background(), apiURL, buttonID)
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
