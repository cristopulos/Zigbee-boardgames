package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	gobutton "github.com/cristopulos/button-hub/go"
)

// Default numInitiatives (Naalu disabled by default)
const defaultNumInitiatives = 8

type TrackerState struct {
	current int
	enabled []bool
	mu      sync.RWMutex
}

func NewTrackerState(start, numInitiatives int) *TrackerState {
	enabled := make([]bool, numInitiatives)
	for i := range enabled {
		enabled[i] = true
	}
	if start < 0 || start >= numInitiatives {
		start = 0
	}
	return &TrackerState{
		current: start,
		enabled: enabled,
	}
}

func (s *TrackerState) Current() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *TrackerState) Enabled(i int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled[i]
}

func (s *TrackerState) SetCurrent(i int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = i
}

func (s *TrackerState) Next() {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.enabled)
	start := s.current
	for {
		s.current = (s.current + 1) % n
		if s.enabled[s.current] {
			return
		}
		if s.current == start {
			return
		}
	}
}

func (s *TrackerState) Prev() {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.enabled)
	start := s.current
	for {
		s.current = (s.current - 1 + n) % n
		if s.enabled[s.current] {
			return
		}
		if s.current == start {
			return
		}
	}
}

func (s *TrackerState) Reset(start int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if start >= 0 && start < len(s.enabled) && s.enabled[start] {
		s.current = start
	} else {
		for i := 0; i < len(s.enabled); i++ {
			if s.enabled[i] {
				s.current = i
				return
			}
		}
	}
}

func (s *TrackerState) ToggleEnabled(i int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i < 0 || i >= len(s.enabled) {
		return
	}
	s.enabled[i] = !s.enabled[i]
	if !s.enabled[i] && s.current == i {
		for j := 0; j < len(s.enabled); j++ {
			idx := (i + 1 + j) % len(s.enabled)
			if s.enabled[idx] {
				s.current = idx
				return
			}
		}
	}
}

func (s *TrackerState) AllEnabled() []bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]bool, len(s.enabled))
	copy(result, s.enabled)
	return result
}

func parseButtonIDs(s string) []string {
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

func main() {
	apiURL := flag.String("api", "http://localhost:3000", "button-hub API URL")
	buttonFlag := flag.String("button", "", "comma-separated button IDs (optional)")
	naalu := flag.Bool("naalu", false, "include Naalu initiative 0")
	startFlag := flag.Int("start", 1, "starting initiative number")
	flag.Parse()

	// With --naalu: 9 cards (0-8), Without --naalu: 8 cards (1-8)
	// Card data always contains all 9 entries, but UI only shows what we need
	numInitiatives := defaultNumInitiatives
	if *naalu {
		numInitiatives = 9 // Include Naalu
	}
	start := *startFlag

	// Validate start against numInitiatives
	// If Naalu excluded (8 cards), valid range is 1-8
	// If Naalu included (9 cards), valid range is 0-8
	if start < 0 || start >= numInitiatives {
		start = 1 // Default to Leadership
	}

	state := NewTrackerState(start, numInitiatives)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	refreshCh := make(chan struct{}, 1)

	buttonIDs := parseButtonIDs(*buttonFlag)
	if len(buttonIDs) > 0 {
		for _, bid := range buttonIDs {
			go func(buttonID string) {
				_ = gobutton.Listen(ctx, *apiURL, buttonID, func(e gobutton.Event) {
					switch e.Action {
					case gobutton.ActionSingle:
						state.Next()
					case gobutton.ActionDouble:
						state.Reset(start)
					}
					select {
					case refreshCh <- struct{}{}:
					default:
					}
				})
			}(bid)
		}
	}

	ui := NewTrackerUI(state, refreshCh, numInitiatives)

	go func() {
		<-quit
		cancel()
		ui.Stop()
	}()

	if len(buttonIDs) > 0 {
		fmt.Printf("Initiative Tracker started, %d initiatives, listening for buttons: %s\n", numInitiatives, strings.Join(buttonIDs, ", "))
	} else {
		fmt.Printf("Initiative Tracker started, %d initiatives (keyboard-only mode)\n", numInitiatives)
	}

	ui.Show()
}