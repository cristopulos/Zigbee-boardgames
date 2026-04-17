package main

import (
	"fmt"
	"sync"
)

// Timer represents a single named timer with an elapsed duration.
type Timer struct {
	name    string
	elapsed int // seconds
}

// TimerManager holds all timers and tracks which one is active.
type TimerManager struct {
	mu          sync.RWMutex
	timers      []Timer
	activeIndex int
	paused      bool
	debug       bool
}

// NewTimerManager creates a manager from a slice of timer names.
func NewTimerManager(names []string) *TimerManager {
	timers := make([]Timer, len(names))
	for i, n := range names {
		timers[i] = Timer{name: n, elapsed: 0}
	}
	return &TimerManager{
		timers:      timers,
		activeIndex: 0,
	}
}

// SetDebug enables or disables debug logging.
func (tm *TimerManager) SetDebug(v bool) {
	tm.debug = v
}

func (tm *TimerManager) logf(format string, args ...interface{}) {
	if tm.debug {
		fmt.Printf("[timer] "+format+"\n", args...)
	}
}

// Cycle moves the active timer to the next one (wraps around).
func (tm *TimerManager) Cycle() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	old := tm.activeIndex
	tm.activeIndex = (tm.activeIndex + 1) % len(tm.timers)
	tm.logf("Cycle: %d -> %d (paused=%v)", old, tm.activeIndex, tm.paused)
}

// SwitchTo makes the timer at the given index active.
func (tm *TimerManager) SwitchTo(index int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if index >= 0 && index < len(tm.timers) {
		old := tm.activeIndex
		tm.activeIndex = index
		tm.logf("SwitchTo(%d): %d -> %d (paused=%v)", index, old, tm.activeIndex, tm.paused)
	} else {
		tm.logf("SwitchTo(%d): ignored out of bounds", index)
	}
}

// Reset sets the active timer's elapsed time to zero.
func (tm *TimerManager) Reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.timers[tm.activeIndex].elapsed = 0
	tm.logf("Reset: timer=%d elapsed=0", tm.activeIndex)
}

// Tick increments the active timer by one second when not paused.
func (tm *TimerManager) Tick() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.paused {
		tm.logf("Tick: skipped (paused=true, timer=%d)", tm.activeIndex)
		return
	}
	tm.timers[tm.activeIndex].elapsed++
	tm.logf("Tick: timer=%d elapsed=%d", tm.activeIndex, tm.timers[tm.activeIndex].elapsed)
}

// TogglePause flips the paused state.
func (tm *TimerManager) TogglePause() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.paused = !tm.paused
	tm.logf("TogglePause: paused=%v (timer=%d)", tm.paused, tm.activeIndex)
}

// IsPaused returns true if the timer is currently paused.
func (tm *TimerManager) IsPaused() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.paused
}

// ActiveIndex returns the index of the currently active timer.
func (tm *TimerManager) ActiveIndex() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.activeIndex
}

// Count returns the number of timers.
func (tm *TimerManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.timers)
}

// TimerName returns the name of the timer at the given index.
func (tm *TimerManager) TimerName(index int) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if index < 0 || index >= len(tm.timers) {
		return ""
	}
	return tm.timers[index].name
}

// TimerElapsed returns the elapsed seconds of the timer at the given index.
func (tm *TimerManager) TimerElapsed(index int) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if index < 0 || index >= len(tm.timers) {
		return 0
	}
	return tm.timers[index].elapsed
}

// FormatElapsed returns "HH:MM:SS" for the given elapsed seconds.
func FormatElapsed(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// Snapshot returns a consistent snapshot of all timer states for UI rendering.
func (tm *TimerManager) Snapshot() (names []string, elapsed []int, active int, paused bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	names = make([]string, len(tm.timers))
	elapsed = make([]int, len(tm.timers))
	for i, t := range tm.timers {
		names[i] = t.name
		elapsed[i] = t.elapsed
	}
	return names, elapsed, tm.activeIndex, tm.paused
}
