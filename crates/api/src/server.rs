use crate::state::AppState;
use axum::{
    routing::{delete, get, post},
    Router,
};
use std::sync::Arc;
use tower_http::{services::ServeDir, trace::TraceLayer};

pub fn build_router(state: Arc<AppState>) -> Router {
    Router::new()
        .route(
            "/",
            get(|| async { axum::response::Redirect::permanent("/dashboard/") }),
        )
        .route("/health", get(crate::handlers::health))
        .route("/buttons", get(crate::handlers::buttons))
        .route("/buttons", post(crate::handlers::register_button))
        .route(
            "/buttons/:button_id",
            delete(crate::handlers::unregister_button),
        )
        .route("/events", get(crate::handlers::events))
        .route(
            "/events/:button_id",
            get(crate::handlers::event_by_button_id),
        )
        .route(
            "/events/:button_id/history",
            get(crate::handlers::history_by_button_id),
        )
        .route("/api/events/stream", get(crate::handlers::events_stream))
        .nest_service("/dashboard", ServeDir::new("crates/dashboard/dist"))
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

pub async fn serve(router: Router, port: u16) -> anyhow::Result<()> {
    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{}", port)).await?;
    tracing::info!("API server listening on 0.0.0.0:{}", port);
    axum::serve(listener, router)
        .with_graceful_shutdown(shutdown_signal())
        .await?;
    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }

    tracing::info!("Shutdown signal received, stopping API server");
}
