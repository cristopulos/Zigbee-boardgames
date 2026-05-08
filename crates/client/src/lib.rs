//! Client library for button-hub.
//!
//! Provides async functions for:
//! - [`Client`] — low-level HTTP/SSE client
//! - [`listen`] — high-level listener with auto-register, reconnect, and backoff
//! - [`register`] / [`unregister`] — button registration with the hub
//!
//! ## Example
//!
//! ```ignore
//! let client = Client::new("http://localhost:3000");
//!
//! client.listen("btn-001", |event| {
//!     println!("Button {:?} pressed: {:?}", event.button_id, event.action);
//! }).await;
//! ```

mod client;
mod listen;
mod registry;

pub use client::Client;
pub use listen::listen;
pub use registry::{register, unregister};
pub use button_core::Event;

// Re-export ActionType for convenience
pub use button_core::ActionType;