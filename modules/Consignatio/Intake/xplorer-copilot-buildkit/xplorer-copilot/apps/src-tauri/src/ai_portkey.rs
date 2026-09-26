//! Explicit native-only Portkey configuration. No discovery, local inference or secret logging.
//! Byline: Claude Code · Opus 5 · 2026-09-18 (Gemini primary + Kimi fallback, reasoning strip)
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

/// Remove model reasoning from user-visible text. Hosted models may inline it as
/// `<think>…</think>` / `<thinking>…</thinking>`, or emit only the closing tag after
/// unlabelled reasoning; everything up to the last closing tag is reasoning.
fn strip_reasoning(text: &str) -> String {
    static BLOCKS: std::sync::OnceLock<regex::Regex> = std::sync::OnceLock::new();
    let blocks = BLOCKS.get_or_init(|| regex::Regex::new(r"(?is)<(think|thinking|reasoning)>.*?</(think|thinking|reasoning)>").unwrap());
    let mut out = blocks.replace_all(text, "").to_string();
    for tag in ["</think>", "</thinking>", "</reasoning>"] {
        if let Some(i) = out.rfind(tag) { out = out[i + tag.len()..].to_string(); }
    }
    // An unclosed opening tag means the reply was cut off inside reasoning.
    for tag in ["<think>", "<thinking>", "<reasoning>"] {
        if let Some(i) = out.find(tag) { out.truncate(i); }
    }
    out.trim().to_string()
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
    let client = reqwest::Client::builder().timeout(Duration::from_secs(175))
        .connect_timeout(Duration::from_secs(10)).redirect(reqwest::redirect::Policy::none())
        .build().map_err(|_| "Cannot create Portkey client")?;
    // No `tools` are sent: the chat's file actions are text blocks the client parses, so no
    // provider tool_calls (or Gemini thought_signatures) ever need to round-trip. Primary/fallback
    // and the per-target model are chosen by the Portkey config (strategy.mode = fallback).
    let started = std::time::Instant::now();
    let mut response = client.post(endpoint).header("x-portkey-config", header)
        .json(&json!({ "model": model, "messages": messages, "max_tokens": 4096, "stream": false }))
        .send().await.map_err(|_| "Portkey request failed or timed out")?;
    let served_by = response.headers().get("x-portkey-last-used-option-index")
        .and_then(|v| v.to_str().ok()).unwrap_or("single").to_string();
    tracing::info!(target: "intake_chat", status = response.status().as_u16(), served_by = %served_by,
        elapsed_ms = started.elapsed().as_millis() as u64, "portkey chat");
    if !response.status().is_success() { return Err(format!("Portkey returned HTTP {}", response.status().as_u16())); }
    let mut body = Vec::new();
    while let Some(chunk) = response.chunk().await.map_err(|_| "Cannot read Portkey response")? {
        if body.len() + chunk.len() > 1024 * 1024 { return Err("Portkey response exceeds 1 MiB".into()); }
        body.extend_from_slice(&chunk);
    }
    let response: Value = serde_json::from_slice(&body).map_err(|_| "Invalid Portkey response JSON")?;
    // Separate reasoning fields (`reasoning_content`, Kimi) are never read; inline reasoning is stripped.
    response.pointer("/choices/0/message/content").and_then(Value::as_str)
        .map(strip_reasoning).filter(|text| !text.is_empty())
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
    fn resolves_keys_inside_fallback_targets() {
        let mut value = json!({"strategy": {"mode": "fallback"}, "targets": [
            {"provider": "openai", "custom_host": "https://generativelanguage.googleapis.com/v1beta/openai", "api_key": "$GEMINI_API_KEY"},
            {"provider": "openai", "custom_host": "https://integrate.api.nvidia.com/v1", "api_key": "$NVIDIA_API_KEY"}]});
        resolve_secrets(&mut value, &|name| Some(format!("k-{name}"))).unwrap();
        assert_eq!(value["targets"][0]["api_key"], "k-GEMINI_API_KEY");
        assert_eq!(value["targets"][1]["api_key"], "k-NVIDIA_API_KEY");
        assert_eq!(value["strategy"]["mode"], "fallback");
    }

    #[test]
    fn strips_inline_reasoning() {
        assert_eq!(strip_reasoning("<think>plan it</think>
Renamed the file."), "Renamed the file.");
        assert_eq!(strip_reasoning("<Thinking>a
b</Thinking>Done"), "Done");
        assert_eq!(strip_reasoning("the user wants x, so...</think>Answer"), "Answer");
        assert_eq!(strip_reasoning("Answer only"), "Answer only");
        assert_eq!(strip_reasoning("Partial <think>never closed"), "Partial");
        assert_eq!(strip_reasoning("<think>only reasoning</think>"), "");
    }

    #[test]
    fn missing_keys_and_local_hosts_fail_closed() {
        assert!(resolve_secrets(&mut json!({"api_key": "$MISSING"}), &|_| None).is_err());
        assert!(resolve_secrets(&mut json!({"custom_host": "http://localhost:11434"}), &|_| None).is_err());
    }
}
