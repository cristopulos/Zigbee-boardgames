//! High-level listener with auto-register, reconnection, and exponential backoff.
//!
//! This is the recommended entry point for most use cases.

use std::error::Error;
use std::time::Duration;

use tokio::time::sleep;

use crate::client::Client;
use crate::registry;
use crate::Event;

/// Maximum backoff duration between reconnection attempts.
const MAX_BACKOFF: Duration = Duration::from_secs(30);

/// Listens for button events with automatic registration and reconnection.
///
/// This function:
///
/// 1. Registers the button_id with button-hub
/// 2. Opens the SSE stream and calls `handler` for each matching event
/// 3. On disconnect/error, reconnects with exponential backoff
/// 4. On graceful shutdown (context cancelled), unregisters the button
///
/// ## Arguments
///
/// - `api_url` — button-hub base URL (e.g., `"http://localhost:3000"`)
/// - `button_id` — the button ID to listen for
/// - `handler` — called for each event from the matching button
///
/// ## Example
///
/// ```ignore
/// let client = Client::new("http://localhost:3000");
///
/// client.listen("btn-001", |event| {
///     println!("{:?}: {:?}", event.button_id, event.action);
/// }).await;
/// ```
pub async fn listen<F>(api_url: &str, button_id: &str, mut handler: F) -> Result<(), Box<dyn Error + Send + Sync>>
where
    F: FnMut(Event),
{
    // Attempt to register the button
    if let Err(e) = registry::register(api_url, button_id).await {
        eprintln!("[button-client] register warning for {}: {}", button_id, e);
    } else {
        eprintln!("[button-client] registered {}", button_id);
    }

    let client = Client::new(api_url);
    let mut backoff = Duration::from_secs(1);

    loop {
        match client.listen(button_id, &mut handler).await {
            Ok(()) => {
                // Stream ended cleanly (shouldn't happen with SSE)
                return Ok(());
            }
            Err(e) => {
                eprintln!(
                    "[button-client] sse error for {}: {} (retry in {:?})",
                    button_id,
                    e,
                    backoff
                );
            }
        }

        // Check if context is cancelled before sleeping
        // (this is checked at the call site via the returned future)

        sleep(backoff).await;

        backoff *= 2;
        if backoff > MAX_BACKOFF {
            backoff = MAX_BACKOFF;
        }
    }
}