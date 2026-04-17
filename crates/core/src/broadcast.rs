use crate::Event;
use tokio::sync::broadcast;

pub type BroadcastRx = broadcast::Receiver<Event>;

pub struct EventBroadcaster {
    tx: broadcast::Sender<Event>,
}

impl EventBroadcaster {
    pub fn new() -> Self {
        let (tx, _rx) = broadcast::channel(256);
        Self { tx }
    }

    pub fn broadcast(&self, event: Event) {
        let _ = self.tx.send(event);
    }

    pub fn subscribe(&self) -> BroadcastRx {
        self.tx.subscribe()
    }
}

impl Default for EventBroadcaster {
    fn default() -> Self {
        Self::new()
    }
}
