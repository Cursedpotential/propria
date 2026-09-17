//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! `catalog://` — the catalog (PG `casebible.raw_duck`) served as an ordinary filesystem.
//!
//! * Tree: `raw_duck.intake_catalog_fs_20260917` / `_dirs_20260917`
//!   (tracked build script `Consignatio/casebible/tools/intake_catalog_fs_20260917.sql`):
//!   `catalog://<source>/<scope>/<recorded path>`, each file resolving to its current vault object.
//! * Writes (owner option A, 2026-09-17 08:58 EDT): the real B2 operation happens on the vault object
//!   through the mount, then one row is appended to `raw_duck.intake_fs_ops_20260917`. Listings
//!   apply that overlay; `source_occurrences` and the vault snapshot rows are never rewritten.
//! * Reads use `metabase_ro`; the overlay is read and appended as `cb_agent` (its own table).

use deadpool_postgres::{Manager, ManagerConfig, Pool, RecyclingMethod};
use serde::Serialize;
use serde_json::{json, Value};
use std::collections::{BTreeMap, HashMap, HashSet};
use std::path::{Path, PathBuf};
use tokio::sync::RwLock;
use tokio_postgres::NoTls;

pub const SCHEME: &str = "catalog://";
const FS: &str = "raw_duck.intake_catalog_fs_20260917";
const DIRS: &str = "raw_duck.intake_catalog_dirs_20260917";
const OPS: &str = "raw_duck.intake_fs_ops_20260917";

#[derive(Clone, Debug)]
pub struct OpRow {
    pub op: String,
    pub kind: String,
    pub from_rel: Option<String>,
    pub to_rel: Option<String>,
    pub vault_key_from: Option<String>,
    pub vault_key_to: Option<String>,
}

pub struct Catalog {
    ro: Pool,
    ops: Pool,
    /// Mount path of the bucket root the vault keys are relative to.
    pub b2_root: PathBuf,
    overlay: RwLock<Vec<OpRow>>,
}

#[derive(Serialize, Clone, Debug)]
pub struct CatalogEntry {
    pub name: String,
    pub path: String,
    pub is_dir: bool,
    pub size: u64,
    pub modified: u64,
    pub file_type: String,
    pub mime_type: Option<String>,
    pub is_readonly: bool,
    /// Present only when something about the entry needs the owner's attention
    /// (e.g. the recorded occurrence has no current B2 object).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub intake_flag: Option<String>,
}

#[derive(Clone, Debug)]
pub struct FileRow {
    pub rel: String,
    pub size: i64,
    pub modtime: Option<chrono::DateTime<chrono::Utc>>,
    pub vault_key: Option<String>,
    pub resolution: String,
}

pub fn is_catalog(p: &str) -> bool {
    p.starts_with(SCHEME)
}

/// `catalog://a/b/` -> `a/b`; `catalog://` -> ``.
pub fn rel_of(p: &str) -> Result<String, String> {
    let rest = p.strip_prefix(SCHEME).ok_or("not a catalog:// path")?;
    let rel = rest.trim_matches('/').to_string();
    if rel.split('/').any(|s| s == "." || s == "..") || rel.contains('\0') {
        return Err("catalog path may not contain '.' or '..' segments".into());
    }
    Ok(rel)
}

pub fn path_of(rel: &str) -> String {
    format!("{SCHEME}{rel}")
}

pub fn parent_of(rel: &str) -> String {
    match rel.rfind('/') {
        Some(i) => rel[..i].to_string(),
        None => String::new(),
    }
}

pub fn name_of(rel: &str) -> String {
    match rel.rfind('/') {
        Some(i) => rel[i + 1..].to_string(),
        None => rel.to_string(),
    }
}

fn join(parent: &str, name: &str) -> String {
    if parent.is_empty() {
        name.to_string()
    } else {
        format!("{parent}/{name}")
    }
}

fn under(path: &str, dir: &str) -> Option<String> {
    if path == dir {
        Some(String::new())
    } else {
        path.strip_prefix(&format!("{dir}/")).map(|s| format!("/{s}"))
    }
}

fn read_secret(env: &str) -> Result<String, String> {
    let file = std::env::var(env).map_err(|_| format!("{env} is not configured"))?;
    std::fs::read_to_string(&file)
        .map(|s| s.trim().to_string())
        .map_err(|e| format!("cannot read {env}: {e}"))
}

fn pool(user_env: &str, pass_env: &str) -> Result<Pool, String> {
    let mut cfg = tokio_postgres::Config::new();
    cfg.host(&std::env::var("INTAKE_CATALOG_PG_HOST").unwrap_or_else(|_| "100.91.190.107".into()))
        .port(
            std::env::var("INTAKE_CATALOG_PG_PORT")
                .ok()
                .and_then(|p| p.parse().ok())
                .unwrap_or(5475),
        )
        .dbname(&std::env::var("INTAKE_CATALOG_PG_DB").unwrap_or_else(|_| "casebible".into()))
        .user(&std::env::var(user_env).map_err(|_| format!("{user_env} is not configured"))?)
        .password(read_secret(pass_env)?)
        .application_name("intake-engine")
        .connect_timeout(std::time::Duration::from_secs(10));
    let mgr = Manager::from_config(
        cfg,
        NoTls,
        ManagerConfig {
            recycling_method: RecyclingMethod::Fast,
        },
    );
    Pool::builder(mgr).max_size(6).build().map_err(|e| e.to_string())
}

impl Catalog {
    pub async fn connect() -> Result<Self, String> {
        let ro = pool("INTAKE_CATALOG_RO_USER", "INTAKE_CATALOG_RO_PASSWORD_FILE")?;
        let ops = pool("INTAKE_CATALOG_OPS_USER", "INTAKE_CATALOG_OPS_PASSWORD_FILE")?;
        let b2_root = PathBuf::from(
            std::env::var("INTAKE_B2_ROOT").unwrap_or_else(|_| "/srv/openlist/b2/salem-data".into()),
        );
        let cat = Catalog {
            ro,
            ops,
            b2_root,
            overlay: RwLock::new(Vec::new()),
        };
        cat.reload_overlay().await?;
        Ok(cat)
    }

    async fn reload_overlay(&self) -> Result<(), String> {
        let client = self.ops.get().await.map_err(|e| format!("catalog ops connection: {e}"))?;
        let rows = client
            .query(
                &format!("select op, kind, from_rel, to_rel, vault_key_from, vault_key_to from {OPS} order by op_id"),
                &[],
            )
            .await
            .map_err(|e| format!("read overlay: {e}"))?;
        let ops = rows
            .iter()
            .map(|r| OpRow {
                op: r.get(0),
                kind: r.get(1),
                from_rel: r.get(2),
                to_rel: r.get(3),
                vault_key_from: r.get(4),
                vault_key_to: r.get(5),
            })
            .collect();
        *self.overlay.write().await = ops;
        Ok(())
    }

    // ── overlay path algebra ────────────────────────────────────────────

    /// Apply one op forward to a path. `None` = the path no longer exists in the tree.
    fn fwd(op: &OpRow, cur: &str) -> Option<String> {
        let (Some(from), to) = (op.from_rel.as_deref(), op.to_rel.as_deref()) else {
            return Some(cur.to_string());
        };
        match op.op.as_str() {
            "rename" | "move" | "delete" | "relocate" => {
                let hit = if op.kind == "dir" {
                    under(cur, from)
                } else if cur == from {
                    Some(String::new())
                } else {
                    None
                };
                match (hit, to) {
                    (None, _) => Some(cur.to_string()),
                    (Some(rest), Some(to)) => Some(format!("{to}{rest}")),
                    (Some(_), None) => None,
                }
            }
            _ => Some(cur.to_string()),
        }
    }

    /// Reverse of `fwd` for one op: which earlier path became `cur`?
    fn back(op: &OpRow, cur: &str) -> Option<String> {
        match op.op.as_str() {
            "rename" | "move" => {
                let (Some(from), Some(to)) = (op.from_rel.as_deref(), op.to_rel.as_deref()) else {
                    return Some(cur.to_string());
                };
                let hit = if op.kind == "dir" {
                    under(cur, to)
                } else if cur == to {
                    Some(String::new())
                } else {
                    None
                };
                match hit {
                    Some(rest) => Some(format!("{from}{rest}")),
                    None => {
                        // The old location was vacated by this op.
                        let vacated = if op.kind == "dir" {
                            under(cur, from).is_some()
                        } else {
                            cur == from
                        };
                        if vacated {
                            None
                        } else {
                            Some(cur.to_string())
                        }
                    }
                }
            }
            "delete" | "relocate" => {
                let Some(from) = op.from_rel.as_deref() else {
                    return Some(cur.to_string());
                };
                let gone = if op.kind == "dir" { under(cur, from).is_some() } else { cur == from };
                if gone {
                    None
                } else {
                    Some(cur.to_string())
                }
            }
            "mkdir" | "import" => {
                if op.to_rel.as_deref() == Some(cur) {
                    None
                } else {
                    Some(cur.to_string())
                }
            }
            _ => Some(cur.to_string()),
        }
    }

    fn to_effective(ops: &[OpRow], base: &str) -> Option<String> {
        let mut cur = base.to_string();
        for op in ops {
            cur = Self::fwd(op, &cur)?;
        }
        Some(cur)
    }

    fn to_base(ops: &[OpRow], eff: &str) -> Option<String> {
        let mut cur = eff.to_string();
        for op in ops.iter().rev() {
            cur = Self::back(op, &cur)?;
        }
        Some(cur)
    }

    /// Current B2 key of a vault object after renames/relocations; `None` if deleted.
    fn current_key(ops: &[OpRow], key: &str) -> Option<String> {
        let mut cur = key.to_string();
        for op in ops {
            if op.kind == "file" && op.vault_key_from.as_deref() == Some(cur.as_str()) {
                match op.op.as_str() {
                    "rename" | "relocate" => {
                        if let Some(to) = &op.vault_key_to {
                            cur = to.clone();
                        }
                    }
                    "delete" => return None,
                    _ => {}
                }
            }
        }
        Some(cur)
    }

    /// Files brought into catalog:// from B2 (`import` ops): (effective path, current B2 key).
    fn imported_files(ops: &[OpRow]) -> Vec<(String, String)> {
        let mut out = Vec::new();
        for (i, op) in ops.iter().enumerate() {
            if op.op != "import" {
                continue;
            }
            let (Some(to), Some(key)) = (&op.to_rel, &op.vault_key_to) else { continue };
            let later = &ops[i + 1..];
            let (Some(eff), Some(cur)) = (Self::to_effective(later, to), Self::current_key(later, key)) else {
                continue;
            };
            out.push((eff, cur));
        }
        out
    }

    /// Directories created by `mkdir` ops, as they are named now.
    fn made_dirs(ops: &[OpRow]) -> Vec<String> {
        let mut out = Vec::new();
        for (i, op) in ops.iter().enumerate() {
            if op.op == "mkdir" {
                if let Some(to) = &op.to_rel {
                    if let Some(eff) = Self::to_effective(&ops[i + 1..], to) {
                        out.push(eff);
                    }
                }
            }
        }
        out
    }

    // ── queries ─────────────────────────────────────────────────────────

    async fn file_rows(&self, rels: &[String]) -> Result<HashMap<String, FileRow>, String> {
        if rels.is_empty() {
            return Ok(HashMap::new());
        }
        let client = self.ro.get().await.map_err(|e| format!("catalog connection: {e}"))?;
        let rows = client
            .query(
                &format!("select rel, size, modtime, vault_key, resolution from {FS} where rel = any($1)"),
                &[&rels],
            )
            .await
            .map_err(|e| format!("catalog query: {e}"))?;
        Ok(rows
            .iter()
            .map(|r| {
                let row = FileRow {
                    rel: r.get(0),
                    size: r.get::<_, Option<i64>>(1).unwrap_or(0),
                    modtime: r.get(2),
                    vault_key: r.get(3),
                    resolution: r.get(4),
                };
                (row.rel.clone(), row)
            })
            .collect())
    }

    async fn dir_exists_base(&self, rel: &str) -> Result<bool, String> {
        if rel.is_empty() {
            return Ok(true);
        }
        let client = self.ro.get().await.map_err(|e| format!("catalog connection: {e}"))?;
        let row = client
            .query_opt(&format!("select 1 from {DIRS} where rel = $1"), &[&rel])
            .await
            .map_err(|e| format!("catalog query: {e}"))?;
        Ok(row.is_some())
    }

    fn entry_for_file(&self, ops: &[OpRow], eff: &str, row: &FileRow) -> CatalogEntry {
        let name = name_of(eff);
        let p = Path::new(&name);
        let current = row.vault_key.as_deref().and_then(|k| Self::current_key(ops, k));
        let flag = match (row.resolution.as_str(), &row.vault_key, &current) {
            ("resolved", Some(_), Some(_)) => None,
            ("resolved", Some(_), None) => Some("The B2 object for this catalog entry was deleted".to_string()),
            ("no_b2_key", _, _) => Some("Recorded occurrence has no B2 copy (no b2_key in the catalog)".into()),
            ("b2_key_not_in_b2_objects", _, _) => {
                Some("Recorded B2 key is not in the B2 listing (b2_objects)".into())
            }
            (other, _, _) => Some(format!("No current vault object for this occurrence ({other})")),
        };
        CatalogEntry {
            name: name.clone(),
            path: path_of(eff),
            is_dir: false,
            size: row.size.max(0) as u64,
            modified: row.modtime.map(|t| t.timestamp().max(0) as u64).unwrap_or(0),
            file_type: xplorer::file_lib::get_file_type(p, false),
            mime_type: xplorer::file_lib::get_mime_type(p),
            is_readonly: false,
            intake_flag: flag,
        }
    }

    fn entry_for_dir(eff: &str, bytes: i64) -> CatalogEntry {
        CatalogEntry {
            name: name_of(eff),
            path: path_of(eff),
            is_dir: true,
            size: bytes.max(0) as u64,
            modified: 0,
            file_type: "directory".into(),
            mime_type: None,
            is_readonly: false,
            intake_flag: None,
        }
    }

    /// List an effective directory.
    pub async fn list(&self, dir_path: &str) -> Result<Vec<CatalogEntry>, String> {
        let dir = rel_of(dir_path)?;
        let ops = self.overlay.read().await.clone();
        let made = Self::made_dirs(&ops);
        let base_dir = Self::to_base(&ops, &dir);
        let exists = match &base_dir {
            Some(b) => self.dir_exists_base(b).await? || made.contains(&dir),
            None => made.contains(&dir),
        };
        if !exists {
            return Err(format!("Directory does not exist: {dir_path}"));
        }

        // Candidate base paths: children of the base dir, plus anything an op moved into this dir.
        let mut base_files: Vec<String> = Vec::new();
        let mut base_dirs: Vec<(String, i64)> = Vec::new();
        if let Some(b) = &base_dir {
            let client = self.ro.get().await.map_err(|e| format!("catalog connection: {e}"))?;
            for r in client
                .query(&format!("select rel from {FS} where parent = $1 order by name"), &[b])
                .await
                .map_err(|e| format!("catalog query: {e}"))?
            {
                base_files.push(r.get(0));
            }
            for r in client
                .query(
                    &format!("select rel, nested_bytes::bigint from {DIRS} where parent = $1 order by name"),
                    &[b],
                )
                .await
                .map_err(|e| format!("catalog query: {e}"))?
            {
                base_dirs.push((r.get(0), r.get(1)));
            }
        }
        for (i, op) in ops.iter().enumerate() {
            if !matches!(op.op.as_str(), "rename" | "move") {
                continue;
            }
            let Some(to) = &op.to_rel else { continue };
            let Some(eff) = Self::to_effective(&ops[i + 1..], to) else { continue };
            if parent_of(&eff) != dir {
                continue;
            }
            if let Some(base) = Self::to_base(&ops, &eff) {
                if op.kind == "dir" {
                    base_dirs.push((base, 0));
                } else {
                    base_files.push(base);
                }
            }
        }

        let mut out: BTreeMap<String, CatalogEntry> = BTreeMap::new();
        let rows = self.file_rows(&base_files).await?;
        for base in &base_files {
            let Some(eff) = Self::to_effective(&ops, base) else { continue };
            if parent_of(&eff) != dir {
                continue;
            }
            if let Some(row) = rows.get(base) {
                out.insert(eff.clone(), self.entry_for_file(&ops, &eff, row));
            }
        }
        for (base, bytes) in &base_dirs {
            let Some(eff) = Self::to_effective(&ops, base) else { continue };
            if parent_of(&eff) != dir {
                continue;
            }
            out.entry(eff.clone()).or_insert_with(|| Self::entry_for_dir(&eff, *bytes));
        }
        for (eff, key) in Self::imported_files(&ops).into_iter().filter(|(e, _)| parent_of(e) == dir) {
            let mount = self.b2_root.join(&key);
            let meta = tokio::fs::metadata(&mount).await.ok();
            let name = name_of(&eff);
            let p = Path::new(&name);
            out.insert(
                eff.clone(),
                CatalogEntry {
                    name: name.clone(),
                    path: path_of(&eff),
                    is_dir: false,
                    size: meta.as_ref().map(|m| m.len()).unwrap_or(0),
                    modified: meta
                        .as_ref()
                        .and_then(|m| m.modified().ok())
                        .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
                        .map(|d| d.as_secs())
                        .unwrap_or(0),
                    file_type: xplorer::file_lib::get_file_type(p, false),
                    mime_type: xplorer::file_lib::get_mime_type(p),
                    is_readonly: false,
                    intake_flag: if meta.is_some() {
                        None
                    } else {
                        Some("Added in Intake; its B2 object is missing".into())
                    },
                },
            );
        }
        for m in made.iter().filter(|m| parent_of(m) == dir) {
            out.entry(m.clone()).or_insert_with(|| Self::entry_for_dir(m, 0));
        }
        let mut list: Vec<CatalogEntry> = out.into_values().collect();
        list.sort_by(|a, b| b.is_dir.cmp(&a.is_dir).then(a.name.to_lowercase().cmp(&b.name.to_lowercase())));
        Ok(list)
    }

    /// What an effective path is right now.
    pub async fn stat(&self, path: &str) -> Result<Option<CatalogEntry>, String> {
        let rel = rel_of(path)?;
        if rel.is_empty() {
            return Ok(Some(Self::entry_for_dir("", 0)));
        }
        let parent = parent_of(&rel);
        let parent_path = path_of(&parent);
        match self.list(&parent_path).await {
            Ok(entries) => Ok(entries.into_iter().find(|e| e.path == path_of(&rel))),
            Err(_) => Ok(None),
        }
    }

    /// Resolve an effective catalog file to its B2 object on the mount.
    pub async fn resolve_file(&self, path: &str) -> Result<(PathBuf, String), String> {
        let rel = rel_of(path)?;
        let ops = self.overlay.read().await.clone();
        if let Some((_, key)) = Self::imported_files(&ops).into_iter().find(|(e, _)| *e == rel) {
            return Ok((self.b2_root.join(&key), key));
        }
        let base = Self::to_base(&ops, &rel).ok_or_else(|| format!("{path} does not exist"))?;
        let rows = self.file_rows(std::slice::from_ref(&base)).await?;
        let row = rows
            .get(&base)
            .ok_or_else(|| format!("{path} is not a file in the catalog"))?;
        let key = row.vault_key.as_deref().ok_or_else(|| {
            format!("{path} has no current B2 object ({}), so it cannot be opened", row.resolution)
        })?;
        let current = Self::current_key(&ops, key)
            .ok_or_else(|| format!("the B2 object behind {path} was deleted"))?;
        Ok((self.b2_root.join(&current), current))
    }

    /// Catalog history for a B2 object on the mount (metadata panel).
    pub async fn lookup_b2_path(&self, mount_path: &Path) -> Result<Value, String> {
        let key = mount_path
            .strip_prefix(&self.b2_root)
            .map_err(|_| format!("{} is not under {}", mount_path.display(), self.b2_root.display()))?
            .to_string_lossy()
            .to_string();
        let ops = self.overlay.read().await.clone();
        // Walk renames backwards so a renamed object still finds its original catalog rows.
        let mut keys = vec![key.clone()];
        let mut cur = key.clone();
        for op in ops.iter().rev() {
            if op.kind == "file" && op.vault_key_to.as_deref() == Some(cur.as_str()) {
                if let Some(from) = &op.vault_key_from {
                    cur = from.clone();
                    keys.push(cur.clone());
                }
            }
        }
        let client = self.ro.get().await.map_err(|e| format!("catalog connection: {e}"))?;
        let vault = client
            .query_opt(
                "select key, size, sha1 from raw_duck.vault_objects_20260916_r4 where key = any($1) limit 1",
                &[&keys],
            )
            .await
            .map_err(|e| format!("catalog query: {e}"))?;
        let rows = client
            .query(
                &format!(
                    "select rel, source, scope, path, source_id, size, modtime, md5, native_hash_kind, native_hash, \
                     disposition, b2_key_recorded, matched_origin, metadata, recorded_at, sha1 \
                     from {FS} where vault_key = any($1) order by source, scope, path limit 500"
                ),
                &[&keys],
            )
            .await
            .map_err(|e| format!("catalog query: {e}"))?;
        let occurrences: Vec<Value> = rows
            .iter()
            .map(|r| {
                let rel: String = r.get(0);
                let eff = Self::to_effective(&ops, &rel);
                json!({
                    "catalog_path": eff.map(|e| path_of(&e)),
                    "source": r.get::<_, String>(1),
                    "scope": r.get::<_, Option<String>>(2),
                    "path": r.get::<_, String>(3),
                    "source_id": r.get::<_, Option<String>>(4),
                    "size": r.get::<_, Option<i64>>(5),
                    "modtime": r.get::<_, Option<chrono::DateTime<chrono::Utc>>>(6),
                    "md5": r.get::<_, Option<String>>(7),
                    "native_hash_kind": r.get::<_, Option<String>>(8),
                    "native_hash": r.get::<_, Option<String>>(9),
                    "disposition": r.get::<_, Option<String>>(10),
                    "b2_key_recorded": r.get::<_, Option<String>>(11),
                    "matched_origin": r.get::<_, Option<String>>(12),
                    "metadata": r.get::<_, Option<Value>>(13),
                    "recorded_at": r.get::<_, Option<chrono::DateTime<chrono::Utc>>>(14),
                    "sha1": r.get::<_, Option<String>>(15),
                })
            })
            .collect();
        let history: Vec<Value> = ops
            .iter()
            .filter(|op| {
                keys.iter().any(|k| {
                    op.vault_key_from.as_deref() == Some(k.as_str()) || op.vault_key_to.as_deref() == Some(k.as_str())
                })
            })
            .map(|op| json!({"op": op.op, "from": op.from_rel, "to": op.to_rel, "vault_key_from": op.vault_key_from, "vault_key_to": op.vault_key_to}))
            .collect();
        Ok(json!({
            "b2_key": key,
            "vault_object": vault.map(|v| json!({
                "key": v.get::<_, String>(0), "size": v.get::<_, Option<i64>>(1), "sha1": v.get::<_, Option<String>>(2)
            })),
            "occurrences": occurrences,
            "count": rows.len(),
            "engine_ops": history,
            "catalog_tables": [FS, "raw_duck.vault_objects_20260916_r4", OPS],
        }))
    }

    // ── writes ──────────────────────────────────────────────────────────

    async fn append(&self, op: OpRow, b2_path_to: Option<String>, detail: Value) -> Result<(), String> {
        let client = self.ops.get().await.map_err(|e| format!("catalog ops connection: {e}"))?;
        client
            .execute(
                &format!(
                    "insert into {OPS} (op, kind, from_rel, to_rel, vault_key_from, vault_key_to, b2_path_to, detail) \
                     values ($1,$2,$3,$4,$5,$6,$7,$8)"
                ),
                &[&op.op, &op.kind, &op.from_rel, &op.to_rel, &op.vault_key_from, &op.vault_key_to, &b2_path_to, &detail],
            )
            .await
            .map_err(|e| format!("B2 change applied but recording it in {OPS} failed: {e}"))?;
        self.overlay.write().await.push(op);
        Ok(())
    }

    fn key_of_mount(&self, p: &Path) -> Option<String> {
        p.strip_prefix(&self.b2_root).ok().map(|k| k.to_string_lossy().to_string())
    }

    /// Effective files under an effective directory (recursive), for directory deletes.
    async fn files_under(&self, dir: &str) -> Result<Vec<String>, String> {
        let mut out = Vec::new();
        let mut stack = vec![dir.to_string()];
        while let Some(d) = stack.pop() {
            for e in self.list(&path_of(&d)).await? {
                let rel = rel_of(&e.path)?;
                if e.is_dir {
                    stack.push(rel);
                } else {
                    out.push(rel);
                }
            }
        }
        Ok(out)
    }

    /// rename / move inside catalog://.
    pub async fn move_within(&self, from_path: &str, to_path: &str) -> Result<(), String> {
        let from = rel_of(from_path)?;
        let to = rel_of(to_path)?;
        if from.is_empty() || to.is_empty() {
            return Err("the catalog root cannot be moved".into());
        }
        let entry = self.stat(from_path).await?.ok_or_else(|| format!("{from_path} does not exist"))?;
        if self.stat(to_path).await?.is_some() {
            return Err(format!("{to_path} already exists"));
        }
        let same_parent = parent_of(&from) == parent_of(&to);
        if !entry.is_dir && same_parent {
            // A rename renames the real B2 object too (same vault folder, new name).
            let (mount, key) = self.resolve_file(from_path).await?;
            let new_mount = mount.with_file_name(name_of(&to));
            if tokio::fs::metadata(&new_mount).await.is_ok() {
                return Err(format!("B2 already has {}", new_mount.display()));
            }
            tokio::fs::rename(&mount, &new_mount)
                .await
                .map_err(|e| format!("B2 rename failed: {e}"))?;
            let new_key = self.key_of_mount(&new_mount);
            return self
                .append(
                    OpRow { op: "rename".into(), kind: "file".into(), from_rel: Some(from), to_rel: Some(to), vault_key_from: Some(key), vault_key_to: new_key },
                    None,
                    json!({}),
                )
                .await;
        }
        // Directory renames and moves between catalog folders change where the entry sits in the
        // catalog tree; the vault object's own B2 location is unchanged.
        self.append(
            OpRow {
                op: if same_parent { "rename".into() } else { "move".into() },
                kind: if entry.is_dir { "dir".into() } else { "file".into() },
                from_rel: Some(from),
                to_rel: Some(to),
                vault_key_from: None,
                vault_key_to: None,
            },
            None,
            json!({"note": "catalog tree position only"}),
        )
        .await
    }

    /// Move a catalog file out to a B2 path: the vault object itself moves there.
    pub async fn relocate_out(&self, from_path: &str, dest: &Path) -> Result<(), String> {
        let from = rel_of(from_path)?;
        let entry = self.stat(from_path).await?.ok_or_else(|| format!("{from_path} does not exist"))?;
        if entry.is_dir {
            return Err("moving a whole catalog folder out to B2 is not supported; move its files".into());
        }
        let (mount, key) = self.resolve_file(from_path).await?;
        if tokio::fs::metadata(dest).await.is_ok() {
            return Err(format!("{} already exists", dest.display()));
        }
        tokio::fs::rename(&mount, dest)
            .await
            .map_err(|e| format!("B2 move failed: {e}"))?;
        self.append(
            OpRow { op: "relocate".into(), kind: "file".into(), from_rel: Some(from), to_rel: None, vault_key_from: Some(key), vault_key_to: self.key_of_mount(dest) },
            Some(dest.to_string_lossy().to_string()),
            json!({}),
        )
        .await
    }

    /// Bring a B2 file into catalog:// (copy or move). catalog:// entries must resolve to a real B2
    /// object, so the object is placed under `intake-catalog-added/<catalog path>` in the bucket
    /// (copied, or moved for a move) and an `import` row records the new entry.
    pub async fn import_in(&self, src: &Path, to_path: &str, is_move: bool) -> Result<(), String> {
        let to = rel_of(to_path)?;
        if to.is_empty() || !to.contains('/') {
            return Err("pick a folder inside catalog://<source>/ to add files to".into());
        }
        let meta = tokio::fs::metadata(src).await.map_err(|e| format!("{}: {e}", src.display()))?;
        if meta.is_dir() {
            return Err("adding a whole B2 folder to catalog:// is not supported; add its files".into());
        }
        if self.stat(to_path).await?.is_some() {
            return Err(format!("{to_path} already exists"));
        }
        let parent = parent_of(&to);
        if self.stat(&path_of(&parent)).await?.map(|e| e.is_dir) != Some(true) {
            return Err(format!("catalog folder {} does not exist", path_of(&parent)));
        }
        let key = format!("intake-catalog-added/{to}");
        let target = self.b2_root.join(&key);
        if tokio::fs::metadata(&target).await.is_ok() {
            return Err(format!("B2 already has {}", target.display()));
        }
        if let Some(dir) = target.parent() {
            tokio::fs::create_dir_all(dir).await.map_err(|e| format!("create {}: {e}", dir.display()))?;
        }
        if is_move {
            tokio::fs::rename(src, &target).await.map_err(|e| format!("B2 move failed: {e}"))?;
        } else {
            tokio::fs::copy(src, &target).await.map_err(|e| format!("B2 copy failed: {e}"))?;
        }
        self.append(
            OpRow { op: "import".into(), kind: "file".into(), from_rel: None, to_rel: Some(to), vault_key_from: self.key_of_mount(src), vault_key_to: Some(key) },
            Some(target.to_string_lossy().to_string()),
            json!({"moved": is_move, "source": src, "bytes": meta.len()}),
        )
        .await
    }

    /// Copy a catalog file out to a B2 path (bytes only; the catalog is unchanged apart from the log row).
    pub async fn copy_out(&self, from_path: &str, dest: &Path) -> Result<(), String> {
        let from = rel_of(from_path)?;
        let entry = self.stat(from_path).await?.ok_or_else(|| format!("{from_path} does not exist"))?;
        let pairs: Vec<(String, PathBuf)> = if entry.is_dir {
            let files = self.files_under(&from).await?;
            files
                .into_iter()
                .map(|f| {
                    let rest = f.strip_prefix(&from).unwrap_or(&f).trim_start_matches('/').to_string();
                    (f, dest.join(rest))
                })
                .collect()
        } else {
            vec![(from.clone(), dest.to_path_buf())]
        };
        let mut copied = 0usize;
        let mut skipped: Vec<String> = Vec::new();
        for (rel, target) in pairs {
            match self.resolve_file(&path_of(&rel)).await {
                Ok((mount, _)) => {
                    if let Some(parent) = target.parent() {
                        tokio::fs::create_dir_all(parent)
                            .await
                            .map_err(|e| format!("create {}: {e}", parent.display()))?;
                    }
                    tokio::fs::copy(&mount, &target)
                        .await
                        .map_err(|e| format!("copy to {} failed: {e}", target.display()))?;
                    copied += 1;
                }
                Err(e) => skipped.push(format!("{rel}: {e}")),
            }
        }
        self.append(
            OpRow { op: "copy".into(), kind: if entry.is_dir { "dir".into() } else { "file".into() }, from_rel: Some(from), to_rel: None, vault_key_from: None, vault_key_to: None },
            Some(dest.to_string_lossy().to_string()),
            json!({"copied": copied, "skipped": skipped}),
        )
        .await?;
        if skipped.is_empty() {
            Ok(())
        } else {
            Err(format!("copied {copied} file(s); {} had no B2 object: {}", skipped.len(), skipped.join("; ")))
        }
    }

    /// Delete: the vault object(s) are deleted on B2, then the entry leaves the tree.
    pub async fn delete(&self, path: &str) -> Result<(), String> {
        let rel = rel_of(path)?;
        if rel.is_empty() {
            return Err("the catalog root cannot be deleted".into());
        }
        let entry = self.stat(path).await?.ok_or_else(|| format!("{path} does not exist"))?;
        if entry.is_dir {
            let files = self.files_under(&rel).await?;
            let mut deleted_keys = HashSet::new();
            for f in &files {
                if let Ok((mount, key)) = self.resolve_file(&path_of(f)).await {
                    if deleted_keys.insert(key.clone()) {
                        tokio::fs::remove_file(&mount)
                            .await
                            .map_err(|e| format!("B2 delete of {} failed: {e}", mount.display()))?;
                        self.append(
                            OpRow { op: "delete".into(), kind: "file".into(), from_rel: Some(f.clone()), to_rel: None, vault_key_from: Some(key), vault_key_to: None },
                            None,
                            json!({"via_dir": rel}),
                        )
                        .await?;
                    }
                }
            }
            return self
                .append(
                    OpRow { op: "delete".into(), kind: "dir".into(), from_rel: Some(rel), to_rel: None, vault_key_from: None, vault_key_to: None },
                    None,
                    json!({"files": files.len(), "b2_objects_deleted": deleted_keys.len()}),
                )
                .await;
        }
        let (mount, key) = match self.resolve_file(path).await {
            Ok(v) => (Some(v.0), Some(v.1)),
            Err(_) => (None, None),
        };
        if let Some(m) = &mount {
            tokio::fs::remove_file(m)
                .await
                .map_err(|e| format!("B2 delete of {} failed: {e}", m.display()))?;
        }
        self.append(
            OpRow { op: "delete".into(), kind: "file".into(), from_rel: Some(rel), to_rel: None, vault_key_from: key, vault_key_to: None },
            None,
            json!({"b2_object_deleted": mount.is_some()}),
        )
        .await
    }

    pub async fn mkdir(&self, path: &str) -> Result<(), String> {
        let rel = rel_of(path)?;
        if rel.is_empty() || self.stat(path).await?.is_some() {
            return Ok(());
        }
        let parent = parent_of(&rel);
        if !parent.is_empty() && self.stat(&path_of(&parent)).await?.is_none() {
            Box::pin(self.mkdir(&path_of(&parent))).await?;
        }
        self.append(
            OpRow { op: "mkdir".into(), kind: "dir".into(), from_rel: None, to_rel: Some(rel), vault_key_from: None, vault_key_to: None },
            None,
            json!({}),
        )
        .await
    }

    /// Unique name in an effective catalog directory ("name (2).ext" style, like the donor).
    pub async fn unique_name(&self, dir_path: &str, file_name: &str) -> Result<String, String> {
        let dir = rel_of(dir_path)?;
        let names: HashSet<String> = self.list(dir_path).await?.into_iter().map(|e| e.name).collect();
        if !names.contains(file_name) {
            return Ok(path_of(&join(&dir, file_name)));
        }
        let (stem, ext) = match file_name.rfind('.') {
            Some(i) if i > 0 => (&file_name[..i], &file_name[i..]),
            _ => (file_name, ""),
        };
        for n in 2..10_000 {
            let candidate = format!("{stem} ({n}){ext}");
            if !names.contains(&candidate) {
                return Ok(path_of(&join(&dir, &candidate)));
            }
        }
        Err("no free name".into())
    }

    pub async fn dir_totals(&self, path: &str) -> Result<(u64, u64, u64), String> {
        let rel = rel_of(path)?;
        let ops = self.overlay.read().await.clone();
        let base = Self::to_base(&ops, &rel).unwrap_or_default();
        let client = self.ro.get().await.map_err(|e| format!("catalog connection: {e}"))?;
        if base.is_empty() {
            let r = client
                .query_one(&format!("select count(*), coalesce(sum(size),0)::bigint from {FS}"), &[])
                .await
                .map_err(|e| format!("catalog query: {e}"))?;
            let dirs: i64 = client
                .query_one(&format!("select count(*) from {DIRS}"), &[])
                .await
                .map_err(|e| format!("catalog query: {e}"))?
                .get(0);
            return Ok((r.get::<_, i64>(1) as u64, r.get::<_, i64>(0) as u64, dirs as u64));
        }
        let r = client
            .query_opt(&format!("select nested_bytes::bigint, nested_files from {DIRS} where rel = $1"), &[&base])
            .await
            .map_err(|e| format!("catalog query: {e}"))?;
        let dirs: i64 = client
            .query_one(&format!("select count(*) from {DIRS} where rel like $1"), &[&format!("{}/%", base.replace('%', "\\%").replace('_', "\\_"))])
            .await
            .map_err(|e| format!("catalog query: {e}"))?
            .get(0);
        Ok(r.map(|r| (r.get::<_, i64>(0) as u64, r.get::<_, i64>(1) as u64, dirs as u64)).unwrap_or((0, 0, 0)))
    }
}
