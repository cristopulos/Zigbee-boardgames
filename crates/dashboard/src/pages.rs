use crate::api::{connect_sse, fetch_buttons, fetch_events, fetch_health};
use crate::components::{ButtonCard, EventTimeline, StatusBar};
use button_core::Event;
use futures_util::StreamExt;
use leptos::prelude::*;
use leptos_meta::*;
use std::collections::HashMap;

#[component]
pub fn DashboardPage() -> impl IntoView {
    let (buttons, set_buttons) = signal(Vec::<String>::new());
    let (events, set_events) = signal(Vec::<Event>::new());
    let (api_ok, set_api_ok) = signal(false);
    let (mqtt_ok, set_mqtt_ok) = signal(false);

    // Initial data load
    Effect::new(move |_| {
        let set_b = set_buttons;
        let set_e = set_events;
        let set_api = set_api_ok;
        let set_mqtt = set_mqtt_ok;
        wasm_bindgen_futures::spawn_local(async move {
            if let Some(health) = fetch_health().await {
                set_api.set(health.status == "ok");
            }
            let btns = fetch_buttons().await;
            set_b.set(btns.clone());
            let evts = fetch_events(20).await;
            set_e.set(evts);
            // Assume MQTT is up if we can fetch buttons/events
            set_mqtt.set(true);
        });
    });

    // Start SSE connection
    Effect::new(move |_| {
        connect_sse(set_events);
    });

    // Periodic health check every 5s
    Effect::new(move |_| {
        let set_api = set_api_ok;
        wasm_bindgen_futures::spawn_local(async move {
            let mut interval = gloo_timers::future::IntervalStream::new(5_000);
            while interval.next().await.is_some() {
                if let Some(health) = fetch_health().await {
                    set_api.set(health.status == "ok");
                } else {
                    set_api.set(false);
                }
            }
        });
    });

    // Compute last event per button
    let last_events = Memo::new(move |_| {
        let mut map = HashMap::<String, Event>::new();
        for e in events.get().iter().rev() {
            map.entry(e.button_id.clone()).or_insert_with(|| e.clone());
        }
        map
    });

    view! {
        <Style id="dashboard-styles">{r#"
            :root {
                --bg: #0f1117;
                --surface: #1a1d27;
                --border: #2d3142;
                --accent: #22d3ee;
                --accent-dim: #0e7490;
                --text: #e2e8f0;
                --text-dim: #94a3b8;
                --success: #4ade80;
                --danger: #f87171;
                --warning: #fbbf24;
            }
            * { box-sizing: border-box; margin: 0; padding: 0; }
            body {
                background: var(--bg);
                color: var(--text);
                font-family: system-ui, -apple-system, sans-serif;
                line-height: 1.5;
                min-height: 100vh;
            }
            .dashboard {
                max-width: 1200px;
                margin: 0 auto;
                padding: 1.5rem;
            }
            .dashboard-header {
                font-size: 1.75rem;
                font-weight: 700;
                margin-bottom: 1rem;
                color: var(--accent);
            }
            .status-bar {
                display: flex;
                gap: 1.5rem;
                margin-bottom: 1.5rem;
                padding: 0.75rem 1rem;
                background: var(--surface);
                border: 1px solid var(--border);
                border-radius: 0.5rem;
            }
            .status-item {
                display: flex;
                align-items: center;
                gap: 0.5rem;
            }
            .status-dot {
                width: 0.6rem;
                height: 0.6rem;
                border-radius: 50%;
                background: var(--text-dim);
            }
            .status-dot.online { background: var(--success); box-shadow: 0 0 6px var(--success); }
            .status-dot.offline { background: var(--danger); }
            .status-label { font-size: 0.875rem; color: var(--text-dim); }
            .section-title {
                font-size: 1.125rem;
                font-weight: 600;
                margin: 1.5rem 0 0.75rem;
                color: var(--text);
            }
            .button-grid {
                display: grid;
                grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
                gap: 1rem;
            }
            .button-card {
                background: var(--surface);
                border: 1px solid var(--border);
                border-radius: 0.5rem;
                padding: 1rem;
            }
            .button-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 0.75rem;
            }
            .button-name { font-weight: 600; font-size: 1rem; }
            .button-body { display: flex; flex-direction: column; gap: 0.5rem; }
            .button-stat { display: flex; justify-content: space-between; font-size: 0.875rem; }
            .stat-label { color: var(--text-dim); }
            .stat-value { font-weight: 500; }
            .stat-value.dim { color: var(--text-dim); }
            .battery-high { color: var(--success); }
            .battery-med { color: var(--warning); }
            .battery-low { color: var(--danger); }
            .timeline {
                background: var(--surface);
                border: 1px solid var(--border);
                border-radius: 0.5rem;
                overflow: hidden;
            }
            .timeline-header {
                padding: 0.75rem 1rem;
                font-weight: 600;
                border-bottom: 1px solid var(--border);
                background: rgba(255,255,255,0.02);
            }
            .timeline-body { max-height: 320px; overflow-y: auto; }
            .event-item {
                display: grid;
                grid-template-columns: 5rem 1fr 1fr;
                gap: 1rem;
                padding: 0.5rem 1rem;
                border-bottom: 1px solid var(--border);
                font-size: 0.875rem;
            }
            .event-item:last-child { border-bottom: none; }
            .event-time { color: var(--text-dim); font-variant-numeric: tabular-nums; }
            .event-button { color: var(--accent); }
            .event-action { text-align: right; }
        "#}</Style>

        <div class="dashboard">
            <div class="dashboard-header">"button-hub Dashboard"</div>

            <StatusBar api_ok=api_ok.into() mqtt_ok=mqtt_ok.into() />

            <div class="section-title">"Buttons"</div>
            <div class="button-grid">
                {
                    let button_list = move || buttons.get();
                    view! {
                        <For
                            each=button_list
                            key=|b: &String| b.clone()
                            children=move |button_id: String| {
                                let button_id_for_memo = button_id.clone();
                                let last = Memo::new(move |_| {
                                    last_events.get().get(&button_id_for_memo).cloned()
                                });
                                view! {
                                    <ButtonCard
                                        button_id=button_id.clone()
                                        last_event=last.into()
                                    />
                                }
                            }
                        />
                    }
                }
            </div>

            <div class="section-title">"Events"</div>
            <EventTimeline events=events.into() />
        </div>
    }
}
