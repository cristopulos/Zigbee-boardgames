use api::{build_router, serve, AppState};
use button_core::{Button, ButtonRegistry, EventBroadcaster};
use mqtt::MqttClient;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    dotenvy::dotenv().ok();

    let mqtt_host = std::env::var("MQTT_BROKER_HOST").unwrap_or_else(|_| "localhost".to_string());
    let mqtt_port = std::env::var("MQTT_BROKER_PORT")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(1883u16);
    let client_id = std::env::var("MQTT_CLIENT_ID").unwrap_or_else(|_| {
        format!("button-hub-{}", std::process::id())
    });
    let api_port = std::env::var("API_PORT")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(3000u16);

    let registry = Arc::new(ButtonRegistry::new(500));
    let broadcaster = Arc::new(EventBroadcaster::new());
    registry.set_broadcaster(broadcaster.clone());

    registry.register(Button::new("button_1", |e| async move {
        tracing::info!(
            "Button 1 pressed: {:?} (battery: {:?}%)",
            e.action,
            e.battery
        );
    }));

    registry.register(Button::new("button_2", |e| async move {
        tracing::info!(
            "Button 2 pressed: {:?} (battery: {:?}%)",
            e.action,
            e.battery
        );
    }));

    let (event_tx, event_rx) = mpsc::channel(64);
    let hub = Arc::new(button_core::Hub::new(registry.clone()));
    let mqtt_client = MqttClient::new(&mqtt_host, mqtt_port, &client_id, event_tx);

    let app_state = Arc::new(AppState {
        registry: registry.clone(),
        broadcaster,
    });
    let router = build_router(app_state);

    let hub_handle = tokio::spawn(async move { hub.run(event_rx).await });
    let mqtt_handle = tokio::spawn(async move { mqtt_client.connect_and_listen().await });
    let api_handle = tokio::spawn(async move {
        if let Err(e) = serve(router, api_port).await {
            tracing::error!("API server error: {:?}", e);
            Err(e)
        } else {
            Ok(())
        }
    });

    tokio::select! {
        result = hub_handle => {
            if let Err(e) = result {
                tracing::error!("Hub task panicked: {:?}", e);
                std::process::exit(1);
            }
        }
        result = mqtt_handle => {
            if let Err(e) = result {
                tracing::error!("MQTT task panicked: {:?}", e);
                std::process::exit(1);
            }
        }
        result = api_handle => {
            match result {
                Ok(Ok(())) => {},
                Ok(Err(e)) => {
                    tracing::error!("API server error: {:?}", e);
                    std::process::exit(1);
                }
                Err(e) => {
                    tracing::error!("API task panicked: {:?}", e);
                    std::process::exit(1);
                }
            }
        }
    }
}
