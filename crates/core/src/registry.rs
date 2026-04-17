use crate::{Button, Event, EventBroadcaster};
use dashmap::DashMap;
use std::collections::VecDeque;
use std::sync::{Arc, Mutex};

pub struct ButtonRegistry {
    buttons: DashMap<String, Arc<Button>>,
    event_log: Arc<Mutex<VecDeque<Event>>>,
    log_capacity: usize,
    broadcaster: Mutex<Option<Arc<EventBroadcaster>>>,
}

impl ButtonRegistry {
    pub fn new(log_capacity: usize) -> Self {
        Self {
            buttons: DashMap::new(),
            event_log: Arc::new(Mutex::new(VecDeque::with_capacity(log_capacity))),
            log_capacity,
            broadcaster: Mutex::new(None),
        }
    }

    pub fn register(&self, button: Button) {
        self.buttons.insert(button.id.clone(), Arc::new(button));
    }

    pub fn register_button_id(&self, button_id: impl Into<String>) {
        let id = button_id.into();
        if id.trim().is_empty() {
            return;
        }
        self.register(Button::new(id, |_e| async move {}));
    }

    pub fn unregister_button_id(&self, button_id: &str) -> bool {
        self.buttons.remove(button_id).is_some()
    }

    pub fn set_broadcaster(&self, bc: Arc<EventBroadcaster>) {
        let mut guard = self.broadcaster.lock().unwrap();
        *guard = Some(bc);
    }

    pub async fn dispatch(&self, event: Event) {
        {
            let mut log = self.event_log.lock().unwrap();
            if log.len() >= self.log_capacity {
                log.pop_front();
            }
            log.push_back(event.clone());
        }

        if let Some(btn) = self.buttons.get(&event.button_id) {
            let handler = btn.handler.clone();
            let evt = event.clone();
            tokio::spawn(async move { handler(evt).await });
        } else {
            tracing::warn!("No handler registered for button_id: {}", event.button_id);
        }

        if let Ok(guard) = self.broadcaster.lock() {
            if let Some(bc) = guard.as_ref() {
                bc.broadcast(event);
            }
        }
    }

    pub fn latest_events(&self, limit: usize) -> Vec<Event> {
        let log = self.event_log.lock().unwrap();
        log.iter().rev().take(limit).rev().cloned().collect()
    }

    pub fn last_event_for(&self, button_id: &str) -> Option<Event> {
        let log = self.event_log.lock().unwrap();
        log.iter().rev().find(|e| e.button_id == button_id).cloned()
    }

    pub fn button_ids(&self) -> Vec<String> {
        self.buttons
            .iter()
            .map(|entry| entry.key().clone())
            .collect()
    }

    pub fn events_for_button(&self, button_id: &str, limit: usize) -> Vec<Event> {
        let log = self.event_log.lock().unwrap();
        let events: Vec<Event> = log
            .iter()
            .filter(|e| e.button_id == button_id)
            .cloned()
            .collect();
        let start = events.len().saturating_sub(limit);
        events[start..].to_vec()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event::ActionType;
    use std::sync::atomic::{AtomicUsize, Ordering};

    fn make_event(button_id: &str, action: &str) -> Event {
        Event {
            button_id: button_id.to_string(),
            action: ActionType::from(action),
            battery: Some(100),
            timestamp: chrono::Utc::now(),
        }
    }

    #[tokio::test]
    async fn dispatch_calls_handler_once() {
        let registry = ButtonRegistry::new(500);
        let counter = Arc::new(AtomicUsize::new(0));
        let c2 = counter.clone();

        registry.register(Button::new("btn1", move |_e| {
            let c = c2.clone();
            async move {
                c.fetch_add(1, Ordering::SeqCst);
            }
        }));

        registry.dispatch(make_event("btn1", "single")).await;
        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;

        assert_eq!(counter.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn dispatch_unknown_button_logs_warning() {
        let registry = ButtonRegistry::new(500);
        registry.dispatch(make_event("unknown", "single")).await;
        // Should not panic; event log still records it.
        let events = registry.latest_events(10);
        assert_eq!(events.len(), 1);
    }

    #[tokio::test]
    async fn event_log_capped_at_capacity() {
        let registry = ButtonRegistry::new(10);
        for i in 0..15 {
            let mut ev = make_event("btn", "single");
            ev.button_id = format!("btn{}", i);
            registry.dispatch(ev).await;
        }
        let log = registry.event_log.lock().unwrap();
        assert_eq!(log.len(), 10);
    }

    #[tokio::test]
    async fn latest_events_returns_chronological_order() {
        let registry = ButtonRegistry::new(500);
        for i in 0..5 {
            let mut ev = make_event("btn", "single");
            ev.button_id = format!("btn{}", i);
            registry.dispatch(ev).await;
        }
        let events = registry.latest_events(3);
        assert_eq!(events.len(), 3);
        assert_eq!(events[0].button_id, "btn2");
        assert_eq!(events[1].button_id, "btn3");
        assert_eq!(events[2].button_id, "btn4");
    }

    #[tokio::test]
    async fn last_event_for_none_when_empty() {
        let registry = ButtonRegistry::new(500);
        assert!(registry.last_event_for("btn1").is_none());
    }

    #[tokio::test]
    async fn register_button_id_adds_to_registry() {
        let registry = ButtonRegistry::new(500);
        registry.register_button_id("btn_x");
        let ids = registry.button_ids();
        assert!(ids.contains(&"btn_x".to_string()));
    }

    #[tokio::test]
    async fn unregister_button_id_removes_from_registry() {
        let registry = ButtonRegistry::new(500);
        registry.register_button_id("btn_y");
        assert!(registry.unregister_button_id("btn_y"));
        let ids = registry.button_ids();
        assert!(!ids.contains(&"btn_y".to_string()));
    }

    #[tokio::test]
    async fn unregister_unknown_button_id_returns_false() {
        let registry = ButtonRegistry::new(500);
        assert!(!registry.unregister_button_id("no_such_btn"));
    }

    #[tokio::test]
    async fn register_empty_button_id_ignored() {
        let registry = ButtonRegistry::new(500);
        registry.register_button_id("   ");
        assert!(registry.button_ids().is_empty());
    }
}
