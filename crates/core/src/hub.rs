use crate::ButtonRegistry;
use crate::Event;
use std::sync::Arc;
use tokio::sync::mpsc::Receiver;

pub struct Hub {
    registry: Arc<ButtonRegistry>,
}

impl Hub {
    pub fn new(registry: Arc<ButtonRegistry>) -> Self {
        Self { registry }
    }

    pub async fn run(&self, mut event_rx: Receiver<Event>) {
        while let Some(event) = event_rx.recv().await {
            self.registry.dispatch(event).await;
        }
        tracing::info!("Hub event channel closed, shutting down");
    }
}
