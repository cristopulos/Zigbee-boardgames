//! Initiative Tracker — egui App rendering strategy card grid.

use crate::tracker::{
    self, card_colors, card_number, clamp, TrackerSnapshot, INITIATIVE_DATA, TRACKER_STATE,
};
use eframe::egui;

const BG: egui::Color32 = egui::Color32::from_rgb(26, 26, 46);
const GREY: egui::Color32 = egui::Color32::from_rgb(136, 136, 136);
const WHITE: egui::Color32 = egui::Color32::WHITE;

/// Main application implementing egui's App trait.
pub struct InitiativeTrackerApp {
    /// Number of cards to show (8 or 9).
    num_cards: usize,
    /// Offset: 0 means show Naalu (0-8), 1 means exclude Naalu (1-8).
    offset: usize,
    /// Whether quit has been requested.
    quit_requested: bool,
}

impl InitiativeTrackerApp {
    pub fn new(_cc: &eframe::CreationContext<'_>, num_cards: usize) -> Self {
        let offset = if num_cards == 8 { 1 } else { 0 };
        Self {
            num_cards,
            offset,
            quit_requested: false,
        }
    }

    /// Returns which card indices to show (0..num_cards mapped to real indices).
    fn show_indices(&self) -> Vec<usize> {
        (0..self.num_cards).map(|i| i + self.offset).collect()
    }
}

impl eframe::App for InitiativeTrackerApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        if self.quit_requested {
            return;
        }

        // Read current state
        let snap: TrackerSnapshot = {
            let state = TRACKER_STATE.read();
            TrackerSnapshot {
                current: state.current(),
                all_enabled: state.all_enabled(),
            }
        };

        // Set background
        ctx.style_mut(|style| {
            style.visuals.window_fill = BG;
        });

        egui::CentralPanel::default().show(ctx, |ui| {
            // Add padding at top
            ui.add_space(16.0);

            // Compute card dimensions — scale vertically to fill available space
            let indices = self.show_indices();
            let spacing = 8.0_f32;
            let card_width = (ui.available_width() - (self.num_cards as f32 - 1.0) * spacing)
                / self.num_cards as f32;
            let hint_height = 20.0_f32;
            let card_height = (ui.available_height() - 28.0 - hint_height).max(80.0);

            ui.horizontal(|ui| {
                ui.spacing_mut().item_spacing.x = spacing;

                for &idx in &indices {
                    let is_active = idx == snap.current;
                    let is_enabled = snap.all_enabled.get(idx).copied().unwrap_or(false);
                    let (bg_color, num_color, name_color) = card_colors(idx, is_active, is_enabled);

                    // Card background
                    let (rect, response) = ui.allocate_at_least(
                        egui::vec2(card_width, card_height),
                        egui::Sense::click(),
                    );
                    ui.painter().rect_filled(rect, 12.0, bg_color);

                    // Border: 3px white for active, 1px grey for inactive
                    let border_color = if is_active { WHITE } else { GREY };
                    let border_width = if is_active { 3.0 } else { 1.0 };
                    ui.painter().rect_stroke(
                        rect,
                        12.0,
                        egui::Stroke::new(border_width, border_color),
                    );

                    // Draw text inside card
                    let inner = rect.shrink(8.0);
                    let painter = ui.painter();

                    // Scale text relative to the smaller card dimension
                    let min_dim = card_width.min(card_height);

                    // Card number (large, centered)
                    let num_size = clamp(min_dim * 0.45);
                    let num_pos =
                        egui::Pos2::new(inner.center().x, inner.center().y + num_size * 0.3);
                    painter.text(
                        num_pos,
                        egui::Align2::CENTER_CENTER,
                        card_number(idx),
                        egui::FontId::proportional(num_size),
                        num_color,
                    );

                    // Card name (small, below center)
                    let name_size = clamp(min_dim * 0.18);
                    let name_y = inner.center().y - num_size * 0.4;
                    let name_pos = egui::Pos2::new(inner.center().x, name_y - name_size * 0.5);
                    painter.text(
                        name_pos,
                        egui::Align2::CENTER_CENTER,
                        INITIATIVE_DATA[idx].1,
                        egui::FontId::proportional(name_size),
                        name_color,
                    );

                    // Handle click → toggle enabled
                    if response.clicked() {
                        tracker::execute(tracker::TrackerCommand::ToggleEnabled(idx));
                    }
                }
            });

            // Hint text at bottom
            ui.add_space(12.0);
            ui.label(
                egui::RichText::new(
                    "SPACE/→/↑: Next  |  ←/↓/⌫: Prev  |  R: Reset  |  0-8: Toggle  |  ESC: Quit",
                )
                .small()
                .color(GREY),
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
                    egui::Key::Space | egui::Key::ArrowRight | egui::Key::ArrowUp => {
                        tracker::execute(tracker::TrackerCommand::Next);
                    }
                    egui::Key::ArrowLeft
                    | egui::Key::ArrowDown
                    | egui::Key::Backspace
                    | egui::Key::Delete => {
                        tracker::execute(tracker::TrackerCommand::Prev);
                    }
                    egui::Key::R => {
                        tracker::execute(tracker::TrackerCommand::Reset);
                    }
                    egui::Key::Escape => {
                        self.quit_requested = true;
                    }
                    // Number keys 0-8
                    egui::Key::Num0 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(0)),
                    egui::Key::Num1 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(1)),
                    egui::Key::Num2 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(2)),
                    egui::Key::Num3 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(3)),
                    egui::Key::Num4 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(4)),
                    egui::Key::Num5 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(5)),
                    egui::Key::Num6 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(6)),
                    egui::Key::Num7 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(7)),
                    egui::Key::Num8 => tracker::execute(tracker::TrackerCommand::ToggleEnabled(8)),
                    _ => {}
                }
            }
        });

        // Request repaint every 100ms so button-press state changes are visible
        ctx.request_repaint_after(std::time::Duration::from_millis(100));
    }
}
