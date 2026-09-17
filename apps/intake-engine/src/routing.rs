//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! Engine-native commands and `catalog://` routing. A command whose path arguments are
//! `catalog://…` is answered here (listing/stat/writes from the catalog, reads by handing the
//! resolved B2 path to the donor handler); everything else goes to the donor handler untouched.

use crate::catalog::{self, is_catalog, CatalogEntry};
use crate::Engine;
use serde_json::{json, Map, Value};
use std::path::{Path, PathBuf};
use std::sync::Arc;

pub type Outcome = Result<Value, (u16, String)>;

fn s(args: &Map<String, Value>, camel: &str, snake: &str) -> Option<String> {
    args.get(camel)
        .or_else(|| args.get(snake))
        .and_then(|v| v.as_str())
        .map(|v| v.to_string())
}

fn bad(msg: impl Into<String>) -> (u16, String) {
    (422, msg.into())
}

fn to_value<T: serde::Serialize>(v: T) -> Outcome {
    serde_json::to_value(v).map_err(|e| (500, e.to_string()))
}

async fn donor(engine: &Arc<Engine>, name: &str, args: Map<String, Value>) -> Outcome {
    let f = engine
        .commands
        .get(name)
        .copied()
        .ok_or_else(|| (501, format!("{name} has no donor handler")))?;
    f(tauri::CommandCtx { app: engine.app.clone(), args })
        .await
        .map_err(|e| (422, e))
}

fn catalog(engine: &Engine) -> Result<&catalog::Catalog, (u16, String)> {
    engine.catalog.as_deref().ok_or_else(|| {
        (
            503,
            format!(
                "catalog:// is unavailable: {}",
                engine.catalog_error.as_deref().unwrap_or("catalog not connected")
            ),
        )
    })
}

/// Engine-native commands (not in the donor).
pub async fn engine_command(engine: &Arc<Engine>, name: &str, args: &Map<String, Value>) -> Option<Outcome> {
    Some(match name {
        "intake_engine_info" => Ok(json!({
            "mount_root": engine.mount_root,
            "b2_root": engine.catalog.as_ref().map(|c| c.b2_root.clone()),
            "catalog": engine.catalog.is_some(),
            "catalog_error": engine.catalog_error,
            "commands": engine.commands.len(),
        })),
        "intake_file_metadata" => {
            let Some(path) = s(args, "path", "path") else {
                return Some(Err(bad("path is required")));
            };
            file_metadata(engine, &path).await
        }
        // Volumes a browser sees: the storage mount, the B2 bucket and catalog://, never the container root.
        "list_drives" => {
            let b2 = engine
                .catalog
                .as_ref()
                .map(|c| c.b2_root.clone())
                .unwrap_or_else(|| engine.mount_root.join("b2/salem-data"));
            Ok(json!([
                {"letter": "", "label": "B2 salem-data", "path": b2, "total_space": 0, "free_space": 0},
                {"letter": "", "label": "Storage (OpenList)", "path": engine.mount_root, "total_space": 0, "free_space": 0},
                {"letter": "", "label": "Catalog", "path": "catalog://", "total_space": 0, "free_space": 0},
            ]))
        }
        "intake_catalog_lookup" => {
            let Some(path) = s(args, "path", "path") else {
                return Some(Err(bad("path is required")));
            };
            catalog_lookup(engine, &path).await
        }
        _ => return None,
    })
}

async fn real_path(engine: &Arc<Engine>, path: &str) -> Result<PathBuf, (u16, String)> {
    if is_catalog(path) {
        let cat = catalog(engine)?;
        cat.resolve_file(path).await.map(|(p, _)| p).map_err(bad)
    } else {
        Ok(PathBuf::from(path))
    }
}

async fn file_metadata(engine: &Arc<Engine>, path: &str) -> Outcome {
    if is_catalog(path) {
        let cat = catalog(engine)?;
        let entry = cat.stat(path).await.map_err(bad)?.ok_or_else(|| bad(format!("{path} does not exist")))?;
        if entry.is_dir {
            let (bytes, files, dirs) = cat.dir_totals(path).await.map_err(bad)?;
            return Ok(json!({ "catalog_entry": entry, "catalog_totals": {"bytes": bytes, "files": files, "dirs": dirs} }));
        }
    }
    let real = real_path(engine, path).await?;
    let hint = is_catalog(path).then(|| catalog::name_of(path.trim_end_matches('/')));
    let mut v = crate::media::file_metadata(&real, hint.as_deref()).await;
    if let Value::Object(m) = &mut v {
        m.insert("resolved_path".into(), json!(real));
    }
    Ok(v)
}

async fn catalog_lookup(engine: &Arc<Engine>, path: &str) -> Outcome {
    let cat = catalog(engine)?;
    let real = real_path(engine, path).await?;
    if !real.starts_with(&cat.b2_root) {
        return Ok(json!({
            "b2_key": null,
            "occurrences": [],
            "count": 0,
            "note": format!("only files under {} are in the catalog", cat.b2_root.display()),
        }));
    }
    cat.lookup_b2_path(&real).await.map_err(bad)
}

/// True when any path-like argument is a catalog:// path.
pub fn touches_catalog(args: &Map<String, Value>) -> bool {
    args.values().any(|v| match v {
        Value::String(s) => is_catalog(s),
        Value::Array(a) => a.iter().any(|x| x.as_str().is_some_and(is_catalog)),
        _ => false,
    })
}

fn file_props(entry: &CatalogEntry) -> Value {
    json!({
        "name": entry.name, "path": entry.path, "size": entry.size, "is_dir": entry.is_dir,
        "created": entry.modified, "modified": entry.modified, "accessed": entry.modified,
        "readonly": false, "hidden": entry.name.starts_with('.'),
        "file_type": entry.file_type, "mime_type": entry.mime_type,
    })
}

async fn stat_required(cat: &catalog::Catalog, path: &str) -> Result<CatalogEntry, (u16, String)> {
    cat.stat(path)
        .await
        .map_err(bad)?
        .ok_or_else(|| (404, format!("{path} does not exist")))
}

/// Replace the path argument with the resolved B2 path and call the donor handler.
async fn donor_on_resolved(engine: &Arc<Engine>, name: &str, mut args: Map<String, Value>, keys: &[&str]) -> Outcome {
    for k in keys {
        if let Some(Value::String(p)) = args.get(*k).cloned() {
            if is_catalog(&p) {
                let real = real_path(engine, &p).await?;
                args.insert((*k).to_string(), json!(real));
            }
        }
    }
    donor(engine, name, args).await
}

async fn transfer(engine: &Arc<Engine>, source: &str, dest: &str, is_move: bool) -> Outcome {
    let cat = catalog(engine)?;
    match (is_catalog(source), is_catalog(dest)) {
        (true, true) if is_move => cat.move_within(source, dest).await.map(|_| Value::Null).map_err(bad),
        (true, true) => Err(bad(
            "catalog:// records where files were found; copying inside it would invent a new occurrence. Copy to a B2 folder instead.",
        )),
        (true, false) if is_move => cat.relocate_out(source, Path::new(dest)).await.map(|_| Value::Null).map_err(bad),
        (true, false) => cat.copy_out(source, Path::new(dest)).await.map(|_| Value::Null).map_err(bad),
        (false, true) => cat.import_in(Path::new(source), dest, is_move).await.map(|_| Value::Null).map_err(bad),
        (false, false) => unreachable!(),
    }
}

/// Handle a donor command whose arguments include a catalog:// path.
pub async fn catalog_command(engine: &Arc<Engine>, name: &str, args: Map<String, Value>) -> Outcome {
    let cat = catalog(engine)?;
    let path = s(&args, "path", "path")
        .or_else(|| s(&args, "dirPath", "dir_path"))
        .or_else(|| s(&args, "filePath", "file_path"))
        .or_else(|| s(&args, "folderPath", "folder_path"))
        .or_else(|| s(&args, "archivePath", "archive_path"))
        .unwrap_or_default();
    match name {
        "read_directory" | "get_files_in_directory" => to_value(cat.list(&path).await.map_err(bad)?),
        "file_exist" => to_value(cat.stat(&path).await.map_err(bad)?.is_some()),
        "is_dir" => to_value(stat_required(cat, &path).await?.is_dir),
        "get_file_properties" | "get_file_meta_data" => {
            let entry = stat_required(cat, &path).await?;
            Ok(file_props(&entry))
        }
        "get_detailed_file_properties" => {
            let entry = stat_required(cat, &path).await?;
            if entry.is_dir {
                let (bytes, _, _) = cat.dir_totals(&path).await.map_err(bad)?;
                return Ok(json!({
                    "path": entry.path, "name": entry.name, "file_type": "Folder", "size": bytes,
                    "size_formatted": format!("{bytes} bytes"), "created": 0, "modified": 0, "accessed": 0,
                    "created_formatted": "", "modified_formatted": "", "accessed_formatted": "",
                    "permissions": {"readable": true, "writable": true, "executable": false, "mode": null},
                    "is_directory": true, "is_hidden": false, "is_readonly": false, "extension": null, "mime_type": null,
                    "attributes": {"archive": false, "compressed": false, "encrypted": false, "hidden": false, "system": false, "temporary": false}
                }));
            }
            let mut v = donor_on_resolved(engine, name, args, &["filePath", "file_path"]).await?;
            if let Value::Object(m) = &mut v {
                m.insert("path".into(), json!(entry.path));
                m.insert("name".into(), json!(entry.name));
            }
            Ok(v)
        }
        "read_text_file" | "read_binary_file" | "extract_document_text" | "compute_file_hash" | "get_image_info"
        | "is_archive" | "get_archive_info" | "list_sqlite_tables" | "get_sqlite_table_columns"
        | "query_sqlite_table" => {
            donor_on_resolved(engine, name, args, &["path", "filePath", "file_path", "archivePath", "archive_path", "dbPath", "db_path"]).await
        }
        "calculate_folder_size" | "get_dir_size" | "get_directory_size" | "get_directory_item_count" => {
            let (bytes, files, dirs) = cat.dir_totals(&path).await.map_err(bad)?;
            match name {
                "calculate_folder_size" => Ok(json!({"total_size": bytes, "file_count": files, "dir_count": dirs, "is_cached": true, "cache_timestamp": 0})),
                "get_dir_size" => Ok(json!({"total_size": bytes, "file_count": files, "dir_count": dirs})),
                "get_directory_size" => Ok(json!(bytes)),
                _ => Ok(json!(files + dirs)),
            }
        }
        "rename" => {
            let old = s(&args, "oldPath", "old_path").ok_or_else(|| bad("oldPath is required"))?;
            let new = s(&args, "newPath", "new_path").ok_or_else(|| bad("newPath is required"))?;
            transfer(engine, &old, &new, true).await
        }
        "move_file" | "move_with_progress" | "copy" | "copy_with_progress" | "accelerated_copy_file"
        | "accelerated_copy_directory" => {
            let src = s(&args, "source", "source").ok_or_else(|| bad("source is required"))?;
            let dst = s(&args, "destination", "destination").ok_or_else(|| bad("destination is required"))?;
            let is_move = name.starts_with("move");
            transfer(engine, &src, &dst, is_move).await?;
            if name.ends_with("_progress") || name.starts_with("accelerated") {
                Ok(json!(format!("catalog-{}", chrono::Utc::now().timestamp_millis())))
            } else {
                Ok(Value::Null)
            }
        }
        "remove_file" | "remove_dir" | "move_to_trash" | "secure_delete" => {
            cat.delete(&path).await.map_err(bad)?;
            Ok(Value::Null)
        }
        "create_dir_recursive" => {
            cat.mkdir(&path).await.map_err(bad)?;
            Ok(Value::Null)
        }
        "get_rename_destination" => {
            let dir = s(&args, "destinationDir", "destination_dir").ok_or_else(|| bad("destinationDir is required"))?;
            let file = s(&args, "fileName", "file_name").ok_or_else(|| bad("fileName is required"))?;
            to_value(cat.unique_name(&dir, &file).await.map_err(bad)?)
        }
        "check_conflicts" => {
            let dir = s(&args, "destinationDir", "destination_dir").ok_or_else(|| bad("destinationDir is required"))?;
            let sources: Vec<String> = args
                .get("sources")
                .and_then(|v| serde_json::from_value(v.clone()).ok())
                .unwrap_or_default();
            let mut conflicts = Vec::new();
            let existing: Vec<CatalogEntry> = if is_catalog(&dir) {
                cat.list(&dir).await.map_err(bad)?
            } else {
                Vec::new()
            };
            for src in sources {
                let src_name = catalog::name_of(src.trim_end_matches('/'));
                let src_entry = if is_catalog(&src) { cat.stat(&src).await.map_err(bad)? } else { None };
                let dest_entry: Option<Value> = if is_catalog(&dir) {
                    existing.iter().find(|e| e.name == src_name).map(|e| json!({"path": e.path, "name": e.name, "is_dir": e.is_dir, "size": e.size, "modified": e.modified}))
                } else {
                    let target = Path::new(&dir).join(&src_name);
                    tokio::fs::metadata(&target).await.ok().map(|m| json!({
                        "path": target, "name": src_name, "is_dir": m.is_dir(), "size": m.len(),
                        "modified": m.modified().ok().and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok()).map(|d| d.as_secs()).unwrap_or(0),
                    }))
                };
                if let Some(dest) = dest_entry {
                    let source = match src_entry {
                        Some(e) => json!({"path": e.path, "name": e.name, "is_dir": e.is_dir, "size": e.size, "modified": e.modified}),
                        None => json!({"path": src, "name": src_name, "is_dir": false, "size": 0, "modified": 0}),
                    };
                    conflicts.push(json!({"source": source, "destination": dest}));
                }
            }
            Ok(Value::Array(conflicts))
        }
        // Git and folder-size caches have nothing to report for catalog entries (not a git work tree,
        // sizes come from the listing itself); tags are keyed by real paths.
        "get_cached_folder_sizes" | "get_file_tags_batch" => Ok(json!({})),
        "find_git_repository" | "get_ai_index_entry" | "trigger_ai_indexing" => Ok(Value::Null),
        "get_git_status" => Ok(json!([])),
        // Harmless UI side effects on a virtual tree.
        "add_to_recent_folders" | "watch_directory" | "unwatch_directory" | "start_watching" | "stop_watching"
        | "set_search_context" | "add_whitelisted_path" | "index_directory" | "add_recent_file" => donor_passthrough_or_null(engine, name, args).await,
        _ => Err((
            501,
            format!(
                "{name} is not available on catalog:// paths (catalog:// supports list, open/preview, metadata, copy/move/rename/delete and new folder)"
            ),
        )),
    }
}

async fn donor_passthrough_or_null(engine: &Arc<Engine>, name: &str, args: Map<String, Value>) -> Outcome {
    if name == "add_to_recent_folders" {
        return donor(engine, name, args).await;
    }
    Ok(Value::Null)
}

/// Argument keys that carry filesystem paths from the browser.
fn is_path_key(k: &str) -> bool {
    let k = k.to_ascii_lowercase();
    ["path", "source", "destination", "dir", "folder", "target", "cwd", "root", "file"]
        .iter()
        .any(|w| k.contains(w))
}

/// Confine every absolute path a browser sends to the storage mount (or catalog://). The engine's
/// own state volume and the container filesystem are not browsable (owner decision 3/6, 2026-09-17).
pub fn confine_args(engine: &Engine, args: &Map<String, Value>) -> Result<(), String> {
    fn check(engine: &Engine, key: &str, v: &str) -> Result<(), String> {
        if !v.starts_with('/') {
            // Bare names ("report.pdf") and catalog:// are fine; a relative or foreign path would
            // resolve against the engine's working directory, outside the storage root.
            if v.contains('/') && !crate::catalog::is_catalog(v) {
                return Err(format!("{key}: {v} must be an absolute path under the storage root or a catalog:// path"));
            }
            return Ok(());
        }
        let p = Path::new(v);
        if p.components().any(|c| matches!(c, std::path::Component::ParentDir)) {
            return Err(format!("{key}: '..' is not allowed in paths"));
        }
        if p.starts_with(&engine.mount_root) {
            Ok(())
        } else {
            Err(format!(
                "{key}: {v} is outside the storage root {} (the hosted engine only works on B2 storage and catalog://)",
                engine.mount_root.display()
            ))
        }
    }
    for (k, v) in args {
        if !is_path_key(k) {
            continue;
        }
        match v {
            Value::String(s) => check(engine, k, s)?,
            Value::Array(a) => {
                for x in a {
                    if let Some(s) = x.as_str() {
                        check(engine, k, s)?;
                    }
                }
            }
            _ => {}
        }
    }
    Ok(())
}
