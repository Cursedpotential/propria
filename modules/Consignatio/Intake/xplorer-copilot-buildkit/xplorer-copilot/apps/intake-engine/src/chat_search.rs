//! Byline: Claude Code · Opus 5 · 2026-09-18
//!
//! One Content Search over the chats index (owner 2026-09-18: "Search yea", then "I have to have
//! one place that I search"). Answers come from indexes only, never from reading B2, and the caller
//! never picks a backend:
//!   * Weaviate hybrid search (BM25 + NIM query vector, target vector `text_nim`) on
//!     `ChatEvents20260918` — the primary index. Its properties are read from the live collection
//!     schema, so fields a later ingest adds (recipients, catalog_path, speaker, topics, …) are
//!     used as soon as they exist, with no engine change.
//!   * The Postgres copy of the same events (`raw_duck.chat_events_20260918`, read as metabase_ro)
//!     with its GIN full-text index. It is queried at the same time because Weaviate holds only
//!     part of the corpus until the ingest finishes; both legs are merged and de-duplicated by
//!     `dedup_key`. When Weaviate covers everything, this leg is dropped server-side
//!     (`INTAKE_CHAT_INDEX_TABLE=""`) and nothing changes in the UI.
//! Person / date / format / speaker / topic filters are applied in both legs, so a filtered search
//! means the same thing whichever leg found the event.

use crate::Engine;
use serde_json::{json, Map, Value};
use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

pub type Outcome = Result<Value, (u16, String)>;

fn env_or(name: &str, default: &str) -> String {
    std::env::var(name).ok().filter(|v| !v.trim().is_empty()).unwrap_or_else(|| default.to_string())
}

/// `schema.table` of the Postgres copy, or `None` once it is retired (empty env value).
fn table() -> Result<Option<String>, (u16, String)> {
    let t = match std::env::var("INTAKE_CHAT_INDEX_TABLE") {
        Ok(v) if v.trim().is_empty() => return Ok(None),
        Ok(v) => v,
        Err(_) => "raw_duck.chat_events_20260918".to_string(),
    };
    if t.chars().all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '.') {
        Ok(Some(t))
    } else {
        Err((500, "INTAKE_CHAT_INDEX_TABLE is not a plain schema.table name".into()))
    }
}

fn provenance_table(events: &str) -> String {
    env_or("INTAKE_CHAT_INDEX_PROVENANCE_TABLE", &events.replace("chat_events_", "chat_event_provenance_"))
}

fn collection() -> String {
    env_or("INTAKE_CHAT_INDEX_COLLECTION", "ChatEvents20260918")
}

fn weaviate_url() -> String {
    env_or("INTAKE_CHAT_INDEX_WEAVIATE_URL", "http://100.91.190.107:8082").trim_end_matches('/').to_string()
}

fn http() -> &'static reqwest::Client {
    static C: OnceLock<reqwest::Client> = OnceLock::new();
    C.get_or_init(|| reqwest::Client::builder().timeout(Duration::from_secs(15)).build().expect("http client"))
}

fn s(args: &Map<String, Value>, camel: &str, snake: &str) -> Option<String> {
    args.get(camel)
        .or_else(|| args.get(snake))
        .and_then(|v| v.as_str())
        .map(|v| v.trim().to_string())
        .filter(|v| !v.is_empty())
}

fn bad(msg: impl Into<String>) -> (u16, String) {
    (422, msg.into())
}

async fn pg(engine: &Engine) -> Result<deadpool_postgres::Object, (u16, String)> {
    let cat = engine
        .catalog
        .as_ref()
        .ok_or_else(|| (503, "the catalog database is not connected".to_string()))?;
    let client = cat.ro_pool().get().await.map_err(|e| (503, format!("catalog connection: {e}")))?;
    client
        .batch_execute("set statement_timeout = '8s'")
        .await
        .map_err(|e| (503, format!("catalog connection: {e}")))?;
    Ok(client)
}

// ---------------------------------------------------------------------------------------------
// Filters: one definition, applied to both legs
// ---------------------------------------------------------------------------------------------

#[derive(Default, Clone)]
struct Filters {
    person: String,
    from: Option<chrono::NaiveDate>,
    to: Option<chrono::NaiveDate>,
    source_format: Option<String>,
    /// "My words only": the AI-chat ingest tags each turn with speaker `owner` / `assistant`.
    speaker: Option<String>,
    topic: Option<String>,
}

fn plain(v: &str, max: usize) -> bool {
    v.len() <= max && v.chars().all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '-')
}

impl Filters {
    fn parse(args: &Map<String, Value>) -> Result<Self, (u16, String)> {
        let person = s(args, "person", "person").unwrap_or_else(|| "all".into()).to_ascii_lowercase();
        if !["all", "katrina", "daughter"].contains(&person.as_str()) {
            return Err(bad("person must be all, katrina or daughter"));
        }
        let date = |k: &str| -> Result<Option<chrono::NaiveDate>, (u16, String)> {
            s(args, k, k)
                .map(|v| chrono::NaiveDate::parse_from_str(&v, "%Y-%m-%d").map_err(|_| bad(format!("{k} must be YYYY-MM-DD"))))
                .transpose()
        };
        let source_format = s(args, "sourceFormat", "source_format").filter(|f| f != "all");
        if source_format.as_deref().is_some_and(|f| !plain(f, 64)) {
            return Err(bad("sourceFormat is not a known format name"));
        }
        let speaker = s(args, "speaker", "speaker").filter(|v| v != "all");
        if speaker.as_deref().is_some_and(|v| !plain(v, 32)) {
            return Err(bad("speaker is not a known speaker"));
        }
        let topic = s(args, "topic", "topic").filter(|v| v != "all");
        if topic.as_deref().is_some_and(|v| !plain(v, 64)) {
            return Err(bad("topic is not a known topic tag"));
        }
        Ok(Filters { person, from: date("from")?, to: date("to")?, source_format, speaker, topic })
    }

    /// Conditions for the Postgres leg; `cols` are the columns that table actually has.
    fn sql(&self, first: usize, cols: &[String]) -> Option<(String, Vec<Box<dyn tokio_postgres::types::ToSql + Sync + Send>>)> {
        let has = |c: &str| cols.iter().any(|x| x == c);
        let mut conds = Vec::new();
        let mut params: Vec<Box<dyn tokio_postgres::types::ToSql + Sync + Send>> = Vec::new();
        match self.person.as_str() {
            "katrina" => conds.push("katrina_conf = 'strong' and katrina_ref_type <> 'group_participant'".to_string()),
            "daughter" => conds.push("daughter_conf is not null".to_string()),
            _ => {}
        }
        if let Some(d) = self.from {
            params.push(Box::new(d.and_hms_opt(0, 0, 0).expect("midnight")));
            conds.push(format!("sort_ts >= ${}", first + params.len() - 1));
        }
        if let Some(d) = self.to {
            params.push(Box::new(d.succ_opt().unwrap_or(d).and_hms_opt(0, 0, 0).expect("midnight")));
            conds.push(format!("sort_ts < ${}", first + params.len() - 1));
        }
        if let Some(f) = &self.source_format {
            params.push(Box::new(f.clone()));
            conds.push(format!("source_format = ${}", first + params.len() - 1));
        }
        // A filter this copy cannot express (its rows predate the column) drops the leg entirely,
        // so a filtered search never mixes in unfiltered rows.
        if let Some(v) = &self.speaker {
            if !has("speaker") {
                return None;
            }
            params.push(Box::new(v.clone()));
            conds.push(format!("speaker = ${}", first + params.len() - 1));
        }
        if let Some(v) = &self.topic {
            if has("topics") {
                params.push(Box::new(v.clone()));
                conds.push(format!("${} = any(topics)", first + params.len() - 1));
            } else if has("topic") {
                params.push(Box::new(v.clone()));
                conds.push(format!("topic = ${}", first + params.len() - 1));
            } else {
                return None;
            }
        }
        Some((conds.iter().map(|c| format!(" and ({c})")).collect::<String>(), params))
    }

    /// Where clause for Weaviate; `props` are the collection's current properties.
    fn weaviate_where(&self, props: &[String]) -> Option<Option<Value>> {
        let has = |p: &str| props.iter().any(|x| x == p);
        let mut ops = Vec::new();
        match self.person.as_str() {
            "katrina" => ops.push(json!({"path": ["katrina_conf"], "operator": "Equal", "valueText": "strong"})),
            "daughter" => ops.push(json!({"path": ["daughter_conf"], "operator": "ContainsAny", "valueText": ["strong", "medium", "weak"]})),
            _ => {}
        }
        if let Some(d) = self.from {
            ops.push(json!({"path": ["sort_ts"], "operator": "GreaterThanEqual", "valueDate": format!("{d}T00:00:00Z")}));
        }
        if let Some(d) = self.to {
            let next = d.succ_opt().unwrap_or(d);
            ops.push(json!({"path": ["sort_ts"], "operator": "LessThan", "valueDate": format!("{next}T00:00:00Z")}));
        }
        if let Some(f) = &self.source_format {
            ops.push(json!({"path": ["source_format"], "operator": "Equal", "valueText": f}));
        }
        if let Some(v) = &self.speaker {
            if !has("speaker") {
                return None;
            }
            ops.push(json!({"path": ["speaker"], "operator": "Equal", "valueText": v}));
        }
        if let Some(v) = &self.topic {
            if has("topics") {
                ops.push(json!({"path": ["topics"], "operator": "ContainsAny", "valueText": [v]}));
            } else if has("topic") {
                ops.push(json!({"path": ["topic"], "operator": "Equal", "valueText": v}));
            } else {
                return None;
            }
        }
        Some(match ops.len() {
            0 => None,
            1 => ops.pop(),
            _ => Some(json!({"operator": "And", "operands": ops})),
        })
    }
}

fn params_ref<'a>(p: &'a [Box<dyn tokio_postgres::types::ToSql + Sync + Send>]) -> Vec<&'a (dyn tokio_postgres::types::ToSql + Sync)> {
    p.iter().map(|b| b.as_ref() as &(dyn tokio_postgres::types::ToSql + Sync)).collect()
}

// ---------------------------------------------------------------------------------------------
// Weaviate leg
// ---------------------------------------------------------------------------------------------

async fn weaviate_props() -> Result<Vec<String>, String> {
    static CACHE: OnceLock<Mutex<Option<(Instant, Vec<String>)>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(None));
    if let Some((at, v)) = cache.lock().unwrap_or_else(|e| e.into_inner()).as_ref() {
        if at.elapsed() < Duration::from_secs(120) {
            return Ok(v.clone());
        }
    }
    let v: Value = http()
        .get(format!("{}/v1/schema/{}", weaviate_url(), collection()))
        .send()
        .await
        .map_err(|e| format!("Weaviate schema: {e}"))?
        .json()
        .await
        .map_err(|e| format!("Weaviate schema: {e}"))?;
    let props: Vec<String> = v
        .get("properties")
        .and_then(|p| p.as_array())
        .map(|a| a.iter().filter_map(|p| p.get("name").and_then(|n| n.as_str()).map(String::from)).collect())
        .unwrap_or_default();
    if props.is_empty() {
        return Err(format!("collection {} has no properties yet", collection()));
    }
    *cache.lock().unwrap_or_else(|e| e.into_inner()) = Some((Instant::now(), props.clone()));
    Ok(props)
}

/// Fields the panel uses; only the ones the collection actually has are requested.
const WANTED: &[&str] = &[
    "dedup_key", "sort_ts", "sender", "recipients", "participants", "conversation_title", "conversation_id",
    "source_format", "event_kind", "direction", "katrina_conf", "katrina_ref_type", "daughter_conf",
    "catrina_class", "n_sources", "vault_key", "catalog_path", "sha1", "extractor", "ingest_run_id",
    "indexed_at", "ts_original", "tz_status", "speaker", "service", "turn_index", "topic", "topics", "body",
];

async fn embed_query(query: &str) -> Result<Vec<f32>, String> {
    let key_file = std::env::var("NVIDIA_API_KEY_FILE").map_err(|_| "NVIDIA_API_KEY_FILE is not configured".to_string())?;
    let key = std::fs::read_to_string(&key_file).map_err(|e| format!("cannot read the NVIDIA key file: {e}"))?;
    let url = format!("{}/embeddings", env_or("INTAKE_CHAT_INDEX_NIM_URL", "https://integrate.api.nvidia.com/v1").trim_end_matches('/'));
    let resp = http()
        .post(url)
        .bearer_auth(key.trim())
        .timeout(Duration::from_secs(6))
        .json(&json!({
            "model": env_or("INTAKE_CHAT_INDEX_EMBED_MODEL", "nvidia/nemotron-3-embed-1b"),
            "input": [query], "input_type": "query", "encoding_format": "float", "truncate": "END",
        }))
        .send()
        .await
        .map_err(|e| format!("embedding request: {e}"))?;
    if !resp.status().is_success() {
        return Err(format!("embedding service answered {}", resp.status()));
    }
    let v: Value = resp.json().await.map_err(|e| e.to_string())?;
    v.pointer("/data/0/embedding")
        .and_then(|e| e.as_array())
        .map(|a| a.iter().filter_map(|x| x.as_f64()).map(|x| x as f32).collect::<Vec<f32>>())
        .filter(|a| !a.is_empty())
        .ok_or_else(|| "embedding service returned no vector".into())
}

/// JSON -> GraphQL input literal (object keys unquoted, operator enums bare).
fn graphql_literal(v: &Value) -> String {
    match v {
        Value::Object(m) => {
            let parts: Vec<String> = m
                .iter()
                .map(|(k, x)| {
                    if k == "operator" {
                        format!("{k}: {}", x.as_str().unwrap_or("Equal"))
                    } else {
                        format!("{k}: {}", graphql_literal(x))
                    }
                })
                .collect();
            format!("{{ {} }}", parts.join(", "))
        }
        Value::Array(a) => format!("[{}]", a.iter().map(graphql_literal).collect::<Vec<_>>().join(", ")),
        other => other.to_string(),
    }
}

/// Weaviate hybrid search. `Ok(None)` means this leg cannot express the filters.
async fn weaviate_search(query: &str, f: &Filters, limit: i64) -> Result<Option<(Vec<Value>, &'static str, Option<String>)>, String> {
    let coll = collection();
    let props = weaviate_props().await?;
    let Some(where_clause) = f.weaviate_where(&props) else { return Ok(None) };
    let fields: Vec<&str> = WANTED.iter().copied().filter(|w| props.iter().any(|p| p == w)).collect();
    if fields.is_empty() {
        return Err(format!("collection {coll} has none of the expected chat-event properties"));
    }
    let (search, mode, note) = match embed_query(query).await {
        Ok(vec) => (
            format!(
                "hybrid: {{ query: {}, alpha: 0.5, targetVectors: [\"text_nim\"], vector: {} }}",
                serde_json::to_string(query).unwrap_or_default(),
                serde_json::to_string(&vec).unwrap_or_default()
            ),
            "hybrid",
            None,
        ),
        Err(e) => (
            format!("bm25: {{ query: {} }}", serde_json::to_string(query).unwrap_or_default()),
            "keyword",
            Some(format!("meaning-based ranking is off right now ({e}); words were matched instead")),
        ),
    };
    let filter = where_clause.map(|w| format!(", where: {}", graphql_literal(&w))).unwrap_or_default();
    let gql = format!(
        "{{ Get {{ {coll}(limit: {limit}, {search}{filter}) {{ {} _additional {{ score }} }} }} }}",
        fields.join(" ")
    );
    let v: Value = http()
        .post(format!("{}/v1/graphql", weaviate_url()))
        .json(&json!({"query": gql}))
        .send()
        .await
        .map_err(|e| format!("Weaviate: {e}"))?
        .json()
        .await
        .map_err(|e| format!("Weaviate: {e}"))?;
    if let Some(err) = v.get("errors") {
        return Err(format!("Weaviate: {}", err.to_string().chars().take(300).collect::<String>()));
    }
    let objs = v.pointer(&format!("/data/Get/{coll}")).and_then(|a| a.as_array()).cloned().unwrap_or_default();
    Ok(Some((objs, mode, note)))
}

// ---------------------------------------------------------------------------------------------
// Hits
// ---------------------------------------------------------------------------------------------

fn tags(katrina_conf: Option<&str>, katrina_ref: Option<&str>, daughter_conf: Option<&str>) -> Vec<Value> {
    let mut out = Vec::new();
    match katrina_conf {
        Some("strong") if katrina_ref != Some("group_participant") && katrina_ref.is_some() => {
            out.push(json!({"tag": "katrina", "confidence": "strong"}))
        }
        Some(c) => out.push(json!({"tag": "katrina", "confidence": c, "ref_type": katrina_ref})),
        None => {}
    }
    if let Some(c) = daughter_conf {
        out.push(json!({"tag": "daughter", "confidence": c}));
    }
    out
}

/// An excerpt around the first query word in the body, matches wrapped in ⟦ ⟧ — the same marks the
/// Postgres leg's `ts_headline` emits, so the panel renders both the same way.
fn snippet(body: &str, query: &str) -> String {
    let terms: Vec<String> = query
        .to_lowercase()
        .split(|c: char| !c.is_alphanumeric() && c != '\'')
        .filter(|t| t.len() > 1)
        .map(String::from)
        .collect();
    let body_lc = body.to_lowercase();
    let start = terms.iter().filter_map(|t| body_lc.find(t.as_str())).min().unwrap_or(0);
    let from = body
        .char_indices()
        .map(|(i, _)| i)
        .filter(|i| *i <= start.saturating_sub(80))
        .next_back()
        .unwrap_or(0);
    let to = body.char_indices().map(|(i, _)| i).find(|i| *i >= from + 300).unwrap_or(body.len());
    let window = &body[from..to];
    let win_lc = window.to_lowercase();
    let mut marks: Vec<(usize, usize)> = Vec::new();
    for t in &terms {
        let mut at = 0;
        while let Some(i) = win_lc[at..].find(t.as_str()) {
            let a = at + i;
            marks.push((a, a + t.len()));
            at = a + t.len();
        }
    }
    marks.sort_unstable();
    let mut out = String::new();
    if from > 0 {
        out.push('…');
    }
    let mut cursor = 0;
    for (a, b) in marks {
        if a < cursor || !window.is_char_boundary(a) || !window.is_char_boundary(b) {
            continue;
        }
        out.push_str(&window[cursor..a]);
        out.push('\u{27E6}');
        out.push_str(&window[a..b]);
        out.push('\u{27E7}');
        cursor = b;
    }
    out.push_str(&window[cursor..]);
    if to < body.len() {
        out.push('…');
    }
    out.trim().to_string()
}

fn str_of(o: &Value, k: &str) -> Option<String> {
    o.get(k).and_then(|v| v.as_str()).map(String::from).filter(|s| !s.is_empty())
}

fn hit_from_weaviate(o: &Value, query: &str, b2_root: &std::path::Path) -> (String, Value) {
    let key = str_of(o, "dedup_key").unwrap_or_default();
    let body = o.get("body").and_then(|b| b.as_str()).unwrap_or("");
    let date = str_of(o, "sort_ts").map(|t| t.replace('T', " ").chars().take(16).collect::<String>());
    let topics = o
        .get("topics")
        .and_then(|t| t.as_array())
        .map(|a| a.iter().filter_map(|x| x.as_str()).collect::<Vec<_>>().join(", "))
        .or_else(|| str_of(o, "topic"));
    (
        key.clone(),
        json!({
            "dedup_key": key,
            "date": date,
            "sender": str_of(o, "sender"),
            "recipients": str_of(o, "recipients"),
            "participants": o.get("participants").and_then(|p| p.as_array())
                .map(|a| a.iter().filter_map(|x| x.as_str()).collect::<Vec<_>>().join(" | "))
                .or_else(|| str_of(o, "participants")),
            "source_format": str_of(o, "source_format"),
            "event_kind": str_of(o, "event_kind"),
            "conversation_title": str_of(o, "conversation_title"),
            "direction": str_of(o, "direction"),
            "speaker": str_of(o, "speaker"),
            "service": str_of(o, "service"),
            "topics": topics,
            "n_sources": o.get("n_sources").and_then(|n| n.as_i64()),
            "tags": tags(str_of(o, "katrina_conf").as_deref(), str_of(o, "katrina_ref_type").as_deref(), str_of(o, "daughter_conf").as_deref()),
            "snippet": snippet(body, query),
            "vault_key": str_of(o, "vault_key"),
            "source_path": str_of(o, "vault_key").map(|k| b2_root.join(k).display().to_string()),
            "catalog_path": str_of(o, "catalog_path"),
        }),
    )
}

/// Columns the Postgres copy has right now (it gains columns as the ingest evolves).
async fn pg_columns(engine: &Engine, t: &str) -> Result<Vec<String>, String> {
    static CACHE: OnceLock<Mutex<Option<(Instant, Vec<String>)>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(None));
    if let Some((at, v)) = cache.lock().unwrap_or_else(|e| e.into_inner()).as_ref() {
        if at.elapsed() < Duration::from_secs(120) {
            return Ok(v.clone());
        }
    }
    let (schema, name) = t.split_once('.').unwrap_or(("public", t));
    let client = pg(engine).await.map_err(|e| e.1)?;
    let cols: Vec<String> = client
        .query(
            "select column_name from information_schema.columns where table_schema = $1 and table_name = $2",
            &[&schema, &name],
        )
        .await
        .map_err(|e| e.to_string())?
        .iter()
        .map(|r| r.get::<_, String>(0))
        .collect();
    *cache.lock().unwrap_or_else(|e| e.into_inner()) = Some((Instant::now(), cols.clone()));
    Ok(cols)
}

/// Postgres leg: full-text search over the same events. `Ok(None)` = this copy cannot express the
/// filters (a column it does not have yet).
async fn pg_search(
    engine: &Arc<Engine>,
    t: &str,
    query: &str,
    f: &Filters,
    limit: i64,
) -> Result<Option<(Vec<(String, Value)>, i64)>, String> {
    let cols = pg_columns(engine, t).await?;
    let Some((cond, params)) = f.sql(3, &cols) else { return Ok(None) };
    let client = pg(engine).await.map_err(|e| e.1)?;
    let sql = format!(
        "select dedup_key, to_char(sort_ts, 'YYYY-MM-DD HH24:MI'), sender, recipients, participants, source_format, \
                event_kind, conversation_title, katrina_conf, katrina_ref_type, daughter_conf, vault_key, catalog_rel, \
                n_sources, direction, \
                ts_headline('english', left(coalesce(body, ''), 20000), q, \
                            'MaxWords=30, MinWords=12, MaxFragments=1, StartSel=\u{27E6}, StopSel=\u{27E7}'), \
                count(*) over () \
         from {t}, websearch_to_tsquery('english', $1) q \
         where to_tsvector('english', coalesce(body, '')) @@ q{cond} \
         order by ts_rank_cd(to_tsvector('english', coalesce(body, '')), q) desc, sort_ts desc limit $2"
    );
    let mut all: Vec<&(dyn tokio_postgres::types::ToSql + Sync)> = vec![&query, &limit];
    all.extend(params_ref(&params));
    let rows = client.query(&sql, &all).await.map_err(|e| e.to_string())?;
    let b2_root = engine.catalog.as_ref().map(|c| c.b2_root.clone()).unwrap_or_else(|| engine.mount_root.join("b2/salem-data"));
    let total = rows.first().map(|r| r.get::<_, i64>(16)).unwrap_or(0);
    let hits = rows
        .iter()
        .map(|r| {
            let key: String = r.get(0);
            let vault_key: Option<String> = r.get(11);
            let kc: Option<String> = r.get(8);
            let kr: Option<String> = r.get(9);
            let dc: Option<String> = r.get(10);
            let snip: String = r.get(15);
            (
                key.clone(),
                json!({
                    "dedup_key": key,
                    "date": r.get::<_, Option<String>>(1),
                    "sender": r.get::<_, Option<String>>(2),
                    "recipients": r.get::<_, Option<String>>(3),
                    "participants": r.get::<_, Option<String>>(4),
                    "source_format": r.get::<_, Option<String>>(5),
                    "event_kind": r.get::<_, Option<String>>(6),
                    "conversation_title": r.get::<_, Option<String>>(7),
                    "direction": r.get::<_, Option<String>>(14),
                    "n_sources": r.get::<_, Option<i32>>(13),
                    "tags": tags(kc.as_deref(), kr.as_deref(), dc.as_deref()),
                    "snippet": if snip.chars().count() > 320 { snip.chars().take(320).collect::<String>() + "…" } else { snip },
                    "vault_key": vault_key.clone(),
                    "source_path": vault_key.map(|k| b2_root.join(k).display().to_string()),
                    "catalog_path": r.get::<_, Option<String>>(12).map(|c| format!("{}{c}", crate::catalog::SCHEME)),
                }),
            )
        })
        .collect();
    Ok(Some((hits, total)))
}

/// Fill the fields one leg does not have from the other leg's copy of the same event.
fn merge_into(base: &mut Value, other: &Value) {
    let (Value::Object(b), Value::Object(o)) = (base, other) else { return };
    for (k, v) in o {
        let missing = b.get(k).map(|x| x.is_null() || x == "").unwrap_or(true);
        if missing && !v.is_null() {
            b.insert(k.clone(), v.clone());
        }
    }
}

// ---------------------------------------------------------------------------------------------
// Scope and search
// ---------------------------------------------------------------------------------------------

async fn weaviate_count() -> Result<i64, String> {
    let q = format!("{{ Aggregate {{ {} {{ meta {{ count }} }} }} }}", collection());
    let v: Value = http()
        .post(format!("{}/v1/graphql", weaviate_url()))
        .json(&json!({"query": q}))
        .send()
        .await
        .map_err(|e| e.to_string())?
        .json()
        .await
        .map_err(|e| e.to_string())?;
    v.pointer(&format!("/data/Aggregate/{}/0/meta/count", collection()))
        .and_then(|c| c.as_i64())
        .ok_or_else(|| "unexpected Weaviate answer".to_string())
}

/// What the one search box covers, in plain words.
pub async fn index_info(engine: &Arc<Engine>) -> Outcome {
    static CACHE: OnceLock<Mutex<Option<(Instant, Value)>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(None));
    if let Some((at, v)) = cache.lock().unwrap_or_else(|e| e.into_inner()).as_ref() {
        if at.elapsed() < Duration::from_secs(120) {
            return Ok(v.clone());
        }
    }
    let (semantic, semantic_error) = match weaviate_count().await {
        Ok(n) => (Some(n), None),
        Err(e) => (None, Some(e)),
    };
    let props = weaviate_props().await.unwrap_or_default();
    let mut events = semantic;
    let mut first = None;
    let mut last = None;
    let mut katrina = None;
    let mut daughter = None;
    let mut formats: Vec<Value> = Vec::new();
    let mut pg_events = None;
    if let Some(t) = table()? {
        if let Ok(client) = pg(engine).await {
            if let Ok(row) = client
                .query_one(
                    &format!(
                        "select count(*), min(sort_ts)::date::text, max(sort_ts)::date::text, \
                         count(*) filter (where katrina_conf = 'strong' and katrina_ref_type <> 'group_participant'), \
                         count(*) filter (where daughter_conf is not null) from {t}"
                    ),
                    &[],
                )
                .await
            {
                pg_events = Some(row.get::<_, i64>(0));
                events = Some(events.unwrap_or(0).max(row.get::<_, i64>(0)));
                first = row.get::<_, Option<String>>(1);
                last = row.get::<_, Option<String>>(2);
                katrina = Some(row.get::<_, i64>(3));
                daughter = Some(row.get::<_, i64>(4));
            }
            if let Ok(rows) = client
                .query(&format!("select source_format, count(*) from {t} group by 1 order by 2 desc"), &[])
                .await
            {
                formats = rows
                    .iter()
                    .map(|r| json!({"format": r.get::<_, Option<String>>(0), "events": r.get::<_, i64>(1)}))
                    .collect();
            }
        }
    }
    // Speaker / topic filters appear only once the ingest has written those properties.
    let mut speakers: Vec<String> = Vec::new();
    let mut topics: Vec<String> = Vec::new();
    if props.iter().any(|p| p == "speaker") {
        speakers = weaviate_values("speaker").await.unwrap_or_default();
    }
    for name in ["topics", "topic"] {
        if props.iter().any(|p| p == name) {
            topics = weaviate_values(name).await.unwrap_or_default();
            break;
        }
    }
    let v = json!({
        "available": events.is_some(),
        "name": "all chats",
        "events": events,
        "semantic_events": semantic,
        "staging_events": pg_events,
        "semantic_error": semantic_error,
        "first": first,
        "last": last,
        "katrina_events": katrina,
        "daughter_events": daughter,
        "formats": formats,
        "speakers": speakers,
        "topics": topics,
        "properties": props,
    });
    *cache.lock().unwrap_or_else(|e| e.into_inner()) = Some((Instant::now(), v.clone()));
    Ok(v)
}

/// Distinct values of a text property, for a filter list (empty when the property is new/unused).
async fn weaviate_values(prop: &str) -> Result<Vec<String>, String> {
    let coll = collection();
    let gql = format!("{{ Aggregate {{ {coll}(groupBy: [\"{prop}\"]) {{ groupedBy {{ value }} meta {{ count }} }} }} }}");
    let v: Value = http()
        .post(format!("{}/v1/graphql", weaviate_url()))
        .json(&json!({"query": gql}))
        .send()
        .await
        .map_err(|e| e.to_string())?
        .json()
        .await
        .map_err(|e| e.to_string())?;
    Ok(v.pointer(&format!("/data/Aggregate/{coll}"))
        .and_then(|a| a.as_array())
        .map(|a| {
            a.iter()
                .filter_map(|g| g.pointer("/groupedBy/value").and_then(|x| x.as_str()).map(String::from))
                .filter(|s| !s.is_empty() && s.len() < 64)
                .take(40)
                .collect()
        })
        .unwrap_or_default())
}

pub async fn search(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let started = Instant::now();
    let query = s(args, "query", "query").ok_or_else(|| bad("query is required"))?;
    if query.chars().count() > 500 {
        return Err(bad("query is longer than 500 characters"));
    }
    let f = Filters::parse(args)?;
    let limit = args.get("limit").and_then(|v| v.as_i64()).unwrap_or(50).clamp(1, 100);
    let fetch = limit * 2;
    let b2_root = engine.catalog.as_ref().map(|c| c.b2_root.clone()).unwrap_or_else(|| engine.mount_root.join("b2/salem-data"));
    let pg_table = table()?;

    let wv_started = Instant::now();
    let wv = async {
        let r = tokio::time::timeout(Duration::from_secs(12), weaviate_search(&query, &f, fetch))
            .await
            .unwrap_or_else(|_| Err("the semantic index did not answer within 12 s".into()));
        (r, wv_started.elapsed().as_millis())
    };
    let pg_started = Instant::now();
    let pgl = async {
        let r = match &pg_table {
            Some(t) => tokio::time::timeout(Duration::from_secs(12), pg_search(engine, t, &query, &f, fetch))
                .await
                .unwrap_or_else(|_| Err("the word index did not answer within 12 s".into())),
            None => Ok(None),
        };
        (r, pg_started.elapsed().as_millis())
    };
    let ((wv_r, wv_ms), (pg_r, pg_ms)) = tokio::join!(wv, pgl);

    let mut notes: Vec<String> = Vec::new();
    let mut fused: HashMap<String, (f64, Vec<&'static str>, Value)> = HashMap::new();
    let mut mode = "keyword";
    let mut legs = 0;
    if let Ok(Some((objs, m, note))) = &wv_r {
        legs += 1;
        mode = m;
        if let Some(n) = note {
            notes.push(n.clone());
        }
        for (i, o) in objs.iter().enumerate() {
            let (key, hit) = hit_from_weaviate(o, &query, &b2_root);
            if key.is_empty() {
                continue;
            }
            let e = fused.entry(key).or_insert((0.0, Vec::new(), hit));
            e.0 += 1.0 / (60.0 + i as f64 + 1.0);
            e.1.push("meaning");
        }
    } else if let Err(e) = &wv_r {
        notes.push(e.clone());
    }
    let mut matches_total: Option<i64> = None;
    if let Ok(Some((hits, total))) = &pg_r {
        legs += 1;
        matches_total = Some(*total);
        for (i, (key, hit)) in hits.iter().enumerate() {
            match fused.get_mut(key) {
                Some(e) => {
                    e.0 += 1.0 / (60.0 + i as f64 + 1.0);
                    e.1.push("words");
                    merge_into(&mut e.2, hit);
                }
                None => {
                    fused.insert(key.clone(), (1.0 / (60.0 + i as f64 + 1.0), vec!["words"], hit.clone()));
                }
            }
        }
    } else if let Err(e) = &pg_r {
        notes.push(e.clone());
    }
    if legs == 0 {
        return Err((503, if notes.is_empty() { "the chats index is not answering".into() } else { notes.join("; ") }));
    }

    let mut ranked: Vec<(f64, Vec<&'static str>, Value)> = fused.into_values().collect();
    ranked.sort_by(|a, b| b.0.partial_cmp(&a.0).unwrap_or(std::cmp::Ordering::Equal));
    let hits: Vec<Value> = ranked
        .into_iter()
        .take(limit as usize)
        .map(|(score, matched, mut hit)| {
            hit["score"] = json!(score);
            hit["matched_by"] = json!(matched);
            hit
        })
        .collect();
    let info = index_info(engine).await.ok();
    Ok(json!({
        "query": query,
        "scope": {
            "name": "all chats",
            "events": info.as_ref().and_then(|i| i.get("events")).cloned(),
            "semantic_events": info.as_ref().and_then(|i| i.get("semantic_events")).cloned(),
        },
        "hits": hits,
        "keyword_matches": matches_total,
        "semantic_mode": mode,
        "timings_ms": {"meaning": wv_ms, "words": pg_ms, "total": started.elapsed().as_millis()},
        "notes": notes,
    }))
}

// ---------------------------------------------------------------------------------------------
// One event in full (Metadata panel)
// ---------------------------------------------------------------------------------------------

pub async fn event(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let key = s(args, "dedupKey", "dedup_key").ok_or_else(|| bad("dedupKey is required"))?;
    // The index itself first; the Postgres copy adds the per-source provenance rows.
    let mut out = match weaviate_event(&key).await {
        Ok(Some(v)) => v,
        _ => Value::Null,
    };
    if let Some(t) = table()? {
        if let Ok(client) = pg(engine).await {
            if let Ok(Some(row)) = client
                .query_opt(
                    &format!(
                        "select to_char(sort_ts, 'YYYY-MM-DD HH24:MI:SS'), tz_status, ts_original, source_format, event_kind, \
                                conversation_title, sender, recipients, participants, direction, counterparty_phone, contact_name, \
                                body, attachments, katrina_ref_type, katrina_conf, daughter_conf, vault_key, catalog_rel, n_sources \
                         from {t} where dedup_key = $1"
                    ),
                    &[&key],
                )
                .await
            {
                let kc: Option<String> = row.get(15);
                let kr: Option<String> = row.get(14);
                let dc: Option<String> = row.get(16);
                let from_pg = json!({
                    "dedup_key": key,
                    "date": row.get::<_, Option<String>>(0),
                    "tz_status": row.get::<_, Option<String>>(1),
                    "ts_original": row.get::<_, Option<String>>(2),
                    "source_format": row.get::<_, Option<String>>(3),
                    "event_kind": row.get::<_, Option<String>>(4),
                    "conversation_title": row.get::<_, Option<String>>(5),
                    "sender": row.get::<_, Option<String>>(6),
                    "recipients": row.get::<_, Option<String>>(7),
                    "participants": row.get::<_, Option<String>>(8),
                    "direction": row.get::<_, Option<String>>(9),
                    "counterparty_phone": row.get::<_, Option<String>>(10),
                    "contact_name": row.get::<_, Option<String>>(11),
                    "body": row.get::<_, Option<String>>(12),
                    "attachments": row.get::<_, Option<String>>(13),
                    "tags": tags(kc.as_deref(), kr.as_deref(), dc.as_deref()),
                    "vault_key": row.get::<_, Option<String>>(17),
                    "catalog_rel": row.get::<_, Option<String>>(18),
                    "n_sources": row.get::<_, Option<i32>>(19),
                });
                if out.is_null() {
                    out = from_pg;
                } else {
                    merge_into(&mut out, &from_pg);
                }
            }
            let prov: Vec<Value> = client
                .query(
                    &format!(
                        "select source_format, extractor, vault_key, sha1, catalog_rel, member_path, record_index, ts_original \
                         from {} where dedup_key = $1 order by vault_key, record_index limit 50",
                        provenance_table(&t)
                    ),
                    &[&key],
                )
                .await
                .map(|rows| {
                    rows.iter()
                        .map(|r| {
                            json!({
                                "source_format": r.get::<_, Option<String>>(0), "extractor": r.get::<_, Option<String>>(1),
                                "vault_key": r.get::<_, Option<String>>(2), "sha1": r.get::<_, Option<String>>(3),
                                "catalog_rel": r.get::<_, Option<String>>(4), "member_path": r.get::<_, Option<String>>(5),
                                "record_index": r.get::<_, Option<i64>>(6), "ts_original": r.get::<_, Option<String>>(7),
                            })
                        })
                        .collect()
                })
                .unwrap_or_default();
            if let Value::Object(m) = &mut out {
                m.insert("provenance".into(), json!(prov));
            }
        }
    }
    if out.is_null() {
        return Err((404, "no such event in the chats index".to_string()));
    }
    if let Value::Object(m) = &mut out {
        m.entry("provenance").or_insert_with(|| json!([]));
        m.insert("index".into(), json!(collection()));
    }
    Ok(out)
}

async fn weaviate_event(key: &str) -> Result<Option<Value>, String> {
    let coll = collection();
    let props = weaviate_props().await?;
    let fields: Vec<&str> = WANTED.iter().copied().filter(|w| props.iter().any(|p| p == w)).collect();
    let gql = format!(
        "{{ Get {{ {coll}(limit: 1, where: {{ path: [\"dedup_key\"], operator: Equal, valueText: {} }}) {{ {} }} }} }}",
        serde_json::to_string(key).unwrap_or_default(),
        fields.join(" ")
    );
    let v: Value = http()
        .post(format!("{}/v1/graphql", weaviate_url()))
        .json(&json!({"query": gql}))
        .send()
        .await
        .map_err(|e| e.to_string())?
        .json()
        .await
        .map_err(|e| e.to_string())?;
    let Some(o) = v.pointer(&format!("/data/Get/{coll}/0")) else { return Ok(None) };
    let kc = str_of(o, "katrina_conf");
    let kr = str_of(o, "katrina_ref_type");
    let dc = str_of(o, "daughter_conf");
    let mut out = o.clone();
    if let Value::Object(m) = &mut out {
        m.insert("date".into(), json!(str_of(o, "sort_ts").map(|t| t.replace('T', " ").chars().take(19).collect::<String>())));
        m.insert("tags".into(), json!(tags(kc.as_deref(), kr.as_deref(), dc.as_deref())));
        if let Some(p) = o.get("participants").and_then(|p| p.as_array()) {
            m.insert("participants".into(), json!(p.iter().filter_map(|x| x.as_str()).collect::<Vec<_>>().join(" | ")));
        }
        if let Some(c) = str_of(o, "catalog_path") {
            m.insert("catalog_rel".into(), json!(c.trim_start_matches(crate::catalog::SCHEME)));
        }
    }
    Ok(Some(out))
}
