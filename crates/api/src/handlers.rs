use crate::state::AppState;
use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::{IntoResponse, Json},
};
use button_core::Event;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::sync::Arc;

#[derive(Serialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: String,
}

#[derive(Serialize)]
pub struct ButtonsResponse {
    pub buttons: Vec<String>,
}

#[derive(Serialize)]
pub struct EventsResponse {
    pub events: Vec<Event>,
    pub count: usize,
}

#[derive(Deserialize)]
pub struct EventsQuery {
    #[serde(default = "default_limit")]
    pub limit: usize,
}

fn default_limit() -> usize {
    20
}

const MAX_LIMIT: usize = 100;

#[derive(Deserialize)]
pub struct RegisterRequest {
    pub button_id: String,
}

#[derive(Serialize)]
pub struct RegisterResponse {
    pub registered: bool,
    pub button_id: String,
}

#[derive(Serialize)]
pub struct UnregisterResponse {
    pub unregistered: bool,
    pub button_id: String,
}

pub async fn health() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "ok".to_string(),
        timestamp: Utc::now().to_rfc3339(),
    })
}

pub async fn buttons(State(state): State<Arc<AppState>>) -> Json<ButtonsResponse> {
    Json(ButtonsResponse {
        buttons: state.registry.button_ids(),
    })
}

pub async fn register_button(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RegisterRequest>,
) -> impl IntoResponse {
    let id = payload.button_id.trim();
    if id.is_empty() || id.len() > 64 {
        return (
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({"error": "button_id must be 1-64 characters"})),
        )
            .into_response();
    }
    state.registry.register_button_id(id);
    (
        StatusCode::OK,
        Json(RegisterResponse {
            registered: true,
            button_id: payload.button_id,
        }),
    )
        .into_response()
}

pub async fn unregister_button(
    State(state): State<Arc<AppState>>,
    Path(button_id): Path<String>,
) -> impl IntoResponse {
    let removed = state.registry.unregister_button_id(&button_id);
    if removed {
        (
            StatusCode::OK,
            Json(UnregisterResponse {
                unregistered: true,
                button_id,
            }),
        )
            .into_response()
    } else {
        (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "not found"})),
        )
            .into_response()
    }
}

pub async fn events(
    State(state): State<Arc<AppState>>,
    Query(query): Query<EventsQuery>,
) -> Json<EventsResponse> {
    let limit = query.limit.min(MAX_LIMIT);
    let events = state.registry.latest_events(limit);
    let count = events.len();
    Json(EventsResponse { events, count })
}

pub async fn event_by_button_id(
    State(state): State<Arc<AppState>>,
    Path(button_id): Path<String>,
) -> impl IntoResponse {
    match state.registry.last_event_for(&button_id) {
        Some(event) => (StatusCode::OK, Json(serde_json::json!(event))).into_response(),
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "not found"})),
        )
            .into_response(),
    }
}

pub async fn history_by_button_id(
    State(state): State<Arc<AppState>>,
    Path(button_id): Path<String>,
    Query(query): Query<EventsQuery>,
) -> Json<EventsResponse> {
    let limit = query.limit.min(MAX_LIMIT);
    let events = state.registry.events_for_button(&button_id, limit);
    let count = events.len();
    Json(EventsResponse { events, count })
}

use axum::response::Sse;
use futures_util::StreamExt;
use tokio_stream::wrappers::BroadcastStream;

pub async fn events_stream(
    State(state): State<Arc<AppState>>,
) -> Sse<
    impl futures_util::Stream<Item = Result<axum::response::sse::Event, std::convert::Infallible>>,
> {
    let rx = state.broadcaster.subscribe();
    let stream = BroadcastStream::new(rx).map(|result| match result {
        Ok(event) => match serde_json::to_string(&event) {
            Ok(json) => Ok(axum::response::sse::Event::default().data(json)),
            Err(e) => {
                tracing::warn!("Failed to serialize event: {:?}", e);
                Ok(axum::response::sse::Event::default().data("{}"))
            }
        },
        Err(tokio_stream::wrappers::errors::BroadcastStreamRecvError::Lagged(n)) => {
            Ok(axum::response::sse::Event::default()
                .data(format!("{{\"type\":\"missed\",\"count\":{}}}", n)))
        }
    });

    Sse::new(stream).keep_alive(
        axum::response::sse::KeepAlive::new()
            .interval(std::time::Duration::from_secs(15))
            .text("keep-alive"),
    )
}
