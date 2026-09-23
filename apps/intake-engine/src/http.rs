//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! Web-mode HTTP contract: `POST /api/{command}`, `GET /api/asset?path=`,
//! `GET /api/events/{event}` (SSE).

use crate::Engine;
use axum::body::{Body, Bytes};
use axum::extract::{Path, Query, State};
use axum::http::{header, HeaderMap, HeaderValue, StatusCode};
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::Router;
use serde::Deserialize;
use serde_json::{Map, Value};
use std::convert::Infallible;
use std::path::{Component, PathBuf};
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncReadExt, AsyncSeekExt};
use tokio::task::JoinHandle;
use tokio_stream::StreamExt;

// Byline: Codex · GPT-6 · 2026-09-23. Dropping an Axum response future
// normally detaches its spawned command task. Hosted folder analysis is
// read-only and must instead stop on client disconnect or deadline.
const ANALYZE_COMMAND_TIMEOUT: Duration = Duration::from_secs(60);

struct CommandTask<T> {
    handle: JoinHandle<T>,
    cancel_on_drop: bool,
}

impl<T> Drop for CommandTask<T> {
    fn drop(&mut self) {
        if self.cancel_on_drop {
            self.handle.abort();
        }
    }
}

pub fn router(engine: Arc<Engine>) -> Router {
    Router::new()
        .route("/healthz", get(healthz))
        .route("/api/asset", get(asset))
        .route("/api/events/{event}", get(events))
        .route("/api/{command}", post(command))
        .layer(axum::extract::DefaultBodyLimit::max(256 * 1024 * 1024))
        .with_state(engine)
}

async fn healthz(State(engine): State<Arc<Engine>>) -> impl IntoResponse {
    let mount_ok = tokio::fs::metadata(&engine.mount_root).await.is_ok();
    let body = serde_json::json!({
        "ok": mount_ok,
        "commands": engine.commands.len(),
        "mount_root": engine.mount_root,
        "mount_readable": mount_ok,
    });
    (
        if mount_ok { StatusCode::OK } else { StatusCode::SERVICE_UNAVAILABLE },
        axum::Json(body),
    )
}

fn text(status: StatusCode, msg: impl Into<String>) -> Response {
    (
        status,
        [(header::CONTENT_TYPE, "text/plain; charset=utf-8")],
        msg.into(),
    )
        .into_response()
}

async fn command(
    State(engine): State<Arc<Engine>>,
    Path(name): Path<String>,
    body: Bytes,
) -> Response {
    if let Some(reason) = engine.browser_unavailable.get(name.as_str()) {
        return text(
            StatusCode::NOT_IMPLEMENTED,
            format!("{name} is not available in the hosted engine: {reason}"),
        );
    }
    let f = engine.commands.get(name.as_str()).copied();
    let args: Map<String, Value> = if body.iter().all(|b| b.is_ascii_whitespace()) {
        Map::new()
    } else {
        match serde_json::from_slice::<Value>(&body) {
            Ok(Value::Object(m)) => m,
            Ok(Value::Null) => Map::new(),
            Ok(_) => return text(StatusCode::BAD_REQUEST, "command arguments must be a JSON object"),
            Err(e) => return text(StatusCode::BAD_REQUEST, format!("invalid JSON body: {e}")),
        }
    };
    let started = std::time::Instant::now();
    let eng = engine.clone();
    let cmd = name.clone();
    let bounded_analysis = name == "analyze_directory";
    let mut task = CommandTask { handle: tokio::spawn(async move {
        crate::routing::confine_args(&eng, &args).map_err(|e| (403, e))?;
        if let Some(out) = crate::routing::engine_command(&eng, &cmd, &args).await {
            return out;
        }
        // A terminal opened while the pane shows catalog:// (a virtual tree) starts at the storage
        // root; terminal commands never go to the catalog handler (2026-09-18).
        let mut args = args;
        if cmd.starts_with("pty_") {
            if let Some(Value::String(cwd)) = args.get("cwd") {
                if crate::catalog::is_catalog(cwd) {
                    args.insert("cwd".into(), Value::String(eng.mount_root.display().to_string()));
                }
            }
        } else if crate::routing::touches_catalog(&args) {
            return crate::routing::catalog_command(&eng, &cmd, args).await;
        }
        let Some(f) = f else {
            return Err((501, format!("{cmd} is not a command of the hosted engine (no donor handler exists for it)")));
        };
        f(tauri::CommandCtx { app: eng.app.clone(), args }).await.map_err(|e| (422, e))
    }), cancel_on_drop: bounded_analysis };
    let joined = if bounded_analysis {
        match tokio::time::timeout(ANALYZE_COMMAND_TIMEOUT, &mut task.handle).await {
            Ok(result) => result,
            Err(_) => return text(StatusCode::GATEWAY_TIMEOUT, "folder analysis exceeded 60 seconds"),
        }
    } else {
        (&mut task.handle).await
    };
    let elapsed = started.elapsed().as_millis();
    match joined {
        Ok(Ok(value)) => {
            tracing::debug!(command = %name, ms = elapsed, "ok");
            let bytes = serde_json::to_vec(&value).unwrap_or_else(|_| b"null".to_vec());
            (
                StatusCode::OK,
                [(header::CONTENT_TYPE, "application/json")],
                bytes,
            )
                .into_response()
        }
        Ok(Err((code, err))) => {
            tracing::info!(command = %name, ms = elapsed, error = %err, "command error");
            text(StatusCode::from_u16(code).unwrap_or(StatusCode::UNPROCESSABLE_ENTITY), err)
        }
        Err(join) => {
            tracing::error!(command = %name, "command panicked: {join}");
            text(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("{name} failed inside the engine: {join}"),
            )
        }
    }
}

#[cfg(test)]
mod analysis_cancellation_tests {
    use super::*;
    use std::future::pending;
    use tokio::sync::{oneshot, Semaphore};

    async fn occupied_task() -> (CommandTask<()>, Arc<Semaphore>) {
        let capacity = Arc::new(Semaphore::new(1));
        let worker_capacity = capacity.clone();
        let (acquired, ready) = oneshot::channel();
        let task = CommandTask {
            handle: tokio::spawn(async move {
                let _permit = worker_capacity.acquire().await.unwrap();
                acquired.send(()).unwrap();
                pending::<()>().await;
            }),
            cancel_on_drop: true,
        };
        ready.await.unwrap();
        (task, capacity)
    }

    #[tokio::test]
    async fn client_disconnect_releases_analysis_capacity() {
        let (task, capacity) = occupied_task().await;
        drop(task);
        let _ = tokio::time::timeout(Duration::from_secs(1), capacity.acquire())
            .await
            .expect("canceled task must release its permit")
            .unwrap();
    }

    #[tokio::test]
    async fn command_timeout_releases_analysis_capacity() {
        let (mut task, capacity) = occupied_task().await;
        assert!(tokio::time::timeout(Duration::from_millis(10), &mut task.handle)
            .await
            .is_err());
        drop(task);
        let _ = tokio::time::timeout(Duration::from_secs(1), capacity.acquire())
            .await
            .expect("timed-out task must release its permit")
            .unwrap();
    }
}

async fn events(
    State(engine): State<Arc<Engine>>,
    Path(event): Path<String>,
) -> Sse<impl tokio_stream::Stream<Item = Result<Event, Infallible>>> {
    let rx = engine.app.subscribe();
    let stream = tokio_stream::wrappers::BroadcastStream::new(rx).filter_map(move |msg| match msg {
        Ok(ev) if ev.name == event => Some(Ok(Event::default().data(ev.payload))),
        _ => None,
    });
    Sse::new(stream).keep_alive(KeepAlive::default())
}

#[derive(Deserialize)]
struct AssetQuery {
    path: String,
}

/// Paths the asset route will stream: the storage mount only.
pub fn confined(engine: &Engine, raw: &str) -> Result<PathBuf, String> {
    let path = PathBuf::from(raw);
    if !path.is_absolute() || path.components().any(|c| matches!(c, Component::ParentDir)) {
        return Err("asset path must be absolute without '..'".into());
    }
    if !path.starts_with(&engine.mount_root) {
        return Err(format!(
            "asset path must be under the storage mount {}",
            engine.mount_root.display()
        ));
    }
    Ok(path)
}

fn mime_for(path: &std::path::Path) -> String {
    let ext = path
        .extension()
        .and_then(|e| e.to_str())
        .map(|e| e.to_ascii_lowercase())
        .unwrap_or_default();
    match ext.as_str() {
        // Guesses mime_guess gets wrong or leaves generic for previewable media.
        "mkv" => "video/x-matroska".into(),
        "mov" => "video/quicktime".into(),
        "m4a" => "audio/mp4".into(),
        "heic" => "image/heic".into(),
        "md" | "markdown" => "text/markdown; charset=utf-8".into(),
        "ts" | "tsx" | "rs" | "py" | "toml" | "yaml" | "yml" | "log" | "ini" | "cfg" => {
            "text/plain; charset=utf-8".into()
        }
        _ => {
            let m = mime_guess::from_path(path).first_or_octet_stream();
            if m.type_() == "text" {
                format!("{m}; charset=utf-8")
            } else {
                m.to_string()
            }
        }
    }
}

fn parse_range(value: &str, len: u64) -> Option<(u64, u64)> {
    let spec = value.strip_prefix("bytes=")?.split(',').next()?.trim();
    let (a, b) = spec.split_once('-')?;
    if len == 0 {
        return None;
    }
    if a.is_empty() {
        let suffix: u64 = b.parse().ok()?;
        let start = len.saturating_sub(suffix);
        return Some((start, len - 1));
    }
    let start: u64 = a.parse().ok()?;
    let end: u64 = if b.is_empty() { len - 1 } else { b.parse::<u64>().ok()?.min(len - 1) };
    (start <= end && start < len).then_some((start, end))
}

async fn asset(
    State(engine): State<Arc<Engine>>,
    Query(q): Query<AssetQuery>,
    headers: HeaderMap,
) -> Response {
    let path = if crate::catalog::is_catalog(&q.path) {
        let Some(cat) = engine.catalog.as_ref() else {
            return text(StatusCode::SERVICE_UNAVAILABLE, "catalog:// is unavailable");
        };
        match cat.resolve_file(&q.path).await {
            Ok((p, _)) => p,
            Err(e) => return text(StatusCode::NOT_FOUND, e),
        }
    } else {
        match confined(&engine, &q.path) {
            Ok(p) => p,
            Err(e) => return text(StatusCode::BAD_REQUEST, e),
        }
    };
    // A catalog entry is served with the type of the name it is listed under, not the vault key's.
    let hint = crate::catalog::is_catalog(&q.path).then(|| crate::catalog::name_of(q.path.trim_end_matches('/')));
    serve_file(path, &headers, hint.as_deref()).await
}

pub async fn serve_file(path: PathBuf, headers: &HeaderMap, name_hint: Option<&str>) -> Response {
    let meta = match tokio::fs::metadata(&path).await {
        Ok(m) if m.is_file() => m,
        Ok(_) => return text(StatusCode::BAD_REQUEST, "asset path is not a file"),
        Err(e) => return text(StatusCode::NOT_FOUND, format!("asset not found: {e}")),
    };
    let len = meta.len();
    let mut file = match tokio::fs::File::open(&path).await {
        Ok(f) => f,
        Err(e) => return text(StatusCode::FORBIDDEN, format!("cannot open asset: {e}")),
    };
    let mime = mime_for(name_hint.map(std::path::Path::new).unwrap_or(&path));
    let range = headers
        .get(header::RANGE)
        .and_then(|v| v.to_str().ok())
        .map(|v| parse_range(v, len));

    let mut builder = Response::builder()
        .header(header::CONTENT_TYPE, mime.as_str())
        .header(header::ACCEPT_RANGES, "bytes")
        .header(header::CACHE_CONTROL, "private, max-age=60")
        .header("X-Content-Type-Options", "nosniff");
    if mime.starts_with("text/html") || mime.starts_with("image/svg") {
        builder = builder.header(
            header::CONTENT_SECURITY_POLICY,
            HeaderValue::from_static("default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox"),
        );
    }

    match range {
        Some(Some((start, end))) => {
            if let Err(e) = file.seek(std::io::SeekFrom::Start(start)).await {
                return text(StatusCode::INTERNAL_SERVER_ERROR, format!("seek failed: {e}"));
            }
            let count = end - start + 1;
            let stream = tokio_util::io::ReaderStream::with_capacity(file.take(count), 256 * 1024);
            builder
                .status(StatusCode::PARTIAL_CONTENT)
                .header(header::CONTENT_LENGTH, count)
                .header(header::CONTENT_RANGE, format!("bytes {start}-{end}/{len}"))
                .body(Body::from_stream(stream))
                .unwrap_or_else(|e| text(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))
        }
        Some(None) => Response::builder()
            .status(StatusCode::RANGE_NOT_SATISFIABLE)
            .header(header::CONTENT_RANGE, format!("bytes */{len}"))
            .body(Body::empty())
            .unwrap_or_else(|e| text(StatusCode::INTERNAL_SERVER_ERROR, e.to_string())),
        None => {
            let stream = tokio_util::io::ReaderStream::with_capacity(file, 256 * 1024);
            builder
                .status(StatusCode::OK)
                .header(header::CONTENT_LENGTH, len)
                .body(Body::from_stream(stream))
                .unwrap_or_else(|e| text(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))
        }
    }
}
