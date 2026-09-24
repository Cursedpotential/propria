//! Native-only, bounded bridge to the hosted Intake catalog/B2 name search.
//! Byline: Codex · GPT-6 · 2026-09-24.
//!
//! The hosted engine owns the catalog joins, B2 key scan, overlay, and custody
//! provenance. The desktop host supplies only the service base URL; the renderer
//! cannot choose an endpoint or send credentials.

use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::net::IpAddr;
use std::sync::LazyLock;
use std::time::Duration;

const MAX_RESPONSE_BYTES: usize = 2 * 1024 * 1024;
const MAX_QUERY_CHARS: usize = 200;
const MAX_PATH_CHARS: usize = 4096;
const MAX_LIMIT: u32 = 1_000;
const SCAN_CAP: u64 = 4_000;
static REQUEST_SLOTS: LazyLock<tokio::sync::Semaphore> =
    LazyLock::new(|| tokio::sync::Semaphore::new(2));

#[derive(Debug, Serialize)]
pub struct NameSearchError {
    pub code: &'static str,
    pub message: String,
    pub status: Option<u16>,
}

fn error(code: &'static str, message: &str) -> NameSearchError {
    NameSearchError {
        code,
        message: message.to_owned(),
        status: None,
    }
}

#[derive(Debug, Serialize)]
struct SearchRequest {
    query: String,
    scope: String,
    path: Option<String>,
    kinds: String,
    sort: String,
    limit: u32,
}

fn request(
    query: String,
    scope: Option<String>,
    path: Option<String>,
    kinds: Option<String>,
    sort: Option<String>,
    limit: Option<u32>,
) -> Result<SearchRequest, NameSearchError> {
    let query = query.trim().to_owned();
    if !(2..=MAX_QUERY_CHARS).contains(&query.chars().count()) || query.contains('\0') {
        return Err(error(
            "invalid_request",
            "Name search needs 2–200 query characters",
        ));
    }
    let scope = scope.unwrap_or_else(|| "everything".into());
    if !matches!(
        scope.as_str(),
        "everything"
            | "this_folder"
            | "this_folder_and_subfolders"
            | "subfolders_only"
            | "one_level_up"
            | "catalog_only"
            | "b2_only"
    ) {
        return Err(error("invalid_request", "Name search scope is invalid"));
    }
    let kinds = kinds.unwrap_or_else(|| "both".into());
    if !matches!(kinds.as_str(), "both" | "all" | "folders" | "files") {
        return Err(error("invalid_request", "Name search kind is invalid"));
    }
    let sort = sort.unwrap_or_else(|| "name".into());
    if !matches!(sort.as_str(), "name" | "path") {
        return Err(error("invalid_request", "Name search sort is invalid"));
    }
    let limit = limit.unwrap_or(200);
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(error("invalid_request", "Name search limit must be 1–1000"));
    }
    let path = path
        .map(|value| value.trim().to_owned())
        .filter(|value| !value.is_empty());
    if path
        .as_ref()
        .is_some_and(|value| value.chars().count() > MAX_PATH_CHARS || value.contains('\0'))
    {
        return Err(error(
            "invalid_request",
            "Name search folder path is invalid",
        ));
    }
    Ok(SearchRequest {
        query,
        scope,
        path,
        kinds,
        sort,
        limit,
    })
}

fn configured_base(value: Option<String>) -> Result<String, NameSearchError> {
    value.filter(|base| !base.trim().is_empty()).ok_or_else(|| {
        error(
            "not_configured",
            "Intake name search is not connected in the native launcher",
        )
    })
}

fn search_url_with_dev(
    base: &str,
    allow_loopback_dev: bool,
) -> Result<reqwest::Url, NameSearchError> {
    let mut url = reqwest::Url::parse(base)
        .map_err(|_| error("invalid_configuration", "Intake name-search URL is invalid"))?;
    if !matches!(url.scheme(), "http" | "https")
        || url.host_str().is_none()
        || !url.username().is_empty()
        || url.password().is_some()
        || url.query().is_some()
        || url.fragment().is_some()
        || url.path() != "/"
    {
        return Err(error(
            "invalid_configuration",
            "Intake name-search base URL must have no path, credentials, query or fragment",
        ));
    }
    let host = url.host_str().unwrap().trim_matches(['[', ']']);
    let allowed = match host.parse::<IpAddr>() {
        Ok(IpAddr::V4(ip)) => {
            let bits = u32::from(ip);
            (bits & 0xffc0_0000) == 0x6440_0000 || (allow_loopback_dev && ip.is_loopback())
        }
        Ok(IpAddr::V6(ip)) => {
            let seg = ip.segments();
            (seg[0], seg[1], seg[2]) == (0xfd7a, 0x115c, 0xa1e0)
                || (allow_loopback_dev && ip.is_loopback())
        }
        Err(_) => {
            (host.ends_with(".ts.net") && url.scheme() == "https")
                || (allow_loopback_dev && host.eq_ignore_ascii_case("localhost"))
        }
    };
    if !allowed {
        return Err(error(
            "invalid_configuration",
            "Intake name-search URL must be a Tailscale address or HTTPS ts.net host",
        ));
    }
    url.set_path("/api/intake_search_names");
    Ok(url)
}

fn search_url(base: &str) -> Result<reqwest::Url, NameSearchError> {
    // Loopback is not a release surface. A debug launcher must opt in explicitly;
    // unit tests exercise loopback transport without changing process globals.
    let allow_loopback_dev = cfg!(test)
        || (cfg!(debug_assertions)
            && std::env::var("INTAKE_NAME_SEARCH_ALLOW_LOOPBACK_DEV").as_deref() == Ok("1"));
    search_url_with_dev(base, allow_loopback_dev)
}

#[derive(Debug, Deserialize)]
struct SearchHit {
    kind: String,
    name: String,
    path: String,
    navigate_to: String,
    select: Option<String>,
    now: Option<String>,
    was: Option<String>,
    source: String,
    matched: String,
    member_of: Option<String>,
    size: Option<i64>,
    modified: Option<u64>,
    flag: Option<String>,
}

#[derive(Debug, Deserialize)]
struct SearchLeg {
    leg: String,
    rows: u64,
    ms: u64,
    capped: bool,
    error: Option<String>,
}

#[derive(Debug, Deserialize)]
struct SearchResponse {
    query: String,
    scope: String,
    kinds: String,
    sort: String,
    folder: Option<String>,
    hits: Vec<SearchHit>,
    shown_folders: u64,
    shown_files: u64,
    total_folders: u64,
    total_files: u64,
    capped: bool,
    scan_cap: u64,
    elapsed_ms: u64,
    legs: Vec<SearchLeg>,
    notes: Vec<String>,
}

fn checked_response(bytes: &[u8], expected: &SearchRequest) -> Result<Value, NameSearchError> {
    let raw: Value = serde_json::from_slice(bytes).map_err(|_| {
        error(
            "invalid_response",
            "Intake name search returned invalid JSON",
        )
    })?;
    // Serde accepts a missing Option field as None. The UI/custody contract,
    // however, distinguishes an explicit null from a field never supplied.
    let complete = raw
        .as_object()
        .is_some_and(|object| object.contains_key("folder"))
        && raw["hits"].as_array().is_some_and(|hits| {
            hits.iter().all(|hit| {
                hit.as_object().is_some_and(|object| {
                    [
                        "select",
                        "now",
                        "was",
                        "member_of",
                        "size",
                        "modified",
                        "flag",
                    ]
                    .iter()
                    .all(|key| object.contains_key(*key))
                })
            })
        })
        && raw["legs"].as_array().is_some_and(|legs| {
            legs.iter().all(|leg| {
                leg.as_object()
                    .is_some_and(|object| object.contains_key("error"))
            })
        });
    if !complete {
        return Err(error(
            "invalid_response",
            "Intake name search omitted custody or provenance fields",
        ));
    }
    let response: SearchResponse = serde_json::from_value(raw.clone()).map_err(|_| {
        error(
            "invalid_response",
            "Intake name search returned an invalid result shape",
        )
    })?;
    let hit_count = response.hits.len() as u64;
    if response.query != expected.query
        || response.scope != expected.scope
        || response.kinds != expected.kinds
        || response.sort != expected.sort
        || response.folder != expected.path
        || hit_count > expected.limit as u64
        || response.shown_folders.saturating_add(response.shown_files) != hit_count
        || response.shown_folders > response.total_folders
        || response.shown_files > response.total_files
        || response.scan_cap != SCAN_CAP
        || response.legs.len() > 5
        || response.legs.iter().any(|leg| leg.rows > SCAN_CAP)
        || response.hits.iter().any(|hit| {
            !matches!(hit.kind.as_str(), "folder" | "file")
                || !matches!(hit.source.as_str(), "catalog" | "b2" | "zip")
                || !matches!(hit.matched.as_str(), "name" | "path")
                || hit.name.is_empty()
                || hit.path.is_empty()
                || hit.navigate_to.is_empty()
        })
    {
        return Err(error(
            "invalid_response",
            "Intake name search result does not match this request",
        ));
    }
    // Typed decoding above also validates every nullable field's value type.
    for hit in &response.hits {
        let _ = (
            &hit.select,
            &hit.now,
            &hit.was,
            &hit.member_of,
            hit.size,
            hit.modified,
            &hit.flag,
        );
    }
    for leg in &response.legs {
        let _ = (&leg.leg, leg.ms, leg.capped, &leg.error);
    }
    let _ = (response.capped, response.elapsed_ms, &response.notes);
    Ok(raw)
}

async fn search_at(base: &str, request: &SearchRequest) -> Result<Value, NameSearchError> {
    let url = search_url(base)?;
    let _slot = REQUEST_SLOTS.try_acquire().map_err(|_| {
        error(
            "busy",
            "Two Intake name searches are already running; retry when one finishes",
        )
    })?;
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(30))
        .connect_timeout(Duration::from_secs(5))
        .redirect(reqwest::redirect::Policy::none())
        .build()
        .map_err(|_| {
            error(
                "client_error",
                "Cannot initialize Intake name-search connection",
            )
        })?;
    let mut response = client
        .post(url)
        .json(request)
        .send()
        .await
        .map_err(|cause| {
            if cause.is_timeout() {
                error("timeout", "Intake name search timed out")
            } else {
                error(
                    "unavailable",
                    "Cannot reach the configured Intake name-search service",
                )
            }
        })?;
    if !response.status().is_success() {
        return Err(NameSearchError {
            code: if response.status().as_u16() == 503 {
                "not_connected"
            } else {
                "backend_error"
            },
            message:
                "Intake name search is unavailable or rejected the request; no result was produced"
                    .into(),
            status: Some(response.status().as_u16()),
        });
    }
    if response
        .content_length()
        .is_some_and(|len| len > MAX_RESPONSE_BYTES as u64)
    {
        return Err(error(
            "response_too_large",
            "Intake name-search response exceeds 2 MiB",
        ));
    }
    let mut bytes = Vec::new();
    while let Some(chunk) = response.chunk().await.map_err(|_| {
        error(
            "response_error",
            "Intake name-search response was interrupted",
        )
    })? {
        if bytes.len().saturating_add(chunk.len()) > MAX_RESPONSE_BYTES {
            return Err(error(
                "response_too_large",
                "Intake name-search response exceeds 2 MiB",
            ));
        }
        bytes.extend_from_slice(&chunk);
    }
    checked_response(&bytes, request)
}

#[tauri::command]
pub async fn intake_search_names(
    query: String,
    scope: Option<String>,
    path: Option<String>,
    kinds: Option<String>,
    sort: Option<String>,
    limit: Option<u32>,
) -> Result<Value, NameSearchError> {
    if !crate::runtime_paths::intake_mode() {
        return Err(error(
            "unavailable",
            "Intake name search is available only in Intake mode",
        ));
    }
    let request = request(query, scope, path, kinds, sort, limit)?;
    let base = configured_base(std::env::var("INTAKE_NAME_SEARCH_API_URL").ok())?;
    search_at(&base, &request).await
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn command_is_registered_with_tauri() {
        assert!(
            include_str!("main.rs").contains("xplorer::intake_name_search::intake_search_names,")
        );
    }

    fn request_for_test() -> SearchRequest {
        request("Takeout".into(), None, None, None, None, Some(2)).unwrap()
    }

    fn fixture() -> Value {
        json!({
            "query":"Takeout", "scope":"everything", "kinds":"both", "sort":"name",
            "folder":null, "hits":[{
                "kind":"file", "name":"Takeout.zip", "path":"catalog://sms/Takeout.zip",
                "navigate_to":"catalog://sms", "select":"catalog://sms/Takeout.zip",
                "now":"/mnt/b2/vault/Takeout.zip", "was":"catalog://old/Takeout.zip",
                "source":"catalog", "matched":"name", "member_of":null,
                "size":42, "modified":1710000000, "flag":null
            }],
            "shown_folders":0, "shown_files":1, "total_folders":0, "total_files":1,
            "capped":false, "scan_cap":4000, "elapsed_ms":12,
            "legs":[{"leg":"catalog_files","rows":1,"ms":5,"capped":false,"error":null}],
            "notes":[]
        })
    }

    #[test]
    fn validates_query_scope_kind_sort_and_bounds() {
        assert_eq!(request_for_test().limit, 2);
        for query in ["", " ", "x"] {
            assert!(request(query.into(), None, None, None, None, None).is_err());
        }
        assert!(request("x".repeat(201), None, None, None, None, None).is_err());
        assert!(request("ab".into(), Some("unknown".into()), None, None, None, None).is_err());
        assert!(request("ab".into(), None, None, Some("unknown".into()), None, None).is_err());
        assert!(request("ab".into(), None, None, None, Some("unknown".into()), None).is_err());
        assert!(request("ab".into(), None, None, None, None, Some(0)).is_err());
        assert!(request("ab".into(), None, None, None, None, Some(1001)).is_err());
        assert!(request("ab".into(), None, Some("x".repeat(4097)), None, None, None).is_err());
    }

    #[test]
    fn url_accepts_only_an_explicit_service_base() {
        assert_eq!(configured_base(None).unwrap_err().code, "not_configured");
        assert_eq!(
            configured_base(Some(" ".into())).unwrap_err().code,
            "not_configured"
        );
        assert_eq!(
            search_url_with_dev("http://100.91.190.107:8790", false)
                .unwrap()
                .as_str(),
            "http://100.91.190.107:8790/api/intake_search_names"
        );
        assert_eq!(
            search_url_with_dev("https://intake.example.ts.net/", false)
                .unwrap()
                .as_str(),
            "https://intake.example.ts.net/api/intake_search_names"
        );
        assert!(search_url_with_dev("http://[fd7a:115c:a1e0::1]:8790", false).is_ok());
        assert!(search_url_with_dev("http://100.127.255.254:8790", false).is_ok());
        assert!(search_url_with_dev("http://100.128.0.1:8790", false).is_err());
        assert!(search_url_with_dev("http://intake.example.ts.net", false).is_err());
        assert!(search_url_with_dev("https://intake.example.ts.net.evil", false).is_err());
        assert!(search_url_with_dev("http://192.168.1.1", false).is_err());
        assert!(search_url_with_dev("http://127.0.0.1:8790", false).is_err());
        assert!(search_url_with_dev("http://127.0.0.1:8790", true).is_ok());
        for base in [
            "file:///data",
            "http://u:p@host",
            "http://host?token=x",
            "http://host#part",
            "http://100.91.190.107:8790/alternate",
        ] {
            assert!(search_url_with_dev(base, false).is_err());
        }
    }

    #[test]
    fn rejects_unavailable_or_mismatched_result_and_preserves_provenance() {
        let request = request_for_test();
        let valid = fixture();
        let checked = checked_response(&serde_json::to_vec(&valid).unwrap(), &request).unwrap();
        assert_eq!(checked["hits"][0]["now"], "/mnt/b2/vault/Takeout.zip");
        assert_eq!(checked["hits"][0]["was"], "catalog://old/Takeout.zip");
        assert_eq!(checked["hits"][0]["source"], "catalog");
        let mut wrong = valid.clone();
        wrong["query"] = json!("different");
        assert!(checked_response(&serde_json::to_vec(&wrong).unwrap(), &request).is_err());
        wrong = valid.clone();
        wrong["scan_cap"] = json!(99);
        assert!(checked_response(&serde_json::to_vec(&wrong).unwrap(), &request).is_err());
        for field in [
            "select",
            "now",
            "was",
            "member_of",
            "size",
            "modified",
            "flag",
        ] {
            wrong = valid.clone();
            wrong["hits"][0].as_object_mut().unwrap().remove(field);
            assert!(checked_response(&serde_json::to_vec(&wrong).unwrap(), &request).is_err());
        }
    }

    #[tokio::test]
    async fn returns_a_real_bounded_http_result() {
        use std::io::{Read, Write};
        let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
        let address = listener.local_addr().unwrap();
        let server = std::thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            stream
                .set_read_timeout(Some(Duration::from_secs(5)))
                .unwrap();
            let mut request_bytes = [0u8; 8192];
            let size = stream.read(&mut request_bytes).unwrap();
            let head = String::from_utf8_lossy(&request_bytes[..size]);
            assert!(head.starts_with("POST /api/intake_search_names HTTP/1.1"));
            let body = serde_json::to_string(&fixture()).unwrap();
            write!(stream, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}", body.len(), body).unwrap();
        });
        let result = search_at(&format!("http://{address}"), &request_for_test())
            .await
            .unwrap();
        assert_eq!(result["hits"][0]["name"], "Takeout.zip");
        server.join().unwrap();
    }

    #[tokio::test]
    async fn unavailable_catalog_fails_closed_without_leaking_backend_body() {
        use std::io::{Read, Write};
        let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
        let address = listener.local_addr().unwrap();
        let server = std::thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            let mut request_bytes = [0u8; 1024];
            let _ = stream.read(&mut request_bytes).unwrap();
            write!(stream, "HTTP/1.1 503 Service Unavailable\r\nContent-Length: 26\r\nConnection: close\r\n\r\nsecret backend detail here").unwrap();
        });
        let error = search_at(&format!("http://{address}"), &request_for_test())
            .await
            .unwrap_err();
        assert_eq!(error.code, "not_connected");
        assert_eq!(error.status, Some(503));
        assert!(!error.message.contains("secret"));
        server.join().unwrap();
    }
}
