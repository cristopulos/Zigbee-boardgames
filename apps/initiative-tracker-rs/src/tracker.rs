//! Tracker State — initiative tracking with enabled/disabled cards.
//!
//! The tracker maintains 9 entries (indices 0-8) always. When --naalu is not
//! used, entry 0 (Naalu) is disabled, leaving 8 visible cards.

use once_cell::sync::Lazy;
use parking_lot::RwLock;
use std::sync::Arc;

use eframe::egui;

// ---------------------------------------------------------------------------
// Card data (verified from TI4 artwork)
// ---------------------------------------------------------------------------

/// Total number of strategy cards (including Naalu at index 0).
pub const NUM_INITIATIVES: usize = 9;

/// Strategy card data: (color_rgb, name).
/// 0=Naalu, 1=Leadership, 2=Diplomacy, 3=Politics, 4=Construction,
/// 5=Trade, 6=Warfare, 7=Technology, 8=Imperial
pub const INITIATIVE_DATA: [(egui::Color32, &'static str); NUM_INITIATIVES] = [
    (egui::Color32::from_rgb(0x00, 0xB4, 0xD8), "Naalu"), // 0 - Teal
    (egui::Color32::from_rgb(0xDF, 0x23, 0x22), "Leadership"), // 1 - Red
    (egui::Color32::from_rgb(0xED, 0x92, 0x37), "Diplomacy"), // 2 - Orange
    (egui::Color32::from_rgb(0xFA, 0xF0, 0x1D), "Politics"), // 3 - Yellow
    (egui::Color32::from_rgb(0x30, 0xAF, 0x60), "Construction"), // 4 - Green
    (egui::Color32::from_rgb(0x03, 0xA6, 0x91), "Trade"), // 5 - Teal
    (egui::Color32::from_rgb(0x1B, 0x8B, 0xCD), "Warfare"), // 6 - Light Blue
    (egui::Color32::from_rgb(0x1B, 0x45, 0x97), "Technology"), // 7 - Dark Blue
    (egui::Color32::from_rgb(0x89, 0x4A, 0xA5), "Imperial"), // 8 - Purple
];

// ---------------------------------------------------------------------------
// Global state
// ---------------------------------------------------------------------------

/// Global tracker state — shared between tokio tasks and egui app.
pub static TRACKER_STATE: Lazy<Arc<RwLock<TrackerState>>> =
    Lazy::new(|| Arc::new(RwLock::new(TrackerState::default())));

/// Commands from the async event handler to the UI.
#[derive(Debug, Clone)]
pub enum TrackerCommand {
    Next,
    Prev,
    Reset,
    ToggleEnabled(usize),
}

/// All initiative tracker state.
#[derive(Debug, Default, Clone)]
pub struct TrackerState {
    /// Active initiative index (0-8). Never equals a disabled index.
    current: usize,
    /// Whether each initiative is enabled (true) or disabled (false).
    enabled: [bool; NUM_INITIATIVES],
    /// Reset target — the initiative to return to on reset.
    start: usize,
}

impl TrackerState {
    /// Create a new tracker state with the given reset target.
    /// All 9 entries start enabled.
    pub fn new(start: usize) -> Self {
        let enabled = [true; NUM_INITIATIVES];
        // Clamp start to valid range
        let start = if start < NUM_INITIATIVES { start } else { 0 };
        Self {
            current: start,
            enabled,
            start,
        }
    }

    /// Returns the current active initiative index.
    pub fn current(&self) -> usize {
        self.current
    }

    /// Returns whether the initiative at the given index is enabled.
    pub fn enabled(&self, index: usize) -> bool {
        self.enabled.get(index).copied().unwrap_or(false)
    }

    /// Advance to the next enabled initiative (wraps around).
    pub fn next(&mut self) {
        let n = NUM_INITIATIVES;
        let start = self.current;
        loop {
            self.current = (self.current + 1) % n;
            if self.enabled[self.current] {
                return;
            }
            if self.current == start {
                return; // No enabled initiatives
            }
        }
    }

    /// Go to the previous enabled initiative (wraps around).
    pub fn prev(&mut self) {
        let n = NUM_INITIATIVES;
        let start = self.current;
        loop {
            self.current = (self.current + n - 1) % n;
            if self.enabled[self.current] {
                return;
            }
            if self.current == start {
                return; // No enabled initiatives
            }
        }
    }

    /// Reset to the configured start initiative (if enabled).
    pub fn reset(&mut self) {
        if self.start < NUM_INITIATIVES && self.enabled[self.start] {
            self.current = self.start;
        } else {
            // Find first enabled initiative
            for i in 0..NUM_INITIATIVES {
                if self.enabled[i] {
                    self.current = i;
                    return;
                }
            }
        }
    }

    /// Toggle the enabled state of the initiative at the given index.
    /// If the current initiative is disabled, advance to the next enabled.
    pub fn toggle_enabled(&mut self, index: usize) {
        if index >= NUM_INITIATIVES {
            return;
        }
        self.enabled[index] = !self.enabled[index];
        if !self.enabled[index] && self.current == index {
            // Advance to next enabled
            for j in 0..NUM_INITIATIVES {
                let idx = (index + 1 + j) % NUM_INITIATIVES;
                if self.enabled[idx] {
                    self.current = idx;
                    return;
                }
            }
        }
    }

    /// Returns a vector of all enabled states (for UI rendering).
    pub fn all_enabled(&self) -> Vec<bool> {
        self.enabled.iter().copied().collect()
    }

    /// Returns the number of enabled initiatives.
    pub fn num_enabled(&self) -> usize {
        self.enabled.iter().filter(|&&e| e).count()
    }
}

/// Clone the current snapshot for UI rendering.
pub fn snapshot() -> TrackerSnapshot {
    let state = TRACKER_STATE.read();
    TrackerSnapshot {
        current: state.current(),
        all_enabled: state.all_enabled(),
    }
}

/// A consistent snapshot of tracker state for UI rendering.
#[derive(Debug, Clone)]
pub struct TrackerSnapshot {
    pub current: usize,
    pub all_enabled: Vec<bool>,
}

/// Execute a tracker command on the global state.
pub fn execute(cmd: TrackerCommand) {
    let mut state = TRACKER_STATE.write();
    match cmd {
        TrackerCommand::Next => state.next(),
        TrackerCommand::Prev => state.prev(),
        TrackerCommand::Reset => state.reset(),
        TrackerCommand::ToggleEnabled(idx) => state.toggle_enabled(idx),
    }
}

/// Returns the color for a card based on its state.
pub fn card_colors(
    index: usize,
    is_active: bool,
    is_enabled: bool,
) -> (egui::Color32, egui::Color32, egui::Color32) {
    if !is_enabled {
        return (
            egui::Color32::from_rgb(0x33, 0x33, 0x33),
            egui::Color32::from_rgb(0x55, 0x55, 0x55),
            egui::Color32::from_rgb(0x44, 0x44, 0x44),
        );
    }

    let card_color = INITIATIVE_DATA[index].0;

    if is_active {
        (card_color, egui::Color32::WHITE, egui::Color32::WHITE)
    } else {
        let dim = dim_color(card_color, 0x4D);
        (
            dim,
            egui::Color32::from_rgb(0x99, 0x99, 0x99),
            egui::Color32::from_rgb(0x88, 0x88, 0x88),
        )
    }
}

/// Dim a color by setting its alpha to the given value (0-255).
fn dim_color(c: egui::Color32, alpha: u8) -> egui::Color32 {
    // Use RGBA directly to avoid internal premultiplication in from_rgba_unmultiplied.
    // The Color32 stored format is already premultiplied internally.
    egui::Color32::from_rgba_unmultiplied(c.r(), c.g(), c.b(), alpha)
}

/// Clamp a font size between 10 and 60.
pub(crate) fn clamp(val: f32) -> f32 {
    val.max(10.0).min(60.0)
}

/// Returns the card number as a string (e.g., "0", "1", "12").
pub fn card_number(index: usize) -> String {
    if index >= NUM_INITIATIVES {
        return String::new();
    }
    if index == 0 {
        return "0".to_string();
    }
    let mut digits = String::new();
    let mut n = index;
    while n > 0 {
        digits.insert(0, (b'0' + (n % 10) as u8) as char);
        n /= 10;
    }
    digits
}

/// Returns the card name for the given index.
pub fn card_name(index: usize) -> &'static str {
    INITIATIVE_DATA
        .get(index)
        .map(|(_, name)| *name)
        .unwrap_or("")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    // --- TrackerState ---

    #[test]
    fn test_new_tracker_state() {
        let state = TrackerState::new(1);
        assert_eq!(state.current(), 1);
        for i in 0..NUM_INITIATIVES {
            assert!(state.enabled(i), "card {} should be enabled initially", i);
        }
    }

    #[test]
    fn test_new_tracker_state_invalid_start() {
        let state = TrackerState::new(99);
        assert_eq!(state.current(), 0, "invalid start should clamp to 0");
        let state = TrackerState::new(usize::MAX);
        assert_eq!(state.current(), 0, "invalid start should clamp to 0");
    }

    #[test]
    fn test_next() {
        let mut state = TrackerState::new(0);
        state.next();
        assert_eq!(state.current(), 1);
        state.next();
        assert_eq!(state.current(), 2);
    }

    #[test]
    fn test_next_wraps() {
        let mut state = TrackerState::new(7);
        state.next();
        assert_eq!(state.current(), 8);
        state.next();
        assert_eq!(state.current(), 0);
    }

    #[test]
    fn test_next_skips_disabled() {
        let mut state = TrackerState::new(0);
        state.enabled[1] = false;
        state.enabled[2] = false;
        state.next();
        assert_eq!(state.current(), 3);
    }

    #[test]
    fn test_next_all_disabled() {
        let mut state = TrackerState::new(0);
        for i in 0..NUM_INITIATIVES {
            state.enabled[i] = false;
        }
        state.next();
        assert_eq!(
            state.current(),
            0,
            "current should not change when all disabled"
        );
    }

    #[test]
    fn test_prev() {
        let mut state = TrackerState::new(3);
        state.prev();
        assert_eq!(state.current(), 2);
        state.prev();
        assert_eq!(state.current(), 1);
    }

    #[test]
    fn test_prev_wraps() {
        let mut state = TrackerState::new(0);
        state.prev();
        assert_eq!(state.current(), 8);
        state.prev();
        assert_eq!(state.current(), 7);
    }

    #[test]
    fn test_prev_skips_disabled() {
        let mut state = TrackerState::new(4);
        state.enabled[3] = false;
        state.enabled[2] = false;
        state.prev();
        assert_eq!(state.current(), 1);
    }

    #[test]
    fn test_prev_all_disabled() {
        let mut state = TrackerState::new(3);
        for i in 0..NUM_INITIATIVES {
            state.enabled[i] = false;
        }
        state.prev();
        assert_eq!(
            state.current(),
            3,
            "current should not change when all disabled"
        );
    }

    #[test]
    fn test_reset() {
        let mut state = TrackerState::new(1);
        state.next();
        state.next();
        assert_eq!(state.current(), 3);
        state.reset();
        assert_eq!(state.current(), 1);
    }

    #[test]
    fn test_reset_invalid_start_uses_first_enabled() {
        let mut state = TrackerState::new(5);
        state.enabled[5] = false;
        state.enabled[6] = false;
        state.reset();
        assert_eq!(state.current(), 0, "should reset to first enabled card");
    }

    #[test]
    fn test_toggle_enabled() {
        let mut state = TrackerState::new(0);
        assert!(state.enabled(3));
        state.toggle_enabled(3);
        assert!(!state.enabled(3), "card 3 should be disabled after toggle");
        state.toggle_enabled(3);
        assert!(
            state.enabled(3),
            "card 3 should be re-enabled after second toggle"
        );
    }

    #[test]
    fn test_toggle_enabled_current_advances() {
        let mut state = TrackerState::new(2);
        assert_eq!(state.current(), 2);
        state.toggle_enabled(2);
        assert_eq!(
            state.current(),
            3,
            "current should advance to 3 after disabling 2"
        );
    }

    #[test]
    fn test_toggle_enabled_out_of_bounds() {
        let mut state = TrackerState::new(0);
        state.toggle_enabled(99);
        state.toggle_enabled(usize::MAX);
        assert_eq!(
            state.current(),
            0,
            "out of bounds toggle should not change current"
        );
    }

    #[test]
    fn test_toggle_enabled_last_card_wraps() {
        let mut state = TrackerState::new(8);
        state.toggle_enabled(8); // Disable 8 (Imperial)
                                 // Search starts at (8+1) % 9 = 0, so Naalu (0) is found first
        assert_eq!(
            state.current(),
            0,
            "current should advance to next enabled (Naalu)"
        );
    }

    #[test]
    fn test_all_enabled() {
        let mut state = TrackerState::new(0);
        state.enabled[4] = false;
        let all = state.all_enabled();
        assert!(!all[4], "all_enabled[4] should be false");
        assert!(all[5], "all_enabled[5] should be true");
        assert_eq!(all.len(), NUM_INITIATIVES);
    }

    // --- Card data ---

    #[test]
    fn test_card_name() {
        let names = [
            "Naalu",
            "Leadership",
            "Diplomacy",
            "Politics",
            "Construction",
            "Trade",
            "Warfare",
            "Technology",
            "Imperial",
        ];
        for (i, expected) in names.iter().enumerate() {
            assert_eq!(
                card_name(i),
                *expected,
                "card_name({}) should be {}",
                i,
                expected
            );
        }
        assert_eq!(
            card_name(99),
            "",
            "out of bounds card_name should return empty"
        );
    }

    #[test]
    fn test_card_number() {
        assert_eq!(card_number(0), "0");
        for i in 1..=8 {
            let num = card_number(i);
            assert!(!num.is_empty(), "card_number({}) should not be empty", i);
        }
    }

    // --- Utilities ---

    #[test]
    fn test_clamp() {
        assert_eq!(clamp(5.0), 10.0);
        assert_eq!(clamp(10.0), 10.0);
        assert_eq!(clamp(30.0), 30.0);
        assert_eq!(clamp(60.0), 60.0);
        assert_eq!(clamp(65.0), 60.0);
        assert_eq!(clamp(0.0), 10.0);
    }

    #[test]
    fn test_dim_color() {
        let c = egui::Color32::from_rgb(0xFF, 0x00, 0x00);
        let dim = dim_color(c, 0x80);
        // Only alpha is semantically meaningful here — from_rgba_unmultiplied
        // internally multiplies RGB by alpha/255, so only alpha is preserved.
        assert_eq!(dim.a(), 0x80, "alpha should be set to the dim value");
    }

    #[test]
    fn test_card_colors_active_enabled() {
        let (bg, num, name) = card_colors(1, true, true);
        // Active + enabled → full color + white text
        assert_eq!(bg, INITIATIVE_DATA[1].0);
        assert_eq!(num, egui::Color32::WHITE);
        assert_eq!(name, egui::Color32::WHITE);
    }

    #[test]
    fn test_card_colors_inactive_enabled() {
        let (bg, _, _) = card_colors(2, false, true);
        // Inactive + enabled → dimmed color
        assert_eq!(bg.a(), 0x4D, "inactive enabled card should have alpha 0x4D");
    }

    #[test]
    fn test_card_colors_disabled() {
        let (bg, num, name) = card_colors(3, false, false);
        assert_eq!(bg, egui::Color32::from_rgb(0x33, 0x33, 0x33));
        assert_eq!(num, egui::Color32::from_rgb(0x55, 0x55, 0x55));
        assert_eq!(name, egui::Color32::from_rgb(0x44, 0x44, 0x44));
    }

    // --- Snapshot ---

    #[test]
    fn test_snapshot() {
        let snap = snapshot();
        assert!(snap.current < NUM_INITIATIVES);
        assert_eq!(snap.all_enabled.len(), NUM_INITIATIVES);
    }
}
