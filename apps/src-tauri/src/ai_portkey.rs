//! Explicit native-only Portkey configuration. No discovery, local inference or secret logging.
use super::ChatMessage;
use serde_json::{json, Value};
use std::time::Duration;

static REQUESTS: tokio::sync::Semaphore = tokio::sync::Semaphore::const_new(2);

fn resolve_secrets(value: &mut Value, lookup: &impl Fn(&str) -> Option<String>) -> Result<(), String> {
    match value {
        Value::String(text) if text.starts_with('$') => {
            let name = &text[1..];
            if name.is_empty() || !name.chars().all(|c| c.is_ascii_uppercase() || c.is_ascii_digit() || c == '_') {
                return Err("Portkey config contains an invalid environment reference".into());
            }
            *text = lookup(name).filter(|s| !s.trim().is_empty())
                .ok_or_else(|| format!("Portkey config requires native environment variable {name}"))?;
        }
        Value::Array(items) => for item in items { resolve_secrets(item, lookup)?; },
        Value::Object(map) => {
            // Strip documentation-only comments, never send those to the gateway.
            map.remove("_comment");
            if let Some(Value::String(host)) = map.get("custom_host") {
                let url = reqwest::Url::parse(host).map_err(|_| "Invalid Portkey provider host")?;
                let hostname = url.host_str().unwrap_or("");
                if url.scheme() != "https" || hostname.eq_ignore_ascii_case("localhost") ||
                    hostname.trim_matches(['[', ']']).parse::<std::net::IpAddr>().is_ok_and(|ip| ip.is_loopback() || ip.is_unspecified()) {
                    return Err("Intake requires remote HTTPS model providers".into());
                }
            }
            for item in map.values_mut() { resolve_secrets(item, lookup)?; }
        }
        _ => {}
    }
    Ok(())
}

pub async fn chat(model: &str, messages: Vec<ChatMessage>) -> Result<String, String> {
    let _permit = REQUESTS.try_acquire().map_err(|_| "Two Intake chat requests are already running")?;
    let base = std::env::var("INTAKE_PORTKEY_URL").map_err(|_| "Configure INTAKE_PORTKEY_URL in the native launcher")?;
    let mut endpoint = reqwest::Url::parse(&base).map_err(|_| "Invalid INTAKE_PORTKEY_URL")?;
    if !matches!(endpoint.scheme(), "http" | "https") || endpoint.host_str().is_none() ||
        !endpoint.username().is_empty() || endpoint.password().is_some() || endpoint.query().is_some() || endpoint.fragment().is_some() {
        return Err("Portkey URL must be an HTTP(S) gateway without embedded credentials, query or fragment".into());
    }
    let path = format!("{}/v1/chat/completions", endpoint.path().trim_end_matches('/'));
    endpoint.set_path(&path);
    let config_path = std::env::var("INTAKE_PORTKEY_CONFIG_FILE").map_err(|_| "Configure INTAKE_PORTKEY_CONFIG_FILE in the native launcher")?;
    let config_path = std::path::Path::new(&config_path);
    if !config_path.is_absolute() { return Err("Portkey config path must be absolute".into()); }
    let metadata = tokio::fs::metadata(config_path).await.map_err(|_| "Cannot inspect Portkey config file")?;
    if metadata.len() > 64 * 1024 { return Err("Portkey config exceeds 64 KiB".into()); }
    let bytes = tokio::fs::read(config_path).await.map_err(|_| "Cannot read Portkey config file")?;
    if bytes.len() > 64 * 1024 { return Err("Portkey config exceeds 64 KiB".into()); }
    let mut config: Value = serde_json::from_slice(&bytes).map_err(|_| "Invalid Portkey config JSON")?;
    if !config.is_object() { return Err("Portkey config must be a JSON object".into()); }
    // `$NAME` resolves from the environment, else from the file named by `NAME_FILE` (the hosted
    // engine keeps model keys in root-only secret files, not its environment; 2026-09-18).
    resolve_secrets(&mut config, &|name| {
        std::env::var(name).ok().or_else(|| {
            let file = std::env::var(format!("{name}_FILE")).ok()?;
            let value = std::fs::read_to_string(file).ok()?;
            let value = value.trim();
            (!value.is_empty()).then(|| value.to_string())
        })
    })?;
    let config_header = serde_json::to_string(&config).map_err(|_| "Cannot encode Portkey config")?;
    let mut header = reqwest::header::HeaderValue::from_str(&config_header).map_err(|_| "Invalid Portkey config header")?;
    header.set_sensitive(true);
    if messages.iter().any(|m| !matches!(m.role.as_str(), "system" | "user" | "assistant")) {
        return Err("Unsupported chat message role".into());
    }
    let client = reqwest::Client::builder().timeout(Duration::from_secs(55))
        .connect_timeout(Duration::from_secs(10)).redirect(reqwest::redirect::Policy::none())
        .build().map_err(|_| "Cannot create Portkey client")?;
    let mut response = client.post(endpoint).header("x-portkey-config", header)
        .json(&json!({ "model": model, "messages": messages, "max_tokens": 4096, "stream": false }))
        .send().await.map_err(|_| "Portkey request failed or timed out")?;
    if !response.status().is_success() { return Err(format!("Portkey returned HTTP {}", response.status().as_u16())); }
    let mut body = Vec::new();
    while let Some(chunk) = response.chunk().await.map_err(|_| "Cannot read Portkey response")? {
        if body.len() + chunk.len() > 1024 * 1024 { return Err("Portkey response exceeds 1 MiB".into()); }
        body.extend_from_slice(&chunk);
    }
    let response: Value = serde_json::from_slice(&body).map_err(|_| "Invalid Portkey response JSON")?;
    response.pointer("/choices/0/message/content").and_then(Value::as_str)
        .filter(|text| !text.trim().is_empty()).map(str::to_owned)
        .ok_or_else(|| "Portkey returned no assistant text".into())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn substitutes_nested_native_secrets_without_changing_model() {
        let mut value = json!({"targets": [{"provider": "openai", "custom_host": "https://ollama.com/v1", "api_key": "$OLLAMA_API_KEY", "override_params": {"model": "nemotron-3-ultra"}}]});
        resolve_secrets(&mut value, &|name| (name == "OLLAMA_API_KEY").then(|| "synthetic-only".into())).unwrap();
        assert_eq!(value["targets"][0]["api_key"], "synthetic-only");
        assert_eq!(value["targets"][0]["override_params"]["model"], "nemotron-3-ultra");
    }

    #[test]
    fn missing_keys_and_local_hosts_fail_closed() {
        assert!(resolve_secrets(&mut json!({"api_key": "$MISSING"}), &|_| None).is_err());
        assert!(resolve_secrets(&mut json!({"custom_host": "http://localhost:11434"}), &|_| None).is_err());
    }
}
