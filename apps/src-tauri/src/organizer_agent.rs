//! Byline: Claude Code · Opus 5 · 2026-09-22
//!
//! Smart Suggestions from the model already configured for the engine
//! (`ai::chat_with_ai` → `ai_portkey.rs`, Gemini primary with the Kimi fallback).
//!
//! What the model is given for a folder: the listing (names, sizes, dates, types),
//! never file contents; what the catalog knows about those files (copies, hashes,
//! recorded original paths), gathered by the engine; and the owner's rules, loaded
//! from the tracked rules file named by `INTAKE_ORGANIZER_RULES_FILE`.
//!
//! What comes back is the existing `FolderSuggestion` shape, so `preview_organization`,
//! `execute_organization` and the panel are unchanged. One model call per analyze,
//! bounded listing, 45 s budget; past that the caller falls back to the fixed rules.
//!
//! A suggestion can only ever move *files* the caller listed into a folder under the
//! analyzed directory. Folders are never named as material, so no suggestion this
//! module produces can rename, merge or dissolve a folder, and the schema has no way
//! to express a delete.

use crate::ai::{chat_with_ai, ChatMessage};
use crate::file_organizer::{categorize_extension, FileInfo, FolderSuggestion};
use serde::Deserialize;
use serde_json::{json, Value};
use std::collections::{BTreeMap, HashMap, HashSet};
use std::path::Path;
use std::time::Duration;

/// Entries of the listing sent to the model. Bigger folders are summarised by the
/// per-type counts, which always cover every file.
pub const MAX_LISTING_ENTRIES: usize = 400;
/// Subfolder names sent as context.
pub const MAX_SUBDIRS: usize = 100;
/// Whole-call budget for one analyze. `ai_portkey` allows far longer; the panel does not.
pub const MODEL_BUDGET: Duration = Duration::from_secs(45);
/// Prefix on a fixed-rule reason so the owner can see the model did not answer.
pub const FALLBACK_LABEL: &str = "Fallback rule";
/// Largest rules file that will be loaded.
const MAX_RULES_BYTES: u64 = 64 * 1024;
/// Largest serialised catalog-facts blob that will be sent.
const MAX_CATALOG_BYTES: usize = 96 * 1024;
const MAX_SUGGESTIONS: usize = 12;
const MAX_FILES_PER_SUGGESTION: usize = 500;
const MAX_FOLDER_NAME: usize = 120;
const MAX_REASON: usize = 400;
/// A folder for a single file is noise, not organization.
const MIN_FILES_PER_SUGGESTION: usize = 2;

/// Everything the model is told about one folder.
pub struct FolderInput<'a> {
    pub dir_path: &'a Path,
    pub files: &'a [FileInfo],
    pub subdirs: &'a [String],
    pub is_project: bool,
    pub project_type: Option<&'a str>,
    /// Catalog facts keyed by file name, gathered by the engine from the read-only
    /// catalog pool. `None` when the catalog is unavailable or the folder is not in it.
    pub catalog_facts: Option<&'a Value>,
}

/// The owner's rules, read from the tracked rules file.
pub fn rules_text() -> Result<String, String> {
    let path = std::env::var("INTAKE_ORGANIZER_RULES_FILE")
        .map_err(|_| "INTAKE_ORGANIZER_RULES_FILE is not configured".to_string())?;
    let path = Path::new(&path);
    if !path.is_absolute() {
        return Err("the organizer rules path must be absolute".into());
    }
    let meta = std::fs::metadata(path).map_err(|e| format!("cannot inspect the organizer rules file: {e}"))?;
    if meta.len() > MAX_RULES_BYTES {
        return Err("the organizer rules file exceeds 64 KiB".into());
    }
    let text = std::fs::read_to_string(path).map_err(|e| format!("cannot read the organizer rules file: {e}"))?;
    if text.trim().is_empty() {
        return Err("the organizer rules file is empty".into());
    }
    Ok(text)
}

/// The chat model the engine is configured with. Step 3 replaces this with the
/// per-run selector; today it is the engine's one setting.
pub fn model_name() -> Result<String, String> {
    let model = std::env::var("INTAKE_CHAT_MODEL")
        .map_err(|_| "INTAKE_CHAT_MODEL is not configured".to_string())?;
    let model = model.trim().to_string();
    if model.is_empty() {
        return Err("INTAKE_CHAT_MODEL is empty".into());
    }
    Ok(model)
}

fn day(modified: u64) -> Option<String> {
    chrono::DateTime::from_timestamp(modified as i64, 0).map(|d| d.format("%Y-%m-%d").to_string())
}

/// The listing as the model sees it: names, sizes, dates and types. Never contents.
pub fn listing_payload(input: &FolderInput<'_>) -> Value {
    let mut types: BTreeMap<String, usize> = BTreeMap::new();
    for file in input.files {
        let ext = Path::new(&file.name)
            .extension()
            .map(|e| e.to_string_lossy().to_lowercase())
            .unwrap_or_default();
        *types.entry(categorize_extension(&ext).to_string()).or_insert(0) += 1;
    }

    let shown: Vec<Value> = input
        .files
        .iter()
        .take(MAX_LISTING_ENTRIES)
        .map(|f| {
            let ext = Path::new(&f.name)
                .extension()
                .map(|e| e.to_string_lossy().to_lowercase())
                .unwrap_or_default();
            json!({
                "name": f.name,
                "size": f.size,
                "modified": day(f.modified),
                "type": categorize_extension(&ext),
            })
        })
        .collect();

    let subdirs: Vec<&String> = input.subdirs.iter().take(MAX_SUBDIRS).collect();

    json!({
        "folder_name": input.dir_path.file_name().map(|n| n.to_string_lossy().to_string()),
        "parent_folder_name": input.dir_path.parent().and_then(|p| p.file_name()).map(|n| n.to_string_lossy().to_string()),
        "looks_like_code_project": input.is_project,
        "project_type": input.project_type,
        "file_count": input.files.len(),
        "files_shown": shown.len(),
        "files_by_type": types,
        "subfolder_count": input.subdirs.len(),
        "subfolders": subdirs,
        "files": shown,
    })
}

/// What the model must return. Kept next to the prompt so the two never drift.
const OUTPUT_CONTRACT: &str = r#"Reply with one JSON object and nothing else. No prose, no markdown fence, no explanation outside the JSON.

{"suggestions": [{"folder": "<new or existing folder name>", "files": ["<file name from the listing>", "..."], "reason": "<plain-language sentence>"}]}

Rules for the reply itself:
- "folder" is a plain folder name, never a path. It is created inside the folder being analyzed.
- "files" holds names exactly as they appear in the listing you were given. Never a path, never a folder name, never a name you were not shown.
- Every suggestion needs at least two files. A file belongs to at most one suggestion.
- "reason" is one sentence in the owner's language explaining what these files have in common.
- No suggestions is a valid and often correct answer: {"suggestions": []}."#;

pub fn build_messages(input: &FolderInput<'_>, rules: &str) -> Vec<ChatMessage> {
    let system = format!(
        "You organize one folder of files for its owner. You are given a folder listing \
         (names, sizes, dates and types — never file contents), what the owner's catalog knows \
         about those files, and the owner's standing rules. Follow the rules exactly; they outrank \
         anything the listing suggests to you.\n\n\
         ===== THE OWNER'S RULES =====\n{}\n\n\
         ===== YOUR REPLY =====\n{}",
        rules, OUTPUT_CONTRACT
    );

    let mut payload = json!({ "listing": listing_payload(input) });
    if let Some(facts) = input.catalog_facts {
        let encoded = serde_json::to_string(facts).unwrap_or_default();
        if !encoded.is_empty() && encoded.len() <= MAX_CATALOG_BYTES {
            payload["catalog"] = facts.clone();
        } else {
            payload["catalog_note"] = json!("the catalog facts for this folder were too large to include");
        }
    } else {
        payload["catalog_note"] = json!("the catalog has nothing recorded for this folder");
    }

    let user = format!(
        "Folder to organize:\n\n{}\n\nReturn the JSON object described above.",
        serde_json::to_string_pretty(&payload).unwrap_or_else(|_| "{}".into())
    );

    vec![
        ChatMessage { role: "system".into(), content: system },
        ChatMessage { role: "user".into(), content: user },
    ]
}

#[derive(Debug, Deserialize)]
struct AgentReply {
    #[serde(default)]
    suggestions: Vec<AgentSuggestion>,
}

#[derive(Debug, Deserialize)]
struct AgentSuggestion {
    #[serde(default, alias = "folder_name", alias = "name", alias = "suggested_name")]
    folder: String,
    #[serde(default, alias = "files_to_move")]
    files: Vec<String>,
    #[serde(default)]
    reason: String,
}

/// The JSON object inside a reply that may still be fenced or prefaced.
fn extract_json(text: &str) -> Option<&str> {
    let start = text.find('{')?;
    let end = text.rfind('}')?;
    (end > start).then(|| &text[start..=end])
}

fn folder_name_ok(name: &str) -> bool {
    !name.is_empty()
        && name.len() <= MAX_FOLDER_NAME
        && name == name.trim()
        && !name.contains('/')
        && !name.contains('\\')
        && !name.contains('\0')
        && !name.contains(':')
        && !name.starts_with('.')
        && name != "."
        && name != ".."
}

/// Turn the model's reply into the existing `FolderSuggestion` shape.
///
/// Only names present in `files` survive, so the model can never move something it
/// was not shown, reach outside the folder, or name a directory as material. A
/// suggestion that breaks a rule is dropped; the rest still stand.
pub fn parse_suggestions(
    text: &str,
    dir_path: &Path,
    files: &[FileInfo],
) -> Result<Vec<FolderSuggestion>, String> {
    let json_text = extract_json(text).ok_or_else(|| "the model returned no JSON object".to_string())?;
    let reply: AgentReply =
        serde_json::from_str(json_text).map_err(|e| format!("the model's JSON did not match the agreed shape: {e}"))?;

    let by_name: HashMap<&str, &FileInfo> = files.iter().map(|f| (f.name.as_str(), f)).collect();
    let mut claimed: HashSet<String> = HashSet::new();
    let mut out = Vec::new();

    for suggestion in reply.suggestions {
        if out.len() >= MAX_SUGGESTIONS {
            break;
        }
        let folder = suggestion.folder.trim().to_string();
        if !folder_name_ok(&folder) {
            continue;
        }

        let mut paths = Vec::new();
        let mut seen: HashSet<&str> = HashSet::new();
        for raw in suggestion.files.iter().take(MAX_FILES_PER_SUGGESTION) {
            let name = raw.trim();
            if name.is_empty() || !seen.insert(name) || claimed.contains(name) {
                continue;
            }
            // Only a file from this listing; a directory name or an invented name has no entry.
            if let Some(file) = by_name.get(name) {
                paths.push(file.path.clone());
            }
        }
        if paths.len() < MIN_FILES_PER_SUGGESTION {
            continue;
        }
        for raw in suggestion.files.iter() {
            let name = raw.trim();
            if by_name.contains_key(name) {
                claimed.insert(name.to_string());
            }
        }

        let mut reason = suggestion.reason.trim().to_string();
        if reason.is_empty() {
            reason = format!("{} files that belong together", paths.len());
        }
        if reason.chars().count() > MAX_REASON {
            reason = reason.chars().take(MAX_REASON).collect();
        }

        out.push(FolderSuggestion {
            suggested_name: folder.clone(),
            target_path: dir_path.join(&folder).to_string_lossy().to_string(),
            files_to_move: paths,
            reason,
            category: "agent".to_string(),
        });
    }

    Ok(out)
}

/// One model call for one folder, inside the analyze budget.
///
/// `Ok((suggestions, model))` on success. `Err(reason)` whenever the caller should
/// fall back to the fixed rules — no rules file, no model configured, a provider
/// error, a timeout, or a reply that could not be parsed.
pub async fn suggest(input: &FolderInput<'_>) -> Result<(Vec<FolderSuggestion>, String), String> {
    let rules = rules_text()?;
    let model = model_name()?;
    let messages = build_messages(input, &rules);

    let started = std::time::Instant::now();
    let reply = tokio::time::timeout(MODEL_BUDGET, chat_with_ai(model.clone(), messages, None))
        .await
        .map_err(|_| format!("the model did not answer within {} s", MODEL_BUDGET.as_secs()))?
        .map_err(|e| format!("the model call failed: {e}"))?;

    // Never log the reply or the prompt: both carry the owner's own file names.
    tracing::info!(
        target: "intake_organizer",
        model = %model,
        elapsed_ms = started.elapsed().as_millis() as u64,
        reply_chars = reply.chars().count(),
        "organizer suggestions"
    );

    let suggestions = parse_suggestions(&reply, input.dir_path, input.files)?;
    Ok((suggestions, model))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn file(name: &str) -> FileInfo {
        FileInfo {
            path: format!("/srv/openlist/folder/{name}"),
            name: name.to_string(),
            size: 2048,
            modified: 1_726_000_000,
        }
    }

    fn files() -> Vec<FileInfo> {
        vec![
            file("invoice-jan.pdf"),
            file("invoice-feb.pdf"),
            file("invoice-mar.pdf"),
            file("holiday.jpg"),
            file("notes.txt"),
        ]
    }

    fn input<'a>(dir: &'a Path, files: &'a [FileInfo], subdirs: &'a [String]) -> FolderInput<'a> {
        FolderInput {
            dir_path: dir,
            files,
            subdirs,
            is_project: false,
            project_type: None,
            catalog_facts: None,
        }
    }

    // ── prompt assembly ─────────────────────────────────────────────────

    #[test]
    fn prompt_carries_the_rules_and_the_listing_but_no_contents() {
        let files = files();
        let subdirs = vec!["Takeout".to_string()];
        let dir = Path::new("/srv/openlist/b2/salem-data/loose");
        let messages = build_messages(&input(dir, &files, &subdirs), "RULE ONE: takeouts are atomic");

        assert_eq!(messages.len(), 2);
        assert_eq!(messages[0].role, "system");
        assert_eq!(messages[1].role, "user");
        assert!(messages[0].content.contains("RULE ONE: takeouts are atomic"));
        assert!(messages[0].content.contains("\"suggestions\""));
        // The listing, by name/size/date/type only.
        assert!(messages[1].content.contains("invoice-jan.pdf"));
        assert!(messages[1].content.contains(&day(1_726_000_000).unwrap()));
        assert!(messages[1].content.contains("Takeout"));
        assert!(messages[1].content.contains("\"loose\""));
        // No absolute host path ever reaches the model.
        assert!(!messages[1].content.contains("/srv/openlist"));
    }

    #[test]
    fn prompt_says_when_the_catalog_had_nothing() {
        let files = files();
        let dir = Path::new("/srv/openlist/loose");
        let messages = build_messages(&input(dir, &files, &[]), "rules");
        assert!(messages[1].content.contains("the catalog has nothing recorded"));
    }

    #[test]
    fn prompt_includes_catalog_facts_when_given() {
        let files = files();
        let dir = Path::new("/srv/openlist/loose");
        let facts = json!({"invoice-jan.pdf": {"copies": 3, "sha1": "abc"}});
        let mut i = input(dir, &files, &[]);
        i.catalog_facts = Some(&facts);
        let messages = build_messages(&i, "rules");
        assert!(messages[1].content.contains("\"copies\""));
        assert!(!messages[1].content.contains("catalog_note"));
    }

    #[test]
    fn prompt_passes_the_project_hint_as_an_input_not_a_block() {
        let files = files();
        let dir = Path::new("/srv/openlist/app");
        let mut i = input(dir, &files, &[]);
        i.is_project = true;
        i.project_type = Some("Rust");
        let messages = build_messages(&i, "rules");
        assert!(messages[1].content.contains("\"looks_like_code_project\": true"));
        assert!(messages[1].content.contains("Rust"));
    }

    #[test]
    fn listing_is_bounded_but_counts_cover_every_file() {
        let many: Vec<FileInfo> = (0..900).map(|i| file(&format!("shot-{i:04}.png"))).collect();
        let dir = Path::new("/srv/openlist/loose");
        let payload = listing_payload(&input(dir, &many, &[]));
        assert_eq!(payload["file_count"], 900);
        assert_eq!(payload["files_shown"], MAX_LISTING_ENTRIES);
        assert_eq!(payload["files"].as_array().unwrap().len(), MAX_LISTING_ENTRIES);
        assert_eq!(payload["files_by_type"]["Images"], 900);
    }

    // ── response parsing ────────────────────────────────────────────────

    #[test]
    fn parses_a_clean_reply() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Invoices","files":["invoice-jan.pdf","invoice-feb.pdf","invoice-mar.pdf"],"reason":"Three monthly invoices from the same year."}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].suggested_name, "Invoices");
        assert_eq!(out[0].files_to_move.len(), 3);
        assert!(out[0].files_to_move[0].ends_with("invoice-jan.pdf"));
        assert_eq!(out[0].category, "agent");
        assert!(out[0].reason.contains("monthly invoices"));
        assert!(out[0].target_path.ends_with("Invoices"));
    }

    #[test]
    fn parses_a_fenced_and_prefaced_reply() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = "Here you go:\n```json\n{\"suggestions\":[{\"folder\":\"Invoices\",\"files\":[\"invoice-jan.pdf\",\"invoice-feb.pdf\"],\"reason\":\"Two invoices.\"}]}\n```\n";
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].files_to_move.len(), 2);
    }

    #[test]
    fn empty_suggestions_is_a_valid_answer() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let out = parse_suggestions(r#"{"suggestions":[]}"#, dir, &files).unwrap();
        assert!(out.is_empty());
    }

    #[test]
    fn non_json_reply_is_an_error_so_the_caller_falls_back() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        assert!(parse_suggestions("I could not organize this folder.", dir, &files).is_err());
        assert!(parse_suggestions("{not json at all}", dir, &files).is_err());
    }

    #[test]
    fn invented_filenames_are_dropped() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Invoices","files":["invoice-jan.pdf","invoice-apr.pdf","invoice-feb.pdf"],"reason":"x"}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out[0].files_to_move.len(), 2, "the file that was never listed is not moved");
        assert!(out[0].files_to_move.iter().all(|p| !p.contains("invoice-apr")));
    }

    #[test]
    fn a_suggestion_naming_only_unknown_files_is_dropped_entirely() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Ghosts","files":["nope-a.pdf","nope-b.pdf"],"reason":"x"}]}"#;
        assert!(parse_suggestions(reply, dir, &files).unwrap().is_empty());
    }

    #[test]
    fn folder_names_that_escape_the_directory_are_refused() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        for bad in ["..", "../elsewhere", "/etc", "a/b", "a\\b", "C:evil", "", " Leading"] {
            let reply = format!(
                r#"{{"suggestions":[{{"folder":"{}","files":["invoice-jan.pdf","invoice-feb.pdf"],"reason":"x"}}]}}"#,
                bad.replace('\\', "\\\\")
            );
            assert!(
                parse_suggestions(&reply, dir, &files).unwrap().is_empty(),
                "folder name {bad:?} should have been refused"
            );
        }
    }

    #[test]
    fn a_file_is_claimed_by_only_one_suggestion() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[
            {"folder":"Invoices","files":["invoice-jan.pdf","invoice-feb.pdf","invoice-mar.pdf"],"reason":"a"},
            {"folder":"Paperwork","files":["invoice-jan.pdf","invoice-feb.pdf"],"reason":"b"}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out.len(), 1, "the second suggestion has no unclaimed files left");
        assert_eq!(out[0].suggested_name, "Invoices");
    }

    #[test]
    fn single_file_suggestions_are_dropped() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Photos","files":["holiday.jpg"],"reason":"one photo"}]}"#;
        assert!(parse_suggestions(reply, dir, &files).unwrap().is_empty());
    }

    #[test]
    fn a_reply_that_tries_to_delete_cannot_express_it() {
        // The schema has no delete verb; extra keys are ignored and the move still
        // only covers listed files.
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Invoices","action":"delete","delete":["notes.txt"],"files":["invoice-jan.pdf","invoice-feb.pdf"],"reason":"x"}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].files_to_move.len(), 2);
        assert!(out[0].files_to_move.iter().all(|p| !p.contains("notes.txt")));
    }

    #[test]
    fn duplicate_names_inside_one_suggestion_move_once() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Invoices","files":["invoice-jan.pdf","invoice-jan.pdf","invoice-feb.pdf"],"reason":"x"}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert_eq!(out[0].files_to_move.len(), 2);
    }

    #[test]
    fn a_missing_reason_gets_a_plain_one() {
        let files = files();
        let dir = Path::new("/srv/openlist/folder");
        let reply = r#"{"suggestions":[{"folder":"Invoices","files":["invoice-jan.pdf","invoice-feb.pdf"]}]}"#;
        let out = parse_suggestions(reply, dir, &files).unwrap();
        assert!(!out[0].reason.trim().is_empty());
    }

    #[test]
    fn too_many_suggestions_are_capped() {
        let many: Vec<FileInfo> = (0..60).map(|i| file(&format!("f{i:02}.txt"))).collect();
        let dir = Path::new("/srv/openlist/folder");
        let groups: Vec<String> = (0..20)
            .map(|g| format!(r#"{{"folder":"G{g}","files":["f{:02}.txt","f{:02}.txt"],"reason":"x"}}"#, g * 2, g * 2 + 1))
            .collect();
        let reply = format!(r#"{{"suggestions":[{}]}}"#, groups.join(","));
        let out = parse_suggestions(&reply, dir, &many).unwrap();
        assert_eq!(out.len(), MAX_SUGGESTIONS);
    }

    // ── configuration gates (the fallback path) ─────────────────────────

    #[test]
    fn an_unset_rules_file_is_an_error_not_a_panic() {
        // The hosted engine sets these; a desktop build without them falls back.
        if std::env::var("INTAKE_ORGANIZER_RULES_FILE").is_err() {
            assert!(rules_text().is_err());
        }
        if std::env::var("INTAKE_CHAT_MODEL").is_err() {
            assert!(model_name().is_err());
        }
    }
}
