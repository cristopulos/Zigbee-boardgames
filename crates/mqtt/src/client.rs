use button_core::{ActionType, Event};
use rumqttc::{AsyncClient, MqttOptions, Packet, QoS};
use std::time::Duration;
use tokio::sync::mpsc::Sender;
use tokio::time::sleep;

pub struct MqttClient {
    options: MqttOptions,
    event_tx: Sender<Event>,
}

impl MqttClient {
    pub fn new(broker_host: &str, broker_port: u16, client_id: &str, tx: Sender<Event>) -> Self {
        let mut options = MqttOptions::new(client_id, broker_host, broker_port);
        options.set_keep_alive(Duration::from_secs(30));
        options.set_clean_session(true);
        Self {
            options,
            event_tx: tx,
        }
    }

    pub async fn connect_and_listen(&self) {
        let cap = 10;
        let (client, mut eventloop) = AsyncClient::new(self.options.clone(), cap);

        if let Err(e) = client.subscribe("zigbee2mqtt/+", QoS::AtLeastOnce).await {
            tracing::error!("Failed to subscribe to zigbee2mqtt/+: {:?}", e);
        }

        loop {
            match eventloop.poll().await {
                Ok(rumqttc::Event::Incoming(Packet::Publish(p))) => {
                    let topic = p.topic;
                    let payload = p.payload;

                    let button_id = topic.strip_prefix("zigbee2mqtt/").unwrap_or(&topic);
                    if button_id == "bridge" || button_id.starts_with("bridge/") {
                        continue;
                    }

                    let json: serde_json::Value = match serde_json::from_slice(&payload) {
                        Ok(v) => v,
                        Err(e) => {
                            tracing::warn!("Failed to parse MQTT payload as JSON: {:?}", e);
                            continue;
                        }
                    };

                    let action = json.get("action").and_then(|v| v.as_str()).unwrap_or("");
                    if action.is_empty() {
                        tracing::debug!("Skipping empty action from topic {}", topic);
                        continue;
                    }

                    let battery = json
                        .get("battery")
                        .and_then(|v| v.as_u64())
                        .map(|v| v as u8);

                    let event = Event {
                        button_id: button_id.to_string(),
                        action: ActionType::from(action),
                        battery,
                        timestamp: chrono::Utc::now(),
                    };

                    if let Err(e) = self.event_tx.try_send(event) {
                        tracing::warn!("Event channel full, dropping event: {:?}", e);
                    }
                }
                Ok(_) => {}
                Err(e) => {
                    tracing::error!("MQTT connection error: {:?}", e);
                    sleep(Duration::from_secs(5)).await;
                }
            }
        }
    }
}
