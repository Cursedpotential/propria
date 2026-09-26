//! Byline: Claude Code · Opus 5 · 2026-09-18
//!
//! Person / date timelines from SurrealDB (`surreal-intake`, ns `consignatio`, db `intake`), owner
//! direction 2026-09-18 20:58 EDT: "timelines, entities and graphs in Surreal".
//!
//! The engine calls whatever timeline functions the database currently defines. Names and argument
//! order are read from `INFO FOR DB` and matched by parameter NAME, so both today's dated functions
//! (`fn::timeline_20260918($from, $to, $person, $source, $lim)`) and a later rename to
//! `fn::timeline($person, $from, $to)` work without an engine change.
//!
//! Until a credential is configured (`INTAKE_SURREAL_PASSWORD_FILE`, root-only file, never env),
//! every call answers `available: false` with the reason, and the panel says the timeline is not
//! loaded yet instead of pretending.

use crate::Engine;
use serde_json::{json, Map, Value};
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

pub type Outcome = Result<Value, (u16, String)>;

fn env_or(name: &str, default: &str) -> String {
    std::env::var(name).ok().filter(|v| !v.trim().is_empty()).unwrap_or_else(|| default.to_string())
}

struct Conn {
    url: String,
    user: String,
    pass: String,
    ns: String,
    db: String,
}

fn conn() -> Result<Conn, String> {
    let url = std::env::var("INTAKE_SURREAL_URL").map_err(|_| "INTAKE_SURREAL_URL is not configured".to_string())?;
    let user = std::env::var("INTAKE_SURREAL_USER").map_err(|_| "INTAKE_SURREAL_USER is not configured".to_string())?;
    let file = std::env::var("INTAKE_SURREAL_PASSWORD_FILE")
        .map_err(|_| "INTAKE_SURREAL_PASSWORD_FILE is not configured".to_string())?;
    let pass = std::fs::read_to_string(&file).map_err(|e| format!("cannot read the Surreal password file: {e}"))?;
    Ok(Conn {
        url: url.trim_end_matches('/').to_string(),
        user,
        pass: pass.trim().to_string(),
        ns: env_or("INTAKE_SURREAL_NS", "consignatio"),
        db: env_or("INTAKE_SURREAL_DB", "intake"),
    })
}

fn http() -> &'static reqwest::Client {
    static C: OnceLock<reqwest::Client> = OnceLock::new();
    C.get_or_init(|| reqwest::Client::builder().timeout(Duration::from_secs(20)).build().expect("http client"))
}

async fn sql(c: &Conn, statement: &str) -> Result<Vec<Value>, String> {
    let resp = http()
        .post(format!("{}/sql", c.url))
        .basic_auth(&c.user, Some(&c.pass))
        .header("Accept", "application/json")
        .header("surreal-ns", &c.ns)
        .header("surreal-db", &c.db)
        .body(statement.to_string())
        .send()
        .await
        .map_err(|e| format!("surreal-intake: {e}"))?;
    if !resp.status().is_success() {
        return Err(format!("surreal-intake answered {}", resp.status()));
    }
    let v: Value = resp.json().await.map_err(|e| format!("surreal-intake: {e}"))?;
    let arr = v.as_array().cloned().unwrap_or_default();
    for r in &arr {
        if r.get("status").and_then(|s| s.as_str()) == Some("ERR") {
            return Err(format!(
                "surreal-intake: {}",
                r.get("result").and_then(|x| x.as_str()).unwrap_or("query failed").chars().take(300).collect::<String>()
            ));
        }
    }
    Ok(arr)
}

#[derive(Clone, Debug)]
struct Fun {
    name: String,
    params: Vec<String>,
}

/// Function definitions currently in the database, by their `fn::` name.
async fn functions(c: &Conn) -> Result<Vec<Fun>, String> {
    static CACHE: OnceLock<Mutex<Option<(Instant, Vec<Fun>)>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(None));
    if let Some((at, v)) = cache.lock().unwrap_or_else(|e| e.into_inner()).as_ref() {
        if at.elapsed() < Duration::from_secs(60) {
            return Ok(v.clone());
        }
    }
    let rows = sql(c, "INFO FOR DB;").await?;
    let defs = rows
        .first()
        .and_then(|r| r.get("result"))
        .and_then(|r| r.get("functions"))
        .and_then(|f| f.as_object())
        .cloned()
        .unwrap_or_default();
    let mut out = Vec::new();
    for (name, def) in defs {
        let text = def.as_str().unwrap_or_default();
        let params = text
            .split_once('(')
            .map(|(_, rest)| rest.split(')').next().unwrap_or(""))
            .unwrap_or("")
            .split(',')
            .filter_map(|p| p.trim().strip_prefix('$').map(|p| p.split(':').next().unwrap_or(p).trim().to_string()))
            .filter(|p| !p.is_empty())
            .collect();
        out.push(Fun { name, params });
    }
    *cache.lock().unwrap_or_else(|e| e.into_inner()) = Some((Instant::now(), out.clone()));
    Ok(out)
}

/// The best match for a role: an exact name, else a dated variant of it.
fn pick<'a>(funs: &'a [Fun], exact: &str, prefix: &str, avoid: &[&str]) -> Option<&'a Fun> {
    funs.iter()
        .find(|f| f.name == exact)
        .or_else(|| {
            funs.iter().find(|f| {
                f.name.starts_with(prefix) && !avoid.iter().any(|a| f.name.starts_with(a)) && f.name != exact
            })
        })
}

/// A value as a SurrealQL literal. `{"__datetime__": "2026-01-01T00:00:00Z"}` becomes `d"…"`.
fn lit(v: &Value) -> String {
    match v {
        Value::Null => "NONE".into(),
        Value::Object(m) if m.contains_key("__datetime__") => {
            format!("d{}", serde_json::to_string(&m["__datetime__"]).unwrap_or_else(|_| "\"\"".into()))
        }
        Value::String(s) => serde_json::to_string(s).unwrap_or_else(|_| "NONE".into()),
        other => other.to_string(),
    }
}

type Values = std::collections::HashMap<String, Value>;

/// Build `fn::name(a, b, …)` from named values, in the order the function declares them.
fn call(f: &Fun, values: &Values) -> String {
    let args: Vec<String> = f.params.iter().map(|p| lit(values.get(p).unwrap_or(&Value::Null))).collect();
    format!("RETURN fn::{}({});", f.name, args.join(", "))
}

/// A YYYY-MM-DD argument as a Surreal datetime value (end dates cover the whole day).
fn date_value(args: &Map<String, Value>, key: &str, end_of_day: bool) -> Result<Value, (u16, String)> {
    match args.get(key).and_then(|v| v.as_str()).map(str::trim).filter(|v| !v.is_empty()) {
        None => Ok(Value::Null),
        Some(v) => {
            chrono::NaiveDate::parse_from_str(v, "%Y-%m-%d").map_err(|_| (422, format!("{key} must be YYYY-MM-DD")))?;
            let time = if end_of_day { "T23:59:59Z" } else { "T00:00:00Z" };
            Ok(json!({"__datetime__": format!("{v}{time}")}))
        }
    }
}

fn not_available(reason: String) -> Value {
    json!({"available": false, "reason": reason, "note": "timeline not loaded yet"})
}

/// `intake_timeline { person, from, to, sourceFormat, limit }`.
pub async fn timeline(_engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let c = match conn() {
        Ok(c) => c,
        Err(e) => return Ok(not_available(e)),
    };
    let funs = match functions(&c).await {
        Ok(f) => f,
        Err(e) => return Ok(not_available(e)),
    };
    let person = args
        .get("person")
        .and_then(|v| v.as_str())
        .unwrap_or("all")
        .to_ascii_lowercase();
    let limit = args.get("limit").and_then(|v| v.as_i64()).unwrap_or(200).clamp(1, 1000);
    let from = date_value(args, "from", false)?;
    let to = date_value(args, "to", true)?;
    let chosen = match person.as_str() {
        "katrina" => pick(&funs, "timeline_katrina", "timeline_katrina", &[]),
        "daughter" => pick(&funs, "timeline_daughter", "timeline_daughter", &[]),
        "all" => pick(&funs, "timeline", "timeline", &["timeline_katrina", "timeline_daughter", "timeline_day", "timeline_search"]),
        other => return Err((422, format!("unknown person {other}"))),
    };
    let Some(f) = chosen else {
        return Ok(not_available(format!(
            "surreal-intake has no timeline function for {person} yet (functions: {})",
            funs.iter().map(|f| f.name.as_str()).collect::<Vec<_>>().join(", ")
        )));
    };
    let mut values = Values::new();
    values.insert("from".into(), from);
    values.insert("to".into(), to);
    values.insert("person".into(), if person == "all" { Value::Null } else { Value::String(person.clone()) });
    values.insert("source".into(), args.get("sourceFormat").cloned().unwrap_or(Value::Null));
    values.insert("lim".into(), json!(limit));
    values.insert("limit".into(), json!(limit));
    values.insert("min_conf".into(), args.get("minConfidence").cloned().unwrap_or(Value::Null));
    match sql(&c, &call(f, &values)).await {
        Ok(rows) => {
            let events = rows.first().and_then(|r| r.get("result")).cloned().unwrap_or(json!([]));
            let n = events.as_array().map(|a| a.len()).unwrap_or(0);
            Ok(json!({"available": true, "function": format!("fn::{}", f.name), "events": events, "count": n}))
        }
        Err(e) => Ok(not_available(e)),
    }
}

/// `intake_timeline_day_counts { person }` — events per day, for a density strip.
pub async fn day_counts(_engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let c = match conn() {
        Ok(c) => c,
        Err(e) => return Ok(not_available(e)),
    };
    let funs = match functions(&c).await {
        Ok(f) => f,
        Err(e) => return Ok(not_available(e)),
    };
    let Some(f) = pick(&funs, "day_counts", "timeline_day_counts", &[]) else {
        return Ok(not_available("surreal-intake has no day-counts function yet".into()));
    };
    let mut values = Values::new();
    let scope = args.get("person").cloned().unwrap_or(Value::Null);
    values.insert("scope".into(), scope.clone());
    values.insert("person".into(), scope);
    match sql(&c, &call(f, &values)).await {
        Ok(rows) => Ok(json!({
            "available": true,
            "function": format!("fn::{}", f.name),
            "days": rows.first().and_then(|r| r.get("result")).cloned().unwrap_or(json!([])),
        })),
        Err(e) => Ok(not_available(e)),
    }
}
