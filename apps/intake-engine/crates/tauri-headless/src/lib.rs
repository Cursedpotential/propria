//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! Headless stand-in for the parts of `tauri` the donor command code uses. The
//! donor handlers compile against this crate unchanged (it is renamed to `tauri`
//! in the engine's Cargo.toml). Nothing here renders a window.

use serde::de::DeserializeOwned;
use serde::Serialize;
use serde_json::{Map, Value};
use std::any::{Any, TypeId};
use std::collections::HashMap;
use std::future::Future;
use std::marker::PhantomData;
use std::ops::Deref;
use std::path::PathBuf;
use std::pin::Pin;
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{Arc, Mutex, RwLock};

pub use tauri_headless_macros::{command, generate_handler};

/// Error type mirroring `tauri::Error` for the calls the donor makes.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("{0}")]
    Message(String),
}

pub type Result<T> = std::result::Result<T, Error>;

pub trait Runtime: Send + Sync + 'static {}

/// The only runtime: no webview.
pub struct Headless;
impl Runtime for Headless {}
pub type Wry = Headless;

/// One emitted event (name + JSON payload), delivered to SSE subscribers.
#[derive(Clone, Debug)]
pub struct EmittedEvent {
    pub name: String,
    pub payload: String,
}

/// In-process listener event (`Listener::listen`).
pub struct Event {
    id: EventId,
    payload: String,
}

impl Event {
    pub fn payload(&self) -> &str {
        &self.payload
    }
    pub fn id(&self) -> EventId {
        self.id
    }
}

pub type EventId = u32;

type ListenerFn = Arc<dyn Fn(Event) + Send + Sync>;

struct Inner {
    events: tokio::sync::broadcast::Sender<EmittedEvent>,
    listeners: Mutex<HashMap<EventId, (String, ListenerFn)>>,
    next_listener: AtomicU32,
    state: RwLock<HashMap<TypeId, Arc<dyn Any + Send + Sync>>>,
    paths: PathResolver,
}

#[derive(Clone)]
pub struct AppHandle {
    inner: Arc<Inner>,
}

/// Directories the Tauri path resolver would hand out, pinned under the engine
/// state volume.
#[derive(Clone, Debug)]
pub struct PathResolver {
    root: PathBuf,
    home: PathBuf,
}

impl PathResolver {
    fn sub(&self, name: &str) -> Result<PathBuf> {
        Ok(self.root.join(name))
    }
    pub fn app_data_dir(&self) -> Result<PathBuf> {
        self.sub("app-data")
    }
    pub fn app_local_data_dir(&self) -> Result<PathBuf> {
        self.sub("local-data")
    }
    pub fn app_config_dir(&self) -> Result<PathBuf> {
        self.sub("config")
    }
    pub fn app_cache_dir(&self) -> Result<PathBuf> {
        self.sub("cache")
    }
    pub fn app_log_dir(&self) -> Result<PathBuf> {
        self.sub("logs")
    }
    pub fn resource_dir(&self) -> Result<PathBuf> {
        self.sub("resources")
    }
    pub fn temp_dir(&self) -> Result<PathBuf> {
        Ok(std::env::temp_dir())
    }
    pub fn home_dir(&self) -> Result<PathBuf> {
        Ok(self.home.clone())
    }
    pub fn document_dir(&self) -> Result<PathBuf> {
        Ok(self.home.clone())
    }
    pub fn download_dir(&self) -> Result<PathBuf> {
        Ok(self.home.clone())
    }
}

impl AppHandle {
    /// `state_root` holds app data; `home` is what "home" means to the UI.
    pub fn new(state_root: PathBuf, home: PathBuf) -> Self {
        let (events, _) = tokio::sync::broadcast::channel(4096);
        AppHandle {
            inner: Arc::new(Inner {
                events,
                listeners: Mutex::new(HashMap::new()),
                next_listener: AtomicU32::new(1),
                state: RwLock::new(HashMap::new()),
                paths: PathResolver {
                    root: state_root,
                    home,
                },
            }),
        }
    }

    pub fn subscribe(&self) -> tokio::sync::broadcast::Receiver<EmittedEvent> {
        self.inner.events.subscribe()
    }

    /// Emit an event whose payload is already JSON (used by engine-native code).
    pub fn emit_raw(&self, name: &str, payload: String) {
        let listeners: Vec<(EventId, ListenerFn)> = self
            .inner
            .listeners
            .lock()
            .map(|m| {
                m.iter()
                    .filter(|(_, (n, _))| n == name)
                    .map(|(id, (_, f))| (*id, f.clone()))
                    .collect()
            })
            .unwrap_or_default();
        for (id, f) in listeners {
            f(Event {
                id,
                payload: payload.clone(),
            });
        }
        let _ = self.inner.events.send(EmittedEvent {
            name: name.to_string(),
            payload,
        });
    }

    pub fn package_info(&self) -> PackageInfo {
        PackageInfo {
            name: "intake-engine".into(),
            version: std::env::var("INTAKE_ENGINE_VERSION").unwrap_or_else(|_| "0.99.1-hosted".into()),
        }
    }

    pub fn window(&self) -> Window {
        Window { app: self.clone() }
    }

    #[doc(hidden)]
    pub fn state_for_command<T: Send + Sync + 'static>(
        &self,
    ) -> std::result::Result<State<'_, T>, String> {
        self.try_state::<T>().ok_or_else(|| {
            format!(
                "state {} is not managed by the hosted engine",
                std::any::type_name::<T>()
            )
        })
    }
}

pub struct PackageInfo {
    pub name: String,
    pub version: String,
}

/// There are no windows in the hosted engine; this only carries the app handle
/// so donor code that emits through a window still reaches SSE subscribers.
#[derive(Clone)]
pub struct Window {
    app: AppHandle,
}
pub type WebviewWindow = Window;

impl Window {
    pub fn app_handle(&self) -> &AppHandle {
        &self.app
    }
}

pub struct State<'r, T: Send + Sync + 'static> {
    value: Arc<T>,
    _life: PhantomData<&'r ()>,
}

impl<'r, T: Send + Sync + 'static> State<'r, T> {
    pub fn inner(&self) -> &T {
        &self.value
    }
}

impl<T: Send + Sync + 'static> Deref for State<'_, T> {
    type Target = T;
    fn deref(&self) -> &T {
        &self.value
    }
}

impl<T: Send + Sync + 'static> Clone for State<'_, T> {
    fn clone(&self) -> Self {
        State {
            value: self.value.clone(),
            _life: PhantomData,
        }
    }
}

pub trait Emitter<R: Runtime = Headless> {
    fn emit<S: Serialize>(&self, event: &str, payload: S) -> Result<()>;
}

fn emit_on(app: &AppHandle, event: &str, payload: impl Serialize) -> Result<()> {
    let json = serde_json::to_string(&payload).map_err(|e| Error::Message(e.to_string()))?;
    app.emit_raw(event, json);
    Ok(())
}

impl Emitter for AppHandle {
    fn emit<S: Serialize>(&self, event: &str, payload: S) -> Result<()> {
        emit_on(self, event, payload)
    }
}

impl Emitter for Window {
    fn emit<S: Serialize>(&self, event: &str, payload: S) -> Result<()> {
        emit_on(&self.app, event, payload)
    }
}

pub trait Listener<R: Runtime = Headless> {
    fn listen<F>(&self, event: impl Into<String>, handler: F) -> EventId
    where
        F: Fn(Event) + Send + Sync + 'static;
    fn unlisten(&self, id: EventId);
}

impl Listener for AppHandle {
    fn listen<F>(&self, event: impl Into<String>, handler: F) -> EventId
    where
        F: Fn(Event) + Send + Sync + 'static,
    {
        let id = self.inner.next_listener.fetch_add(1, Ordering::Relaxed);
        if let Ok(mut map) = self.inner.listeners.lock() {
            map.insert(id, (event.into(), Arc::new(handler)));
        }
        id
    }
    fn unlisten(&self, id: EventId) {
        if let Ok(mut map) = self.inner.listeners.lock() {
            map.remove(&id);
        }
    }
}

pub trait Manager<R: Runtime = Headless> {
    fn app_handle(&self) -> &AppHandle;
    fn path(&self) -> &PathResolver {
        &self.app_handle().inner.paths
    }
    fn manage<T: Send + Sync + 'static>(&self, state: T) -> bool {
        let mut map = match self.app_handle().inner.state.write() {
            Ok(m) => m,
            Err(_) => return false,
        };
        let key = TypeId::of::<T>();
        if map.contains_key(&key) {
            return false;
        }
        map.insert(key, Arc::new(state));
        true
    }
    fn try_state<T: Send + Sync + 'static>(&self) -> Option<State<'_, T>> {
        let map = self.app_handle().inner.state.read().ok()?;
        let any = map.get(&TypeId::of::<T>())?.clone();
        let value = any.downcast::<T>().ok()?;
        Some(State {
            value,
            _life: PhantomData,
        })
    }
    fn state<T: Send + Sync + 'static>(&self) -> State<'_, T> {
        self.try_state::<T>()
            .unwrap_or_else(|| panic!("state {} not managed", std::any::type_name::<T>()))
    }
}

impl Manager for AppHandle {
    fn app_handle(&self) -> &AppHandle {
        self
    }
}

impl Manager for Window {
    fn app_handle(&self) -> &AppHandle {
        &self.app
    }
}

// ---------------------------------------------------------------------------
// Command dispatch
// ---------------------------------------------------------------------------

pub type CommandFuture =
    Pin<Box<dyn Future<Output = std::result::Result<Value, String>> + Send + 'static>>;
pub type CommandFn = fn(CommandCtx) -> CommandFuture;

pub trait Command {
    fn call(ctx: CommandCtx) -> CommandFuture;
}

pub struct CommandCtx {
    pub app: AppHandle,
    pub args: Map<String, Value>,
}

impl CommandCtx {
    /// Tauri looks arguments up by their camelCase name; accept snake_case too.
    pub fn arg<T: DeserializeOwned>(&self, camel: &str, snake: &str) -> std::result::Result<T, String> {
        let raw = self
            .args
            .get(camel)
            .or_else(|| self.args.get(snake))
            .cloned()
            .unwrap_or(Value::Null);
        serde_json::from_value(raw).map_err(|e| {
            format!("invalid args `{camel}` for command: {e}")
        })
    }
}

#[doc(hidden)]
pub mod __private {
    pub use serde_json::Value;
    use serde::Serialize;

    pub fn block_in_place<F: FnOnce() -> R, R>(f: F) -> R {
        tokio::task::block_in_place(f)
    }

    pub fn from_value<T: Serialize>(v: T) -> Result<Value, String> {
        serde_json::to_value(v).map_err(|e| e.to_string())
    }

    pub fn from_result<T: Serialize, E: Serialize>(r: Result<T, E>) -> Result<Value, String> {
        match r {
            Ok(v) => from_value(v),
            Err(e) => Err(match serde_json::to_value(&e) {
                Ok(Value::String(s)) => s,
                Ok(other) => other.to_string(),
                Err(err) => err.to_string(),
            }),
        }
    }
}

pub mod async_runtime {
    pub use tokio::spawn;
    pub use tokio::task::spawn_blocking;
}
