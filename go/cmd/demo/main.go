// demo is a reference CLI that listens for button events and prints them to stdout.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	gobutton "github.com/cristopulos/button-hub/go"
)

func main() {
	apiURL := flag.String("api", "http://localhost:3000", "button-hub API base URL")
	buttonID := flag.String("button", "", "button_id to listen for (required)")
	flag.Parse()

	if *buttonID == "" {
		fmt.Fprintln(os.Stderr, "Usage: demo --button <button_id> [--api <url>]")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("[demo] listening for button_id=%s on %s\n", *buttonID, *apiURL)

	if err := gobutton.Listen(ctx, *apiURL, *buttonID, func(e gobutton.Event) {
		battery := "?"
		if e.Battery != nil {
			battery = fmt.Sprintf("%d%%", *e.Battery)
		}
		fmt.Printf("[demo] event: action=%s battery=%s at %s\n", e.Action, battery, e.Timestamp)
	}); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "[demo] error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[demo] shutting down")
}
