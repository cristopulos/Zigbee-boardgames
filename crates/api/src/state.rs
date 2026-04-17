use button_core::{ButtonRegistry, EventBroadcaster};
use std::sync::Arc;

pub struct AppState {
    pub registry: Arc<ButtonRegistry>,
    pub broadcaster: Arc<EventBroadcaster>,
}
