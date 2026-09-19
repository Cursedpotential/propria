//! Byline: Claude Code · Opus 5 · 2026-09-18
//!
//! Index-first Content Search (owner 2026-09-18 19:56 EDT "Search yea"). Answers come from the
//! indexes only, never from reading B2:
//!   * Postgres full-text search (GIN `to_tsvector('english', body)`) on the chat timeline table
//!     `raw_duck.chat_events_20260918` (catalog PG, read as metabase_ro);
//!   * Weaviate hybrid search (BM25 + NIM query vector, target vector `text_nim`) on
//!     `ChatEvents20260918`, same embedder as `casebible/tools/chat_timeline_mvp/embed_weaviate.py`
//!     (nvidia/nemotron-3-embed-1b, input_type=query).
//! The two ranked lists are merged by reciprocal rank fusion; every displayed row, and the person /
//! date / format filters, come from the PG table so both sources obey the same definitions as the
//! `timeline_katrina_20260918` / `timeline_daughter_20260918` views.

use crate::Engine;
use serde_json::{json, Map, Value};
use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

pub type Outcome = Result<Value, (u16, String)>;

fn env_or(name: &str, default: &str) -> String {
    std::env::var(name).ok().filter(|v| !v.trim().is_empty()).unwrap_or_else(|| default.to_string())
}

/// `schema.table` made of identifier characters only (it is spliced into SQL).
fn table() -> Result<String, (u16, String)> {
    let t = env_or("INTAKE_CHAT_INDEX_TABLE", "raw_duck.chat_events_20260918");
    if t.chars().all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '.') {
        Ok(t)
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
    C.get_or_init(|| reqwest::Client::builder().timeout(Duration::from_secs(10)).build().expect("http client"))
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
        .ok_or_else(|| (503, "the catalog database is not connected, so the chats index is unavailable".to_string()))?;
    let client = cat.ro_pool().get().await.map_err(|e| (503, format!("catalog connection: {e}")))?;
    client
        .batch_execute("set statement_timeout = '8s'")
        .await
        .map_err(|e| (503, format!("catalog connection: {e}")))?;
    Ok(client)
}

// ---------------------------------------------------------------------------------------------
// Filters (one definition, applied in PG; Weaviate gets a coarse pre-filter only)
// ---------------------------------------------------------------------------------------------

#[derive(Default, Clone)]
struct Filters {
    person: String,
    from: Option<chrono::NaiveDate>,
    to: Option<chrono::NaiveDate>,
    source_format: Option<String>,
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
        if let Some(f) = &source_format {
            if f.len() > 64 || !f.chars().all(|c| c.is_ascii_alphanumeric() || c == '_') {
                return Err(bad("sourceFormat is not a known format name"));
            }
        }
        Ok(Filters { person, from: date("from")?, to: date("to")?, source_format })
    }

    /// SQL conditions + parameters, numbered from `first`.
    fn sql(&self, first: usize) -> (String, Vec<Box<dyn tokio_postgres::types::ToSql + Sync + Send>>) {
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
        let sql = conds.iter().map(|c| format!(" and ({c})")).collect::<String>();
        (sql, params)
    }

    fn weaviate_where(&self) -> Option<Value> {
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
        match ops.len() {
            0 => None,
            1 => ops.pop(),
            _ => Some(json!({"operator": "And", "operands": ops})),
        }
    }
}

fn params_ref<'a>(p: &'a [Box<dyn tokio_postgres::types::ToSql + Sync + Send>]) -> Vec<&'a (dyn tokio_postgres::types::ToSql + Sync)> {
    p.iter().map(|b| b.as_ref() as &(dyn tokio_postgres::types::ToSql + Sync)).collect()
}

// ---------------------------------------------------------------------------------------------
// Scope ("Searching: chats index (N events)")
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
        .ok_or_else(|| format!("unexpected Weaviate answer: {}", v.to_string().chars().take(200).collect::<String>()))
}

pub async fn index_info(engine: &Arc<Engine>) -> Outcome {
    static CACHE: OnceLock<Mutex<Option<(Instant, Value)>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(None));
    if let Some((at, v)) = cache.lock().unwrap_or_else(|e| e.into_inner()).as_ref() {
        if at.elapsed() < Duration::from_secs(300) {
            return Ok(v.clone());
        }
    }
    let t = table()?;
    let client = pg(engine).await?;
    let totals = client
        .query_one(
            &format!(
                "select count(*), min(sort_ts)::date::text, max(sort_ts)::date::text, \
                 count(*) filter (where katrina_conf = 'strong' and katrina_ref_type <> 'group_participant'), \
                 count(*) filter (where daughter_conf is not null) from {t}"
            ),
            &[],
        )
        .await
        .map_err(|e| (503, format!("chats index: {e}")))?;
    let formats: Vec<Value> = client
        .query(&format!("select source_format, count(*) from {t} group by 1 order by 2 desc"), &[])
        .await
        .map_err(|e| (503, format!("chats index: {e}")))?
        .iter()
        .map(|r| json!({"format": r.get::<_, Option<String>>(0), "events": r.get::<_, i64>(1)}))
        .collect();
    let (semantic, semantic_error) = match weaviate_count().await {
        Ok(n) => (Some(n), None),
        Err(e) => (None, Some(e)),
    };
    let v = json!({
        "available": true,
        "name": "chats index",
        "table": t,
        "collection": collection(),
        "events": totals.get::<_, i64>(0),
        "first": totals.get::<_, Option<String>>(1),
        "last": totals.get::<_, Option<String>>(2),
        "katrina_events": totals.get::<_, i64>(3),
        "daughter_events": totals.get::<_, i64>(4),
        "semantic_events": semantic,
        "semantic_error": semantic_error,
        "formats": formats,
    });
    *cache.lock().unwrap_or_else(|e| e.into_inner()) = Some((Instant::now(), v.clone()));
    Ok(v)
}

// ---------------------------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------------------------

async fn fts_keys(engine: &Engine, query: &str, f: &Filters, limit: i64) -> Result<(Vec<String>, i64), String> {
    let t = table().map_err(|e| e.1)?;
    let client = pg(engine).await.map_err(|e| e.1)?;
    let (cond, params) = f.sql(3);
    let sql = format!(
        "select dedup_key, count(*) over () from {t}, websearch_to_tsquery('english', $1) q \
         where to_tsvector('english', coalesce(body, '')) @@ q{cond} \
         order by ts_rank_cd(to_tsvector('english', coalesce(body, '')), q) desc, sort_ts desc limit $2"
    );
    let mut all: Vec<&(dyn tokio_postgres::types::ToSql + Sync)> = vec![&query, &limit];
    all.extend(params_ref(&params));
    let rows = client.query(&sql, &all).await.map_err(|e| e.to_string())?;
    let total = rows.first().map(|r| r.get::<_, i64>(1)).unwrap_or(0);
    Ok((rows.iter().map(|r| r.get::<_, String>(0)).collect(), total))
}

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

/// Weaviate candidates: hybrid (BM25 + vector) when the query embeds, BM25 alone otherwise.
async fn weaviate_keys(query: &str, f: &Filters, limit: i64) -> Result<(Vec<String>, &'static str, Option<String>), String> {
    let coll = collection();
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
            "bm25",
            Some(format!("semantic ranking unavailable ({e}); keyword ranking used")),
        ),
    };
    let filter = f.weaviate_where().map(|w| format!(", where: {}", graphql_literal(&w))).unwrap_or_default();
    let gql = format!("{{ Get {{ {coll}(limit: {limit}, {search}{filter}) {{ dedup_key }} }} }}");
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
    let keys = v
        .pointer(&format!("/data/Get/{coll}"))
        .and_then(|a| a.as_array())
        .map(|a| a.iter().filter_map(|o| o.get("dedup_key").and_then(|k| k.as_str()).map(String::from)).collect())
        .unwrap_or_default();
    Ok((keys, mode, note))
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

pub async fn search(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let started = Instant::now();
    let query = s(args, "query", "query").ok_or_else(|| bad("query is required"))?;
    if query.chars().count() > 500 {
        return Err(bad("query is longer than 500 characters"));
    }
    let f = Filters::parse(args)?;
    let limit = args.get("limit").and_then(|v| v.as_i64()).unwrap_or(50).clamp(1, 100);
    let fetch = limit * 2;

    let t_fts = Instant::now();
    let fts = async {
        let r = fts_keys(engine, &query, &f, fetch).await;
        (r, t_fts.elapsed().as_millis())
    };
    let t_wv = Instant::now();
    let wv = async {
        let r = tokio::time::timeout(Duration::from_secs(9), weaviate_keys(&query, &f, fetch))
            .await
            .unwrap_or_else(|_| Err("Weaviate did not answer within 9 s".into()));
        (r, t_wv.elapsed().as_millis())
    };
    let ((fts_r, fts_ms), (wv_r, wv_ms)) = tokio::join!(fts, wv);

    let mut notes: Vec<String> = Vec::new();
    let mut fused: HashMap<String, (f64, Vec<&'static str>)> = HashMap::new();
    let mut fts_total = None;
    match &fts_r {
        Ok((keys, total)) => {
            fts_total = Some(*total);
            for (i, k) in keys.iter().enumerate() {
                let e = fused.entry(k.clone()).or_insert((0.0, Vec::new()));
                e.0 += 1.0 / (60.0 + i as f64 + 1.0);
                e.1.push("keyword");
            }
        }
        Err(e) => notes.push(format!("full-text search failed: {e}")),
    }
    let mut semantic_mode = "unavailable";
    match &wv_r {
        Ok((keys, mode, note)) => {
            semantic_mode = mode;
            if let Some(n) = note {
                notes.push(n.clone());
            }
            for (i, k) in keys.iter().enumerate() {
                let e = fused.entry(k.clone()).or_insert((0.0, Vec::new()));
                e.0 += 1.0 / (60.0 + i as f64 + 1.0);
                e.1.push(if *mode == "hybrid" { "semantic" } else { "bm25" });
            }
        }
        Err(e) => notes.push(format!("Weaviate search failed: {e}")),
    }
    if fts_r.is_err() && wv_r.is_err() {
        return Err((503, notes.join("; ")));
    }

    let mut ranked: Vec<(String, f64, Vec<&'static str>)> = fused.into_iter().map(|(k, (s, m))| (k, s, m)).collect();
    ranked.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
    let keys: Vec<String> = ranked.iter().map(|r| r.0.clone()).collect();

    // Display rows + the exact filters, from PG.
    let t = table()?;
    let client = pg(engine).await?;
    let (cond, params) = f.sql(3);
    let sql = format!(
        "select dedup_key, to_char(sort_ts, 'YYYY-MM-DD HH24:MI'), sender, recipients, participants, source_format, \
                event_kind, conversation_title, katrina_conf, katrina_ref_type, daughter_conf, vault_key, catalog_rel, \
                n_sources, direction, \
                ts_headline('english', left(coalesce(body, ''), 20000), websearch_to_tsquery('english', $2), \
                            'MaxWords=30, MinWords=12, MaxFragments=1, StartSel=\u{27E6}, StopSel=\u{27E7}') \
         from {t} where dedup_key = any($1){cond}"
    );
    let mut all: Vec<&(dyn tokio_postgres::types::ToSql + Sync)> = vec![&keys, &query];
    all.extend(params_ref(&params));
    let rows = client.query(&sql, &all).await.map_err(|e| (503, format!("chats index: {e}")))?;
    let b2_root = engine.catalog.as_ref().map(|c| c.b2_root.clone()).unwrap_or_else(|| engine.mount_root.join("b2/salem-data"));
    let mut by_key: HashMap<String, Value> = HashMap::new();
    for r in rows {
        let key: String = r.get(0);
        let vault_key: Option<String> = r.get(11);
        let catalog_rel: Option<String> = r.get(12);
        let kc: Option<String> = r.get(8);
        let kr: Option<String> = r.get(9);
        let dc: Option<String> = r.get(10);
        let snippet: String = r.get(15);
        let snippet: String = if snippet.chars().count() > 320 { snippet.chars().take(320).collect::<String>() + "…" } else { snippet };
        by_key.insert(
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
                "snippet": snippet,
                "vault_key": vault_key,
                "source_path": vault_key.as_ref().map(|k| b2_root.join(k).display().to_string()),
                "catalog_path": catalog_rel.map(|c| format!("{}{c}", crate::catalog::SCHEME)),
            }),
        );
    }
    let hits: Vec<Value> = ranked
        .into_iter()
        .filter_map(|(k, score, matched)| {
            by_key.remove(&k).map(|mut v| {
                v["score"] = json!(score);
                v["matched_by"] = json!(matched);
                v
            })
        })
        .take(limit as usize)
        .collect();
    let info = index_info(engine).await.ok();
    Ok(json!({
        "query": query,
        "scope": {
            "name": "chats index",
            "events": info.as_ref().and_then(|i| i.get("events")).cloned(),
            "semantic_events": info.as_ref().and_then(|i| i.get("semantic_events")).cloned(),
        },
        "hits": hits,
        "keyword_matches": fts_total,
        "semantic_mode": semantic_mode,
        "timings_ms": {"keyword": fts_ms, "semantic": wv_ms, "total": started.elapsed().as_millis()},
        "notes": notes,
    }))
}

/// One event in full (Metadata panel) with every source row it was deduplicated from.
pub async fn event(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let key = s(args, "dedupKey", "dedup_key").ok_or_else(|| bad("dedupKey is required"))?;
    let t = table()?;
    let client = pg(engine).await?;
    let row = client
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
        .map_err(|e| (503, format!("chats index: {e}")))?
        .ok_or_else(|| (404, "no such event in the chats index".to_string()))?;
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
    let kc: Option<String> = row.get(15);
    let kr: Option<String> = row.get(14);
    let dc: Option<String> = row.get(16);
    Ok(json!({
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
        "provenance": prov,
        "table": t,
    }))
}
