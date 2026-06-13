//! Timer Manager — manages named timers with active selection, pause, and tick.

use once_cell::sync::Lazy;
use parking_lot::RwLock;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread::{self, JoinHandle};
use std::time::{Duration, Instant};

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
pub static TIMER_STATE: Lazy<Arc<RwLock<TimerState>>> =
    Lazy::new(|| Arc::new(RwLock::new(TimerState::default())));

/// Shutdown flag for the background timer ticker thread.
static TIMER_RUNNING: Lazy<Arc<AtomicBool>> = Lazy::new(|| Arc::new(AtomicBool::new(false)));

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

/// RAII handle for the background timer ticker thread.
///
/// On drop, the ticker thread is signaled to stop and joined. This guarantees
/// the process never hangs waiting for a sleeping ticker on early exit
/// (e.g., `eframe::run_native` returning an error or panicking).
///
/// The thread is decoupled from the egui event loop so the timer advances
/// even when the window is unfocused (egui/eframe pause `update()` on
/// unfocused windows on some platforms, which previously halted the timer).
pub struct TimerThread {
    running: Arc<AtomicBool>,
    handle: Option<JoinHandle<()>>,
}

impl TimerThread {
    /// Start the background ticker thread. Returns `None` if a ticker is
    /// already running — the existing thread is kept and a dummy handle is
    /// returned that is a no-op on drop.
    pub fn start() -> Self {
        let running = Arc::clone(&TIMER_RUNNING);
        // Try to claim the running flag using a compare-and-swap.
        // If another ticker is already running, the existing one is reused
        // and this handle's drop is a no-op (we hold no real JoinHandle).
        if running
            .compare_exchange(false, true, Ordering::AcqRel, Ordering::Acquire)
            .is_err()
        {
            return Self {
                running,
                handle: None,
            };
        }
        let thread_flag = Arc::clone(&running);
        let handle = thread::spawn(move || {
            while thread_flag.load(Ordering::Acquire) {
                tick();
                thread::sleep(Duration::from_secs(1));
            }
        });
        Self {
            running,
            handle: Some(handle),
        }
    }
}

impl Drop for TimerThread {
    fn drop(&mut self) {
        // Release semantics on the store ensure the thread observes the
        // false flag on its next Acquire load.
        self.running.store(false, Ordering::Release);
        if let Some(handle) = self.handle.take() {
            // Best-effort join: the thread sleeps for at most 1s between
            // iterations, so join() returns promptly.
            let _ = handle.join();
        }
    }
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

    // -----------------------------------------------------------------------
    // Tests for the background ticker thread (TimerThread RAII guard).
    // These exercise the real time-gated tick() path by sleeping on a real
    // wall-clock, and use a single shared mutex to keep the global
    // TIMER_STATE / TIMER_RUNNING stable across tests.
    // -----------------------------------------------------------------------

    /// Tests that touch the global TIMER_STATE or the ticker thread must
    /// hold this lock so the parallel test runner doesn't observe each
    /// other's mutations.
    static THREAD_TEST_LOCK: Lazy<parking_lot::Mutex<()>> =
        Lazy::new(|| parking_lot::Mutex::new(()));

    /// Reset the globals back to a known state for each test. Must be
    /// called while holding `THREAD_TEST_LOCK` so no other ticker is
    /// mutating TIMER_STATE.
    fn reset_globals(names: Vec<String>) {
        // Clear the running flag and wait for any in-flight ticker to exit
        // before we rewrite TIMER_STATE.
        TIMER_RUNNING.store(false, Ordering::Release);
        std::thread::sleep(Duration::from_millis(50));
        let mut state = TIMER_STATE.write();
        *state = TimerState::new(names);
    }

    /// The background thread should tick the active timer at least once
    /// after we sleep long enough for one 1-second interval to elapse.
    #[test]
    fn test_start_timer_thread_advances_active_timer() {
        let _guard = THREAD_TEST_LOCK.lock();
        reset_globals(vec!["Thread A".to_string(), "Thread B".to_string()]);

        let _thread = TimerThread::start();

        // Wait long enough for the loop to call tick() at least once.
        // The loop ticks immediately, then sleeps 1s, so 1.5s is enough
        // to observe either 1 or 2 ticks depending on scheduling.
        std::thread::sleep(Duration::from_millis(1500));

        let elapsed = TIMER_STATE.read().timers[0].elapsed;
        assert!(
            (1..=2).contains(&elapsed),
            "expected 1-2 ticks after 1.5s, got {}",
            elapsed
        );
        // _thread's drop will stop and join the ticker.
    }

    /// Dropping the TimerThread must cause the spawned thread to exit
    /// promptly, proving the shutdown flag is honored.
    #[test]
    fn test_stop_timer_thread_joins_cleanly() {
        let _guard = THREAD_TEST_LOCK.lock();
        reset_globals(vec!["A".to_string()]);

        {
            let _thread = TimerThread::start();
            // Let the thread start its loop.
            std::thread::sleep(Duration::from_millis(100));
        } // _thread drops here; the thread is joined before this scope ends.

        // If drop didn't join cleanly, TIMER_RUNNING would still be true
        // and a subsequent start would be a no-op.
        assert!(
            !TIMER_RUNNING.load(Ordering::Acquire),
            "TIMER_RUNNING should be false after TimerThread was dropped"
        );
    }

    /// The thread must not advance the timer while the manager is paused.
    /// We pause first, let the thread loop several times, then check that
    /// elapsed stayed at 0.
    #[test]
    fn test_timer_thread_respects_paused_state() {
        let _guard = THREAD_TEST_LOCK.lock();
        reset_globals(vec!["Paused Timer".to_string()]);

        // Pause before starting the thread so any tick() calls are no-ops.
        TIMER_STATE.write().toggle_pause();
        assert!(TIMER_STATE.read().paused);

        let _thread = TimerThread::start();

        // Sleep for slightly more than 2 tick intervals to give the loop
        // multiple chances to (incorrectly) tick.
        std::thread::sleep(Duration::from_millis(2300));

        let elapsed = TIMER_STATE.read().timers[0].elapsed;
        assert_eq!(
            elapsed, 0,
            "timer should not advance while paused, got elapsed={}",
            elapsed
        );
    }

    /// Starting and stopping the thread multiple times in succession must
    /// work — no panics, no leaked JoinHandles, the second run still ticks.
    #[test]
    fn test_timer_thread_can_start_and_stop_repeatedly() {
        let _guard = THREAD_TEST_LOCK.lock();
        reset_globals(vec!["Cycle".to_string()]);

        // First run.
        {
            let _thread = TimerThread::start();
            std::thread::sleep(Duration::from_millis(1500));
        }
        let after_first = TIMER_STATE.read().timers[0].elapsed;
        assert!(
            after_first >= 1,
            "first run should have ticked at least once, got {}",
            after_first
        );

        // Reset elapsed, then run again from a known state.
        TIMER_STATE.write().reset();
        assert_eq!(TIMER_STATE.read().timers[0].elapsed, 0);

        // Second run.
        {
            let _thread = TimerThread::start();
            std::thread::sleep(Duration::from_millis(1500));
        }
        let after_second = TIMER_STATE.read().timers[0].elapsed;
        assert!(
            after_second >= 1,
            "second run should have ticked at least once, got {}",
            after_second
        );
    }

    /// After TimerThread is dropped, no further ticks should land on
    /// TIMER_STATE. This protects against any path that might miss the
    /// running flag (e.g., a tick already in flight when stop is called).
    #[test]
    fn test_no_ticks_after_stop() {
        let _guard = THREAD_TEST_LOCK.lock();
        reset_globals(vec!["Quiet".to_string()]);

        {
            let _thread = TimerThread::start();
            std::thread::sleep(Duration::from_millis(1500));
        }

        let snapshot_before = TIMER_STATE.read().timers[0].elapsed;

        // Wait for a couple of "would-be" tick intervals. Since the thread
        // is dropped, the elapsed value must remain constant.
        std::thread::sleep(Duration::from_millis(2200));

        let snapshot_after = TIMER_STATE.read().timers[0].elapsed;
        assert_eq!(
            snapshot_before, snapshot_after,
            "timer advanced from {} to {} after thread was stopped",
            snapshot_before, snapshot_after
        );
    }
}
