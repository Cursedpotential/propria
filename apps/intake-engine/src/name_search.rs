//! Byline: Claude Code · Opus 5 · 2026-09-22
//!
//! Name search across everything, not just the folder that happens to be open (owner 2026-09-22
//! 18:54 EDT: "When I click the magnifying glass I should be able to search for a folder name …
//! the whole level, every fucking thing").
//!
//! What is searched, all from the catalog database, never by reading B2:
//!   * `raw_duck.intake_catalog_dirs_20260917` — every recorded folder (the headline case).
//!   * `raw_duck.intake_catalog_fs_20260917`   — every recorded file, matched on the name it was
//!     recorded under AND on the basename of the vault object it resolves to now.
//!   * `raw_duck.vault_objects_20260916_r4`    — current B2 truth, so objects the catalog never
//!     recorded are still found, and B2 folders are derived from their keys.
//!   * the chat provenance table's `member_path` — names of files INSIDE a zip, flagged
//!     "inside <zip>" (owner 2026-09-22 18:58 EDT).
//!
//! Every hit carries both places a file has lived (owner 2026-09-22 18:58 EDT): `was` is the
//! original recorded path (`catalog://<source>/<scope>/<recorded path>`) and `now` is the current
//! vault object on the mount. The export/dedupe passes renamed and moved things; the owner
//! remembers the old name.
//!
//! Scope is the dropdown every file search has had for twenty-five years (owner 2026-09-22 18:55
//! EDT), translated to a path-prefix predicate so the database does the work.
//!
//! Cost: none of these columns is indexed for a leading-wildcard match, so each leg is a scan that
//! stops at `SCAN_CAP` matching rows. A trigram index would remove that; it is a catalog change and
//! is proposed, not applied — see `docs/URGENT-TODO.md`.

use crate::catalog::{self, Catalog};
use crate::Engine;
use serde_json::{json, Map, Value};
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Instant;

pub type Outcome = Result<Value, (u16, String)>;

/// Matching rows one leg reads before it stops. A leading-wildcard match cannot use an index, so
/// this is what keeps a common word ("Takeout") from scanning 1.5 M rows to the end.
pub const SCAN_CAP: i64 = 4_000;
const DEFAULT_LIMIT: i64 = 200;
const MAX_LIMIT: i64 = 1_000;
/// A leg that runs longer than this gives up and says so, instead of hanging the search box.
const STATEMENT_TIMEOUT: &str = "25s";

const VAULT: &str = "raw_duck.vault_objects_20260916_r4";
const VAULT_DELETED: &str = "raw_duck.vault_onecopy_pilot_delete_20260916";
const DEFAULT_PROVENANCE: &str = "raw_duck.chat_event_provenance_20260918";

// ─────────────────────────────────────────────────────────────────────────────────────────────
// Patterns: what the user typed, as SQL and as an engine-side matcher
// ─────────────────────────────────────────────────────────────────────────────────────────────

/// `*` and `?` are the wildcards people type at a file-search box; `%` and `_` are ordinary
/// characters in a filename and stay literal. Returns the SQL `like` body and whether the user
/// typed a wildcard at all.
fn translate(query: &str) -> (String, bool) {
    let mut out = String::with_capacity(query.len() + 2);
    let mut wild = false;
    for c in query.chars() {
        match c {
            '*' => {
                out.push('%');
                wild = true;
            }
            '?' => {
                out.push('_');
                wild = true;
            }
            '%' | '_' | '\\' => {
                out.push('\\');
                out.push(c);
            }
            _ => out.push(c),
        }
    }
    (out, wild)
}

/// The pattern a NAME is matched against: plain text means "contains", a wildcard query means
/// exactly what it says (`*.pdf` ends with .pdf, `kat*` starts with kat).
pub fn name_like(query: &str) -> String {
    let (body, wild) = translate(query);
    if wild {
        body
    } else {
        format!("%{body}%")
    }
}

/// The pattern a FULL PATH is matched against: always "contains", because a path match is a match
/// anywhere along the path.
pub fn path_like(query: &str) -> String {
    let (body, _) = translate(query);
    format!("%{body}%")
}

/// A literal string used as a `like` prefix (a folder path can itself contain `%` or `_`).
fn escape_like(literal: &str) -> String {
    let mut out = String::with_capacity(literal.len());
    for c in literal.chars() {
        if matches!(c, '%' | '_' | '\\') {
            out.push('\\');
        }
        out.push(c);
    }
    out
}

/// The query as a whole-string wildcard pattern, for matches the engine decides itself (which
/// segment of a B2 key is the folder the user meant).
pub fn name_glob(query: &str) -> String {
    if query.contains('*') || query.contains('?') {
        query.to_string()
    } else {
        format!("*{query}*")
    }
}

/// Case-insensitive `*` / `?` match over the whole string.
pub fn wildcard_match(pattern: &str, text: &str) -> bool {
    let p: Vec<char> = pattern.to_lowercase().chars().collect();
    let t: Vec<char> = text.to_lowercase().chars().collect();
    let (mut pi, mut ti) = (0usize, 0usize);
    let mut star: Option<usize> = None;
    let mut mark = 0usize;
    while ti < t.len() {
        if pi < p.len() && (p[pi] == '?' || p[pi] == t[ti]) {
            pi += 1;
            ti += 1;
        } else if pi < p.len() && p[pi] == '*' {
            star = Some(pi);
            mark = ti;
            pi += 1;
        } else if let Some(s) = star {
            pi = s + 1;
            mark += 1;
            ti = mark;
        } else {
            return false;
        }
    }
    while pi < p.len() && p[pi] == '*' {
        pi += 1;
    }
    pi == p.len()
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// Scope
// ─────────────────────────────────────────────────────────────────────────────────────────────

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Scope {
    Everything,
    ThisFolder,
    ThisFolderAndSubfolders,
    SubfoldersOnly,
    OneLevelUp,
    CatalogOnly,
    B2Only,
}

impl Scope {
    pub fn parse(raw: &str) -> Option<Scope> {
        let k = raw.trim().to_ascii_lowercase().replace([' ', '-'], "_");
        match k.as_str() {
            "" | "everything" | "all" => Some(Scope::Everything),
            "folder" | "this_folder" | "this_folder_only" => Some(Scope::ThisFolder),
            "folder_subfolders" | "this_folder_and_subfolders" | "subtree" => {
                Some(Scope::ThisFolderAndSubfolders)
            }
            "subfolders" | "subfolders_only" => Some(Scope::SubfoldersOnly),
            "parent" | "one_level_up" | "up" => Some(Scope::OneLevelUp),
            "catalog" | "catalog_only" => Some(Scope::CatalogOnly),
            "b2" | "b2_only" | "vault" => Some(Scope::B2Only),
            _ => None,
        }
    }

    pub fn id(self) -> &'static str {
        match self {
            Scope::Everything => "everything",
            Scope::ThisFolder => "this_folder",
            Scope::ThisFolderAndSubfolders => "this_folder_and_subfolders",
            Scope::SubfoldersOnly => "subfolders_only",
            Scope::OneLevelUp => "one_level_up",
            Scope::CatalogOnly => "catalog_only",
            Scope::B2Only => "b2_only",
        }
    }

    /// Scopes that mean nothing without knowing which folder is open.
    fn needs_folder(self) -> bool {
        matches!(
            self,
            Scope::ThisFolder
                | Scope::ThisFolderAndSubfolders
                | Scope::SubfoldersOnly
                | Scope::OneLevelUp
        )
    }

    fn depth(self) -> Depth {
        match self {
            Scope::ThisFolder => Depth::Direct,
            Scope::SubfoldersOnly => Depth::Below,
            _ => Depth::Here,
        }
    }
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
enum Depth {
    /// Entries whose parent IS this folder.
    Direct,
    /// This folder and everything under it.
    Here,
    /// Strictly under this folder (a file sitting directly in it is the folder's own file).
    Below,
}

/// Which path space the scope prefix lives in: the catalog tree, or B2 vault keys.
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
enum Space {
    Catalog,
    Vault,
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// SQL assembly
// ─────────────────────────────────────────────────────────────────────────────────────────────

/// A statement being built, with its bound parameters. Values are always parameters, never
/// interpolated text.
struct Sql {
    params: Vec<String>,
}

impl Sql {
    fn new() -> Self {
        Sql { params: Vec::new() }
    }

    fn bind(&mut self, v: impl Into<String>) -> String {
        self.params.push(v.into());
        format!("${}", self.params.len())
    }

    fn refs(&self) -> Vec<&(dyn tokio_postgres::types::ToSql + Sync)> {
        self.params
            .iter()
            .map(|p| p as &(dyn tokio_postgres::types::ToSql + Sync))
            .collect()
    }
}

/// The scope as a predicate over a path column and its parent-directory expression. `None` means
/// "no restriction" (the whole space).
fn scope_clause(
    sql: &mut Sql,
    path_col: &str,
    parent_expr: &str,
    prefix: &str,
    depth: Depth,
    is_file_leg: bool,
) -> Option<String> {
    match depth {
        Depth::Direct => {
            if prefix.is_empty() {
                Some(format!("strpos({path_col}, '/') = 0"))
            } else {
                let k = sql.bind(prefix);
                Some(format!("{parent_expr} = {k}"))
            }
        }
        Depth::Here => {
            if prefix.is_empty() {
                None
            } else {
                let k = sql.bind(prefix);
                let p = sql.bind(format!("{}/%", escape_like(prefix)));
                Some(format!("({path_col} = {k} or {path_col} like {p} escape '\\')"))
            }
        }
        Depth::Below => {
            if prefix.is_empty() {
                Some(format!("strpos({path_col}, '/') > 0"))
            } else {
                let p = sql.bind(format!("{}/%", escape_like(prefix)));
                let mut clause = format!("{path_col} like {p} escape '\\'");
                if is_file_leg {
                    let k = sql.bind(prefix);
                    clause = format!("{clause} and {parent_expr} <> {k}");
                }
                Some(clause)
            }
        }
    }
}

/// `where` text from the match predicate plus an optional scope predicate.
fn where_of(matched: &str, scope: Option<String>) -> String {
    match scope {
        Some(s) => format!("where ({matched}) and ({s})"),
        None => format!("where ({matched})"),
    }
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// Hits
// ─────────────────────────────────────────────────────────────────────────────────────────────

#[derive(Clone)]
struct Hit {
    kind: &'static str,
    name: String,
    /// The full path shown in the row, and the thing the row is identified by.
    path: String,
    /// The folder the active pane opens when the row is clicked.
    navigate_to: String,
    /// The file to select once that folder has loaded; `None` for a folder row.
    select: Option<String>,
    /// Where the bytes are now (mount path of the current vault object).
    now: Option<String>,
    /// Where the catalog recorded it originally.
    was: Option<String>,
    source: &'static str,
    matched: &'static str,
    member_of: Option<String>,
    size: Option<i64>,
    modified: Option<u64>,
    flag: Option<String>,
}

impl Hit {
    fn to_json(&self) -> Value {
        json!({
            "kind": self.kind,
            "name": self.name,
            "path": self.path,
            "navigate_to": self.navigate_to,
            "select": self.select,
            "now": self.now,
            "was": self.was,
            "source": self.source,
            "matched": self.matched,
            "member_of": self.member_of,
            "size": self.size,
            "modified": self.modified,
            "flag": self.flag,
        })
    }
}

fn parent_of_mount(path: &str) -> String {
    match path.rfind('/') {
        Some(0) => "/".to_string(),
        Some(i) => path[..i].to_string(),
        None => path.to_string(),
    }
}

fn basename(path: &str) -> String {
    match path.rfind('/') {
        Some(i) => path[i + 1..].to_string(),
        None => path.to_string(),
    }
}

fn mount_path(b2_root: &PathBuf, key: &str) -> String {
    format!("{}/{}", b2_root.display().to_string().trim_end_matches('/'), key)
}

/// The deepest segment of a directory path whose own name matches: the folder the user meant when
/// a whole B2 key matched. `None` when no segment matches by name.
fn folder_hit_from_dir(dir: &str, glob: &str) -> Option<String> {
    let segs: Vec<&str> = dir.split('/').filter(|s| !s.is_empty()).collect();
    let mut found: Option<usize> = None;
    for (i, seg) in segs.iter().enumerate() {
        if wildcard_match(glob, seg) {
            found = Some(i);
        }
    }
    found.map(|i| segs[..=i].join("/"))
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// Legs
// ─────────────────────────────────────────────────────────────────────────────────────────────

struct Leg {
    name: &'static str,
    hits: Vec<Hit>,
    rows: usize,
    ms: u128,
    capped: bool,
    error: Option<String>,
}

impl Leg {
    fn empty(name: &'static str) -> Leg {
        Leg { name, hits: Vec::new(), rows: 0, ms: 0, capped: false, error: None }
    }

    fn to_json(&self) -> Value {
        json!({
            "leg": self.name,
            "rows": self.rows,
            "ms": self.ms,
            "capped": self.capped,
            "error": self.error,
        })
    }
}

struct Ctx {
    name_like: String,
    path_like: String,
    name_glob: String,
    space: Option<Space>,
    prefix: String,
    depth: Depth,
    cap: i64,
    b2_root: PathBuf,
    ops: Vec<catalog::OpRow>,
}

impl Ctx {
    /// The scope predicate for a leg whose paths live in `space`; `Err(())` when this leg cannot
    /// express the active scope at all (a catalog folder has no vault key, and vice versa).
    fn scope_for(
        &self,
        sql: &mut Sql,
        space: Space,
        path_col: &str,
        parent_expr: &str,
        is_file_leg: bool,
    ) -> Result<Option<String>, ()> {
        match self.space {
            None => Ok(None),
            Some(s) if s == space => Ok(scope_clause(
                sql,
                path_col,
                parent_expr,
                &self.prefix,
                self.depth,
                is_file_leg,
            )),
            Some(_) => Err(()),
        }
    }
}

async fn client(engine: &Engine) -> Result<deadpool_postgres::Object, (u16, String)> {
    let cat = engine
        .catalog
        .as_ref()
        .ok_or_else(|| (503, "the catalog database is not connected".to_string()))?;
    let c = cat
        .ro_pool()
        .get()
        .await
        .map_err(|e| (503, format!("catalog connection: {e}")))?;
    c.batch_execute(&format!("set statement_timeout = '{STATEMENT_TIMEOUT}'"))
        .await
        .map_err(|e| (503, format!("catalog connection: {e}")))?;
    Ok(c)
}

/// Folders the catalog recorded. The dirs table is small (~80 k rows), so this is the fast leg —
/// and it is the one the owner's question is about.
async fn catalog_folders(engine: &Arc<Engine>, ctx: &Ctx) -> Leg {
    let mut leg = Leg::empty("catalog_folders");
    let started = Instant::now();
    let mut sql = Sql::new();
    let n = sql.bind(ctx.name_like.clone());
    let p = sql.bind(ctx.path_like.clone());
    let scope = match ctx.scope_for(&mut sql, Space::Catalog, "d.rel", "d.parent", false) {
        Ok(s) => s,
        Err(()) => return leg,
    };
    let matched = format!("d.name ilike {n} escape '\\' or d.rel ilike {p} escape '\\'");
    let text = format!(
        "select d.rel, d.name, d.nested_bytes::bigint, d.nested_files::bigint, \
         (d.name ilike {n} escape '\\') as name_hit \
         from {dirs} d {where_} limit {cap}",
        dirs = catalog::DIRS_TABLE,
        where_ = where_of(&matched, scope),
        cap = ctx.cap,
    );
    let c = match client(engine).await {
        Ok(c) => c,
        Err((_, e)) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let params = sql.refs();
    let rows = match c.query(text.as_str(), &params[..]).await {
        Ok(r) => r,
        Err(e) => {
            leg.error = Some(format!("catalog folders: {e}"));
            leg.ms = started.elapsed().as_millis();
            return leg;
        }
    };
    leg.rows = rows.len();
    leg.capped = rows.len() as i64 >= ctx.cap;
    for r in &rows {
        let base: String = r.get(0);
        let Some(eff) = Catalog::effective_path(&ctx.ops, &base) else { continue };
        let bytes: Option<i64> = r.get(2);
        let files: Option<i64> = r.get(3);
        let name_hit: bool = r.get(4);
        let path = catalog::path_of(&eff);
        leg.hits.push(Hit {
            kind: "folder",
            name: catalog::name_of(&eff),
            path: path.clone(),
            navigate_to: path.clone(),
            select: None,
            now: None,
            was: Some(path),
            source: "catalog",
            matched: if name_hit { "name" } else { "path" },
            member_of: None,
            size: bytes,
            modified: None,
            flag: files.map(|f| format!("{f} files recorded")),
        });
    }
    leg.ms = started.elapsed().as_millis();
    leg
}

/// Files the catalog recorded, matched on the recorded name AND on the basename of the vault object
/// they resolve to now — the two names the same file has had.
async fn catalog_files(engine: &Arc<Engine>, ctx: &Ctx) -> Leg {
    let mut leg = Leg::empty("catalog_files");
    let started = Instant::now();
    let mut sql = Sql::new();
    let n = sql.bind(ctx.name_like.clone());
    let p = sql.bind(ctx.path_like.clone());
    let (path_col, parent_expr) = match ctx.space {
        Some(Space::Vault) => ("f.vault_key", "regexp_replace(f.vault_key, '/[^/]*$', '')"),
        _ => ("f.rel", "f.parent"),
    };
    let space = match ctx.space {
        Some(s) => s,
        None => Space::Catalog,
    };
    let scope = match ctx.scope_for(&mut sql, space, path_col, parent_expr, true) {
        Ok(s) => s,
        Err(()) => return leg,
    };
    let name_expr = format!(
        "(f.name ilike {n} escape '\\' or regexp_replace(coalesce(f.vault_key, ''), '^.*/', '') ilike {n} escape '\\')"
    );
    let matched = format!(
        "{name_expr} or f.rel ilike {p} escape '\\' or f.vault_key ilike {p} escape '\\'"
    );
    let text = format!(
        "select f.rel, f.name, f.size, f.modtime, f.vault_key, f.resolution, {name_expr} as name_hit \
         from {fs} f {where_} limit {cap}",
        fs = catalog::FS_TABLE,
        where_ = where_of(&matched, scope),
        cap = ctx.cap,
    );
    let c = match client(engine).await {
        Ok(c) => c,
        Err((_, e)) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let params = sql.refs();
    let rows = match c.query(text.as_str(), &params[..]).await {
        Ok(r) => r,
        Err(e) => {
            leg.error = Some(format!("catalog files: {e}"));
            leg.ms = started.elapsed().as_millis();
            return leg;
        }
    };
    leg.rows = rows.len();
    leg.capped = rows.len() as i64 >= ctx.cap;
    for r in &rows {
        let base: String = r.get(0);
        let Some(eff) = Catalog::effective_path(&ctx.ops, &base) else { continue };
        let size: Option<i64> = r.get(2);
        let modtime: Option<chrono::DateTime<chrono::Utc>> = r.get(3);
        let vault_key: Option<String> = r.get(4);
        let resolution: String = r.get(5);
        let name_hit: bool = r.get(6);
        let current = vault_key
            .as_deref()
            .and_then(|k| Catalog::effective_vault_key(&ctx.ops, k));
        let was = catalog::path_of(&eff);
        let now = current.as_deref().map(|k| mount_path(&ctx.b2_root, k));
        // The row opens where the bytes actually are; an occurrence with no current object can
        // still be opened in the catalog tree, where its flag explains why it has no bytes.
        let target = now.clone().unwrap_or_else(|| was.clone());
        let flag = match (resolution.as_str(), &current) {
            ("resolved", Some(_)) => None,
            ("resolved", None) => Some("the B2 object for this entry was deleted".to_string()),
            ("no_b2_key", _) => Some("recorded occurrence has no B2 copy".to_string()),
            ("b2_key_not_in_b2_objects", _) => {
                Some("recorded B2 key is not in the B2 listing".to_string())
            }
            (other, _) => Some(format!("no current vault object ({other})")),
        };
        leg.hits.push(Hit {
            kind: "file",
            name: catalog::name_of(&eff),
            path: target.clone(),
            navigate_to: if catalog::is_catalog(&target) {
                catalog::path_of(&catalog::parent_of(&eff))
            } else {
                parent_of_mount(&target)
            },
            select: Some(target),
            now,
            was: Some(was),
            source: "catalog",
            matched: if name_hit { "name" } else { "path" },
            member_of: None,
            size,
            modified: catalog::recorded_time(modtime),
            flag,
        });
    }
    leg.ms = started.elapsed().as_millis();
    leg
}

/// Objects that are in B2 right now, whether or not an occurrence was ever recorded for them.
async fn vault_files(engine: &Arc<Engine>, ctx: &Ctx) -> Leg {
    let mut leg = Leg::empty("b2_files");
    let started = Instant::now();
    let mut sql = Sql::new();
    let n = sql.bind(ctx.name_like.clone());
    let p = sql.bind(ctx.path_like.clone());
    let scope = match ctx.scope_for(
        &mut sql,
        Space::Vault,
        "v.key",
        "regexp_replace(v.key, '/[^/]*$', '')",
        true,
    ) {
        Ok(s) => s,
        Err(()) => return leg,
    };
    let name_expr = format!("regexp_replace(v.key, '^.*/', '') ilike {n} escape '\\'");
    let matched = format!("{name_expr} or v.key ilike {p} escape '\\'");
    let text = format!(
        "select v.key, v.size, {name_expr} as name_hit from {vault} v {where_} \
         and not exists (select 1 from {deleted} d where d.key = v.key) limit {cap}",
        vault = VAULT,
        deleted = VAULT_DELETED,
        where_ = where_of(&matched, scope),
        cap = ctx.cap,
    );
    let c = match client(engine).await {
        Ok(c) => c,
        Err((_, e)) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let params = sql.refs();
    let rows = match c.query(text.as_str(), &params[..]).await {
        Ok(r) => r,
        Err(e) => {
            leg.error = Some(format!("B2 files: {e}"));
            leg.ms = started.elapsed().as_millis();
            return leg;
        }
    };
    leg.rows = rows.len();
    leg.capped = rows.len() as i64 >= ctx.cap;
    for r in &rows {
        let key: String = r.get(0);
        let size: Option<i64> = r.get(1);
        let name_hit: bool = r.get(2);
        let Some(current) = Catalog::effective_vault_key(&ctx.ops, &key) else { continue };
        let path = mount_path(&ctx.b2_root, &current);
        leg.hits.push(Hit {
            kind: "file",
            name: basename(&current),
            path: path.clone(),
            navigate_to: parent_of_mount(&path),
            select: Some(path.clone()),
            now: Some(path),
            was: None,
            source: "b2",
            matched: if name_hit { "name" } else { "path" },
            member_of: None,
            size,
            modified: None,
            flag: None,
        });
    }
    leg.ms = started.elapsed().as_millis();
    leg
}

/// B2 folders. There is no folder table for the vault — a folder is a prefix of the keys — so the
/// distinct directories whose path matches come back and the deepest segment that matches BY NAME
/// is the folder the user meant.
async fn vault_folders(engine: &Arc<Engine>, ctx: &Ctx) -> Leg {
    let mut leg = Leg::empty("b2_folders");
    let started = Instant::now();
    let mut sql = Sql::new();
    let p = sql.bind(ctx.path_like.clone());
    let dir_expr = "regexp_replace(v.key, '/[^/]*$', '')";
    let scope = match ctx.scope_for(&mut sql, Space::Vault, "v.key", dir_expr, false) {
        Ok(s) => s,
        Err(()) => return leg,
    };
    let matched = format!("{dir_expr} ilike {p} escape '\\'");
    let text = format!(
        "select distinct {dir_expr} as dir from {vault} v {where_} \
         and not exists (select 1 from {deleted} d where d.key = v.key) limit {cap}",
        vault = VAULT,
        deleted = VAULT_DELETED,
        where_ = where_of(&matched, scope),
        cap = ctx.cap,
    );
    let c = match client(engine).await {
        Ok(c) => c,
        Err((_, e)) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let params = sql.refs();
    let rows = match c.query(text.as_str(), &params[..]).await {
        Ok(r) => r,
        Err(e) => {
            leg.error = Some(format!("B2 folders: {e}"));
            leg.ms = started.elapsed().as_millis();
            return leg;
        }
    };
    leg.rows = rows.len();
    leg.capped = rows.len() as i64 >= ctx.cap;
    let mut seen: HashMap<String, bool> = HashMap::new();
    for r in &rows {
        let dir: String = r.get(0);
        if dir.is_empty() {
            continue;
        }
        let (folder, by_name) = match folder_hit_from_dir(&dir, &ctx.name_glob) {
            Some(f) => (f, true),
            None => (dir, false),
        };
        // A folder found by name wins over the same folder found only by its path.
        match seen.get(&folder) {
            Some(true) => continue,
            Some(false) if !by_name => continue,
            _ => {}
        }
        seen.insert(folder, by_name);
    }
    for (folder, by_name) in seen {
        let path = mount_path(&ctx.b2_root, &folder);
        leg.hits.push(Hit {
            kind: "folder",
            name: basename(&folder),
            path: path.clone(),
            navigate_to: path,
            select: None,
            now: None,
            was: None,
            source: "b2",
            matched: if by_name { "name" } else { "path" },
            member_of: None,
            size: None,
            modified: None,
            flag: None,
        });
    }
    leg.ms = started.elapsed().as_millis();
    leg
}

/// `schema.table` of the chat provenance table, which is where the catalog records the names of
/// files INSIDE a zip. Empty env value retires the leg.
fn provenance_table() -> Result<Option<String>, String> {
    let t = match std::env::var("INTAKE_CHAT_INDEX_PROVENANCE_TABLE") {
        Ok(v) if v.trim().is_empty() => return Ok(None),
        Ok(v) => v,
        Err(_) => DEFAULT_PROVENANCE.to_string(),
    };
    if t.chars().all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '.') {
        Ok(Some(t))
    } else {
        Err("INTAKE_CHAT_INDEX_PROVENANCE_TABLE is not a plain schema.table name".into())
    }
}

/// Names of files inside a zip, where the catalog recorded them. The hit opens the zip itself,
/// flagged with the member it matched.
async fn zip_members(engine: &Arc<Engine>, ctx: &Ctx) -> Leg {
    let mut leg = Leg::empty("zip_members");
    let table = match provenance_table() {
        Ok(Some(t)) => t,
        Ok(None) => return leg,
        Err(e) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let started = Instant::now();
    let mut sql = Sql::new();
    let n = sql.bind(ctx.name_like.clone());
    let p = sql.bind(ctx.path_like.clone());
    let (path_col, parent_expr) = match ctx.space {
        Some(Space::Vault) => ("m.vault_key", "regexp_replace(m.vault_key, '/[^/]*$', '')"),
        _ => ("m.catalog_rel", "regexp_replace(m.catalog_rel, '/[^/]*$', '')"),
    };
    let space = match ctx.space {
        Some(s) => s,
        None => Space::Catalog,
    };
    let scope = match ctx.scope_for(&mut sql, space, path_col, parent_expr, true) {
        Ok(s) => s,
        Err(()) => return leg,
    };
    let matched = format!(
        "regexp_replace(m.member_path, '^.*/', '') ilike {n} escape '\\' \
         or m.member_path ilike {p} escape '\\'"
    );
    let text = format!(
        "select distinct m.vault_key, m.catalog_rel, m.member_path from {table} m \
         {where_} and m.member_path is not null limit {cap}",
        where_ = where_of(&matched, scope),
        cap = ctx.cap,
    );
    let c = match client(engine).await {
        Ok(c) => c,
        Err((_, e)) => {
            leg.error = Some(e);
            return leg;
        }
    };
    let params = sql.refs();
    let rows = match c.query(text.as_str(), &params[..]).await {
        Ok(r) => r,
        Err(e) => {
            leg.error = Some(format!("zip members: {e}"));
            leg.ms = started.elapsed().as_millis();
            return leg;
        }
    };
    leg.rows = rows.len();
    leg.capped = rows.len() as i64 >= ctx.cap;
    for r in &rows {
        let vault_key: Option<String> = r.get(0);
        let catalog_rel: Option<String> = r.get(1);
        let member_path: String = r.get(2);
        let member = basename(&member_path);
        let member_hit = wildcard_match(&ctx.name_glob, &member);
        let current = vault_key
            .as_deref()
            .and_then(|k| Catalog::effective_vault_key(&ctx.ops, k));
        let now = current.as_deref().map(|k| mount_path(&ctx.b2_root, k));
        let was = catalog_rel
            .as_deref()
            .and_then(|rel| Catalog::effective_path(&ctx.ops, rel))
            .map(|eff| catalog::path_of(&eff));
        let Some(target) = now.clone().or_else(|| was.clone()) else { continue };
        let container = basename(&target);
        leg.hits.push(Hit {
            kind: "file",
            name: member,
            // `<the zip>!/<entry>`: one row per entry, and the row still says which object holds it.
            path: format!("{target}!/{member_path}"),
            navigate_to: if catalog::is_catalog(&target) {
                catalog::path_of(&catalog::parent_of(&catalog::rel_of(&target).unwrap_or_default()))
            } else {
                parent_of_mount(&target)
            },
            select: Some(target),
            now,
            was,
            source: "zip",
            matched: if member_hit { "name" } else { "path" },
            member_of: Some(container.clone()),
            size: None,
            modified: None,
            flag: Some(format!("inside {container}")),
        });
    }
    leg.ms = started.elapsed().as_millis();
    leg
}

// ─────────────────────────────────────────────────────────────────────────────────────────────
// Command
// ─────────────────────────────────────────────────────────────────────────────────────────────

fn s(args: &Map<String, Value>, camel: &str, snake: &str) -> Option<String> {
    args.get(camel)
        .or_else(|| args.get(snake))
        .and_then(|v| v.as_str())
        .map(|v| v.trim().to_string())
        .filter(|v| !v.is_empty())
}

/// `intake_search_names { query, scope?, path?, kinds?, sort?, limit? }`.
pub async fn search_names(engine: &Arc<Engine>, args: &Map<String, Value>) -> Outcome {
    let query = s(args, "query", "query").ok_or_else(|| (422, "query is required".to_string()))?;
    if query.chars().count() < 2 {
        return Err((422, "type at least two characters".to_string()));
    }
    let scope_raw = s(args, "scope", "scope").unwrap_or_else(|| "everything".into());
    let scope = Scope::parse(&scope_raw)
        .ok_or_else(|| (422, format!("{scope_raw} is not one of the search scopes")))?;
    let kinds = s(args, "kinds", "kinds").unwrap_or_else(|| "both".into());
    let (want_folders, want_files) = match kinds.as_str() {
        "both" | "all" => (true, true),
        "folders" => (true, false),
        "files" => (false, true),
        other => return Err((422, format!("{other} is not one of: both, folders, files"))),
    };
    let sort = s(args, "sort", "sort").unwrap_or_else(|| "name".into());
    if sort != "name" && sort != "path" {
        return Err((422, format!("{sort} is not one of: name, path")));
    }
    let limit = args
        .get("limit")
        .and_then(|v| v.as_i64())
        .unwrap_or(DEFAULT_LIMIT)
        .clamp(1, MAX_LIMIT);

    let cat = engine.catalog.as_deref().ok_or_else(|| {
        (
            503,
            format!(
                "the catalog database is not connected: {}",
                engine.catalog_error.as_deref().unwrap_or("no connection")
            ),
        )
    })?;

    let mut notes: Vec<String> = Vec::new();
    let folder = s(args, "path", "path");

    // Where the scope prefix lives, and what it is.
    let (mut space, mut prefix) = (None, String::new());
    if scope.needs_folder() {
        match folder.as_deref() {
            None => notes.push(
                "no folder was open, so this searched everything instead of that folder".into(),
            ),
            Some(f) if catalog::is_catalog(f) => {
                let rel = catalog::rel_of(f).map_err(|e| (422, e))?;
                space = Some(Space::Catalog);
                prefix = if scope == Scope::OneLevelUp { catalog::parent_of(&rel) } else { rel };
            }
            Some(f) => {
                let root = cat.b2_root.display().to_string();
                let root = root.trim_end_matches('/');
                match f.strip_prefix(root).map(|r| r.trim_start_matches('/').to_string()) {
                    Some(key) => {
                        space = Some(Space::Vault);
                        prefix = if scope == Scope::OneLevelUp {
                            match key.rfind('/') {
                                Some(i) => key[..i].to_string(),
                                None => String::new(),
                            }
                        } else {
                            key
                        };
                    }
                    None => notes.push(format!(
                        "{f} is not in the catalog or the B2 vault, so this searched everything instead of that folder"
                    )),
                }
            }
        }
    }

    let ctx = Ctx {
        name_like: name_like(&query),
        path_like: path_like(&query),
        name_glob: name_glob(&query),
        space,
        prefix,
        depth: scope.depth(),
        cap: SCAN_CAP,
        b2_root: cat.b2_root.clone(),
        ops: cat.overlay_ops().await,
    };

    // Which legs this scope reads at all.
    let catalog_leg = scope != Scope::B2Only;
    let vault_leg = scope != Scope::CatalogOnly;
    let zip_leg = scope != Scope::B2Only;
    if space == Some(Space::Catalog) && vault_leg && scope.needs_folder() {
        notes.push(
            "a catalog:// folder has no B2 prefix, so B2 objects are matched through the catalog rows that resolve into it".into(),
        );
    }
    if space == Some(Space::Vault) && catalog_leg && scope.needs_folder() {
        notes.push(
            "a B2 folder has no catalog prefix, so recorded folders are matched through the vault keys under it".into(),
        );
    }

    let started = Instant::now();
    let (cf, cfi, vf, vfi, zm) = tokio::join!(
        async {
            if catalog_leg && want_folders && space != Some(Space::Vault) {
                catalog_folders(engine, &ctx).await
            } else {
                Leg::empty("catalog_folders")
            }
        },
        async {
            if catalog_leg && want_files {
                catalog_files(engine, &ctx).await
            } else {
                Leg::empty("catalog_files")
            }
        },
        async {
            if vault_leg && want_folders && space != Some(Space::Catalog) {
                vault_folders(engine, &ctx).await
            } else {
                Leg::empty("b2_folders")
            }
        },
        async {
            if vault_leg && want_files {
                vault_files(engine, &ctx).await
            } else {
                Leg::empty("b2_files")
            }
        },
        async {
            if zip_leg && want_files {
                zip_members(engine, &ctx).await
            } else {
                Leg::empty("zip_members")
            }
        },
    );
    let legs = [cf, cfi, vf, vfi, zm];

    // Merge. The same file reached through the catalog and through the vault listing is one hit,
    // and the catalog's is the one that knows where the file used to be.
    let mut merged: Vec<Hit> = Vec::new();
    let mut index: HashMap<(&'static str, String), usize> = HashMap::new();
    for leg in &legs {
        if let Some(e) = &leg.error {
            notes.push(e.clone());
        }
        for hit in &leg.hits {
            let key = (hit.kind, hit.path.clone());
            match index.get(&key) {
                Some(&i) => {
                    if merged[i].was.is_none() && hit.was.is_some() {
                        merged[i] = hit.clone();
                    } else if merged[i].matched == "path" && hit.matched == "name" {
                        merged[i].matched = "name";
                    }
                }
                None => {
                    index.insert(key, merged.len());
                    merged.push(hit.clone());
                }
            }
        }
    }

    let total_folders = merged.iter().filter(|h| h.kind == "folder").count();
    let total_files = merged.len() - total_folders;
    let capped = legs.iter().any(|l| l.capped);

    // Folders first (the owner asked for folder names); a name match before a path-only match;
    // then the chosen sort.
    merged.sort_by(|a, b| {
        let rank = |h: &Hit| (h.kind == "file", h.matched == "path");
        let key = |h: &Hit| {
            if sort == "path" {
                h.path.to_lowercase()
            } else {
                h.name.to_lowercase()
            }
        };
        rank(a)
            .cmp(&rank(b))
            .then_with(|| key(a).cmp(&key(b)))
            .then_with(|| a.path.cmp(&b.path))
    });
    merged.truncate(limit as usize);

    let shown_folders = merged.iter().filter(|h| h.kind == "folder").count();
    if capped {
        notes.push(format!(
            "more than {SCAN_CAP} rows matched in at least one place; counts are what was read, not the whole catalog"
        ));
    }

    Ok(json!({
        "query": query,
        "scope": scope.id(),
        "kinds": kinds,
        "sort": sort,
        "folder": folder,
        "hits": merged.iter().map(Hit::to_json).collect::<Vec<_>>(),
        "shown_folders": shown_folders,
        "shown_files": merged.len() - shown_folders,
        "total_folders": total_folders,
        "total_files": total_files,
        "capped": capped,
        "scan_cap": SCAN_CAP,
        "elapsed_ms": started.elapsed().as_millis(),
        "legs": legs.iter().map(Leg::to_json).collect::<Vec<_>>(),
        "notes": notes,
    }))
}

/// The scopes the UI offers, named by the engine so both sides cannot drift apart.
pub fn scopes() -> Value {
    json!([
        {"id": "everything", "label": "Everything"},
        {"id": "this_folder", "label": "This folder only"},
        {"id": "this_folder_and_subfolders", "label": "This folder and subfolders"},
        {"id": "subfolders_only", "label": "Subfolders only"},
        {"id": "one_level_up", "label": "One level up"},
        {"id": "catalog_only", "label": "Catalog only"},
        {"id": "b2_only", "label": "B2 only"},
    ])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn plain_text_means_contains() {
        assert_eq!(name_like("Katrina"), "%Katrina%");
        assert_eq!(path_like("Katrina"), "%Katrina%");
        assert_eq!(name_glob("Katrina"), "*Katrina*");
    }

    #[test]
    fn wildcards_are_taken_literally_for_names_and_widened_for_paths() {
        assert_eq!(name_like("*.pdf"), "%.pdf");
        assert_eq!(name_like("sms-?.xml"), "sms-_.xml");
        assert_eq!(path_like("*.pdf"), "%%.pdf%");
        assert_eq!(name_glob("*.pdf"), "*.pdf");
    }

    #[test]
    fn percent_and_underscore_are_ordinary_characters_in_a_filename() {
        assert_eq!(name_like("100%_done"), "%100\\%\\_done%");
        assert_eq!(escape_like("a_b%c"), "a\\_b\\%c");
    }

    #[test]
    fn wildcard_match_is_case_insensitive_and_whole_string() {
        assert!(wildcard_match("*katrina*", "2019 KATRINA texts"));
        assert!(wildcard_match("*.pdf", "Order.PDF"));
        assert!(!wildcard_match("*.pdf", "Order.pdf.txt"));
        assert!(wildcard_match("sms-?.xml", "sms-3.xml"));
        assert!(!wildcard_match("sms-?.xml", "sms-33.xml"));
        assert!(wildcard_match("*", "anything"));
        assert!(!wildcard_match("takeout", "Takeout 2"));
    }

    #[test]
    fn every_scope_the_dropdown_offers_parses() {
        for scope in scopes().as_array().unwrap() {
            let id = scope["id"].as_str().unwrap();
            assert_eq!(Scope::parse(id).map(|s| s.id()), Some(id), "{id}");
        }
        assert_eq!(Scope::parse(""), Some(Scope::Everything));
        assert_eq!(Scope::parse("This Folder Only"), Some(Scope::ThisFolder));
        assert_eq!(Scope::parse("nonsense"), None);
    }

    #[test]
    fn only_folder_relative_scopes_need_a_folder() {
        assert!(!Scope::Everything.needs_folder());
        assert!(!Scope::CatalogOnly.needs_folder());
        assert!(!Scope::B2Only.needs_folder());
        assert!(Scope::ThisFolder.needs_folder());
        assert!(Scope::OneLevelUp.needs_folder());
    }

    #[test]
    fn this_folder_only_is_a_parent_equality() {
        let mut sql = Sql::new();
        let clause = scope_clause(&mut sql, "f.rel", "f.parent", "gdrive/salemnet", Depth::Direct, true);
        assert_eq!(clause.as_deref(), Some("f.parent = $1"));
        assert_eq!(sql.params, vec!["gdrive/salemnet".to_string()]);
    }

    #[test]
    fn this_folder_and_subfolders_is_the_folder_plus_its_prefix() {
        let mut sql = Sql::new();
        let clause =
            scope_clause(&mut sql, "f.rel", "f.parent", "gdrive/salemnet", Depth::Here, true);
        assert_eq!(clause.as_deref(), Some("(f.rel = $1 or f.rel like $2 escape '\\')"));
        assert_eq!(sql.params[1], "gdrive/salemnet/%");
    }

    #[test]
    fn subfolders_only_excludes_the_folders_own_files_but_not_its_folders() {
        let mut sql = Sql::new();
        let files = scope_clause(&mut sql, "f.rel", "f.parent", "a/b", Depth::Below, true);
        assert_eq!(files.as_deref(), Some("f.rel like $1 escape '\\' and f.parent <> $2"));
        let mut sql = Sql::new();
        let dirs = scope_clause(&mut sql, "d.rel", "d.parent", "a/b", Depth::Below, false);
        assert_eq!(dirs.as_deref(), Some("d.rel like $1 escape '\\'"));
    }

    #[test]
    fn everything_has_no_prefix_predicate_at_all() {
        let mut sql = Sql::new();
        assert_eq!(scope_clause(&mut sql, "f.rel", "f.parent", "", Depth::Here, true), None);
        assert!(sql.params.is_empty());
    }

    #[test]
    fn a_folder_path_containing_a_wildcard_character_is_escaped_as_a_prefix() {
        let mut sql = Sql::new();
        scope_clause(&mut sql, "f.rel", "f.parent", "100%_files", Depth::Here, true);
        assert_eq!(sql.params[1], "100\\%\\_files/%");
    }

    #[test]
    fn a_b2_folder_hit_is_the_deepest_segment_that_matches_by_name() {
        assert_eq!(
            folder_hit_from_dir("consignatio/vault/v1/Takeout/Drive/My Drive", "*takeout*"),
            Some("consignatio/vault/v1/Takeout".to_string())
        );
        assert_eq!(
            folder_hit_from_dir("a/Takeout/b/Takeout backup/c", "*takeout*"),
            Some("a/Takeout/b/Takeout backup".to_string())
        );
        assert_eq!(folder_hit_from_dir("a/b/c", "*takeout*"), None);
    }

    #[test]
    fn paths_split_the_way_the_ui_navigates() {
        assert_eq!(parent_of_mount("/srv/openlist/b2/salem-data/a/b.txt"), "/srv/openlist/b2/salem-data/a");
        assert_eq!(basename("a/b/c.txt"), "c.txt");
        assert_eq!(basename("c.txt"), "c.txt");
        assert_eq!(
            mount_path(&PathBuf::from("/srv/openlist/b2/salem-data"), "a/b.txt"),
            "/srv/openlist/b2/salem-data/a/b.txt"
        );
    }

    #[test]
    fn a_where_clause_keeps_match_and_scope_separate() {
        assert_eq!(where_of("a or b", None), "where (a or b)");
        assert_eq!(where_of("a or b", Some("c".into())), "where (a or b) and (c)");
    }
}
