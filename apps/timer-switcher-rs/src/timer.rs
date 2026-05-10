//! Timer Manager — manages named timers with active selection, pause, and tick.

use std::time::{Instant, Duration};
use std::sync::Arc;
use parking_lot::RwLock;
use once_cell::sync::Lazy;

/// Format elapsed seconds as "HH:MM:SS".
pub fn format_elapsed(seconds: u64) -> String {
    let h = seconds / 3600;
    let m = (seconds % 3600) / 60;
    let s = seconds % 60;
    format!("{:02}:{:02}:{:02}", h, m, s)
}

/// A single named timer with elapsed duration in seconds.
#[derive(Debug, Clone)]
pub struct Timer {
    pub name: String,
    pub elapsed: u64,
}

/// Global timer state — shared between tokio tasks and the egui app.
pub static TIMER_STATE: Lazy<Arc<RwLock<TimerState>>> = Lazy::new(|| {
    Arc::new(RwLock::new(TimerState::default()))
});

/// All timer state, including which one is active and paused status.
#[derive(Debug, Default)]
pub struct TimerState {
    pub timers: Vec<Timer>,
    pub active_index: usize,
    pub paused: bool,
    last_tick: Option<Instant>,
}

/// Commands from the async event handler to the UI.
#[derive(Debug, Clone)]
pub enum TimerCommand {
    Cycle,
    SwitchTo(usize),
    TogglePause,
    Reset,
}

impl TimerState {
    /// Create a new manager with the given timer names.
    pub fn new(names: Vec<String>) -> Self {
        let timers = names
            .into_iter()
            .map(|name| Timer { name, elapsed: 0 })
            .collect();
        Self {
            timers,
            active_index: 0,
            paused: false,
            last_tick: None,
        }
    }

    /// Move the active timer to the next one (wraps around).
    pub fn cycle(&mut self) {
        if !self.timers.is_empty() {
            self.active_index = (self.active_index + 1) % self.timers.len();
        }
    }

    /// Make the timer at the given index active.
    pub fn switch_to(&mut self, index: usize) {
        if index < self.timers.len() {
            self.active_index = index;
        }
    }

    /// Set the active timer's elapsed time to zero.
    pub fn reset(&mut self) {
        if !self.timers.is_empty() {
            self.timers[self.active_index].elapsed = 0;
        }
    }

    /// Increment the active timer by one real second (gated to 1 Hz).
    pub fn tick(&mut self) {
        if self.paused || self.timers.is_empty() {
            return;
        }
        let now = Instant::now();
        let should_tick = match self.last_tick {
            None => true,
            Some(last) => now.duration_since(last) >= Duration::from_secs(1),
        };
        if should_tick {
            self.timers[self.active_index].elapsed += 1;
            self.last_tick = Some(now);
        }
    }

    /// Increment the active timer by one second (test-only, no time gating).
    #[cfg(test)]
    pub fn tick_test(&mut self) {
        if !self.paused && !self.timers.is_empty() {
            self.timers[self.active_index].elapsed += 1;
        }
    }

    /// Flip the paused state.
    pub fn toggle_pause(&mut self) {
        self.paused = !self.paused;
    }

    /// Number of timers.
    pub fn count(&self) -> usize {
        self.timers.len()
    }

    /// Clone the current snapshot for UI rendering.
    pub fn snapshot(&self) -> TimerSnapshot {
        TimerSnapshot {
            names: self.timers.iter().map(|t| t.name.clone()).collect(),
            elapsed: self.timers.iter().map(|t| t.elapsed).collect(),
            active_index: self.active_index,
            paused: self.paused,
        }
    }
}

/// A consistent snapshot of all timer states for UI rendering.
#[derive(Debug, Clone)]
pub struct TimerSnapshot {
    pub names: Vec<String>,
    pub elapsed: Vec<u64>,
    pub active_index: usize,
    pub paused: bool,
}

impl TimerSnapshot {
    /// Number of timers in this snapshot.
    pub fn count(&self) -> usize {
        self.names.len()
    }
}

/// Execute a timer command on the global state.
pub fn execute(cmd: TimerCommand) {
    let mut state = TIMER_STATE.write();
    match cmd {
        TimerCommand::Cycle => state.cycle(),
        TimerCommand::SwitchTo(idx) => state.switch_to(idx),
        TimerCommand::TogglePause => state.toggle_pause(),
        TimerCommand::Reset => state.reset(),
    }
}

/// Tick the active timer (called from the UI timer).
pub fn tick() {
    TIMER_STATE.write().tick();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_elapsed() {
        assert_eq!(format_elapsed(0), "00:00:00");
        assert_eq!(format_elapsed(5), "00:00:05");
        assert_eq!(format_elapsed(65), "00:01:05");
        assert_eq!(format_elapsed(3661), "01:01:01");
    }

    #[test]
    fn test_new_manager() {
        let state = TimerState::new(vec!["A".to_string(), "B".to_string()]);
        assert_eq!(state.count(), 2);
        assert_eq!(state.active_index, 0);
        assert!(!state.paused);
    }

    #[test]
    fn test_cycle() {
        let mut state = TimerState::new(vec!["A".to_string(), "B".to_string(), "C".to_string()]);
        assert_eq!(state.active_index, 0);
        state.cycle();
        assert_eq!(state.active_index, 1);
        state.cycle();
        assert_eq!(state.active_index, 2);
        state.cycle();
        assert_eq!(state.active_index, 0);
    }

    #[test]
    fn test_tick_only_active() {
        let mut state = TimerState::new(vec!["A".to_string(), "B".to_string()]);
        state.tick_test();
        state.tick_test();
        let snap = state.snapshot();
        assert_eq!(snap.elapsed[0], 2);
        assert_eq!(snap.elapsed[1], 0);
    }

    #[test]
    fn test_tick_skipped_when_paused() {
        let mut state = TimerState::new(vec!["A".to_string()]);
        state.tick_test();
        state.tick_test();
        state.toggle_pause();
        state.tick_test();
        state.tick_test();
        state.toggle_pause();
        state.tick_test();
        let snap = state.snapshot();
        assert_eq!(snap.elapsed[0], 3);
    }

    #[test]
    fn test_snapshot() {
        let mut state = TimerState::new(vec!["A".to_string(), "B".to_string()]);
        state.tick_test();
        state.cycle();
        state.tick_test();

        let snap = state.snapshot();
        assert_eq!(snap.names, vec!["A", "B"]);
        assert_eq!(snap.elapsed, vec![1, 1]);
        assert_eq!(snap.active_index, 1);
        assert!(!snap.paused);
    }
}