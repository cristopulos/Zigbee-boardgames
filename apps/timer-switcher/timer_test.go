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

	names, elapsed, active := tm.Snapshot()
	if len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Fatalf("unexpected names: %v", names)
	}
	if elapsed[0] != 1 || elapsed[1] != 2 {
		t.Fatalf("unexpected elapsed: %v", elapsed)
	}
	if active != 1 {
		t.Fatalf("unexpected active: %d", active)
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
