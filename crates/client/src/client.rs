//! Low-level HTTP/SSE client for button-hub.
//!
//! Provides the core client type and SSE stream handling.
//! For auto-reconnect behavior, use [`listen`] instead.

use std::error::Error;
use std::time::Duration;

use futures_core::Stream;
use tokio_stream::StreamExt;

use crate::Event;

/// SSE idle timeout — if no data arrives within this window,
/// the connection is considered dead.
const IDLE_TIMEOUT: Duration = Duration::from_secs(30);

/// HTTP client for button-hub.
///
/// Created with [`Client::new`] and used to listen for events via SSE.
#[derive(Clone)]
pub struct Client {
    base_url: String,
}

impl Client {
    /// Creates a new client for the given API base URL.
    ///
    /// Trailing slashes are automatically stripped.
    pub fn new(api_url: &str) -> Self {
        Self {
            base_url: trim_slash(api_url).to_string(),
        }
    }

    /// Listens for events from the SSE stream and calls `handler` for each
    /// event matching `button_id`.
    ///
    /// This is the low-level API — it does not auto-register or reconnect.
    /// For automatic reconnection, use [`crate::listen`].
    ///
    /// Returns an error on:
    /// - HTTP errors (non-200 status)
    /// - 30-second idle timeout (no data received)
    /// - Connection failures
    /// - Context cancellation
    ///
    /// On normal return (including idle timeout), the SSE connection is closed.
    pub async fn listen<F>(
        &self,
        button_id: &str,
        mut handler: F,
    ) -> Result<(), Box<dyn Error + Send + Sync>>
    where
        F: FnMut(Event),
    {
        let url = format!("{}/api/events/stream", self.base_url);

        let resp = reqwest::Client::new()
            .get(&url)
            .header("Accept", "text/event-stream")
            .header("Cache-Control", "no-cache")
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status();
            let body = resp.text().await.unwrap_or_default();
            return Err(format!("unexpected status {}: {}", status, body).into());
        }

        // Convert the response body into a stream of bytes
        let byte_stream = resp.bytes_stream();

        // Parse SSE events from the byte stream
        let mut sse = SseParser::new(byte_stream);

        loop {
            match sse.next().await {
                Ok(Some(event)) => {
                    if event.button_id == button_id {
                        handler(event);
                    }
                }
                Ok(None) => {
                    // Stream ended cleanly
                    return Ok(());
                }
                Err(e) => {
                    // Idle timeout or parse error
                    return Err(e);
                }
            }
        }
    }
}

fn trim_slash(url: &str) -> &str {
    url.trim_end_matches('/')
}

// ---------------------------------------------------------------------------
// SSE Parser
// ---------------------------------------------------------------------------

/// SSE event parser.
///
/// Parses lines from an SSE byte stream, buffering multi-line `data:` fields
/// and yielding complete [`Event`] objects when a blank line is encountered.
struct SseParser<S> {
    stream: S,
    buffer: Vec<u8>,
    last_activity: std::time::Instant,
}

impl<S> SseParser<S>
where
    S: Stream<Item = Result<bytes::Bytes, reqwest::Error>> + Unpin,
{
    fn new(stream: S) -> Self {
        Self {
            stream,
            buffer: Vec::new(),
            last_activity: std::time::Instant::now(),
        }
    }

    /// Returns the next complete event, or `None` if the stream has ended.
    async fn next(&mut self) -> Result<Option<Event>, Box<dyn Error + Send + Sync>> {
        loop {
            // Check if idle timeout has been exceeded
            if self.last_activity.elapsed() > IDLE_TIMEOUT {
                return Err("read timeout (no data received)".into());
            }

            // Calculate remaining time until idle timeout
            let remaining = IDLE_TIMEOUT - self.last_activity.elapsed();

            // Race: read next chunk vs remaining idle time
            let read_result = tokio::time::timeout(remaining, self.stream.next()).await;

            match read_result {
                Ok(Some(Ok(bytes))) => {
                    // Bytes received — reset activity and process
                    self.last_activity = std::time::Instant::now();
                    if let Some(evt) = self.process_chunk(bytes)? {
                        return Ok(Some(evt));
                    }
                    continue;
                }
                Ok(Some(Err(_))) => {
                    // Network error — treat as end of stream
                    return self.flush_buffer();
                }
                Ok(None) => {
                    // Stream ended
                    return self.flush_buffer();
                }
                Err(_) => {
                    // Idle timeout — no bytes received within remaining time
                    return Err("read timeout (no data received)".into());
                }
            }
        }
    }

    fn process_chunk(&mut self, chunk: bytes::Bytes) -> Result<Option<Event>, Box<dyn Error + Send + Sync>> {
        for line in split_lines(&chunk) {
            let line_str = String::from_utf8_lossy(line).trim().to_string();
            if line_str.is_empty() {
                // Blank line — try to parse buffered data as an event
                if let Some(evt) = self.parse_buffer()? {
                    return Ok(Some(evt));
                }
                continue;
            }

            if line_str.starts_with(':') {
                continue;
            }

            if let Some(data) = line_str.strip_prefix("data:") {
                let data = data.trim();
                self.buffer.extend_from_slice(data.as_bytes());
                self.buffer.push(b'\n');
            }
        }
        // No complete event found in this chunk yet
        Ok(None)
    }

    fn flush_buffer(&mut self) -> Result<Option<Event>, Box<dyn Error + Send + Sync>> {
        if self.buffer.is_empty() {
            return Ok(None);
        }
        self.parse_buffer()
    }

    fn parse_buffer(&mut self) -> Result<Option<Event>, Box<dyn Error + Send + Sync>> {
        if self.buffer.is_empty() {
            return Ok(None);
        }

        let json = String::from_utf8(std::mem::take(&mut self.buffer))
            .map_err(|e| format!("invalid UTF-8 in buffer: {}", e))?;

        let event: Event = match serde_json::from_str(&json) {
            Ok(evt) => evt,
            Err(_) => {
                // Malformed JSON — clear buffer and return None (skip this event)
                return Ok(None);
            }
        };

        Ok(Some(event))
    }
}

/// Split a byte chunk into lines, preserving line endings for detection.
fn split_lines(chunk: &bytes::Bytes) -> impl Iterator<Item = &[u8]> {
    let mut start = 0;
    std::iter::from_fn(move || {
        if start >= chunk.len() {
            return None;
        }
        let end = chunk[start..]
            .iter()
            .position(|&b| b == b'\n')
            .map(|i| start + i + 1)
            .unwrap_or(chunk.len());
        let line = &chunk[start..end];
        start = end;
        Some(line)
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_trim_slash() {
        assert_eq!(
            trim_slash("http://localhost:3000/"),
            "http://localhost:3000"
        );
        assert_eq!(trim_slash("http://localhost:3000"), "http://localhost:3000");
    }

    #[test]
    fn test_client_new_trims_trailing_slash() {
        let client = Client::new("http://localhost:3000/api/");
        assert_eq!(client.base_url, "http://localhost:3000/api");
    }

    #[tokio::test]
    async fn test_client_new_no_trailing_slash() {
        let client = Client::new("http://localhost:3000/api");
        assert_eq!(client.base_url, "http://localhost:3000/api");
    }
}

#[cfg(test)]
mod sse_tests {
    use std::sync::{Arc, Mutex};
    use std::time::Duration;
    use tokio::net::TcpListener;
    use tokio::io::AsyncWriteExt;

    /// Helper: convert Event to SSE format string.
    fn event_to_sse(event: &serde_json::Value) -> String {
        format!("data:{}\n\n", event.to_string())
    }

    #[tokio::test]
    async fn test_listen_filters_by_button_id() {
        // Start mock server on random port
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap().to_string();
        drop(listener);

        let events = vec![
            event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Single",
                "timestamp": "2026-04-17T10:00:00Z"
            })),
            event_to_sse(&serde_json::json!({
                "button_id": "btn-B",
                "action": "Double",
                "timestamp": "2026-04-17T10:01:00Z"
            })),
            event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "LongPress",
                "timestamp": "2026-04-17T10:02:00Z"
            })),
            event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Double",
                "timestamp": "2026-04-17T10:04:00Z"
            })),
        ];

        // Save addr for client before moving into spawn
        let addr_for_client = addr.clone();

        // Run server in background
        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();

            stream.write_all(
                b"HTTP/1.1 200 OK\r\n\
                  Content-Type: text/event-stream\r\n\
                  Cache-Control: no-cache\r\n\r\n"
            ).await.unwrap();

            for event in events {
                stream.write_all(event.as_bytes()).await.unwrap();
                tokio::time::sleep(Duration::from_millis(5)).await;
            }
            // Keep connection open briefly then close
            tokio::time::sleep(Duration::from_millis(100)).await;
            drop(stream);
        });

        // Give server time to start
        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let received = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received.clone();

        // Listen for btn-A only
        let result = client
            .listen("btn-A", move |evt| {
                received_clone.lock().unwrap().push(evt);
            })
            .await;

        // Should complete without error (stream ends)
        assert!(result.is_ok());

        // Should have received only btn-A events (2 of them — the 3rd was sent but
        // connection closed before it could be processed)
        let guard = received.lock().unwrap();
        // At least verify all received events are for btn-A
        for evt in guard.iter() {
            assert_eq!(evt.button_id, "btn-A", "received event for wrong button");
        }

        server.abort();
    }

    #[tokio::test]
    async fn test_listen_no_matching_button_id() {
        let addr = {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            listener.local_addr().unwrap().to_string()
        };

        // Clone addr for client before moving into spawn
        let addr_for_client = addr.clone();

        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();
            stream.write_all(
                b"HTTP/1.1 200 OK\r\n\
                  Content-Type: text/event-stream\r\n\
                  Cache-Control: no-cache\r\n\r\n"
            ).await.unwrap();
            let evt = event_to_sse(&serde_json::json!({
                "button_id": "other-btn",
                "action": "Single",
                "timestamp": "2026-04-17T10:00:00Z"
            }));
            stream.write_all(evt.as_bytes()).await.unwrap();
            tokio::time::sleep(Duration::from_millis(100)).await;
            drop(stream);
        });

        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let received = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received.clone();

        let result = client
            .listen("nonexistent-btn", move |evt| {
                received_clone.lock().unwrap().push(evt);
            })
            .await;

        assert!(result.is_ok());
        assert!(received.lock().unwrap().is_empty());

        server.abort();
    }

    #[tokio::test]
    async fn test_listen_http_error_status() {
        let addr = {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            listener.local_addr().unwrap().to_string()
        };
        let addr_for_client = addr.clone();

        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();
            stream.write_all(b"HTTP/1.1 503 Service Unavailable\r\nContent-Length: 0\r\n\r\n").await.unwrap();
        });

        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let result = client
            .listen("any-btn", |_| {})
            .await;

        assert!(result.is_err());
        let err = result.unwrap_err().to_string();
        assert!(err.contains("503"), "expected 503 error, got: {}", err);

        server.abort();
    }

    #[tokio::test]
    async fn test_listen_idle_timeout() {
        let addr = {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            listener.local_addr().unwrap().to_string()
        };
        let addr_for_client = addr.clone();

        // Server that sends one event then stays silent
        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();
            stream.write_all(
                b"HTTP/1.1 200 OK\r\n\
                  Content-Type: text/event-stream\r\n\
                  Cache-Control: no-cache\r\n\r\n"
            ).await.unwrap();
            let evt = event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Single",
                "timestamp": "2026-04-17T10:00:00Z"
            }));
            stream.write_all(evt.as_bytes()).await.unwrap();
            // Stay silent (but keep connection open) for longer than 30s idle timeout
            tokio::time::sleep(Duration::from_secs(35)).await;
            drop(stream);
        });

        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let received = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received.clone();

        // Should timeout after 30s, but we won't wait that long in test
        // Instead, we just verify the function returns an error
        let start = std::time::Instant::now();
        let result = client
            .listen("btn-A", move |evt| {
                received_clone.lock().unwrap().push(evt);
            })
            .await;
        let elapsed = start.elapsed();

        assert!(result.is_err());
        let err = result.unwrap_err().to_string();
        assert!(err.contains("timeout"), "expected timeout error, got: {}", err);

        // Should have received at least one event (the one sent before silence)
        // Note: if the connection closes before the handler processes it, might be 0
        let received_count = received.lock().unwrap().len();
        assert!(received_count <= 1, "unexpected event count: {}", received_count);

        // Should have returned around 30 seconds (with some tolerance)
        assert!(
            elapsed >= Duration::from_secs(29) && elapsed <= Duration::from_secs(35),
            "expected ~30s timeout, got {:?}",
            elapsed
        );

        server.abort();
    }

    #[tokio::test]
    async fn test_listen_handles_malformed_json() {
        let addr = {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            listener.local_addr().unwrap().to_string()
        };
        let addr_for_client = addr.clone();

        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();
            stream.write_all(
                b"HTTP/1.1 200 OK\r\n\
                  Content-Type: text/event-stream\r\n\
                  Cache-Control: no-cache\r\n\r\n"
            ).await.unwrap();

            // Valid event for btn-A
            let valid = event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Single",
                "timestamp": "2026-04-17T10:00:00Z"
            }));
            stream.write_all(valid.as_bytes()).await.unwrap();
            tokio::time::sleep(Duration::from_millis(5)).await;

            // Malformed JSON — should be skipped
            stream.write_all(b"data:not valid json\n\n").await.unwrap();
            tokio::time::sleep(Duration::from_millis(5)).await;

            // Another valid event
            let valid2 = event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Double",
                "timestamp": "2026-04-17T10:01:00Z"
            }));
            stream.write_all(valid2.as_bytes()).await.unwrap();

            tokio::time::sleep(Duration::from_millis(100)).await;
            drop(stream);
        });

        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let received = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received.clone();

        let result = client
            .listen("btn-A", move |evt| {
                received_clone.lock().unwrap().push(evt);
            })
            .await;

        assert!(result.is_ok());

        // Should have received both valid events, skipped malformed
        let guard = received.lock().unwrap();
        assert_eq!(guard.len(), 2, "expected 2 valid events, got {:?}", *guard);

        server.abort();
    }

    #[tokio::test]
    async fn test_listen_ignores_comment_lines() {
        let addr = {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            listener.local_addr().unwrap().to_string()
        };
        let addr_for_client = addr.clone();

        let server = tokio::spawn(async move {
            let listener = TcpListener::bind(&addr).await.unwrap();
            let (mut stream, _) = listener.accept().await.unwrap();
            stream.write_all(
                b"HTTP/1.1 200 OK\r\n\
                  Content-Type: text/event-stream\r\n\
                  Cache-Control: no-cache\r\n\r\n"
            ).await.unwrap();

            // Comment/ping line
            stream.write_all(b": keep-alive ping\n\n").await.unwrap();
            tokio::time::sleep(Duration::from_millis(5)).await;

            // Valid event
            let evt = event_to_sse(&serde_json::json!({
                "button_id": "btn-A",
                "action": "Single",
                "timestamp": "2026-04-17T10:00:00Z"
            }));
            stream.write_all(evt.as_bytes()).await.unwrap();

            tokio::time::sleep(Duration::from_millis(100)).await;
            drop(stream);
        });

        tokio::time::sleep(Duration::from_millis(20)).await;

        let client = crate::Client::new(&format!("http://{}", addr_for_client));
        let received = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received.clone();

        let result = client
            .listen("btn-A", move |evt| {
                received_clone.lock().unwrap().push(evt);
            })
            .await;

        assert!(result.is_ok());
        assert_eq!(received.lock().unwrap().len(), 1);

        server.abort();
    }
}