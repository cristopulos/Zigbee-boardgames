//! Button registration and unregistration with the hub API.

use std::error::Error;

/// Registers a button ID with button-hub.
///
/// This is best-effort — errors are logged but not fatal.
pub async fn register(api_url: &str, button_id: &str) -> Result<(), Box<dyn Error + Send + Sync>> {
    let url = format!("{}/buttons", trim_slash(api_url));
    let body = serde_json::json!({ "button_id": button_id });

    let client = reqwest::Client::new();
    let resp = client
        .post(&url)
        .json(&body)
        .send()
        .await?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("register failed: {} {}", status, body).into());
    }

    Ok(())
}

/// Unregisters a button ID from button-hub.
pub async fn unregister(
    api_url: &str,
    button_id: &str,
) -> Result<(), Box<dyn Error + Send + Sync>> {
    let url = format!("{}/buttons/{}", trim_slash(api_url), button_id);

    let client = reqwest::Client::new();
    let resp = client.delete(&url).send().await?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("unregister failed: {} {}", status, body).into());
    }

    Ok(())
}

fn trim_slash(url: &str) -> &str {
    url.trim_end_matches('/')
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_trim_slash() {
        assert_eq!(trim_slash("http://localhost:3000/"), "http://localhost:3000");
        assert_eq!(trim_slash("http://localhost:3000"), "http://localhost:3000");
        assert_eq!(trim_slash("http://localhost:3000//"), "http://localhost:3000");
    }
}