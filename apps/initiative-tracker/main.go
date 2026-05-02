package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	gobutton "github.com/cristopulos/button-hub/go"
)

const numInitiatives = 9

type TrackerState struct {
	current int
	enabled [numInitiatives]bool
	mu      sync.RWMutex
}

func NewTrackerState(start int) *TrackerState {
	enabled := [numInitiatives]bool{true, true, true, true, true, true, true, true, true}
	if start < 0 || start >= numInitiatives {
		start = 1
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

	start := s.current
	for {
		s.current = (s.current + 1) % numInitiatives
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

	start := s.current
	for {
		s.current = (s.current - 1 + numInitiatives) % numInitiatives
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
	if start >= 0 && start < numInitiatives && s.enabled[start] {
		s.current = start
	} else {
		for i := 0; i < numInitiatives; i++ {
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
	if i < 0 || i >= numInitiatives {
		return
	}
	s.enabled[i] = !s.enabled[i]
	if !s.enabled[i] && s.current == i {
		for j := 0; j < numInitiatives; j++ {
			idx := (i + 1 + j) % numInitiatives
			if s.enabled[idx] {
				s.current = idx
				return
			}
		}
	}
}

func (s *TrackerState) AllEnabled() [numInitiatives]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

func main() {
	apiURL := flag.String("api", "http://localhost:3000", "button-hub API URL")
	buttonID := flag.String("button", "", "Button ID (optional; keyboard-only if empty)")
	startFlag := flag.Int("start", 1, "Starting initiative number (0-8)")
	flag.Parse()

	state := NewTrackerState(*startFlag)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	refreshCh := make(chan struct{}, 1)

	if *buttonID != "" {
		go func() {
			err := gobutton.Listen(ctx, *apiURL, *buttonID, func(e gobutton.Event) {
				switch e.Action {
				case gobutton.ActionSingle:
					state.Next()
				case gobutton.ActionDouble:
					state.Reset(*startFlag)
				}
				select {
				case refreshCh <- struct{}{}:
				default:
				}
			})
			if err != nil && ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "button listen error: %v\n", err)
			}
		}()
	}

	ui := NewTrackerUI(state, refreshCh)

	go func() {
		<-quit
		cancel()
		ui.Stop()
	}()

	ui.Show()
}