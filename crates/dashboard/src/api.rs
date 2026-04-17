use button_core::{ActionType, Event};
use leptos::prelude::*;
use serde::{Deserialize, Serialize};
use wasm_bindgen::prelude::*;
use web_sys::{EventSource, MessageEvent};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ButtonsResponse {
    pub buttons: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventsResponse {
    pub events: Vec<Event>,
    pub count: usize,
}

fn api_base() -> String {
    // In dev (Trunk), API is proxied at same origin
    // In prod, same origin
    String::new()
}

pub async fn fetch_health() -> Option<HealthResponse> {
    let url = format!("{}/health", api_base());
    let resp = gloo_net::http::Request::get(&url).send().await.ok()?;
    resp.json().await.ok()
}

pub async fn fetch_buttons() -> Vec<String> {
    let url = format!("{}/buttons", api_base());
    if let Ok(resp) = gloo_net::http::Request::get(&url).send().await {
        if let Ok(data) = resp.json::<ButtonsResponse>().await {
            return data.buttons;
        }
    }
    Vec::new()
}

pub async fn fetch_events(limit: usize) -> Vec<Event> {
    let url = format!("{}/events?limit={}", api_base(), limit);
    if let Ok(resp) = gloo_net::http::Request::get(&url).send().await {
        if let Ok(data) = resp.json::<EventsResponse>().await {
            return data.events;
        }
    }
    Vec::new()
}

pub fn connect_sse(set_events: WriteSignal<Vec<Event>>) {
    let url = format!("{}/api/events/stream", api_base());
    let es = match EventSource::new(&url) {
        Ok(es) => es,
        Err(e) => {
            tracing::error!("Failed to create EventSource: {:?}", e);
            return;
        }
    };

    let callback = Closure::wrap(Box::new(move |e: web_sys::Event| {
        let msg = e.dyn_ref::<MessageEvent>().unwrap();
        if let Some(data) = msg.data().as_string() {
            if let Ok(event) = serde_json::from_str::<Event>(&data) {
                set_events.update(|events| {
                    events.insert(0, event);
                    if events.len() > 100 {
                        events.truncate(100);
                    }
                });
            } else if let Ok(val) = serde_json::from_str::<serde_json::Value>(&data) {
                if val.get("type").and_then(|v| v.as_str()) == Some("missed") {
                    tracing::warn!("SSE lagged, missed events");
                }
            }
        }
    }) as Box<dyn FnMut(_)>);

    es.add_event_listener_with_callback("message", callback.as_ref().unchecked_ref())
        .unwrap();

    // Prevent closure from being dropped
    callback.forget();
}

pub fn format_action(action: &ActionType) -> String {
    match action {
        ActionType::Single => "Single".to_string(),
        ActionType::Double => "Double".to_string(),
        ActionType::LongPress => "Long Press".to_string(),
        ActionType::Unknown(s) => format!("Unknown ({})", s),
    }
}

pub fn format_time(timestamp: &chrono::DateTime<chrono::Utc>) -> String {
    timestamp.format("%H:%M:%S").to_string()
}
