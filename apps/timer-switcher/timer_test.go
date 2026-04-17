package main

import (
	"testing"
	"time"
)

func TestNewTimerManager(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B", "C"})
	if tm.Count() != 3 {
		t.Fatalf("expected 3 timers, got %d", tm.Count())
	}
	if tm.ActiveIndex() != 0 {
		t.Fatalf("expected active index 0, got %d", tm.ActiveIndex())
	}
	if tm.TimerName(0) != "A" {
		t.Fatalf("expected name A, got %s", tm.TimerName(0))
	}
}

func TestCycle(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B", "C"})
	tm.Cycle()
	if tm.ActiveIndex() != 1 {
		t.Fatalf("expected active index 1, got %d", tm.ActiveIndex())
	}
	tm.Cycle()
	if tm.ActiveIndex() != 2 {
		t.Fatalf("expected active index 2, got %d", tm.ActiveIndex())
	}
	tm.Cycle()
	if tm.ActiveIndex() != 0 {
		t.Fatalf("expected wrap to 0, got %d", tm.ActiveIndex())
	}
}

func TestSwitchTo(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B", "C"})
	tm.SwitchTo(2)
	if tm.ActiveIndex() != 2 {
		t.Fatalf("expected active index 2, got %d", tm.ActiveIndex())
	}
	// Out of bounds should be ignored
	tm.SwitchTo(99)
	if tm.ActiveIndex() != 2 {
		t.Fatalf("expected active index still 2, got %d", tm.ActiveIndex())
	}
}

func TestReset(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})
	tm.Tick()
	tm.Tick()
	tm.Tick()
	if tm.TimerElapsed(0) != 3 {
		t.Fatalf("expected 3 seconds, got %d", tm.TimerElapsed(0))
	}
	tm.Reset()
	if tm.TimerElapsed(0) != 0 {
		t.Fatalf("expected 0 after reset, got %d", tm.TimerElapsed(0))
	}
}

func TestTickOnlyActive(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})
	tm.Tick()
	tm.Tick()
	if tm.TimerElapsed(0) != 2 {
		t.Fatalf("expected active timer 2s, got %d", tm.TimerElapsed(0))
	}
	if tm.TimerElapsed(1) != 0 {
		t.Fatalf("expected inactive timer 0s, got %d", tm.TimerElapsed(1))
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "00:00:00"},
		{5, "00:00:05"},
		{60, "00:01:00"},
		{65, "00:01:05"},
		{3600, "01:00:00"},
		{3661, "01:01:01"},
		{86399, "23:59:59"},
	}
	for _, tt := range tests {
		got := FormatElapsed(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatElapsed(%d) = %s, want %s", tt.seconds, got, tt.want)
		}
	}
}

func TestSnapshot(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})
	tm.Tick()
	tm.Cycle()
	tm.Tick()
	tm.Tick()

	names, elapsed, active, paused := tm.Snapshot()
	if len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Fatalf("unexpected names: %v", names)
	}
	if elapsed[0] != 1 || elapsed[1] != 2 {
		t.Fatalf("unexpected elapsed: %v", elapsed)
	}
	if active != 1 {
		t.Fatalf("unexpected active: %d", active)
	}
	if paused {
		t.Fatalf("expected not paused")
	}
}

func TestTimerConcurrency(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})
	// Simulate rapid ticks and cycles from different goroutines
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			tm.Tick()
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 50; i++ {
			tm.Cycle()
			time.Sleep(time.Millisecond * 2)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	// If we get here without deadlock or panic, concurrency is safe
}

func TestParseTimerNames(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"A,B,C", []string{"A", "B", "C"}},
		{" A , B , C ", []string{"A", "B", "C"}},
		{"A", []string{"A"}},
		{"", []string{}},
		{",,", []string{}},
	}
	for _, tt := range tests {
		got := parseTimerNames(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseTimerNames(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseTimerNames(%q)[%d] = %s, want %s", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// TestParseButtonIDs tests the parseButtonIDs helper function
func TestParseButtonIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single button ID",
			input: "btn1",
			want:  []string{"btn1"},
		},
		{
			name:  "multiple comma-separated IDs",
			input: "btn1,btn2,btn3",
			want:  []string{"btn1", "btn2", "btn3"},
		},
		{
			name:  "IDs with whitespace around commas",
			input: " btn1 , btn2 , btn3 ",
			want:  []string{"btn1", "btn2", "btn3"},
		},
		{
			name:  "empty string returns empty slice",
			input: "",
			want:  []string{},
		},
		{
			name:  "trailing comma",
			input: "btn1,btn2,",
			want:  []string{"btn1", "btn2"},
		},
		{
			name:  "leading comma",
			input: ",btn1,btn2",
			want:  []string{"btn1", "btn2"},
		},
		{
			name:  "multiple consecutive commas",
			input: "btn1,,,btn2",
			want:  []string{"btn1", "btn2"},
		},
		{
			name:  "mixed whitespace and multiple commas",
			input: " btn1 ,  , btn2 ",
			want:  []string{"btn1", "btn2"},
		},
		{
			name:  "single ID with spaces",
			input: " btn1 ",
			want:  []string{"btn1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseButtonIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseButtonIDs(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseButtonIDs(%q)[%d] = %s, want %s", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestDirectMapVsCycleLogic tests that direct mapping and cycling behave correctly
func TestDirectMapVsCycleLogic(t *testing.T) {
	// Test 1: 3 buttons and 3 timers — direct mapping
	// Button index 0 should switch to timer 0, button index 2 to timer 2
	t.Run("direct map 3x3", func(t *testing.T) {
		tm := NewTimerManager([]string{"Timer 0", "Timer 1", "Timer 2"})
		// Simulate button 0 (index 0) direct mapping to timer 0
		tm.SwitchTo(0)
		if tm.ActiveIndex() != 0 {
			t.Fatalf("direct map: expected active index 0, got %d", tm.ActiveIndex())
		}
		// Simulate button 2 (index 2) direct mapping to timer 2
		tm.SwitchTo(2)
		if tm.ActiveIndex() != 2 {
			t.Fatalf("direct map: expected active index 2, got %d", tm.ActiveIndex())
		}
	})

	// Test 2: 2 buttons and 3 timers — both buttons should cycle
	t.Run("cycle 2 buttons 3 timers", func(t *testing.T) {
		tm := NewTimerManager([]string{"Timer 0", "Timer 1", "Timer 2"})
		// Both buttons cycle, so verify cycling behavior
		tm.Cycle()
		if tm.ActiveIndex() != 1 {
			t.Fatalf("cycle: expected active index 1, got %d", tm.ActiveIndex())
		}
		tm.Cycle()
		if tm.ActiveIndex() != 2 {
			t.Fatalf("cycle: expected active index 2, got %d", tm.ActiveIndex())
		}
		tm.Cycle()
		if tm.ActiveIndex() != 0 {
			t.Fatalf("cycle: expected wrap to 0, got %d", tm.ActiveIndex())
		}
	})

	// Test 3: 1 button and 3 timers — cycles (backward compatible)
	t.Run("cycle 1 button 3 timers", func(t *testing.T) {
		tm := NewTimerManager([]string{"Timer 0", "Timer 1", "Timer 2"})
		// Single button always cycles
		tm.Cycle()
		if tm.ActiveIndex() != 1 {
			t.Fatalf("single button cycle: expected active index 1, got %d", tm.ActiveIndex())
		}
		tm.Cycle()
		if tm.ActiveIndex() != 2 {
			t.Fatalf("single button cycle: expected active index 2, got %d", tm.ActiveIndex())
		}
		tm.Cycle()
		if tm.ActiveIndex() != 0 {
			t.Fatalf("single button cycle: expected wrap to 0, got %d", tm.ActiveIndex())
		}
	})
}

// TestDirectMapBoundaryCases tests edge cases for direct mapping
func TestDirectMapBoundaryCases(t *testing.T) {
	// Test switching to first timer in a larger set
	t.Run("switch to first in 5 timers", func(t *testing.T) {
		tm := NewTimerManager([]string{"A", "B", "C", "D", "E"})
		tm.SwitchTo(0)
		if tm.ActiveIndex() != 0 {
			t.Fatalf("expected active index 0, got %d", tm.ActiveIndex())
		}
	})

	// Test switching to last timer in a larger set
	t.Run("switch to last in 5 timers", func(t *testing.T) {
		tm := NewTimerManager([]string{"A", "B", "C", "D", "E"})
		tm.SwitchTo(4)
		if tm.ActiveIndex() != 4 {
			t.Fatalf("expected active index 4, got %d", tm.ActiveIndex())
		}
	})

	// Test cycling with same count (should use direct map in real code)
	t.Run("equal button timer count", func(t *testing.T) {
		tm := NewTimerManager([]string{"A", "B"})
		// When len(buttonIDs) == len(names), direct map is used
		// This means button 0 -> SwitchTo(0), button 1 -> SwitchTo(1)
		tm.SwitchTo(0)
		if tm.ActiveIndex() != 0 {
			t.Fatalf("expected active index 0, got %d", tm.ActiveIndex())
		}
		tm.SwitchTo(1)
		if tm.ActiveIndex() != 1 {
			t.Fatalf("expected active index 1, got %d", tm.ActiveIndex())
		}
	})
}

// TestOutOfBoundsSwitch tests that out-of-bounds indices are safely ignored
func TestOutOfBoundsSwitch(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B", "C"})

	// Out of bounds should be ignored
	prevIndex := tm.ActiveIndex()
	tm.SwitchTo(99)
	if tm.ActiveIndex() != prevIndex {
		t.Fatalf("out of bounds should be ignored, expected %d, got %d", prevIndex, tm.ActiveIndex())
	}

	tm.SwitchTo(-1)
	if tm.ActiveIndex() != prevIndex {
		t.Fatalf("negative index should be ignored, expected %d, got %d", prevIndex, tm.ActiveIndex())
	}

	// Valid switch after out of bounds should still work
	tm.SwitchTo(2)
	if tm.ActiveIndex() != 2 {
		t.Fatalf("expected active index 2 after valid switch, got %d", tm.ActiveIndex())
	}
}

// TestTogglePause verifies that TogglePause flips the paused state on/off/on
func TestTogglePause(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})

	// Initial state should be unpaused (false)
	if tm.IsPaused() {
		t.Fatalf("expected initial paused state to be false")
	}

	// First toggle: should become paused
	tm.TogglePause()
	if !tm.IsPaused() {
		t.Fatalf("expected paused state to be true after first toggle")
	}

	// Second toggle: should become unpaused
	tm.TogglePause()
	if tm.IsPaused() {
		t.Fatalf("expected paused state to be false after second toggle")
	}

	// Third toggle: should become paused again
	tm.TogglePause()
	if !tm.IsPaused() {
		t.Fatalf("expected paused state to be true after third toggle")
	}
}

// TestIsPaused verifies the initial state is false and true after toggle
func TestIsPaused(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})

	// Initial state should be false
	if tm.IsPaused() != false {
		t.Fatalf("expected initial paused state to be false, got true")
	}

	// After toggle, should be true
	tm.TogglePause()
	if tm.IsPaused() != true {
		t.Fatalf("expected paused state to be true after toggle, got false")
	}

	// Toggle again, back to false
	tm.TogglePause()
	if tm.IsPaused() != false {
		t.Fatalf("expected paused state to be false after second toggle, got true")
	}
}

// TestTickWhenPaused verifies Tick increments normally, stops when paused, resumes after unpause
func TestTickWhenPaused(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})

	// Tick normally while not paused
	tm.Tick()
	tm.Tick()
	tm.Tick()
	if tm.TimerElapsed(0) != 3 {
		t.Fatalf("expected 3 seconds after 3 ticks, got %d", tm.TimerElapsed(0))
	}

	// Pause the timer
	tm.TogglePause()
	if tm.TimerElapsed(0) != 3 {
		t.Fatalf("elapsed should not change on pause, expected 3, got %d", tm.TimerElapsed(0))
	}

	// Tick while paused - should not increment
	tm.Tick()
	tm.Tick()
	tm.Tick()
	if tm.TimerElapsed(0) != 3 {
		t.Fatalf("expected elapsed to remain 3 while paused, got %d", tm.TimerElapsed(0))
	}

	// Unpause the timer
	tm.TogglePause()

	// Tick while unpaused - should resume incrementing
	tm.Tick()
	tm.Tick()
	if tm.TimerElapsed(0) != 5 {
		t.Fatalf("expected 5 seconds after resuming, got %d", tm.TimerElapsed(0))
	}
}

// TestSnapshotPaused verifies that Snapshot returns the correct paused state
func TestSnapshotPaused(t *testing.T) {
	tm := NewTimerManager([]string{"A", "B"})
	tm.Tick()
	tm.Tick()

	// Snapshot when not paused should return false for paused
	names, elapsed, activeIdx, paused := tm.Snapshot()
	if paused {
		t.Fatalf("expected paused to be false in initial snapshot")
	}
	if len(names) != 2 || len(elapsed) != 2 {
		t.Fatalf("unexpected snapshot length: names=%d, elapsed=%d", len(names), len(elapsed))
	}
	if activeIdx != 0 {
		t.Fatalf("expected active index 0, got %d", activeIdx)
	}

	// Pause and verify snapshot returns true
	tm.TogglePause()
	_, _, _, paused = tm.Snapshot()
	if !paused {
		t.Fatalf("expected paused to be true after toggle")
	}

	// Unpause and verify snapshot returns false again
	tm.TogglePause()
	_, _, _, paused = tm.Snapshot()
	if paused {
		t.Fatalf("expected paused to be false after second toggle")
	}
}
