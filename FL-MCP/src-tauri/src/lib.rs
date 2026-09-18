// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Tauri shell: spawns the Node sidecar (../sidecar/server.mjs) as a plain
// child process (not a Tauri "sidecar binary" — that mechanism expects a
// compiled, target-triple-suffixed executable; our sidecar dynamically
// loads @surrealdb/node, which ships a native .node addon that cannot be
// embedded into a single-executable bundle the way pure JS can, so true
// `externalBin` packaging was investigated and set aside — see
// docs/ARCHITECTURE.md "Open items" for the honest writeup). We set PORT=0
// so the OS assigns a real ephemeral port, read that port back off the
// child's stdout (the single `SIDECAR_READY {"port":N}` line it prints),
// and expose it to the webview ONLY through the `sidecar_port` command
// below — never baked into the frontend bundle, never passed as a URL
// query param.
//
// The Claude Code OAuth token, if configured, lives entirely inside the
// sidecar's own process environment (see ../sidecar/lib/auth.mjs) and is
// never read, forwarded, or logged by this Rust process.

use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader};
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex};
use tauri::{Manager, State};

#[derive(Default)]
struct SidecarState {
    port: Mutex<Option<u16>>,
    child: Mutex<Option<Child>>,
}

#[derive(Serialize, Deserialize)]
struct SidecarReady {
    port: u16,
}

// Installed, standalone-binary location (owner order 2026-09-07 15:29:
// `C:\Users\matts\bin\family-court-workbench.exe`) — this app is a personal,
// single-machine local tool (not a distributed installer), so hardcoding
// the one machine's known source-tree location as a fallback for "where is
// the sidecar script" is a deliberate, documented simplification rather
// than an oversight. See docs/ARCHITECTURE.md.
const KNOWN_WORKBENCH_ROOT: &str = r"E:\AI_Workspace\Projects\the-platform-workspace\family-court-workbench";

fn sidecar_script_path(app: &tauri::AppHandle) -> Option<PathBuf> {
    let resource_dir = app.path().resource_dir().ok();
    let candidates = [
        // dev (`tauri dev` / `cargo run`, cwd = src-tauri/): ../sidecar/server.mjs
        std::env::current_dir().ok().map(|d| d.join("..").join("sidecar").join("server.mjs")),
        // installed standalone .exe (owner's ~/bin copy) — see KNOWN_WORKBENCH_ROOT above.
        Some(PathBuf::from(KNOWN_WORKBENCH_ROOT).join("sidecar").join("server.mjs")),
        // future: a real bundled resource, if externalBin packaging is revisited.
        resource_dir.map(|d| d.join("sidecar").join("server.mjs")),
    ];
    candidates.into_iter().flatten().find(|p| p.exists())
}

fn spawn_sidecar(app: &tauri::AppHandle, state: Arc<SidecarState>) {
    let Some(script) = sidecar_script_path(app) else {
        eprintln!("[family-court-workbench] sidecar/server.mjs not found — chat/store panes will show a connection error.");
        return;
    };

    let mut cmd = Command::new("node");
    cmd.arg(&script)
        .env("PORT", "0") // request a real OS-assigned ephemeral port
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit());

    let mut child = match cmd.spawn() {
        Ok(c) => c,
        Err(err) => {
            eprintln!("[family-court-workbench] failed to spawn sidecar (`node {}`): {err}", script.display());
            return;
        }
    };

    if let Some(stdout) = child.stdout.take() {
        let state_for_thread = state.clone();
        std::thread::spawn(move || {
            let reader = BufReader::new(stdout);
            for line in reader.lines().map_while(Result::ok) {
                if let Some(json) = line.strip_prefix("SIDECAR_READY ") {
                    if let Ok(ready) = serde_json::from_str::<SidecarReady>(json) {
                        *state_for_thread.port.lock().unwrap() = Some(ready.port);
                    }
                }
            }
        });
    }

    *state.child.lock().unwrap() = Some(child);
}

#[tauri::command]
async fn sidecar_port(state: State<'_, Arc<SidecarState>>) -> Result<u16, String> {
    // Short poll loop: the sidecar prints its ready line within a few hundred
    // ms of spawn; cap the wait so a genuinely dead sidecar surfaces as a
    // clear error in the UI instead of hanging the chat/store panes forever.
    for _ in 0..100 {
        if let Some(port) = *state.port.lock().unwrap() {
            return Ok(port);
        }
        std::thread::sleep(std::time::Duration::from_millis(100));
    }
    Err("sidecar did not report a port within 10s — see the app's stderr log".into())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let state = Arc::new(SidecarState::default());

    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .manage(state.clone())
        .setup(move |app| {
            spawn_sidecar(&app.handle().clone(), state.clone());
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![sidecar_port])
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::Destroyed = event {
                if let Some(state) = window.app_handle().try_state::<Arc<SidecarState>>() {
                    if let Some(mut child) = state.child.lock().unwrap().take() {
                        let _ = child.kill();
                    }
                }
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running the family-court-workbench Tauri application");
}
