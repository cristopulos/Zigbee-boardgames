# AGENTS.md — timer-switcher-rs

Compact instructions for AI agents working in this app.

## Overview

A Rust/egui app that cycles through named timers via Zigbee button presses or keyboard input. Connects to button-hub via SSE using the `button-client` crate.

## Architecture

```
timer.rs (logic, global state) → main.rs (async wiring) → app.rs (egui UI)
```

- **`timer.rs`**: `TimerState`, `TimerCommand`, `TIMER_STATE` global, tick gating
- **`main.rs`**: CLI parsing, tokio async button listeners, egui app runner
- **`app.rs`**: `TimerCard` widget, `TimerSwitcherApp`, keyboard/mouse handling

## Button Mapping

| Condition | Mode | Behavior |
|-----------|------|----------|
| `len(button_ids) == len(timers)` | Direct map (1:1) | Button N activates timer N |
| Otherwise | Cycle mode | Any button cycles to next timer |

**1:1 fallback**: If pressing the button for the already-active timer, falls back to cycling.

## Timer Behavior

### 1 Hz Tick Gating

`TimerState::tick()` uses `Instant` to ensure only one increment per real second:

```rust
let should_tick = match self.last_tick {
    None => true,
    Some(last) => now.duration_since(last) >= Duration::from_secs(1),
};
```

This prevents timer drift when `tick()` is called from a faster UI loop.

### 100 ms Repaint Interval

The UI calls `ctx.request_repaint_after(100ms)` to ensure button-press state changes are visible promptly. Without this, egui only redraws on input events.

## Thread Safety

Global `TIMER_STATE` (`Arc<RwLock<TimerState>>`) bridges tokio async tasks and egui's main thread:

- Async button handlers call `execute(TimerCommand::...)` which locks `TIMER_STATE` and modifies state
- UI reads state via `state.snapshot()` in `update()`
- Never call UI methods directly from async tasks

## Key Files

| File | Purpose |
|------|---------|
| `src/timer.rs` | Timer logic, global state, commands |
| `src/app.rs` | egui UI, widget rendering |
| `src/main.rs` | Entry point, async wiring, CLI |

## Developer Commands

```bash
# Run the app
cargo run -p timer-switcher -- [--button <ids>] [--timers <names>] [--debug]

# Tests
cargo test -p timer-switcher
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api` | `http://localhost:3000` | button-hub API URL |
| `--button` | (required) | Comma-separated button IDs |
| `--timers` | `Timer 1,Timer 2,Timer 3` | Comma-separated timer names |
| `--debug` | `false` | Verbose event logging |

## Known Quirks

1. **1 Hz gating**: `tick()` must be called every frame; `Instant` comparison handles the rest. Tests use `tick_test()` which skips gating.
2. **Paused + active = amber**: When the active timer is paused, time text turns amber.
3. **`ViewportBuilder`**: Always uses `with_resizable(true)` — without this, window resize fails silently.
4. **`Color32`**: Colors use standard RGB constructors. No premultiplication issues in this app.
