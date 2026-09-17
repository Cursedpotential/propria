//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! Extracted metadata for the native Metadata panel (owner decision 6c):
//! image EXIF (kamadak-exif), audio tags + properties (lofty), audio/video container and
//! stream info (ffprobe from the image), document text (the donor's `document_extractor`).

use lofty::file::{AudioFile, TaggedFileExt};
use lofty::tag::ItemValue;
use serde_json::{json, Value};
use std::path::Path;
use std::time::Duration;

const TEXT_EXCERPT_CHARS: usize = 20_000;
const DOC_MAX_BYTES: u64 = 64 * 1024 * 1024;

fn ext(path: &Path) -> String {
    path.extension()
        .and_then(|e| e.to_str())
        .map(|e| e.to_ascii_lowercase())
        .unwrap_or_default()
}

const EXIF_EXT: &[&str] = &["jpg", "jpeg", "tif", "tiff", "heic", "heif", "png", "webp", "avif", "dng", "cr2", "nef", "arw"];
const AUDIO_EXT: &[&str] = &["mp3", "flac", "m4a", "aac", "ogg", "opus", "wav", "aiff", "aif", "wma", "ape", "wv", "mpc"];
const VIDEO_EXT: &[&str] = &["mp4", "m4v", "mov", "mkv", "webm", "avi", "wmv", "flv", "3gp", "mts", "m2ts", "ts", "mpg", "mpeg"];
const DOC_EXT: &[&str] = &["pdf", "docx", "doc", "xlsx", "xls", "pptx", "ppt", "rtf", "odt", "ods", "odp"];

pub fn exif(path: &Path) -> Result<Value, String> {
    let file = std::fs::File::open(path).map_err(|e| e.to_string())?;
    let mut reader = std::io::BufReader::new(file);
    let data = exif::Reader::new()
        .read_from_container(&mut reader)
        .map_err(|e| format!("no EXIF: {e}"))?;
    let fields: Vec<Value> = data
        .fields()
        .map(|f| {
            json!({
                "tag": f.tag.to_string(),
                "ifd": format!("{:?}", f.ifd_num),
                "value": f.display_value().with_unit(&data).to_string(),
            })
        })
        .collect();
    Ok(json!({ "fields": fields, "count": fields.len() }))
}

pub fn audio(path: &Path) -> Result<Value, String> {
    let tagged = lofty::read_from_path(path).map_err(|e| format!("no audio tags: {e}"))?;
    let props = tagged.properties();
    let mut tags = Vec::new();
    for tag in tagged.tags() {
        for item in tag.items() {
            let value = match item.value() {
                ItemValue::Text(t) | ItemValue::Locator(t) => t.clone(),
                ItemValue::Binary(b) => format!("<{} bytes>", b.len()),
            };
            tags.push(json!({
                "tag_type": format!("{:?}", tag.tag_type()),
                "key": format!("{:?}", item.key()),
                "value": value,
            }));
        }
    }
    Ok(json!({
        "file_type": format!("{:?}", tagged.file_type()),
        "duration_ms": props.duration().as_millis() as u64,
        "overall_bitrate_kbps": props.overall_bitrate(),
        "audio_bitrate_kbps": props.audio_bitrate(),
        "sample_rate": props.sample_rate(),
        "bit_depth": props.bit_depth(),
        "channels": props.channels(),
        "pictures": tagged.tags().iter().map(|t| t.pictures().len()).sum::<usize>(),
        "tags": tags,
    }))
}

pub async fn ffprobe(path: &Path) -> Result<Value, String> {
    let out = tokio::time::timeout(
        Duration::from_secs(45),
        tokio::process::Command::new("ffprobe")
            .args(["-v", "error", "-print_format", "json", "-show_format", "-show_streams"])
            .arg(path)
            .kill_on_drop(true)
            .output(),
    )
    .await
    .map_err(|_| "ffprobe timed out after 45 s".to_string())?
    .map_err(|e| format!("ffprobe could not run: {e}"))?;
    if !out.status.success() {
        return Err(format!(
            "ffprobe failed: {}",
            String::from_utf8_lossy(&out.stderr).trim()
        ));
    }
    serde_json::from_slice(&out.stdout).map_err(|e| format!("ffprobe output: {e}"))
}

/// Everything the Metadata panel shows for one real file on the mount.
pub async fn file_metadata(path: &Path) -> Value {
    let p = path.to_string_lossy().to_string();
    let e = ext(path);
    let mut out = serde_json::Map::new();

    out.insert(
        "properties".into(),
        match xplorer::operations::get_detailed_file_properties(p.clone()).await {
            Ok(v) => serde_json::to_value(v).unwrap_or(Value::Null),
            Err(err) => json!({ "error": err }),
        },
    );
    let is_dir = tokio::fs::metadata(path).await.map(|m| m.is_dir()).unwrap_or(false);
    if is_dir {
        return Value::Object(out);
    }

    if EXIF_EXT.contains(&e.as_str()) {
        let pb = path.to_path_buf();
        let r = tokio::task::spawn_blocking(move || exif(&pb)).await.unwrap_or_else(|j| Err(j.to_string()));
        out.insert("exif".into(), r.unwrap_or_else(|err| json!({ "error": err })));
        if let Ok(info) = xplorer::operations::get_image_info(p.clone()).await {
            out.insert("image".into(), serde_json::to_value(info).unwrap_or(Value::Null));
        }
    }
    if AUDIO_EXT.contains(&e.as_str()) {
        let pb = path.to_path_buf();
        let r = tokio::task::spawn_blocking(move || audio(&pb)).await.unwrap_or_else(|j| Err(j.to_string()));
        out.insert("audio".into(), r.unwrap_or_else(|err| json!({ "error": err })));
    }
    if AUDIO_EXT.contains(&e.as_str()) || VIDEO_EXT.contains(&e.as_str()) {
        out.insert(
            "media".into(),
            ffprobe(path).await.unwrap_or_else(|err| json!({ "error": err })),
        );
    }
    if DOC_EXT.contains(&e.as_str()) {
        let size = tokio::fs::metadata(path).await.map(|m| m.len()).unwrap_or(0);
        let doc = if size > DOC_MAX_BYTES {
            json!({ "error": format!("document is {} MB; text extraction is limited to {} MB in the panel", size / 1_048_576, DOC_MAX_BYTES / 1_048_576) })
        } else {
            match xplorer::operations::extract_document_text(p.clone()).await {
                Ok(text) => {
                    let chars = text.chars().count();
                    let excerpt: String = text.chars().take(TEXT_EXCERPT_CHARS).collect();
                    json!({ "chars": chars, "truncated": chars > TEXT_EXCERPT_CHARS, "text": excerpt })
                }
                Err(err) => json!({ "error": err }),
            }
        };
        out.insert("document_text".into(), doc);
    }
    Value::Object(out)
}
