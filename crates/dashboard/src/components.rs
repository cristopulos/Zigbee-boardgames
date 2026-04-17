use crate::api::{format_action, format_time};
use button_core::Event;
use leptos::prelude::*;

#[component]
pub fn StatusBar(api_ok: Signal<bool>, mqtt_ok: Signal<bool>) -> impl IntoView {
    view! {
        <div class="status-bar">
            <div class="status-item">
                <span class={move || if api_ok.get() { "status-dot online" } else { "status-dot offline" }} />
                <span class="status-label">{move || if api_ok.get() { "API Online" } else { "API Offline" }}</span>
            </div>
            <div class="status-item">
                <span class={move || if mqtt_ok.get() { "status-dot online" } else { "status-dot offline" }} />
                <span class="status-label">{move || if mqtt_ok.get() { "MQTT Online" } else { "MQTT Offline" }}</span>
            </div>
        </div>
    }
}

#[component]
pub fn ButtonCard(button_id: String, last_event: Signal<Option<Event>>) -> impl IntoView {
    view! {
        <div class="button-card">
            <div class="button-header">
                <span class="button-name">{button_id.clone()}</span>
                <span class={move || {
                    if last_event.get().is_some() { "status-dot online" } else { "status-dot offline" }
                }} />
            </div>
            <div class="button-body">
                <div class="button-stat">
                    <span class="stat-label">"Last Action"</span>
                    <span class="stat-value">{
                        move || last_event.get().as_ref().map(|e| format_action(&e.action)).unwrap_or_else(|| "—".to_string())
                    }</span>
                </div>
                <div class="button-stat">
                    <span class="stat-label">"Battery"</span>
                    <span class={move || {
                        last_event.get().and_then(|e| e.battery).map(|b| {
                            let class = if b > 50 { "battery-high" } else if b > 20 { "battery-med" } else { "battery-low" };
                            format!("stat-value battery {}", class)
                        }).unwrap_or_else(|| "stat-value".to_string())
                    }}>{
                        move || last_event.get().and_then(|e| e.battery).map(|b| format!("{}%", b)).unwrap_or_else(|| "—".to_string())
                    }</span>
                </div>
                <div class="button-stat">
                    <span class="stat-label">"Last Seen"</span>
                    <span class="stat-value dim">{
                        move || last_event.get().as_ref().map(|e| format_time(&e.timestamp)).unwrap_or_else(|| "—".to_string())
                    }</span>
                </div>
            </div>
        </div>
    }
}

#[component]
pub fn EventTimeline(events: Signal<Vec<Event>>) -> impl IntoView {
    view! {
        <div class="timeline">
            <div class="timeline-header">"Recent Events"</div>
            <div class="timeline-body">
                {
                    let event_list = move || events.get().into_iter().take(20).collect::<Vec<Event>>();
                    view! {
                        <For
                            each=event_list
                            key=|e: &Event| format!("{}-{}", e.button_id, e.timestamp.to_rfc3339())
                            children=move |event: Event| {
                                view! {
                                    <div class="event-item">
                                        <span class="event-time">{format_time(&event.timestamp)}</span>
                                        <span class="event-button">{event.button_id.clone()}</span>
                                        <span class="event-action">{format_action(&event.action)}</span>
                                    </div>
                                }
                            }
                        />
                    }
                }
            </div>
        </div>
    }
}
