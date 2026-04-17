pub mod handlers;
pub mod server;
pub mod state;

#[cfg(test)]
mod tests;

pub use server::{build_router, serve};
pub use state::AppState;
