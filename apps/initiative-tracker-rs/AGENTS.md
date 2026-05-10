# AGENTS.md — initiative-tracker-rs

Compact instructions for AI agents working in this app.

## Overview

A Rust/egui app that cycles through Twilight Imperium 4 strategy cards. Supports keyboard controls and Zigbee button input via the `button-client` crate.

## Architecture

```
tracker.rs (logic, global state) → main.rs (async wiring) → app.rs (egui UI)
```

- **`tracker.rs`**: `TrackerState`, `TrackerCommand`, `TRACKER_STATE` global, card data
- **`lib.rs`**: Module re-exports for library use
- **`main.rs`**: CLI parsing, tokio async button listeners, egui app runner
- **`app.rs`**: `InitiativeTrackerApp`, keyboard/mouse handling

## Card Data

9 strategy cards defined in `INITIATIVE_DATA`:

| Index | Color | Name |
|-------|-------|------|
| 0 | Teal | Naalu |
| 1 | Red | Leadership |
| 2 | Orange | Diplomacy |
| 3 | Yellow | Politics |
| 4 | Green | Construction |
| 5 | Teal | Trade |
| 6 | Blue | Warfare |
| 7 | Dark Blue | Technology |
| 8 | Purple | Imperial |

## Naalu Flag and Offset Logic

By default, Naalu (index 0) is disabled:

```rust
if !args.naalu {
    state.toggle_enabled(0);
}
```

The `offset` field controls which indices are shown:

| Flag | `num_cards` | `offset` | Shown |
|------|-------------|----------|-------|
| Default (no `--naalu`) | 8 | 1 | indices 1-8 |
| `--naalu` | 9 | 0 | indices 0-8 |

The `show_indices()` method maps display positions to real card indices:

```rust
fn show_indices(&self) -> Vec<usize> {
    (0..self.num_cards).map(|i| i + self.offset).collect()
}
```

## Button Behavior

| Action | Effect |
|--------|--------|
| Single press | Next enabled initiative |
| Double press | Reset to starting initiative |
| Long press | Ignored |

## Thread Safety

Global `TRACKER_STATE` (`Arc<RwLock<TrackerState>>`) bridges tokio async tasks and egui's main thread, same pattern as timer-switcher.

## Key Files

| File | Purpose |
|------|---------|
| `src/tracker.rs` | Tracker logic, global state, commands, card data |
| `src/app.rs` | egui UI, keyboard/mouse handling |
| `src/main.rs` | Entry point, async wiring, CLI |
| `src/lib.rs` | Module re-exports |

## Developer Commands

```bash
# Run the app
cargo run -p initiative-tracker -- [--button <ids>] [--naalu] [--start N] [--debug]

# Tests
cargo test -p initiative-tracker
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api` | `http://localhost:3000` | button-hub API URL |
| `--button` | (optional) | Comma-separated button IDs |
| `--naalu` | `false` | Include Naalu (default: 8 cards shown) |
| `--start` | `1` | Starting initiative (Leadership) |
| `--debug` | `false` | Verbose event logging |

## Keyboard Controls

| Key | Action |
|-----|--------|
| `SPACE` / `→` / `↑` | Next initiative |
| `←` / `↓` / `⌫` | Previous initiative |
| `R` | Reset to starting initiative |
| `0-8` | Toggle that card on/off |
| `ESC` | Quit |

## Known Quirks

1. **100 ms repaint**: `ctx.request_repaint_after(100ms)` ensures button-press state changes are visible promptly.
2. **Disabled cards**: Clicking a card toggles its enabled state. If the current card is disabled, `toggle_enabled` advances to the next enabled card.
3. **Color32 premultiplication**: `from_rgba_unmultiplied` stores values internally premultiplied. Used in `dim_color()` to set alpha on already-RGB-known colors.
4. **`ViewportBuilder`**: Always uses `with_resizable(true)`.
5. **Keyboard input loop**: Scans `input.events` for `Key { pressed: true }`. Only processes the first pressed key per frame.
