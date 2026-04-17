package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	gobutton "github.com/cristopulos/button-hub/go"
)

func main() {
	apiURL := flag.String("api", "http://localhost:3000", "button-hub API base URL")
	buttonID := flag.String("button", "", "button_id to listen for (required)")
	timersFlag := flag.String("timers", "Timer 1,Timer 2,Timer 3", "comma-separated timer names")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *buttonID == "" {
		fmt.Fprintln(os.Stderr, "Usage: timer-switcher --button <button_id> [--api <url>] [--timers <names>] [--debug]")
		os.Exit(1)
	}

	names := parseTimerNames(*timersFlag)
	if len(names) < 2 {
		fmt.Fprintf(os.Stderr, "Error: need at least 2 timers, got %d\n", len(names))
		os.Exit(1)
	}

	tm := NewTimerManager(names)
	ui := NewTimerUI(tm)

	// Start button listener in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = gobutton.Listen(ctx, *apiURL, *buttonID, func(e gobutton.Event) {
			if *debug {
				fmt.Printf("[remote] received: button_id=%s action=%s\n", e.ButtonID, e.Action)
			}
			if e.Action == gobutton.ActionSingle {
				tm.Cycle()
				ui.refreshAll()
			} else if *debug {
				fmt.Printf("[remote] ignored: expected Single, got %s\n", e.Action)
			}
		})
	}()

	fmt.Printf("Timer Switcher started with %d timers, listening for button '%s'\n", len(names), *buttonID)
	fmt.Println("Controls: SPACE = switch, ENTER = reset, ESC = quit")

	ui.Show()
}

func parseTimerNames(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
