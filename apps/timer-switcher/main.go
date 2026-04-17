// Timer Switcher — a TUI app that cycles through named timers via Zigbee button presses.
//
// Usage:
//
//	timer-switcher --button <id>[,<id>...] [--timers <names>] [--api <url>] [--debug]
//
// Button behavior:
//   - When the number of buttons matches the number of timers, each button
//     maps directly to its corresponding timer (1:1 mode).
//   - Otherwise all buttons cycle through the timers in sequence (cycle mode).
//   - Double-click any button to pause/resume the active timer.
//
// TUI controls:
//   - SPACE: switch to the next timer
//   - ENTER: reset timer
//   - P: pause/resume
//   - ESC: quit
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	gobutton "github.com/cristopulos/button-hub/go"
)

// main parses flags, starts a button listener per ID, and runs the TUI.
// Button presses trigger timer switches; see package-level documentation for
// how the mapping mode is selected based on button/timer counts.
func main() {
	apiURL := flag.String("api", "http://localhost:3000", "button-hub API base URL")
	buttonFlag := flag.String("button", "", "comma-separated button IDs to listen for (required)")
	timersFlag := flag.String("timers", "Timer 1,Timer 2,Timer 3", "comma-separated timer names")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	buttonIDs := parseButtonIDs(*buttonFlag)
	if len(buttonIDs) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: timer-switcher --button <button_id>[,<button_id>...] [--api <url>] [--timers <names>] [--debug]")
		os.Exit(1)
	}

	names := parseTimerNames(*timersFlag)
	if len(names) < 2 {
		fmt.Fprintf(os.Stderr, "Error: need at least 2 timers, got %d\n", len(names))
		os.Exit(1)
	}

	tm := NewTimerManager(names)
	tm.SetDebug(*debug)
	ui := NewTimerUI(tm)
	ui.debug = *debug

	// Start a button listener for each button ID
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	directMap := len(buttonIDs) == len(names)
	if *debug {
		fmt.Printf("[main] directMap=%v (buttons=%d, timers=%d)\n", directMap, len(buttonIDs), len(names))
	}
	if !directMap && len(buttonIDs) > 1 {
		fmt.Printf("Note: %d buttons with %d timers — all buttons will cycle\n", len(buttonIDs), len(names))
	}
	for i, bid := range buttonIDs {
		idx := i // capture for closure
		go func(buttonID string) {
			if *debug {
				fmt.Printf("[main] starting listener for button=%s idx=%d\n", buttonID, idx)
			}
			_ = gobutton.Listen(ctx, *apiURL, buttonID, func(e gobutton.Event) {
				if *debug {
					fmt.Printf("[remote] received: button_id=%s action=%s battery=%v\n", e.ButtonID, e.Action, e.Battery)
				}
				switch e.Action {
				case gobutton.ActionSingle:
					if *debug {
						fmt.Printf("[remote] handling Single: button=%s idx=%d directMap=%v\n", e.ButtonID, idx, directMap)
					}
					if directMap {
						tm.SwitchTo(idx)
					} else {
						tm.Cycle()
					}
					ui.refreshAll()
					if *debug {
						fmt.Printf("[remote] Single handled: active=%d paused=%v\n", tm.ActiveIndex(), tm.IsPaused())
					}
				case gobutton.ActionDouble:
					if *debug {
						fmt.Printf("[remote] handling Double: button=%s -> TogglePause\n", e.ButtonID)
					}
					tm.TogglePause()
					ui.refreshAll()
				default:
					if *debug {
						fmt.Printf("[remote] ignored: expected Single/Double, got %s\n", e.Action)
					}
				}
			})
		}(bid)
	}

	mode := "cycle"
	if directMap {
		mode = "direct map (1:1)"
	}
	fmt.Printf("Timer Switcher started with %d timers, %d buttons, mode=%s\n", len(names), len(buttonIDs), mode)
	fmt.Printf("Listening for buttons: %s\n", strings.Join(buttonIDs, ", "))
	fmt.Println("Controls: SPACE = switch, ENTER = reset, P = pause, ESC = quit")

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

func parseButtonIDs(s string) []string {
	return parseTimerNames(s) // same comma-split and trim logic
}
