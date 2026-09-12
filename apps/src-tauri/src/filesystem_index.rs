//! Read-only bridge to Intake's configured filesystem index, not evidence search.
use serde::{Deserialize, Serialize};
use std::sync::LazyLock;
use std::time::Duration;

const MAX_RESPONSE_BYTES: usize = 2 * 1024 * 1024;
static REQUEST_SLOTS: LazyLock<tokio::sync::Semaphore> =
    LazyLock::new(|| tokio::sync::Semaphore::new(2));

#[derive(Debug, Serialize)]
pub struct FilesystemSearchError {
    pub code: &'static str,
    pub message: String,
    pub status: Option<u16>,
}

fn error(code: &'static str, message: &str) -> FilesystemSearchError {
    FilesystemSearchError {
        code,
        message: message.to_owned(),
        status: None,
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct FilesystemHit {
    pub object_id: String,
    pub source_id: String,
    pub source_path: String,
    pub document_id: String,
    pub chunk_id: String,
    pub filename: String,
    pub text: String,
    pub score: f64,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct FilesystemSearchResponse {
    pub query: String,
    pub backend: String,
    pub collection: String,
    pub target_vector: Option<String>,
    pub coverage: String,
    pub hits: Vec<FilesystemHit>,
}

fn search_url(base: &str) -> Result<reqwest::Url, FilesystemSearchError> {
    let mut url = reqwest::Url::parse(base)
        .map_err(|_| error("invalid_configuration", "Filesystem index URL is invalid"))?;
    if !matches!(url.scheme(), "http" | "https")
        || url.host_str().is_none()
        || !url.username().is_empty()
        || url.password().is_some()
        || url.query().is_some()
        || url.fragment().is_some()
    {
        return Err(error(
            "invalid_configuration",
            "Filesystem index URL must be HTTP(S), without embedded credentials, query or fragment",
        ));
    }
    let path = format!("{}/filesystem/search", url.path().trim_end_matches('/'));
    url.set_path(&path);
    Ok(url)
}

fn validate_request(query: &str, mode: &str, limit: u32) -> Result<(), FilesystemSearchError> {
    if query.trim().is_empty()
        || query.chars().count() > 4096
        || !matches!(mode, "keyword" | "hybrid")
        || !(1..=100).contains(&limit)
    {
        return Err(error(
            "invalid_request",
            "Use a nonempty query up to 4096 characters, keyword/hybrid mode and limit 1–100",
        ));
    }
    Ok(())
}

#[tauri::command]
pub async fn filesystem_index_search(
    query: String,
    mode: String,
    limit: u32,
) -> Result<FilesystemSearchResponse, FilesystemSearchError> {
    validate_request(&query, &mode, limit)?;
    let base = std::env::var("INTAKE_FILESYSTEM_API_URL")
        .ok().filter(|v| !v.trim().is_empty())
        .ok_or_else(|| error("not_configured", "Filesystem index is not connected. Configure INTAKE_FILESYSTEM_API_URL in the native launcher; ordinary browsing remains available."))?;
    let url = search_url(&base)?;
    let _slot = REQUEST_SLOTS.try_acquire().map_err(|_| {
        error(
            "busy",
            "Two filesystem searches are already running; retry when one finishes",
        )
    })?;
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(25))
        .connect_timeout(Duration::from_secs(5))
        .redirect(reqwest::redirect::Policy::none())
        .build()
        .map_err(|_| {
            error(
                "client_error",
                "Cannot initialize filesystem index connection",
            )
        })?;
    let mut request = client.post(url).json(&serde_json::json!({
        "query": query, "mode": mode, "limit": limit
    }));
    if let Ok(token) = std::env::var("INTAKE_FILESYSTEM_API_TOKEN") {
        if !token.is_empty() {
            request = request.bearer_auth(token);
        }
    }
    let mut response = request.send().await.map_err(|e| {
        if e.is_timeout() {
            error("timeout", "Filesystem index request timed out")
        } else {
            error(
                "unavailable",
                "Cannot reach the configured filesystem index",
            )
        }
    })?;
    if !response.status().is_success() {
        return Err(FilesystemSearchError {
            code: "backend_error",
            message: "Filesystem index rejected the request or is not ready; no search result was produced".into(),
            status: Some(response.status().as_u16()),
        });
    }
    if response
        .content_length()
        .is_some_and(|len| len > MAX_RESPONSE_BYTES as u64)
    {
        return Err(error(
            "response_too_large",
            "Filesystem search response exceeds the 2 MiB transport limit",
        ));
    }
    let mut bytes = Vec::new();
    while let Some(chunk) = response.chunk().await.map_err(|_| {
        error(
            "response_error",
            "Filesystem search response was interrupted or timed out",
        )
    })? {
        if bytes.len().saturating_add(chunk.len()) > MAX_RESPONSE_BYTES {
            return Err(error(
                "response_too_large",
                "Filesystem search response exceeds the 2 MiB transport limit",
            ));
        }
        bytes.extend_from_slice(&chunk);
    }
    let result: FilesystemSearchResponse = serde_json::from_slice(&bytes).map_err(|_| {
        error(
            "invalid_response",
            "Filesystem index returned an invalid response shape",
        )
    })?;
    if result.hits.len() > limit as usize || result.backend != "weaviate" || result.query != query {
        return Err(error(
            "invalid_response",
            "Filesystem index response does not match this request",
        ));
    }
    Ok(result)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn url_preserves_api_prefix() {
        assert_eq!(
            search_url("http://127.0.0.1:4319/").unwrap().as_str(),
            "http://127.0.0.1:4319/filesystem/search"
        );
        assert_eq!(
            search_url("https://index.example/api").unwrap().as_str(),
            "https://index.example/api/filesystem/search"
        );
    }
    #[test]
    fn rejects_unsafe_or_ambiguous_urls() {
        for base in [
            "file:///E:/data",
            "http://user:secret@host",
            "http://host?key=value",
            "http://host#fragment",
        ] {
            assert!(search_url(base).is_err());
        }
    }
    #[test]
    fn request_bounds() {
        assert!(validate_request("files", "keyword", 20).is_ok());
        assert!(validate_request(" ", "hybrid", 20).is_err());
        assert!(validate_request("files", "other", 20).is_err());
        assert!(validate_request("files", "hybrid", 101).is_err());
        assert!(validate_request(&"x".repeat(4097), "hybrid", 1).is_err());
    }
}
