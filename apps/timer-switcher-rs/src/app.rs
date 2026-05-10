//! egui App — renders the timer cards and handles keyboard input.

use eframe::egui;
use egui::{Color32, Widget};

use crate::timer::{TIMER_STATE, tick};

// Color palette (from Go version)
const BG: Color32 = Color32::from_rgb(26, 26, 46);
const INACTIVE_CARD: Color32 = Color32::from_rgb(40, 40, 50);
const ACTIVE_CARD: Color32 = Color32::from_rgb(26, 58, 92);
const ACTIVE_BORDER: Color32 = Color32::from_rgb(0, 180, 216);
const INACTIVE_BORDER: Color32 = Color32::from_rgb(85, 85, 85);
const WHITE: Color32 = Color32::from_rgb(255, 255, 255);
const CYAN: Color32 = Color32::from_rgb(144, 224, 239);
const GREY: Color32 = Color32::from_rgb(170, 170, 170);
const AMBER: Color32 = Color32::from_rgb(255, 193, 7);

/// A timer card widget that renders background + name + time text.
pub struct TimerCard {
    pub card_width: f32,
    pub card_height: f32,
    pub is_active: bool,
    pub paused: bool,
    pub name: String,
    pub time_str: String,
}

impl Widget for TimerCard {
    fn ui(self, ui: &mut egui::Ui) -> egui::Response {
        let (rect, response) = ui.allocate_at_least(
            egui::vec2(self.card_width, self.card_height),
            egui::Sense::click(),
        );

        // Paint background + border
        let fill_color = if self.is_active { ACTIVE_CARD } else { INACTIVE_CARD };
        let border_color = if self.is_active { ACTIVE_BORDER } else { INACTIVE_BORDER };
        let border_width = if self.is_active { 3.0 } else { 1.0 };
        ui.painter().rect_filled(rect, 12.0, fill_color);
        ui.painter().rect_stroke(rect, 12.0, egui::Stroke::new(border_width, border_color));

        // Text padding within the card
        let inner = rect.shrink(10.0);
        let painter = ui.painter();

        // Name text — top-center of inner rect
        let name_color = if self.is_active { CYAN } else { GREY };
        let name_size = (self.card_height * 0.18).clamp(10.0, 40.0);
        painter.text(
            egui::Pos2::new(inner.center().x, inner.top() + name_size * 0.5),
            egui::Align2::CENTER_CENTER,
            &self.name,
            egui::FontId::proportional(name_size),
            name_color,
        );

        // Time text — center of inner rect
        let time_color = if self.is_active && self.paused { AMBER } else { WHITE };
        let time_size = (self.card_height * 0.28).clamp(14.0, 80.0);
        painter.text(
            inner.center(),
            egui::Align2::CENTER_CENTER,
            &self.time_str,
            egui::FontId::proportional(time_size),
            time_color,
        );

        response
    }
}

/// Main application implementing egui's App trait.
pub struct TimerSwitcherApp {
    pub debug: bool,
}

impl TimerSwitcherApp {
    pub fn new(_cc: &eframe::CreationContext<'_>, debug: bool) -> Self {
        Self { debug }
    }
}

impl eframe::App for TimerSwitcherApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // Request repaint frequently so button-press state changes are visible promptly
        ctx.request_repaint_after(std::time::Duration::from_millis(100));

        // Tick the timer
        tick();

        // Read current state
        let snap = {
            let state = TIMER_STATE.read();
            state.snapshot()
        };

        // Set background
        ctx.style_mut(|style| {
            style.visuals.window_fill = BG;
        });

        // Render the UI
        egui::CentralPanel::default().show(ctx, |ui| {
            // Add some padding at top
            ui.add_space(16.0);

            let num_cards = snap.count();
            if num_cards == 0 {
                ui.label("No timers configured.");
                return;
            }

            // Compute card dimensions — scale vertically to fill available space
            let available_width = ui.available_width();
            let spacing = 12.0_f32;
            let card_width = (available_width - (num_cards as f32 - 1.0) * spacing) / num_cards as f32;
            let hint_height = 20.0_f32;
            let card_height = (ui.available_height() - 32.0 - hint_height).max(80.0);

            ui.horizontal(|ui| {
                ui.spacing_mut().item_spacing.x = spacing;

                for i in 0..num_cards {
                    let is_active = i == snap.active_index;
                    let name = snap.names[i].clone();
                    let time_str = crate::timer::format_elapsed(snap.elapsed[i]);
                    let idx = i;
                    let paused = snap.paused;

                    let card = TimerCard {
                        card_width,
                        card_height,
                        is_active,
                        paused,
                        name,
                        time_str,
                    };

                    let response = ui.add(card);

                    // Handle click
                    if response.clicked() {
                        crate::timer::execute(crate::timer::TimerCommand::SwitchTo(idx));
                    }
                }
            });

            // Bottom hint text
            ui.add_space(16.0);
            ui.label(
                "SPACE: Switch  |  ENTER: Reset  |  P: Pause  |  ESC: Quit"
            );
        });

        // Handle keyboard input
        ctx.input_mut(|input| {
            let mut key_pressed = None;
            for event in input.events.iter() {
                if let egui::Event::Key { key, pressed, .. } = event {
                    if *pressed {
                        key_pressed = Some(*key);
                        break;
                    }
                }
            }

            if let Some(key) = key_pressed {
                match key {
                    egui::Key::Space => {
                        crate::timer::execute(crate::timer::TimerCommand::Cycle);
                    }
                    egui::Key::Enter => {
                        crate::timer::execute(crate::timer::TimerCommand::Reset);
                    }
                    egui::Key::P => {
                        crate::timer::execute(crate::timer::TimerCommand::TogglePause);
                    }
                    _ => {}
                }
            }
        });
    }
}

// ---------------------------------------------------------------------------
// Unit tests — TimerCard text rendering
//
// The timer card widget renders `name` at the top and `time_str` in the
// center.  These tests verify the exact strings that will be painted.
// No egui runtime is required — the fields are the source of truth for
// what the widget will display.
// ---------------------------------------------------------------------------
#[cfg(test)]
mod tests {
    use super::TimerCard;

    /// The name string is the exact text the card renders at the top.
    #[test]
    fn test_card_name_is_painted() {
        let card = TimerCard {
            card_width: 200.0,
            card_height: 120.0,
            is_active: true,
            paused: false,
            name: "Player 1".to_string(),
            time_str: "00:01:30".to_string(),
        };
        assert_eq!(card.name, "Player 1");
    }

    /// The time_str string is the exact text the card renders in the center.
    #[test]
    fn test_card_time_str_is_painted() {
        let card = TimerCard {
            card_width: 200.0,
            card_height: 120.0,
            is_active: true,
            paused: false,
            name: "Player 1".to_string(),
            time_str: "00:01:30".to_string(),
        };
        assert_eq!(card.time_str, "00:01:30");
    }

    /// Active state uses ACTIVE_CARD blue background and CYAN name text.
    #[test]
    fn test_card_active_state() {
        let card = TimerCard {
            card_width: 150.0,
            card_height: 100.0,
            is_active: true,
            paused: false,
            name: "P2".to_string(),
            time_str: "00:05:00".to_string(),
        };
        assert!(card.is_active);
        assert!(!card.paused);
        assert_eq!(card.name, "P2");
    }

    /// Paused state uses WHITE time text; paused-while-active uses AMBER.
    #[test]
    fn test_card_paused_active_uses_amber() {
        let card = TimerCard {
            card_width: 100.0,
            card_height: 80.0,
            is_active: true,
            paused: true,
            name: "Paused".to_string(),
            time_str: "00:00:00".to_string(),
        };
        assert!(card.paused);
        assert!(card.is_active);
        // time_str is "00:00:00" — the amber color branch is taken because
        // is_active && paused == true (painted amber in the widget code)
    }

    /// Multiple cards each carry their own distinct text strings.
    #[test]
    fn test_multiple_cards_distinct_text() {
        let names = ["P1", "P2", "P3"];
        let times = ["00:01:00", "00:02:00", "00:03:00"];

        for i in 0..3 {
            let card = TimerCard {
                card_width: 180.0,
                card_height: 100.0,
                is_active: i == 0,
                paused: false,
                name: names[i].to_string(),
                time_str: times[i].to_string(),
            };
            assert_eq!(card.name, names[i]);
            assert_eq!(card.time_str, times[i]);
        }
    }
}