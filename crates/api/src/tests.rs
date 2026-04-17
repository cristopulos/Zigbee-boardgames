#[cfg(test)]
mod tests {
    use crate::{build_router, AppState};
    use axum::{
        body::{to_bytes, Body},
        http::Request,
    };
    use button_core::{ActionType, Button, ButtonRegistry, Event, EventBroadcaster};
    use std::sync::Arc;
    use tower::ServiceExt;

    fn setup_router() -> (axum::Router, Arc<ButtonRegistry>) {
        let registry = Arc::new(ButtonRegistry::new(500));
        let broadcaster = Arc::new(EventBroadcaster::new());
        registry.register(Button::new("button_1", |_e| async move {}));
        registry.register(Button::new("button_2", |_e| async move {}));
        let state = Arc::new(AppState {
            registry: registry.clone(),
            broadcaster,
        });
        let router = build_router(state);
        (router, registry)
    }

    #[tokio::test]
    async fn health_returns_ok() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .uri("/health")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["status"], "ok");
        assert!(json.get("timestamp").is_some());
    }

    #[tokio::test]
    async fn buttons_returns_registered_ids() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .uri("/buttons")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let buttons = json["buttons"].as_array().unwrap();
        assert_eq!(buttons.len(), 2);
        assert!(buttons.iter().any(|b| b == "button_1"));
        assert!(buttons.iter().any(|b| b == "button_2"));
    }

    #[tokio::test]
    async fn events_empty_initially() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .uri("/events")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["events"].as_array().unwrap().len(), 0);
        assert_eq!(json["count"], 0);
    }

    #[tokio::test]
    async fn events_returns_dispatched_event() {
        let (router, registry) = setup_router();
        let event = Event {
            button_id: "button_1".to_string(),
            action: ActionType::Single,
            battery: Some(85),
            timestamp: chrono::Utc::now(),
        };
        registry.dispatch(event).await;
        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;

        let response = router
            .oneshot(
                Request::builder()
                    .uri("/events")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["count"], 1);
        assert_eq!(json["events"][0]["button_id"], "button_1");
    }

    #[tokio::test]
    async fn event_by_unknown_id_returns_404() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .uri("/events/unknown_button")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 404);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["error"], "not found");
    }

    #[tokio::test]
    async fn events_limit_respected() {
        let (router, registry) = setup_router();
        for i in 0..10 {
            let event = Event {
                button_id: format!("btn{}", i),
                action: ActionType::Single,
                battery: None,
                timestamp: chrono::Utc::now(),
            };
            registry.dispatch(event).await;
        }
        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;

        let response = router
            .oneshot(
                Request::builder()
                    .uri("/events?limit=5")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let events = json["events"].as_array().unwrap();
        assert_eq!(events.len(), 5);
        // Should return most recent 5 in chronological order
        assert_eq!(events[0]["button_id"], "btn5");
        assert_eq!(events[4]["button_id"], "btn9");
    }

    #[tokio::test]
    async fn register_button_adds_id() {
        let (router, _) = setup_router();
        let response = router
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/buttons")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"button_id":"new_btn"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["registered"], true);
        assert_eq!(json["button_id"], "new_btn");

        let buttons = router
            .oneshot(
                Request::builder()
                    .uri("/buttons")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        let body = to_bytes(buttons.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let ids = json["buttons"].as_array().unwrap();
        assert!(ids.iter().any(|b| b == "new_btn"));
    }

    #[tokio::test]
    async fn unregister_button_removes_id() {
        let (router, _) = setup_router();
        let response = router
            .clone()
            .oneshot(
                Request::builder()
                    .method("DELETE")
                    .uri("/buttons/button_1")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["unregistered"], true);
        assert_eq!(json["button_id"], "button_1");

        let buttons = router
            .oneshot(
                Request::builder()
                    .uri("/buttons")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        let body = to_bytes(buttons.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        let ids = json["buttons"].as_array().unwrap();
        assert!(!ids.iter().any(|b| b == "button_1"));
    }

    #[tokio::test]
    async fn unregister_unknown_button_returns_404() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .method("DELETE")
                    .uri("/buttons/unknown")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 404);
    }

    #[tokio::test]
    async fn register_empty_button_id_returns_400() {
        let (router, _) = setup_router();
        let response = router
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/buttons")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"button_id":"  "}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 400);
    }

    #[tokio::test]
    async fn register_button_id_too_long_returns_400() {
        let (router, _) = setup_router();
        // button_id with 65 characters (exceeds 64 char limit)
        let long_button_id = "a".repeat(65);
        let response = router
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/buttons")
                    .header("content-type", "application/json")
                    .body(Body::from(format!(r#"{{"button_id":"{}"}}"#, long_button_id)))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 400);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["error"], "button_id must be 1-64 characters");
    }

    #[tokio::test]
    async fn register_button_id_exactly_64_chars_succeeds() {
        let (router, _) = setup_router();
        // button_id with exactly 64 characters (boundary case)
        let max_button_id = "a".repeat(64);
        let response = router
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/buttons")
                    .header("content-type", "application/json")
                    .body(Body::from(format!(r#"{{"button_id":"{}"}}"#, max_button_id)))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), 200);
        let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
        let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
        assert_eq!(json["registered"], true);
        assert_eq!(json["button_id"].as_str().unwrap().len(), 64);
    }
}
