//! Byline: Claude Code · Opus 5 · 2026-09-18
//!
//! "Search this folder live": a bounded text scan of ONE storage folder, read from B2 through the
//! rclone mount. It is an explicit action, never the default search (owner 2026-09-18 19:56 EDT):
//! current folder (and its subfolders) only, at most 2,000 files and 200 MB read, files over 20 MB
//! skipped, never on catalog:// and never on the storage root or a bucket root. The donor's
//! `grep_search` (used by several panels) is routed here too, so no caller can walk the whole mount.

use crate::Engine;
use serde_json::{json, Map, Value};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{Duration, Instant};

pub type Outcome = Result<Value, (u16, String)>;

pub const MAX_FILES: u64 = 2_000;
pub const MAX_BYTES: u64 = 200 * 1024 * 1024;
pub const MAX_FILE_BYTES: u64 = 20 * 1024 * 1024;
/// Directory entries listed before the walk stops (listing a B2 folder is remote calls too).
const MAX_ENTRIES: u64 = 20_000;
const MAX_WALL: Duration = Duration::from_secs(180);
pub const PROGRESS_EVENT: &str = "intake-live-search-progress";

fn s(args: &Map<String, Value>, camel: &str, snake: &str) -> Option<String> {
    args.get(camel)
        .or_else(|| args.get(snake))
        .and_then(|v| v.as_str())
        .map(|v| v.to_string())
}

/// The folder must be a real storage folder at least three levels below the mount root
/// (e.g. `b2/salem-data/<folder>`), so a live scan never spans a whole bucket or the mount.
fn check_root(engine: &Engine, raw: &str) -> Result<PathBuf, (u16, String)> {
    if crate::catalog::is_catalog(raw) {
        return Err((
            422,
            "Live folder search reads real files from storage; catalog:// is a virtual tree built from the catalog. \
             Use the chats index search, or open the folder under B2 salem-data and search there."
                .into(),
        ));
    }
    let p = PathBuf::from(raw);
    let rel = p.strip_prefix(&engine.mount_root).map_err(|_| {
        (403, format!("{raw} is outside the storage root {}", engine.mount_root.display()))
    })?;
    if rel.components().count() < 3 {
        return Err((
            422,
            format!(
                "{raw} is too broad for a live search (it would read a whole bucket or the storage mount). \
                 Open a specific folder first."
            ),
        ));
    }
    Ok(p)
}

/// Media and archive extensions: a text search can never match inside them, and on this mount every
/// read is a B2 download, so they are skipped without being fetched.
const BINARY_EXT: &[&str] = &[
    "mp3", "m4a", "wav", "aac", "ogg", "opus", "flac", "amr", "mp4", "mkv", "mov", "avi", "webm", "wmv", "3gp",
    "jpg", "jpeg", "png", "gif", "bmp", "tiff", "heic", "webp", "ico", "psd", "zip", "gz", "tgz", "bz2", "xz",
    "7z", "rar", "tar", "pdf", "docx", "xlsx", "pptx", "odt", "ods", "exe", "dll", "so", "dylib", "db",
    "sqlite", "sqlite3", "bin", "iso", "dmg", "apk", "jar", "class", "woff", "woff2", "ttf", "otf",
];

fn is_binary_ext(path: &Path) -> bool {
    path.extension()
        .and_then(|e| e.to_str())
        .map(|e| e.to_ascii_lowercase())
        .is_some_and(|e| BINARY_EXT.contains(&e.as_str()))
}

/// Read a file as text, sniffing the first 8 KB: a binary file costs 8 KB of B2 traffic, not its
/// whole size. Returns the text and the bytes actually downloaded.
fn read_text(path: &Path, remaining: u64) -> (Option<String>, u64) {
    use std::io::Read;
    let Ok(mut f) = std::fs::File::open(path) else { return (None, 0) };
    let mut head = vec![0u8; 8192];
    let n = match f.read(&mut head) {
        Ok(n) => n,
        Err(_) => return (None, 0),
    };
    head.truncate(n);
    let read = n as u64;
    if head.contains(&0) {
        // A NUL byte in the first 8 KB: binary, and the rest is never downloaded.
        return (None, read);
    }
    if n < 8192 {
        return (String::from_utf8(head).ok(), read);
    }
    let mut rest = Vec::new();
    if f.take(remaining.saturating_sub(read)).read_to_end(&mut rest).is_err() {
        return (None, read);
    }
    let total = read + rest.len() as u64;
    head.extend_from_slice(&rest);
    (String::from_utf8(head).ok(), total)
}

struct Scan {
    matches: Vec<Value>,
    files_read: u64,
    bytes_read: u64,
    skipped_large: u64,
    skipped_binary: u64,
    entries: u64,
    stopped: Option<String>,
}

fn scan(root: &Path, query: &str, max_results: usize, mut progress: impl FnMut(&Scan, &str, bool)) -> Scan {
    let needle = query.to_lowercase();
    let started = Instant::now();
    let mut last = Instant::now();
    let mut st = Scan {
        matches: Vec::new(),
        files_read: 0,
        bytes_read: 0,
        skipped_large: 0,
        skipped_binary: 0,
        entries: 0,
        stopped: None,
    };
    for entry in walkdir::WalkDir::new(root).follow_links(false).sort_by_file_name() {
        st.entries += 1;
        if st.entries > MAX_ENTRIES {
            st.stopped = Some(format!("listed {MAX_ENTRIES} entries in this folder tree; stopped before reading more"));
            break;
        }
        if started.elapsed() > MAX_WALL {
            st.stopped = Some(format!("time limit of {} s reached", MAX_WALL.as_secs()));
            break;
        }
        let Ok(entry) = entry else { continue };
        if !entry.file_type().is_file() {
            continue;
        }
        let Ok(meta) = entry.metadata() else { continue };
        let size = meta.len();
        if size > MAX_FILE_BYTES {
            st.skipped_large += 1;
            continue;
        }
        if is_binary_ext(entry.path()) {
            st.skipped_binary += 1;
            continue;
        }
        if st.files_read >= MAX_FILES {
            st.stopped = Some(format!("file cap reached: {MAX_FILES} files read"));
            break;
        }
        if st.bytes_read + size > MAX_BYTES {
            st.stopped = Some(format!("size cap reached: {} MB read", MAX_BYTES / 1_048_576));
            break;
        }
        let current = entry.path().display().to_string();
        if last.elapsed() > Duration::from_millis(250) {
            progress(&st, &current, false);
            last = Instant::now();
        }
        let (text, downloaded) = read_text(entry.path(), MAX_BYTES - st.bytes_read);
        st.bytes_read += downloaded;
        let Some(text) = text else {
            st.skipped_binary += 1;
            continue;
        };
        st.files_read += 1;
        if let Some((i, line)) = text.lines().enumerate().find(|(_, l)| l.to_lowercase().contains(&needle)) {
            let content = if line.len() > 500 {
                let mut end = 500;
                while !line.is_char_boundary(end) {
                    end -= 1;
                }
                format!("{}...", &line[..end])
            } else {
                line.to_string()
            };
            st.matches.push(json!({
                "file": current,
                "line": i + 1,
                "content": content,
                "filename": entry.file_name().to_string_lossy(),
            }));
            if st.matches.len() >= max_results {
                st.stopped = Some(format!("result limit reached: {max_results} matching files"));
                break;
            }
        }
    }
    progress(&st, "", true);
    st
}

async fn run(engine: &Arc<Engine>, root_raw: &str, query: &str, max_results: usize, search_id: Option<String>) -> Result<(PathBuf, Scan, u128), (u16, String)> {
    let query = query.trim().to_string();
    if query.is_empty() {
        return Err((422, "query is required".into()));
    }
    let root = check_root(engine, root_raw)?;
    match tokio::fs::metadata(&root).await {
        Ok(m) if m.is_dir() => {}
        Ok(_) => return Err((422, format!("{root_raw} is not a folder"))),
        Err(e) => return Err((404, format!("{root_raw}: {e}"))),
    }
    let app = engine.app.clone();
    let started = Instant::now();
    let root2 = root.clone();
    let root_s = root.display().to_string();
    let st = tokio::task::spawn_blocking(move || {
        scan(&root2, &query, max_results, |st, current, done| {
            let payload = json!({
                "searchId": search_id, "root": root_s, "current": current, "done": done,
                "files_read": st.files_read, "bytes_read": st.bytes_read,
                "skipped_large": st.skipped_large, "matches": st.matches.len(),
                "max_files": MAX_FILES, "max_bytes": MAX_BYTES,
            });
            app.emit_raw(PROGRESS_EVENT, payload.to_string());
        })
    })
    .await
    .map_err(|e| (500, format!("live search failed: {e}")))?;
    Ok((root, st, started.elapsed().as_millis()))
}

/// `intake_live_folder_search { path, query, maxResults?, searchId? }`.
pub async fn live_folder_search(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let path = s(args, "path", "path").ok_or_else(|| (422, "path is required".to_string()))?;
    let query = s(args, "query", "query").unwrap_or_default();
    let max_results = args.get("maxResults").and_then(|v| v.as_u64()).unwrap_or(200).clamp(1, 1000) as usize;
    let search_id = s(args, "searchId", "search_id");
    let (root, st, ms) = run(engine, &path, &query, max_results, search_id).await?;
    tracing::info!(root = %root.display(), files = st.files_read, bytes = st.bytes_read, ms, "live folder search");
    Ok(json!({
        "root": root,
        "query": query.trim(),
        "matches": st.matches,
        "files_read": st.files_read,
        "bytes_read": st.bytes_read,
        "skipped_large": st.skipped_large,
        "skipped_binary": st.skipped_binary,
        "entries_listed": st.entries,
        "stopped": st.stopped,
        "caps": {"max_files": MAX_FILES, "max_bytes": MAX_BYTES, "max_file_bytes": MAX_FILE_BYTES},
        "elapsed_ms": ms,
    }))
}

/// The donor's `grep_search { query, searchPath, maxResults }`, answered by the capped scan.
pub async fn grep_search(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let path = s(args, "searchPath", "search_path").ok_or_else(|| (422, "searchPath is required".to_string()))?;
    let query = s(args, "query", "query").unwrap_or_default();
    let max_results = args.get("maxResults").and_then(|v| v.as_u64()).unwrap_or(200).clamp(1, 1000) as usize;
    let (_, st, _) = run(engine, &path, &query, max_results, None).await?;
    Ok(Value::Array(st.matches))
}
