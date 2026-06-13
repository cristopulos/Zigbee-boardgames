# AGENTS.md — timer-switcher-rs

Compact instructions for AI agents working in this app.

## Overview

A Rust/egui app that cycles through named timers via Zigbee button presses or keyboard input. Connects to button-hub via SSE using the `button-client` crate.

## Architecture

```
timer.rs (logic, global state) → main.rs (async wiring) → app.rs (egui UI)
                                       ↓
                                 background ticker thread
```

- **`timer.rs`**: `TimerState`, `TimerCommand`, `TIMER_STATE` global, tick gating, `TimerThread` RAII guard
- **`main.rs`**: CLI parsing, tokio async button listeners, egui app runner, owns the `TimerThread` guard
- **`app.rs`**: `TimerCard` widget, `TimerSwitcherApp`, keyboard/mouse handling (no longer drives the timer)
- **Background thread**: spawned by `TimerThread::start()`, runs the 1 Hz tick loop independently of egui

## Button Mapping

| Condition | Mode | Behavior |
|-----------|------|----------|
| `len(button_ids) == len(timers)` | Direct map (1:1) | Button N activates timer N |
| Otherwise | Cycle mode | Any button cycles to next timer |

**1:1 fallback**: If pressing the button for the already-active timer, falls back to cycling.

## Timer Behavior

### Background Ticker Thread

The 1 Hz tick is driven by a dedicated `std::thread`, not by the egui `update()` callback. **Why:** egui pauses `update()` while the window is unfocused, which previously halted timer progress. The background thread keeps ticking regardless of window focus.

**How it works:**

- `timer::TimerThread::start()` spawns a thread that loops `tick(); thread::sleep(1s)` until shutdown.
- A `TIMER_RUNNING` `AtomicBool` (Acquire/Release ordering) signals the loop to exit.
- `TimerThread` is an **RAII guard**: its `Drop` impl flips the flag and `join()`s the thread, guaranteeing clean shutdown on normal exit, early return, or panic.
- `main.rs` binds it to `let _timer_thread = timer::TimerThread::start();` **before** `eframe::run_native` and keeps the binding alive for the whole process lifetime.

### 1 Hz Tick Gating

`TimerState::tick()` uses `Instant` to ensure only one increment per real second:

```rust
let should_tick = match self.last_tick {
    None => true,
    Some(last) => now.duration_since(last) >= Duration::from_secs(1),
};
```

This prevents timer drift even if the ticker thread wakes up early or is preempted.

### 100 ms Repaint Interval

The UI calls `ctx.request_repaint_after(100ms)` to ensure button-press state changes are visible promptly. Without this, egui only redraws on input events. **Note:** the repaint interval no longer drives the timer — it exists solely to surface state mutations (button presses, pause/resume) to the user.

## Thread Safety

Global `TIMER_STATE` (`Arc<RwLock<TimerState>>`) is shared across three threads:

- **Tokio async tasks**: button handlers call `execute(TimerCommand::...)` which locks `TIMER_STATE` and modifies state
- **Background ticker thread**: calls `tick()` once per second to advance the active timer
- **egui main thread**: reads state via `state.snapshot()` in `update()` to render

Never call UI methods directly from async tasks or the ticker thread. The ticker never invokes `tick()` while holding the write lock during a UI snapshot — `RwLock` handles concurrent access, but keep critical sections short.

## Key Files

| File | Purpose |
|------|---------|
| `src/timer.rs` | Timer logic, global state, commands, `TimerThread` background ticker |
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

1. **Ticker starts immediately**: The background thread begins ticking at app start and calls `tick_test`-style logic, so elapsed time accumulates even before the window is visible or focused. If the app launches and is never shown, time still passes.
2. **`TimerThread` guard lifetime**: The `let _timer_thread = ...` binding in `main.rs` must outlive `eframe::run_native`. Do not move it into a narrower scope (e.g., a helper function returning early) — dropping the guard before the event loop returns will kill the ticker.
3. **1 Hz gating**: `tick()` must be called every second; `Instant` comparison handles jitter. Tests use `tick_test()` which skips gating.
4. **Paused + active = amber**: When the active timer is paused, time text turns amber.
5. **`ViewportBuilder`**: Always uses `with_resizable(true)` — without this, window resize fails silently.
6. **`Color32`**: Colors use standard RGB constructors. No premultiplication issues in this app.
