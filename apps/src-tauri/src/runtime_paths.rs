//! Intake's explicit runtime boundary. Never change Windows known-folder settings.
use std::path::{Component, Path, PathBuf};

pub fn intake_mode() -> bool {
    std::env::var("XPLORER_INTAKE_MODE").as_deref() == Ok("1")
}

fn validate_root(path: &Path) -> Result<PathBuf, String> {
    if !path.is_absolute() || path.components().any(|c| matches!(c, Component::ParentDir)) {
        return Err("Intake runtime root must be an absolute path without parent traversal".into());
    }
    #[cfg(windows)]
    if !path
        .to_string_lossy()
        .to_ascii_lowercase()
        .starts_with("e:\\")
        && !path
            .to_string_lossy()
            .to_ascii_lowercase()
            .starts_with("e:/")
    {
        return Err("Intake runtime root must be on the approved E: development drive".into());
    }
    Ok(path.to_path_buf())
}

pub fn runtime_root() -> Result<PathBuf, String> {
    let raw = std::env::var_os("XPLORER_RUNTIME_DIR")
        .filter(|s| !s.is_empty())
        .ok_or_else(|| {
            "Intake requires XPLORER_RUNTIME_DIR; no OS-directory fallback is allowed".to_string()
        })?;
    validate_root(Path::new(&raw))
}

/// Called before Tauri constructs its WebView or any persistent subsystem.
pub fn validate_startup() -> Result<(), String> {
    if intake_mode() {
        let root = runtime_root()?;
        std::fs::create_dir_all(&root)
            .map_err(|e| format!("Cannot create Intake runtime root: {e}"))?;
        let resolved = std::fs::canonicalize(&root)
            .map_err(|e| format!("Cannot resolve Intake runtime root: {e}"))?;
        #[cfg(windows)]
        if !resolved
            .to_string_lossy()
            .to_ascii_lowercase()
            .starts_with("\\\\?\\e:\\")
        {
            return Err(
                "Resolved Intake runtime root must remain on E: (no redirected junction)".into(),
            );
        }
        for child in [
            "app-data",
            "local-data",
            "roaming-data",
            "webview",
            "documents",
        ] {
            let path = root.join(child);
            if path.exists() {
                let target = std::fs::canonicalize(&path)
                    .map_err(|e| format!("Cannot resolve runtime directory {child}: {e}"))?;
                if !target.starts_with(&resolved) {
                    return Err(format!(
                        "Runtime directory {child} redirects outside the isolated root"
                    ));
                }
            }
            std::fs::create_dir_all(&path)
                .map_err(|e| format!("Cannot create runtime directory {child}: {e}"))?;
        }
    }
    Ok(())
}

/// Hold this file for the process lifetime. OS locks release on exit/crash;
/// the marker is intentionally retained and never permanently deleted.
pub fn acquire_instance_lock() -> Result<Option<std::fs::File>, String> {
    if !intake_mode() {
        return Ok(None);
    }
    let file = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .truncate(false)
        .open(runtime_root()?.join("intake.instance.lock"))
        .map_err(|e| format!("Cannot open Intake instance lock: {e}"))?;
    file.try_lock()
        .map_err(|e| format!("Intake runtime is already in use or cannot be locked: {e}"))?;
    Ok(Some(file))
}

pub fn app_data_dir<R: tauri::Runtime>(app: &impl tauri::Manager<R>) -> Result<PathBuf, String> {
    if intake_mode() {
        runtime_root().map(|p| p.join("app-data"))
    } else {
        app.path().app_data_dir().map_err(|e| e.to_string())
    }
}

// Existing Option-based callers have fallback logic. In Intake mode an invalid
// root must fail closed rather than allowing that fallback to write elsewhere.
pub fn data_local_dir() -> Option<PathBuf> {
    if intake_mode() {
        Some(
            runtime_root()
                .expect("Intake runtime must be validated before storage access")
                .join("local-data"),
        )
    } else {
        dirs::data_local_dir()
    }
}

pub fn data_dir() -> Option<PathBuf> {
    if intake_mode() {
        Some(
            runtime_root()
                .expect("Intake runtime must be validated before storage access")
                .join("roaming-data"),
        )
    } else {
        dirs::data_dir()
    }
}

pub fn require_legacy_search() -> Result<(), String> {
    if intake_mode() {
        Err("Legacy Xplorer indexing/search is disabled in Intake mode. CocoIndex search is not connected yet; filesystem browsing remains available.".into())
    } else {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn rejects_relative_and_parent_paths() {
        assert!(validate_root(Path::new("runtime")).is_err());
        #[cfg(windows)]
        assert!(validate_root(Path::new("E:\\runtime\\..\\outside")).is_err());
    }
    #[test]
    #[cfg(windows)]
    fn requires_approved_drive() {
        assert!(validate_root(Path::new("C:\\runtime")).is_err());
        assert!(validate_root(Path::new("E:\\AI_Workspace\\.intake-dev\\runtime")).is_ok());
    }
}
