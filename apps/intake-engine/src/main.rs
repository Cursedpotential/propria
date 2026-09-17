//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! Hosted Intake engine: the donor's Rust command handlers served over the
//! documented web-mode contract (`apps/web/content/docs/architecture/web-mode.mdx`).

mod catalog;
mod donor_commands;
mod http;
mod media;
mod routing;

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use tauri::{AppHandle, CommandFn, Manager};

pub struct Engine {
    pub app: AppHandle,
    pub commands: HashMap<&'static str, CommandFn>,
    pub browser_unavailable: HashMap<&'static str, &'static str>,
    pub mount_root: PathBuf,
    pub catalog: Option<Arc<catalog::Catalog>>,
    pub catalog_error: Option<String>,
}

fn env_path(name: &str, default: &str) -> PathBuf {
    PathBuf::from(std::env::var(name).unwrap_or_else(|_| default.to_string()))
}

/// Commands that act on the viewer's own desktop/OS. A browser tab talking to a
/// server has no such desktop, so they answer 501 with the reason.
fn browser_unavailable() -> HashMap<&'static str, &'static str> {
    const DESKTOP: &str = "needs the viewer's own desktop OS; the hosted engine runs on a server and cannot open apps, windows or OS shells on your PC";
    HashMap::from([
        ("open_file", DESKTOP),
        ("open_url", "opens a URL in the server's desktop browser; open links in your own browser tab instead"),
        ("open_in_terminal", DESKTOP),
        ("show_in_folder", DESKTOP),
        ("open_recycle_bin", DESKTOP),
        ("open_file_with_application", DESKTOP),
        ("set_default_application", DESKTOP),
        ("get_system_applications", DESKTOP),
        ("eject_volume", "ejecting removable media is a desktop OS action"),
        ("install_cli", "installs the xplorer CLI into the desktop OS PATH"),
        ("set_default_folder_handler", "Windows Explorer shell integration (Windows desktop only)"),
        ("add_context_menu_entry", "Windows Explorer shell integration (Windows desktop only)"),
        ("remove_context_menu_entry", "Windows Explorer shell integration (Windows desktop only)"),
        ("get_shell_integration_status", "Windows Explorer shell integration (Windows desktop only)"),
        ("register_global_shortcuts", "OS-wide global shortcuts need a desktop session; in-page shortcuts still work"),
        ("unregister_global_shortcuts", "OS-wide global shortcuts need a desktop session; in-page shortcuts still work"),
        ("toggle_global_shortcuts", "OS-wide global shortcuts need a desktop session; in-page shortcuts still work"),
    ])
}

#[tokio::main(flavor = "multi_thread")]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .init();

    let state_root = env_path("INTAKE_ENGINE_STATE", "/state");
    let mount_root = env_path("INTAKE_MOUNT_ROOT", "/srv/openlist");
    let bind = std::env::var("INTAKE_ENGINE_BIND").unwrap_or_else(|_| "0.0.0.0:8790".into());

    // The donor's isolated-runtime switch: all app data under the state volume.
    std::env::set_var("XPLORER_INTAKE_MODE", "1");
    std::env::set_var("XPLORER_RUNTIME_DIR", &state_root);
    if let Err(e) = xplorer::runtime_paths::validate_startup() {
        eprintln!("intake-engine: {e}");
        std::process::exit(1);
    }

    let app = AppHandle::new(state_root.clone(), mount_root.clone());

    let progress = xplorer::operations::ProgressManager::new().with_app_handle(app.clone());
    app.manage(Arc::new(progress));

    let app_data = xplorer::runtime_paths::app_data_dir(&app).expect("app data dir");
    let shortcuts_dir = app_data.join("shortcuts");
    std::fs::create_dir_all(&shortcuts_dir).ok();
    xplorer::shortcuts::init_shortcuts_manager(shortcuts_dir.to_str().unwrap_or("shortcuts"));
    std::fs::create_dir_all(app_data.join("extensions")).ok();
    xplorer::extensions::init_extension_manager(app_data.to_str().unwrap_or("/state/app-data"));

    let mut commands: HashMap<&'static str, CommandFn> = HashMap::new();
    for (name, f) in donor_commands::donor_commands() {
        commands.insert(name, f);
    }

    let (catalog, catalog_error) = match catalog::Catalog::connect().await {
        Ok(c) => (Some(Arc::new(c)), None),
        Err(e) => {
            tracing::error!("catalog:// unavailable: {e}");
            (None, Some(e))
        }
    };

    let engine = Arc::new(Engine {
        app,
        commands,
        browser_unavailable: browser_unavailable(),
        mount_root,
        catalog,
        catalog_error,
    });

    let listener = tokio::net::TcpListener::bind(&bind)
        .await
        .unwrap_or_else(|e| panic!("bind {bind}: {e}"));
    tracing::info!(
        "intake-engine listening on {bind}; {} commands",
        engine.commands.len()
    );
    axum::serve(listener, http::router(engine))
        .await
        .expect("server");
}
