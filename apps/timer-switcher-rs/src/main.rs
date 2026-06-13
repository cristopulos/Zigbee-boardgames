//! Timer Switcher — a Rust app that cycles through named timers via Zigbee button presses.
//!
//! Usage:
//!     cargo run -p timer-switcher -- --button <id>[,<id>...] [--timers <names>] [--api <url>]
//!
//! Button behavior:
//!   - When the number of buttons matches the number of timers, each button
//!     maps directly to its corresponding timer (1:1 mode).
//!   - Otherwise all buttons cycle through the timers in sequence (cycle mode).
//!   - Double-click any button to pause/resume the active timer.

use clap::Parser;

mod app;
mod timer;

use app::TimerSwitcherApp;
use timer::{execute, TimerCommand, TimerState, TIMER_STATE};

/// Parse comma-separated string into a Vec<String>.
fn parse_ids(s: &str) -> Vec<String> {
    s.split(',')
        .map(|p| p.trim().to_string())
        .filter(|p| !p.is_empty())
        .collect()
}

#[derive(Parser, Debug)]
#[command(
    author,
    version,
    about = "Timer Switcher — cycles timers via Zigbee buttons"
)]
struct Args {
    /// button-hub API base URL
    #[arg(long, default_value = "http://localhost:3000")]
    api: String,

    /// Comma-separated button IDs to listen for
    #[arg(long)]
    button: String,

    /// Comma-separated timer names (at least 2 required)
    #[arg(long, default_value = "Timer 1,Timer 2,Timer 3")]
    timers: String,

    /// Include debug logging
    #[arg(long, short)]
    debug: bool,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    let button_ids = parse_ids(&args.button);
    if button_ids.is_empty() {
        anyhow::bail!("--button is required");
    }

    let timer_names = parse_ids(&args.timers);
    if timer_names.len() < 2 {
        anyhow::bail!("need at least 2 timers, got {}", timer_names.len());
    }

    // Initialize the shared timer state
    {
        let mut state = TIMER_STATE.write();
        *state = TimerState::new(timer_names.clone());
    }

    // Determine mapping mode
    let direct_map = button_ids.len() == timer_names.len();
    let mode = if direct_map {
        "direct map (1:1)"
    } else {
        "cycle"
    };

    println!(
        "Timer Switcher started with {} timers, {} buttons, mode={}",
        timer_names.len(),
        button_ids.len(),
        mode
    );
    println!("Listening for buttons: {}", button_ids.join(", "));
    println!("Controls: SPACE = switch, ENTER = reset, P = pause, ESC = quit");

    // Spawn async button listeners
    let api_url = args.api.clone();
    for (idx, button_id) in button_ids.iter().enumerate() {
        let api_url = api_url.clone();
        let button_id = button_id.clone();
        let direct_map = direct_map;
        let idx = idx;
        let debug = args.debug;

        tokio::spawn(async move {
            if debug {
                println!(
                    "[main] starting listener for button={} idx={}",
                    button_id, idx
                );
            }

            let client = button_client::Client::new(&api_url);

            // Use low-level listen with error handling
            let result = client.listen(&button_id, move |event| {
                if debug {
                    println!("[remote] received: button_id={} action={:?}", event.button_id, event.action);
                }

                match event.action {
                    button_client::ActionType::Single => {
                        if debug {
                            let active = TIMER_STATE.read().active_index;
                            println!("[remote] handling Single: button={} idx={} directMap={} active={}", 
                                event.button_id, idx, direct_map, active);
                        }

                        if direct_map {
                            // If this button's timer is already active, cycle instead of no-op
                            let active = TIMER_STATE.read().active_index;
                            if active == idx {
                                execute(TimerCommand::Cycle);
                            } else {
                                execute(TimerCommand::SwitchTo(idx));
                            }
                        } else {
                            execute(TimerCommand::Cycle);
                        }

                        if debug {
                            let active = TIMER_STATE.read().active_index;
                            let paused = TIMER_STATE.read().paused;
                            println!("[remote] Single handled: active={} paused={}", active, paused);
                        }
                    }
                    button_client::ActionType::Double => {
                        if debug {
                            println!("[remote] handling Double: button={} -> TogglePause", event.button_id);
                        }
                        execute(TimerCommand::TogglePause);
                    }
                    _ => {
                        if debug {
                            println!("[remote] ignored: expected Single/Double, got {:?}", event.action);
                        }
                    }
                }
            }).await;

            if debug {
                match result {
                    Ok(()) => println!("[main] listener for {} ended normally", button_id),
                    Err(e) => println!("[main] listener for {} error: {}", button_id, e),
                }
            }
        });
    }

    // Background timer ticker so the timer advances even when the window is
    // unfocused (egui/eframe pause `update()` on unfocused windows). The
    // RAII guard stops and joins the thread on any exit path, including
    // errors and panics.
    let _timer_thread = timer::TimerThread::start();

    // Run the egui app
    let options = eframe::NativeOptions {
        viewport: eframe::egui::ViewportBuilder::default()
            .with_inner_size([900.0, 280.0])
            .with_resizable(true),
        ..Default::default()
    };

    eframe::run_native(
        "Timer Switcher",
        options,
        Box::new(|cc| Ok(Box::new(TimerSwitcherApp::new(cc, args.debug)))),
    )
    .map_err(|e| anyhow::anyhow!("egui error: {}", e))?;

    Ok(())
}
