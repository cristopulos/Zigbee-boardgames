//! Initiative Tracker — cycles through TI4 strategy cards via Zigbee buttons.
//!
//! Usage:
//!     cargo run -p initiative-tracker -- [--button <id>[,<id>...]] [--naalu] [--start N]
//!
//! Button behavior:
//!   - Single press → Next initiative
//!   - Double press → Reset to starting initiative
//!   - Long press → ignored
//!
//! Keyboard controls:
//!   - SPACE/→/↑ → Next
//!   - ←/↓/⌫ → Prev
//!   - R → Reset
//!   - 0-8 → Toggle that card
//!   - ESC → Quit

use clap::Parser;

mod tracker;
mod app;

use tracker::{TrackerState, TrackerCommand, execute, TRACKER_STATE, NUM_INITIATIVES};
use app::InitiativeTrackerApp;

/// Parse comma-separated string into a Vec<String>.
fn parse_ids(s: &str) -> Vec<String> {
    s.split(',')
        .map(|p| p.trim().to_string())
        .filter(|p| !p.is_empty())
        .collect()
}

#[derive(Parser, Debug)]
#[command(author, version, about = "Initiative Tracker — cycles TI4 strategy cards via Zigbee buttons")]
struct Args {
    /// button-hub API base URL
    #[arg(long, default_value = "http://localhost:3000")]
    api: String,

    /// Comma-separated button IDs to listen for (optional)
    #[arg(long)]
    button: Option<String>,

    /// Include Naalu initiative 0 (default: 8 cards, Naalu excluded)
    #[arg(long, short)]
    naalu: bool,

    /// Starting initiative number (default: 1 for Leadership)
    #[arg(long, short, default_value = "1")]
    start: usize,

    /// Include debug logging
    #[arg(long, short)]
    debug: bool,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    // Determine number of cards
    let num_cards = if args.naalu { 9 } else { 8 };

    // Validate and clamp start
    let start = if args.start < NUM_INITIATIVES { args.start } else { 1 };

    // Initialize the shared tracker state
    {
        let mut state = TRACKER_STATE.write();
        *state = TrackerState::new(start);
        // Disable Naalu (index 0) if not using --naalu
        if !args.naalu {
            state.toggle_enabled(0);
        }
    }

    // Parse button IDs
    let button_ids: Vec<String> = args.button.as_ref()
        .map(|s| parse_ids(s))
        .filter(|v| !v.is_empty())
        .unwrap_or_default();

    let mode_str = if button_ids.is_empty() {
        "keyboard-only mode"
    } else if button_ids.len() == 1 {
        "single button → Next"
    } else {
        "multiple buttons"
    };

    println!("Initiative Tracker started, {} initiatives, {}", num_cards, mode_str);
    if !button_ids.is_empty() {
        println!("Listening for buttons: {}", button_ids.join(", "));
    }
    println!("Controls: SPACE=Next, ←/↓=Prev, R=Reset, 0-8=Toggle, ESC=Quit");

    // Spawn async button listeners
    let api_url = args.api.clone();
    for button_id in &button_ids {
        let api_url = api_url.clone();
        let button_id = button_id.clone();
        let debug = args.debug;

        tokio::spawn(async move {
            if debug {
                println!("[main] starting listener for button={}", button_id);
            }

            let client = button_client::Client::new(&api_url);
            let result = client.listen(&button_id, move |event| {
                if debug {
                    println!("[remote] received: button_id={} action={:?}", event.button_id, event.action);
                }

                match event.action {
                    button_client::ActionType::Single => {
                        if debug {
                            println!("[remote] handling Single -> Next");
                        }
                        execute(TrackerCommand::Next);
                    }
                    button_client::ActionType::Double => {
                        if debug {
                            println!("[remote] handling Double -> Reset");
                        }
                        execute(TrackerCommand::Reset);
                    }
                    _ => {
                        if debug {
                            println!("[remote] ignored: {:?}", event.action);
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

    // Run the egui app
    let options = eframe::NativeOptions {
        viewport: eframe::egui::ViewportBuilder::default()
            .with_inner_size([1200.0, 220.0])
            .with_resizable(true),
        ..Default::default()
    };

    eframe::run_native(
        "Initiative Tracker",
        options,
        Box::new(|cc| Ok(Box::new(InitiativeTrackerApp::new(cc, num_cards)))),
    )
    .map_err(|e| anyhow::anyhow!("egui error: {}", e))?;

    Ok(())
}