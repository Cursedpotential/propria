// Byline: Claude Code · Fable 5.1 · 2026-09-07
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — owner orders 13:09-13:16: filed/draft
// works registers, court-event vs master-timeline split, memos (analysis/strategy/
// weakness/direction), reference data (behavior patterns/ontology/lexicon), an evidence
// log, evals/reports, a case_status singleton, per-record source provenance, a
// case-extract/v1 importer, and a platform (probata) NDJSON export bundle. All additive
// — every pre-existing table, edge, tool, and record shape keeps working unchanged.
//
// Embedded SurrealDB case store: search + graph over the family-court-toolkit's
// case data (people, children, orders, hearings, deadlines, events, messages,
// exhibits, factors, sources, notes, court events, filings, drafts, memos,
// reference data, an evidence log, evals, and case status) plus the edges relating them.
//
// This file is deliberately separate from src/server.ts and src/core.ts (both
// were being edited by another agent concurrently when this was written) — it
// has no dependency on either and exports only functions + `registerStoreTools`
// lives in store-tools.ts, which is the file to wire into server.ts.
//
// `surrealdb` + `@surrealdb/node` are esbuild-external (see build.mjs) because
// @surrealdb/node ships native .node binaries that cannot be bundled. Every
// use of them goes through a dynamic `await import(...)` wrapped in try/catch
// so a missing/failed native module degrades every store tool to
// `{ available: false, reason }` instead of crashing the whole MCP server at
// startup. See README-store.md for the required `npm ci --omit=dev` step.
//
// Verified interactively against a live mem:// SurrealDB (surrealdb@2.0.8 +
// @surrealdb/node@3.0.3) before this file was written: table names that are
// SurrealQL keywords (e.g. `order`) need backtick-quoting in DDL/RELATE text;
// full-text index syntax on this build is `FIELDS <f> FULLTEXT ANALYZER <a>
// BM25` (not `SEARCH ANALYZER`); `type::record`/`type::thing` do not exist on
// this build — record refs are built with the `RecordId` class instead;
// `SELECT ... FROM type::table($t)` cannot use a FULLTEXT index (the planner
// needs a literal table name for that one call site — table names there come
// only from the SEARCHABLE_TABLES whitelist, never raw user input); a plain
// JS string passed as a date is stored as a string, not a real `datetime` —
// occurred_at/known_at must be cast to `Date` objects before they cross into
// SurrealQL for `<=`/`ORDER BY` comparisons to work correctly.

import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { randomUUID } from "node:crypto";
import { dirname, join } from "node:path";
import { migrateLibrary, migratePersonalCaseContext, putPersonalCaseSource, putVersionedPersonalRecord, retainLibraryImport } from "./case-library.js";
import { migrateLibrarySync } from "./library-sync-backend.js";
import { configuredLibrarySyncScope } from "./library-sync-integration.js";

// ---------------------------------------------------------------------------
// Tables, edges, factors
// ---------------------------------------------------------------------------

export const DATA_TABLES = [
  "person", "child", "order", "hearing", "deadline", "event",
  "message", "exhibit", "factor", "source", "note", "court",
  // Owner orders 2026-09-07 13:09-13:16 — additive registers:
  "court_event", "filing", "draft", "memo", "reference", "evidence_log", "eval", "case_status",
] as const;
export type DataTable = (typeof DATA_TABLES)[number];

export const EDGE_TABLES = [
  "evidences", "supports_factor", "contradicts_factor", "sent_by", "sent_to", "filed_in",
  // Owner orders 2026-09-07 13:09-13:16 — additive edges:
  "drafted_as", "responds_to", "entered_at", "logs", "evaluates", "about", "matches_pattern",
] as const;
export type EdgeTable = (typeof EDGE_TABLES)[number];

export const SEARCHABLE_TABLES = [
  "event", "message", "note", "exhibit",
  "memo", "filing", "draft", "court_event", "reference", "eval",
] as const;
export type SearchableTable = (typeof SEARCHABLE_TABLES)[number];

export const VECTOR_TABLES = ["event", "message", "note", "memo", "eval"] as const;
export type VectorTable = (typeof VECTOR_TABLES)[number];

const SEARCH_FIELD: Record<SearchableTable, string> = {
  event: "description",
  message: "body",
  note: "text",
  exhibit: "label",
  memo: "text",
  filing: "title",
  draft: "title",
  court_event: "title",
  reference: "definition",
  eval: "text",
};

/** Tables (beyond the two-clock occurred_at/known_at set) carrying their own
 * single date field(s) that must be cast to real `Date` objects before they
 * cross into SurrealQL (see the file-head comment). */
const SINGLE_DATE_FIELDS: Partial<Record<DataTable, string[]>> = {
  court_event: ["date"],
  filing: ["date"],
  draft: ["date"],
  evidence_log: ["logged_at"],
  eval: ["created_at"],
};

const SUMMARY_FIELD_CANDIDATES = ["title", "description", "label", "letter", "initials", "body", "text", "definition", "pattern", "county"];

// Tables carrying the platform's two-clock discipline (occurred_at = when the
// thing happened; known_at = when the owner learned it). Never treated as a
// horizon predicate elsewhere in the platform, but `known_by` filtering on
// this small local store is exactly the local analogue of that discipline.
const TWO_CLOCK_TABLES = new Set<DataTable>(["event", "message", "exhibit"]);

export const FACTORS: ReadonlyArray<{ letter: string; title: string }> = [
  { letter: "a", title: "Love, affection, and emotional ties" },
  { letter: "b", title: "Capacity and disposition to provide love, affection, and guidance" },
  { letter: "c", title: "Capacity to provide material needs" },
  { letter: "d", title: "Length of time in a stable, satisfactory environment" },
  { letter: "e", title: "Permanence of the proposed custodial home" },
  { letter: "f", title: "Moral fitness of the parties" },
  { letter: "g", title: "Mental and physical health of the parties" },
  { letter: "h", title: "Child's home, school, and community record" },
  { letter: "i", title: "Child's reasonable preference" },
  { letter: "j", title: "Willingness to facilitate a relationship with the other parent" },
  { letter: "k", title: "Domestic violence" },
  { letter: "l", title: "Any other relevant factor" },
];

const DEFAULT_EMBED_DIM = 2048;
const EMBED_MODEL = "nvidia/nemotron-3-embed-1b";
const EMBED_URL = "https://integrate.api.nvidia.com/v1/embeddings";
// Kept in sync with .claude-plugin/plugin.json's "version" by hand (store.ts has no
// build-time access to that file) — bump both together.
const PLUGIN_VERSION = "3.1.0";

function ident(table: string): string {
  return "`" + table.replace(/`/g, "") + "`";
}

function isDataTable(value: unknown): value is DataTable {
  return typeof value === "string" && (DATA_TABLES as readonly string[]).includes(value);
}

function isEdgeTable(value: unknown): value is EdgeTable {
  return typeof value === "string" && (EDGE_TABLES as readonly string[]).includes(value);
}

function isSearchableTable(value: unknown): value is SearchableTable {
  return typeof value === "string" && (SEARCHABLE_TABLES as readonly string[]).includes(value);
}

// ---------------------------------------------------------------------------
// Config: db path, embedding dimension, NVIDIA/NIM key discovery
// ---------------------------------------------------------------------------

export function defaultDbPath(): string {
  return join(homedir(), ".config", "family-court-toolkit", "case.db");
}

/**
 * `scheme://C:/Users/...` mis-parses the Windows drive letter as a URL
 * "authority" (host) and silently drops it — verified live: connecting to
 * `rocksdb://C:/Users/matts/...case.db` "succeeds" but actually creates the
 * database at `<process cwd>/C/Users/matts/...case.db` (a RELATIVE path,
 * losing the drive entirely), not the intended absolute location. The fix is
 * the scheme's OPAQUE form — no `//` — under which `rocksdb:C:/Users/...`
 * parses correctly as one absolute path (verified live, same content
 * appears at the real `C:\Users\...` location). This only ever applies to a
 * Windows absolute drive path; relative paths and `mem://` are untouched.
 */
function toOpaqueIfWindowsDriveAbsolute(url: string): string {
  const match = /^([a-z][a-z0-9+.-]*):\/\/([A-Za-z]:[/\\].*)$/.exec(url);
  return match ? `${match[1]}:${match[2]}` : url;
}

/** Resolve an explicitly selected shared endpoint without creating a private fallback database.
 * Inputs: configured URL and whether an isolated caller explicitly supplied a local engine.
 * Outputs: normalized database URL. Effects: none; rejects missing or implicit local configuration.
 * Choose before opening a connection so configuration failure cannot split the shared case store.
 * Byline: Codex · GPT-6 · 2026-10-04.
 */
export function resolveConfiguredDbUrl(raw: string | undefined, explicitLocal = false): string {
  const value = raw?.trim();
  if (!value) throw new Error("Shared case store is not configured: set CUSTODY_CASE_DB to the hosted SurrealDB endpoint");
  if (/^(wss?|https?):\/\//i.test(value)) return value;
  if (!explicitLocal) throw new Error("CUSTODY_CASE_DB must select the shared hosted SurrealDB endpoint; local engines require an explicit isolated override");
  const url = value.includes("://") || /^[a-z][a-z0-9+.-]*:[A-Za-z]:[/\\]/i.test(value)
    ? value : `rocksdb://${value.replace(/\\/g, "/")}`;
  return toOpaqueIfWindowsDriveAbsolute(url);
}

function resolveDbUrl(override?: string): string {
  const raw = override ?? (process.env.CUSTODY_CASE_DB?.trim() || findSecretsValue("CUSTODY_CASE_DB"));
  return resolveConfiguredDbUrl(raw, override !== undefined);
}

function parsePositiveInt(value: string | undefined): number | null {
  if (!value) return null;
  const n = Number.parseInt(value, 10);
  return Number.isInteger(n) && n > 0 ? n : null;
}

/** Tolerant `KEY = value` line parser — never uses `source`, never prints values. */
function parseEnvLine(line: string, keys: string[]): { key: string; value: string } | null {
  const match = /^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$/.exec(line);
  if (!match) return null;
  const [, key, rawValue] = match;
  if (!keys.includes(key)) return null;
  let value = rawValue;
  if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
    value = value.slice(1, -1);
  }
  return value ? { key, value } : null;
}

/** First value of `key` across ~/.secrets/*.env (tolerant line parse; never `source`d, never printed). */
function findSecretsValue(key: string): string | undefined {
  const dir = join(homedir(), ".secrets");
  if (!existsSync(dir)) return undefined;
  let files: string[] = [];
  try {
    files = readdirSync(dir).filter((f) => f.endsWith(".env"));
  } catch {
    return undefined;
  }
  for (const file of files) {
    try {
      for (const line of readFileSync(join(dir, file), "utf8").split(/\r?\n/)) {
        const hit = parseEnvLine(line, [key]);
        if (hit) return hit.value;
      }
    } catch {
      // unreadable — skip
    }
  }
  return undefined;
}

/** Root credentials for a shared SurrealDB server (ws:// mode). Same discovery rules as the
 * NVIDIA key: env first, then `~/.secrets/*.env` parsed line-by-line. Values are never logged. */
function findServerCredentials(): { username: string; password: string } | undefined {
  const fromEnv = { username: process.env.CUSTODY_CASE_DB_USER, password: process.env.CUSTODY_CASE_DB_PASS };
  if (fromEnv.username && fromEnv.password) return { username: fromEnv.username, password: fromEnv.password };
  const dir = join(homedir(), ".secrets");
  if (!existsSync(dir)) return undefined;
  let files: string[] = [];
  try {
    files = readdirSync(dir).filter((f) => f.endsWith(".env"));
  } catch {
    return undefined;
  }
  const found: Record<string, string> = {};
  for (const file of files) {
    try {
      for (const line of readFileSync(join(dir, file), "utf8").split(/\r?\n/)) {
        const hit = parseEnvLine(line, ["CUSTODY_CASE_DB_USER", "CUSTODY_CASE_DB_PASS"]);
        if (hit && !found[hit.key]) found[hit.key] = hit.value;
      }
    } catch {
      // unreadable file — skip
    }
    if (found.CUSTODY_CASE_DB_USER && found.CUSTODY_CASE_DB_PASS) break;
  }
  return found.CUSTODY_CASE_DB_USER && found.CUSTODY_CASE_DB_PASS
    ? { username: found.CUSTODY_CASE_DB_USER, password: found.CUSTODY_CASE_DB_PASS }
    : undefined;
}

function findNvidiaApiKey(): string | undefined {
  if (process.env.NVIDIA_API_KEY) return process.env.NVIDIA_API_KEY;
  if (process.env.NIM_API_KEY) return process.env.NIM_API_KEY;
  const dir = join(homedir(), ".secrets");
  if (!existsSync(dir)) return undefined;
  let files: string[] = [];
  try {
    files = readdirSync(dir).filter((f) => f.endsWith(".env"));
  } catch {
    return undefined;
  }
  for (const file of files) {
    try {
      const content = readFileSync(join(dir, file), "utf8");
      for (const line of content.split(/\r?\n/)) {
        const found = parseEnvLine(line, ["NVIDIA_API_KEY", "NIM_API_KEY"]);
        if (found) return found.value;
      }
    } catch {
      // unreadable file — skip, never throw on secret discovery
    }
  }
  return undefined;
}

function stripDataUris(text: string): string {
  return text.replace(/data:image\/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+/g, "[image omitted]");
}

/** Test-only escape hatch: force "no embeddings configured" regardless of what
 * ~/.secrets/*.env holds on the developer's machine, so degrade-to-text tests
 * are deterministic. Never set this in production. */
function embeddingsDisabledForTests(): boolean {
  return process.env.CUSTODY_DISABLE_EMBEDDINGS === "1";
}

/**
 * Calls the NVIDIA embeddings API if a key is configured; returns null (never
 * throws) when no key is available so callers can degrade to text-only search.
 * Embeddings are optional everywhere in this store.
 */
export async function embed(text: string): Promise<number[] | null> {
  if (embeddingsDisabledForTests()) return null;
  const apiKey = findNvidiaApiKey();
  if (!apiKey) return null;
  const cleaned = stripDataUris(text);
  const res = await fetch(EMBED_URL, {
    method: "POST",
    headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
    body: JSON.stringify({ input: [cleaned], model: EMBED_MODEL }),
  });
  if (!res.ok) {
    throw new Error(`NVIDIA embeddings API returned HTTP ${res.status} for model ${EMBED_MODEL}`);
  }
  const json = (await res.json()) as { data?: Array<{ embedding?: number[] }> };
  const vector = json.data?.[0]?.embedding;
  if (!Array.isArray(vector)) throw new Error("NVIDIA embeddings API returned no embedding vector");
  return vector;
}

export function embeddingsConfigured(): boolean {
  return !embeddingsDisabledForTests() && Boolean(findNvidiaApiKey());
}

// ---------------------------------------------------------------------------
// Connection + migration (lazy, cached, gracefully degrading)
// ---------------------------------------------------------------------------

// Minimal structural types for the pieces of the surrealdb driver we touch —
// kept local so this file has zero *static* dependency on the `surrealdb`
// package (only ever loaded via dynamic import).
interface RecordIdLike {
  table: string;
  id: unknown;
  toString(): string;
}
interface SurrealLike {
  connect(url: string, opts?: unknown): Promise<unknown>;
  signin?(auth: { username: string; password: string }): Promise<unknown>;
  use(opts: { namespace: string; database: string }): Promise<unknown>;
  query<T = unknown[]>(surql: string, params?: Record<string, unknown>): Promise<T[]>;
  close(): Promise<unknown>;
}
interface SurrealModule {
  Surreal: new (opts: { engines: unknown }) => SurrealLike;
  RecordId: new (table: string, id: unknown) => RecordIdLike;
}

let cachedRecordIdCtor: SurrealModule["RecordId"] | null = null;

export interface StoreOk {
  available: true;
  db: SurrealLike;
  dim: number;
}
export interface StoreErr {
  available: false;
  reason: string;
}
export type StoreResult = StoreOk | StoreErr;

let cachedStore: Promise<StoreResult> | null = null;
let cachedUrl: string | null = null;
const openedConnections = new Set<SurrealLike>();

/** Test-only: drops the cached connection so a fresh mem:// store can be opened. */
export function resetStoreForTests(): void {
  cachedStore = null;
  cachedUrl = null;
}

/** Closes every store connection opened via getStore() during this process.
 * The embedded engine's native connection otherwise keeps the event loop
 * alive (mem://) or, worse, can leave a rocksdb LOCK file held past an
 * abrupt process exit (file-backed) — CLI scripts (case-cli.mjs,
 * import-vincent-schema.mjs) call this once before exiting; tests call it
 * from an `after()` hook for the same reason (verified live: without this,
 * `node --test tests/store.test.mjs` finishes all assertions in well under a
 * second but the process itself never exits on its own). */
export async function closeAllStoresForTests(): Promise<void> {
  resetStoreForTests();
  const closes = [...openedConnections].map((db) => db.close().catch(() => undefined));
  openedConnections.clear();
  await Promise.all(closes);
}

/** Production-facing alias for CLI scripts (case-cli.mjs,
 * import-vincent-schema.mjs): call once before `process.exit()` so a
 * file-backed (rocksdb://) connection releases its LOCK file cleanly. */
export const closeStore = closeAllStoresForTests;

async function loadDriver(): Promise<{ ok: true; mod: SurrealModule } | { ok: false; reason: string }> {
  try {
    const surrealdb = (await import("surrealdb")) as unknown as SurrealModule;
    const node = (await import("@surrealdb/node")) as unknown as { createNodeEngines: () => unknown };
    cachedRecordIdCtor = surrealdb.RecordId;
    return { ok: true, mod: { Surreal: surrealdb.Surreal, RecordId: surrealdb.RecordId } as SurrealModule & { createNodeEngines?: unknown } };
  } catch (err) {
    return {
      ok: false,
      reason:
        `surrealdb/@surrealdb/node failed to load: ${err instanceof Error ? err.message : String(err)}. ` +
        `Run "npm install surrealdb @surrealdb/node" then "npm ci --omit=dev" in mcp-app — ` +
        `@surrealdb/node ships native .node binaries that are not bundled and must ship as real node_modules. ` +
        `See README-store.md.`,
    };
  }
}

async function openStore(url: string): Promise<StoreResult> {
  const driver = await loadDriver();
  if (!driver.ok) return { available: false, reason: driver.reason };
  let createNodeEngines: () => unknown;
  try {
    const nodeMod = (await import("@surrealdb/node")) as { createNodeEngines: () => unknown };
    createNodeEngines = nodeMod.createNodeEngines;
  } catch (err) {
    return { available: false, reason: `@surrealdb/node failed to load: ${err instanceof Error ? err.message : String(err)}` };
  }

  try {
    const remote = /^(wss?|https?):\/\//i.test(url);
    if (!remote && !url.startsWith("mem://")) {
      // rocksdb:// (and any other file-backed engine) URLs carry a plain
      // filesystem path after the scheme, forward-slashed even on Windows
      // (verified live) — node's path.dirname() accepts both separators.
      // Strips either the normal (scheme://path) or opaque/Windows-drive
      // (scheme:path, see toOpaqueIfWindowsDriveAbsolute) scheme prefix.
      const fsPath = url.replace(/^[a-z0-9]+:\/{0,2}/i, "");
      const parent = dirname(fsPath);
      if (parent && parent !== "." && !existsSync(parent)) {
        mkdirSync(parent, { recursive: true });
      }
    }
    // Remote URLs (ws/wss/http/https -> the shared SurrealDB service in ~) must use the SDK's
    // built-in engines: passing createNodeEngines() REPLACES the engine map with mem/rocksdb/
    // surrealkv only, which is why ws:// failed with 'The engine "ws" is not supported' (2026-09-07).
    const db = remote
      ? new (driver.mod.Surreal as unknown as new () => SurrealLike)()
      : new driver.mod.Surreal({ engines: createNodeEngines() });
    // Shared-server mode (owner ruling 2026-09-07 15:30: one SurrealDB service in ~ so every
    // harness shares the same data). Embedded rocksdb:/mem: URLs need no auth; a remote
    // ws://|wss://|http(s):// URL authenticates with root credentials resolved from env
    // (CUSTODY_CASE_DB_USER / CUSTODY_CASE_DB_PASS) else ~/.secrets/*.env (tolerant parse,
    // never `source`d, never printed). Missing creds -> connect stays unauthenticated so a
    // `--unauthenticated` dev server still works.
    // The credentials go to connect() as the `authentication` provider, not to a one-off
    // .signin(): a signin token expires (1 h) and the driver then has nothing to renew with,
    // so the long-running console fell to "Anonymous access not allowed" on every store call
    // (found 2026-10-02). The provider is re-invoked whenever the session expires or the
    // socket reconnects. (Claude Code · Opus 5.5 · 2026-10-02)
    const creds = remote ? findServerCredentials() : null;
    await db.connect(url, creds ? { authentication: () => creds, namespace: "fct", database: "case" } : undefined);
    await db.use({ namespace: "fct", database: "case" });
    const dim = await migrate(db, driver.mod);
    openedConnections.add(db);
    return { available: true, db, dim };
  } catch (err) {
    return { available: false, reason: `failed to open case store at ${url}: ${err instanceof Error ? err.message : String(err)}` };
  }
}

export async function getStore(urlOverride?: string): Promise<StoreResult> {
  let url: string;
  try { url = resolveDbUrl(urlOverride); }
  catch (error) { return { available: false, reason: error instanceof Error ? error.message : String(error) }; }
  if (cachedStore && cachedUrl === url) return cachedStore;
  cachedUrl = url;
  cachedStore = openStore(url);
  return cachedStore;
}

async function migrate(db: SurrealLike, mod: SurrealModule): Promise<number> {
  await migrateLibrary({ db });
  if (configuredLibrarySyncScope()) {
    await migrateLibrarySync({ db });
    await db.query("DEFINE TABLE IF NOT EXISTS library_file_alias SCHEMALESS PERMISSIONS NONE; DEFINE INDEX IF NOT EXISTS library_file_alias_unique ON library_file_alias FIELDS record_id, binding_id UNIQUE;");
  }
  await migratePersonalCaseContext({ db });
  await db.query("DEFINE TABLE IF NOT EXISTS meta SCHEMALESS;");

  const existing = await db.query<Array<{ embed_dim?: number }>>("SELECT embed_dim FROM meta:config;");
  const existingDim = existing?.[0]?.[0]?.embed_dim;
  const envDim = parsePositiveInt(process.env.CUSTODY_EMBED_DIM) ?? DEFAULT_EMBED_DIM;
  const dim = typeof existingDim === "number" && existingDim > 0 ? existingDim : envDim;

  await db.query("UPSERT meta:config MERGE $data;", { data: { embed_dim: dim, embed_model: EMBED_MODEL, updated_at: new Date() } });

  const ddl: string[] = [];
  for (const table of DATA_TABLES) {
    if (table === "child") continue; // personal-context migration retains existing fields above
    ddl.push(`DEFINE TABLE IF NOT EXISTS ${ident(table)} SCHEMALESS;`);
  }
  for (const edge of EDGE_TABLES) {
    ddl.push(`DEFINE TABLE IF NOT EXISTS ${ident(edge)} TYPE RELATION SCHEMALESS;`);
  }

  ddl.push("DEFINE ANALYZER IF NOT EXISTS en TOKENIZERS class FILTERS lowercase, snowball(english);");
  for (const table of SEARCHABLE_TABLES) {
    const field = SEARCH_FIELD[table];
    ddl.push(`DEFINE INDEX IF NOT EXISTS ${table}_text ON ${ident(table)} FIELDS ${field} FULLTEXT ANALYZER en BM25;`);
  }
  for (const table of VECTOR_TABLES) {
    ddl.push(`DEFINE INDEX IF NOT EXISTS ${table}_vec ON ${ident(table)} FIELDS embedding HNSW DIMENSION ${dim} DIST COSINE;`);
  }

  for (const stmt of ddl) {
    await db.query(stmt);
  }

  // Seed the 12 MCL 722.23 factors (idempotent: MERGE onto a stable id).
  for (const factor of FACTORS) {
    await db.query("IF (SELECT VALUE id FROM ONLY $rid) = NONE { CREATE $rid CONTENT $data; };", { rid: new mod.RecordId("factor", factor.letter), data: { letter: factor.letter, title: factor.title } });
  }
  // One court record, per the "keep it simple" instruction — created empty
  // so `filed_in` edges and case_summary always have a stable target.
  await db.query("UPSERT $rid MERGE $data;", { rid: new mod.RecordId("court", "main"), data: {} });
  // Singleton case_status record — created empty so case_status/case_docket always
  // have a stable target even before the owner has set anything (D-2026-09-07 orders).
  await db.query("UPSERT $rid MERGE $data;", { rid: new mod.RecordId("case_status", "current"), data: {} });

  return dim;
}

// ---------------------------------------------------------------------------
// Record reference helpers
// ---------------------------------------------------------------------------

export type RecordRef = string | { table: string; id: string };

function recordIdCtor(): SurrealModule["RecordId"] {
  if (!cachedRecordIdCtor) throw new Error("store not connected yet — call getStore() first");
  return cachedRecordIdCtor;
}

/**
 * This SurrealDB build's `RecordId.toString()` — and therefore every id this
 * store hands back to a caller via `normalize()`/`refToString()` — renders an
 * id containing characters outside `[A-Za-z0-9_]` (a hyphen, most commonly:
 * the case-extract/v1 `<row>-<seq>` id shape, e.g. "A1-7") wrapped in
 * SurrealQL's own quoted-identifier brackets, e.g. `memo:⟨probe-memo-1⟩`
 * rather than `memo:probe-memo-1` — verified live. Without unwrapping those
 * brackets here, feeding a ref STRING BACK that this store itself just
 * displayed (search hit id, docket entry id, timeline entry id, case_source's
 * own `id`, ...) into case_source/case_graph/case_put's `relations` would
 * silently construct a DIFFERENT record (literal brackets baked into the id)
 * than the one actually stored. Strip a matching leading `⟨`/trailing `⟩`
 * pair before constructing the RecordId so the round trip is exact.
 */
function unwrapQuotedId(idPart: string): string {
  if (idPart.startsWith("⟨") && idPart.endsWith("⟩") && idPart.length >= 2) return idPart.slice(1, -1);
  return idPart;
}

/** Applies `unwrapQuotedId` to the id half of a full "table:id" string, so
 * every DISPLAYED ref (search hits, docket/timeline entries, case_source,
 * graph neighbors, ...) shows the clean id a caller actually wrote, not
 * SurrealDB's internal quoted form. */
function unwrapQuotedRefString(str: string): string {
  const idx = str.indexOf(":");
  if (idx < 1) return str;
  return `${str.slice(0, idx)}:${unwrapQuotedId(str.slice(idx + 1))}`;
}

export function parseRef(ref: RecordRef): RecordIdLike {
  const RecordId = recordIdCtor();
  if (typeof ref === "string") {
    const idx = ref.indexOf(":");
    if (idx < 1 || idx === ref.length - 1) throw new Error(`Invalid record reference "${ref}"; expected "table:id".`);
    return new RecordId(ref.slice(0, idx), unwrapQuotedId(ref.slice(idx + 1)));
  }
  return new RecordId(ref.table, ref.id);
}

export function refToString(ref: RecordIdLike | string): string {
  const raw = typeof ref === "string" ? ref : ref.toString();
  return unwrapQuotedRefString(raw);
}

/** Deep-converts RecordId and SurrealDB `Datetime` wrapper values into plain
 * JSON-safe strings so tool results and test assertions never see driver
 * internals. */
export function normalize(value: unknown): unknown {
  if (value === null || value === undefined) return value;
  if (Array.isArray(value)) return value.map(normalize);
  if (value instanceof Date) return value.toISOString();
  if (typeof value === "object") {
    // Record refs (`RecordId` instances): duck-typed by property presence,
    // NOT `typeof x.table === "string"` — the driver's `.table` getter
    // returns a `Table`-like wrapper object, not a raw string primitive
    // (verified live: `typeof recordId.table === "object"` even though it
    // prints/stringifies as the table name).
    if ("table" in (value as object) && "id" in (value as object) && typeof (value as RecordIdLike).toString === "function") {
      let asString: string | undefined;
      try {
        asString = (value as RecordIdLike).toString();
      } catch {
        asString = undefined;
      }
      // unwrapQuotedRefString: strips SurrealDB's own quoted-identifier
      // brackets (⟨...⟩) around an id containing non-simple characters (e.g.
      // a hyphen — every case-extract/v1 id, "<row>-<seq>") so callers see
      // the plain id they wrote, and so the displayed ref round-trips
      // correctly back through parseRef.
      if (typeof asString === "string" && /^[^:\s]+:.+$/.test(asString)) return unwrapQuotedRefString(asString);
    }
    // SurrealDB `Datetime` wrapper values duck-type on `toISOString()`.
    const maybeDatetime = value as { toISOString?: () => string };
    if (typeof maybeDatetime.toISOString === "function") {
      try {
        return maybeDatetime.toISOString();
      } catch {
        // fall through to generic object handling
      }
    }
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) out[k] = normalize(v);
    return out;
  }
  return value;
}

/** Casts every date-shaped field on `table` (the two-clock occurred_at/known_at
 * pair on TWO_CLOCK_TABLES, plus any single date field(s) listed in
 * SINGLE_DATE_FIELDS — court_event.date, filing.date, draft.date,
 * evidence_log.logged_at, eval.created_at) from an incoming ISO string to a
 * real `Date` object, so `<=`/`ORDER BY` work in SurrealQL. Renamed from
 * castTwoClocks (2026-09-07) when the single-date-field tables were added;
 * behavior for the original two-clock tables is unchanged. */
function castDateFields(table: DataTable, data: Record<string, unknown>): Record<string, unknown> {
  const fields: string[] = [
    ...(TWO_CLOCK_TABLES.has(table) ? ["occurred_at", "known_at"] : []),
    ...(SINGLE_DATE_FIELDS[table] ?? []),
  ];
  if (fields.length === 0) return data;
  const out = { ...data };
  for (const field of fields) {
    const value = out[field];
    if (typeof value === "string") {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) throw new Error(`Invalid ${field} value "${value}" on table ${table}; expected an ISO 8601 datetime string.`);
      out[field] = date;
    }
  }
  return out;
}

// ---------------------------------------------------------------------------
// case_put
// ---------------------------------------------------------------------------

export interface CasePutRelation {
  edge: EdgeTable;
  from: RecordRef;
  to: RecordRef;
  data?: Record<string, unknown>;
}

export interface CasePutInput {
  table: DataTable;
  id?: string;
  expected_version?: string;
  data: Record<string, unknown>;
  relations?: CasePutRelation[];
}

export interface CasePutResult {
  table: DataTable;
  id: string;
  record: Record<string, unknown>;
  relations: Array<{ edge: EdgeTable; from: string; to: string }>;
}

/** Save admitted private case fields and optionally retain an exact-version edit atomically.
 * Inputs: table/key, changed fields, optional expected version and relations. Outputs: shared saved record and relation identities.
 * Effects: shared store merge/revision and requested edges. Choose for personal records; legal library edits use library proposals.
 * Byline: Codex · GPT-6 · 2026-10-04.
 */
export async function casePut(store: StoreOk, input: CasePutInput): Promise<CasePutResult> {
  if (input.table === "reference" || (input.table === "source" && input.data?.kind !== "case_document")) {
    throw new Error("Library changes require library_propose and version-bound citation validation before library_publish");
  }
  if (!isDataTable(input.table)) throw new Error(`Unknown table "${input.table}". Known tables: ${DATA_TABLES.join(", ")}.`);
  if (input.expected_version !== undefined && !input.id) throw new Error("Version-bound edits require an explicit record id");
  const casted = castDateFields(input.table, input.data ?? {});

  // court_event.status: computed from date vs now WHEN ABSENT FROM THIS CALL'S
  // OWN data — never when this put is a partial patch that didn't touch `date`
  // (that would clobber an already-set status, e.g. "adjourned", on a MERGE
  // that only meant to update `outcome`). Owner order 2026-09-07 13:09-13:16.
  if (input.table === "court_event" && !Object.prototype.hasOwnProperty.call(input.data ?? {}, "status")) {
    const dateVal = casted.date;
    if (dateVal instanceof Date) {
      casted.status = dateVal.getTime() < Date.now() ? "past" : "upcoming";
    }
  }

  let record: Record<string, unknown>;
  const idString = input.id ?? randomUUID();
  const current = input.expected_version === undefined && input.id
    ? await caseRecord(store, { table: input.table, id: idString }) : null;
  const expected = input.expected_version ?? current?.version ?? "absent";
  // Legacy callers still retain revisions; exact-version clients additionally bind their observed edit base.
  if (input.table === "source") record = await putPersonalCaseSource(store, idString, casted, expected);
  else record = await putVersionedPersonalRecord(store, input.table, idString, casted, expected);

  const relations: Array<{ edge: EdgeTable; from: string; to: string }> = [];
  for (const rel of input.relations ?? []) {
    if (!isEdgeTable(rel.edge)) throw new Error(`Unknown edge "${rel.edge}". Known edges: ${EDGE_TABLES.join(", ")}.`);
    const from = parseRef(rel.from);
    const to = parseRef(rel.to);
    const query = rel.data && Object.keys(rel.data).length > 0
      ? `RELATE $from->${ident(rel.edge)}->$to CONTENT $data;`
      : `RELATE $from->${ident(rel.edge)}->$to;`;
    await store.db.query(query, { from, to, data: rel.data ?? {} });
    relations.push({ edge: rel.edge, from: refToString(from), to: refToString(to) });
  }

  return { table: input.table, id: idString, record: normalize(record) as Record<string, unknown>, relations };
}

// ---------------------------------------------------------------------------
// case_query
// ---------------------------------------------------------------------------

const ROW_CAP = 200;

export interface CaseQueryInput {
  surql: string;
  params?: Record<string, unknown>;
  write?: boolean;
}

export interface CaseQueryResult {
  results: unknown[];
  truncated: boolean;
}

function assertQueryAllowed(surql: string, write: boolean | undefined): void {
  if (write) throw new Error("Raw case queries are read-only; use governed record or library tools for writes");
  const lexical = surql.replace(/'(?:\\.|[^'\\])*'|"(?:\\.|[^"\\])*"|`(?:\\.|[^`\\])*`|\/\*[\s\S]*?\*\/|--[^\r\n]*/g, " ");
  const statements = lexical.split(";").map(s => s.trim()).filter(Boolean);
  if (!statements.length || statements.some(s => !/^SELECT\b/i.test(s))
    || /\b(?:CREATE|INSERT|UPSERT|UPDATE|DELETE|REMOVE|DEFINE|RELATE|ALTER|REBUILD|FUNCTION|FOR|LET|BEGIN|COMMIT|CANCEL)\b|\b(?:fn|http|api|script)\s*::/i.test(lexical))
    throw new Error("Raw case queries accept SELECT-only inspection without mutation or external function calls");
}

export async function caseQuery(store: StoreOk, input: CaseQueryInput): Promise<CaseQueryResult> {
  assertQueryAllowed(input.surql, input.write);
  const raw = await store.db.query<unknown[]>(input.surql, input.params ?? {});
  let truncated = false;
  const capped = raw.map((statementResult) => {
    if (Array.isArray(statementResult) && statementResult.length > ROW_CAP) {
      truncated = true;
      return statementResult.slice(0, ROW_CAP);
    }
    return statementResult;
  });
  return { results: normalize(capped) as unknown[], truncated };
}

// ---------------------------------------------------------------------------
// case_search
// ---------------------------------------------------------------------------

export interface CaseSearchInput {
  query: string;
  tables?: SearchableTable[];
  k?: number;
  mode?: "text" | "vector" | "hybrid";
}

export interface CaseSearchHit {
  id: string;
  table: SearchableTable;
  snippet: string;
  score: number;
  occurred_at: string | null;
  known_at: string | null;
  /** Provenance block ({path, sha256, r2_path, locator, url, row, extract_id}),
   * when the record carries one — owner order 2026-09-07: every record links back
   * to its original source document. */
  source: Record<string, unknown> | null;
}

export interface CaseSearchResult {
  hits: CaseSearchHit[];
  mode: "text" | "vector" | "hybrid";
  degraded: boolean;
  degraded_reason: string | null;
}

function snippetOf(text: unknown): string {
  const s = typeof text === "string" ? text : "";
  return s.length > 220 ? `${s.slice(0, 217)}...` : s;
}

/**
 * `search::score()` on this SurrealDB build uses the classical (non-smoothed)
 * Robertson-Sparck-Jones IDF term `ln((N - n + 0.5) / (n + 0.5))`, which is
 * exactly zero (or negative, clamped to zero) whenever the matching term
 * appears in >= 50% of the indexed corpus — verified live by growing a test
 * corpus one document at a time: score stayed exactly 0 through 2 total
 * documents (1 match / 2 = 50%), then became positive and grew normally from
 * 3 documents on. This is NOT a query-construction bug (the `@1@` reference
 * number already matches `search::score(1)`'s argument, and explicit
 * `BM25(1.2,0.75)` parameters make no difference) — it is a real, documented
 * property of that IDF formula that is mathematically guaranteed to degenerate
 * for exactly the corpus sizes a case store has for its first few records.
 * Since the row was only returned because it matched the FULLTEXT predicate,
 * a reported score of 0 would misleadingly read as "not relevant" rather
 * than "BM25 can't yet differentiate this tiny corpus". This floor makes
 * every match report a genuine positive, differentiated score even before
 * BM25's statistics have enough documents to do that on their own.
 */
function naiveOverlapScore(query: string, text: string): number {
  const queryTerms = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (queryTerms.length === 0) return 0.01;
  const lowerText = text.toLowerCase();
  const matched = queryTerms.filter((term) => lowerText.includes(term)).length;
  return Math.max(matched / queryTerms.length, 0.01);
}

async function textSearch(store: StoreOk, tables: SearchableTable[], query: string, k: number): Promise<CaseSearchHit[]> {
  const hits: CaseSearchHit[] = [];
  for (const table of tables) {
    const field = SEARCH_FIELD[table];
    const hasClocks = TWO_CLOCK_TABLES.has(table);
    const clockSelect = hasClocks ? ", occurred_at, known_at" : "";
    const rows = await store.db.query<Array<Record<string, unknown>>>(
      `SELECT id, ${field} AS text${clockSelect}, source, search::score(1) AS score FROM ${ident(table)} WHERE ${field} @1@ $q ORDER BY score DESC LIMIT $k;`,
      { q: query, k },
    );
    for (const row of rows.at(-1) ?? []) {
      const text = typeof row.text === "string" ? row.text : "";
      const rawScore = typeof row.score === "number" ? row.score : 0;
      hits.push({
        id: refToString(row.id as RecordIdLike),
        table,
        snippet: snippetOf(row.text),
        score: rawScore > 0 ? rawScore : naiveOverlapScore(query, text),
        occurred_at: row.occurred_at ? (normalize(row.occurred_at) as string) : null,
        known_at: row.known_at ? (normalize(row.known_at) as string) : null,
        source: row.source ? (normalize(row.source) as Record<string, unknown>) : null,
      });
    }
  }
  return hits;
}

async function vectorSearch(store: StoreOk, tables: SearchableTable[], vector: number[], k: number): Promise<CaseSearchHit[]> {
  const hits: CaseSearchHit[] = [];
  const eligible = tables.filter((t): t is VectorTable => (VECTOR_TABLES as readonly string[]).includes(t));
  for (const table of eligible) {
    const field = SEARCH_FIELD[table];
    const hasClocks = TWO_CLOCK_TABLES.has(table);
    const clockSelect = hasClocks ? ", occurred_at, known_at" : "";
    const rows = await store.db.query<Array<Record<string, unknown>>>(
      `SELECT id, ${field} AS text${clockSelect}, source, vector::distance::knn() AS distance FROM ${ident(table)} WHERE embedding <|${k},COSINE|> $v ORDER BY distance;`,
      { v: vector },
    );
    for (const row of rows.at(-1) ?? []) {
      const distance = typeof row.distance === "number" ? row.distance : 1;
      hits.push({
        id: refToString(row.id as RecordIdLike),
        table,
        snippet: snippetOf(row.text),
        score: 1 - distance, // cosine distance -> similarity-ish score for merging
        occurred_at: row.occurred_at ? (normalize(row.occurred_at) as string) : null,
        known_at: row.known_at ? (normalize(row.known_at) as string) : null,
        source: row.source ? (normalize(row.source) as Record<string, unknown>) : null,
      });
    }
  }
  return hits;
}

/**
 * Reciprocal-rank fusion across the text and vector hit lists. RRF's own
 * per-item contribution (`1/(60+rank)`) is an internal ranking weight, not a
 * meaningful relevance number on the scale a caller would expect from
 * `score` — text (BM25) and vector (1 - cosine distance) scores are on
 * different scales, so RRF's rank-based combination is the correct way to
 * ORDER results, but the `score` a caller sees should be the REAL underlying
 * text or vector score, not the arbitrary RRF weight. When a record is
 * found by both searches, the larger of its two real scores is kept.
 */
function fuse(textHits: CaseSearchHit[], vectorHits: CaseSearchHit[], k: number): CaseSearchHit[] {
  const RRF_K = 60;
  const combined = new Map<string, { hit: CaseSearchHit; rank: number }>();
  const apply = (list: CaseSearchHit[]) => {
    list.forEach((hit, index) => {
      const rank = index + 1;
      const contribution = 1 / (RRF_K + rank);
      const existing = combined.get(hit.id);
      if (existing) {
        existing.rank += contribution;
        if (hit.score > existing.hit.score) existing.hit = hit;
      } else {
        combined.set(hit.id, { hit, rank: contribution });
      }
    });
  };
  apply(textHits);
  apply(vectorHits);
  return [...combined.values()]
    .sort((a, b) => b.rank - a.rank)
    .slice(0, k)
    .map((entry) => entry.hit);
}

export async function caseSearch(store: StoreOk, input: CaseSearchInput): Promise<CaseSearchResult> {
  const tables = (input.tables?.filter(isSearchableTable) ?? [...SEARCHABLE_TABLES]) as SearchableTable[];
  const k = input.k ?? 10;
  const mode = input.mode ?? "hybrid";

  const textHits = mode === "vector" ? [] : await textSearch(store, tables, input.query, k);

  if (mode === "text") {
    return { hits: textHits.slice(0, k), mode, degraded: false, degraded_reason: null };
  }

  let vector: number[] | null = null;
  let degradedReason: string | null = null;
  try {
    vector = await embed(input.query);
    if (!vector) degradedReason = "No NVIDIA_API_KEY/NIM_API_KEY configured; embeddings are unavailable.";
  } catch (err) {
    degradedReason = `Embedding call failed: ${err instanceof Error ? err.message : String(err)}`;
  }

  if (!vector) {
    if (mode === "vector") return { hits: [], mode, degraded: true, degraded_reason: degradedReason };
    return { hits: textHits.slice(0, k), mode: "text", degraded: true, degraded_reason: degradedReason };
  }

  const vectorHits = await vectorSearch(store, tables, vector, k);
  if (mode === "vector") return { hits: vectorHits.slice(0, k), mode, degraded: false, degraded_reason: null };
  return { hits: fuse(textHits, vectorHits, k), mode, degraded: false, degraded_reason: null };
}

// ---------------------------------------------------------------------------
// case_graph + case_factor_map
// ---------------------------------------------------------------------------

export interface CaseGraphNeighbor {
  edge: EdgeTable;
  direction: "out" | "in";
  id: string;
  summary: string;
}

export interface CaseGraphResult {
  id: string;
  depth: 1 | 2;
  neighbors: CaseGraphNeighbor[];
}

function summarize(record: Record<string, unknown> | undefined, id: string): string {
  if (!record) return id;
  for (const field of SUMMARY_FIELD_CANDIDATES) {
    const value = record[field];
    if (typeof value === "string" && value.trim()) return value;
  }
  return id;
}

async function neighborsOf(store: StoreOk, rid: RecordIdLike, edges: EdgeTable[]): Promise<Array<{ edge: EdgeTable; direction: "out" | "in"; target: RecordIdLike }>> {
  const out: Array<{ edge: EdgeTable; direction: "out" | "in"; target: RecordIdLike }> = [];
  for (const edge of edges) {
    const outgoing = await store.db.query<Array<{ out: RecordIdLike }>>(`SELECT out FROM ${ident(edge)} WHERE in = $rid;`, { rid });
    for (const row of outgoing.at(-1) ?? []) out.push({ edge, direction: "out", target: row.out });
    const incoming = await store.db.query<Array<{ in: RecordIdLike }>>(`SELECT in FROM ${ident(edge)} WHERE out = $rid;`, { rid });
    for (const row of incoming.at(-1) ?? []) out.push({ edge, direction: "in", target: row.in });
  }
  return out;
}

async function resolveSummaries(store: StoreOk, ids: RecordIdLike[]): Promise<Map<string, Record<string, unknown>>> {
  const map = new Map<string, Record<string, unknown>>();
  if (ids.length === 0) return map;
  const rows = await store.db.query<Array<Record<string, unknown>>>("SELECT * FROM $ids;", { ids });
  for (const row of rows.at(-1) ?? []) {
    map.set(refToString(row.id as RecordIdLike), row);
  }
  return map;
}

export async function caseGraph(store: StoreOk, input: { id: RecordRef; depth?: 1 | 2; edges?: EdgeTable[] }): Promise<CaseGraphResult> {
  const rid = parseRef(input.id);
  const edges = (input.edges?.filter(isEdgeTable) as EdgeTable[] | undefined) ?? [...EDGE_TABLES];
  const depth = input.depth ?? 1;

  const level1 = await neighborsOf(store, rid, edges);
  const visited = new Set<string>([refToString(rid)]);
  const collected = [...level1];

  if (depth === 2) {
    const seen = new Set(level1.map((n) => refToString(n.target)));
    for (const neighbor of level1) {
      const key = refToString(neighbor.target);
      if (visited.has(key)) continue;
      visited.add(key);
      const level2 = await neighborsOf(store, neighbor.target, edges);
      for (const n2 of level2) {
        const targetKey = refToString(n2.target);
        if (targetKey === refToString(rid) || seen.has(targetKey)) continue;
        seen.add(targetKey);
        collected.push(n2);
      }
      if (collected.length > 200) break; // simple bound against pathological fan-out
    }
  }

  const idsToResolve = collected.map((n) => n.target);
  const summaries = await resolveSummaries(store, idsToResolve);

  const neighbors: CaseGraphNeighbor[] = collected.map((n) => {
    const key = refToString(n.target);
    return { edge: n.edge, direction: n.direction, id: key, summary: summarize(summaries.get(key), key) };
  });

  return { id: refToString(rid), depth, neighbors: normalize(neighbors) as CaseGraphNeighbor[] };
}

export interface FactorMapEntry {
  letter: string;
  title: string;
  support_count: number;
  contradict_count: number;
  top_supporting: Array<{ id: string; summary: string; weight: number | null }>;
  top_contradicting: Array<{ id: string; summary: string; weight: number | null }>;
}

async function topFor(store: StoreOk, edge: "supports_factor" | "contradicts_factor", factorRid: RecordIdLike): Promise<Array<{ id: string; summary: string; weight: number | null }>> {
  const rows = await store.db.query<Array<{ in: RecordIdLike; weight?: number }>>(
    `SELECT in, weight FROM ${ident(edge)} WHERE out = $rid ORDER BY weight DESC LIMIT 5;`,
    { rid: factorRid },
  );
  const list = rows.at(-1) ?? [];
  const summaries = await resolveSummaries(store, list.map((r) => r.in));
  return list.map((r) => {
    const key = refToString(r.in);
    return { id: key, summary: summarize(summaries.get(key), key), weight: typeof r.weight === "number" ? r.weight : null };
  });
}

export async function caseFactorMap(store: StoreOk): Promise<FactorMapEntry[]> {
  const RecordId = recordIdCtor();
  const entries: FactorMapEntry[] = [];
  for (const factor of FACTORS) {
    const rid = new RecordId("factor", factor.letter);
    const supportCountRows = await store.db.query<Array<{ count: number }>>(
      `SELECT count() AS count FROM ${ident("supports_factor")} WHERE out = $rid GROUP ALL;`,
      { rid },
    );
    const contradictCountRows = await store.db.query<Array<{ count: number }>>(
      `SELECT count() AS count FROM ${ident("contradicts_factor")} WHERE out = $rid GROUP ALL;`,
      { rid },
    );
    entries.push({
      letter: factor.letter,
      title: factor.title,
      support_count: supportCountRows.at(-1)?.[0]?.count ?? 0,
      contradict_count: contradictCountRows.at(-1)?.[0]?.count ?? 0,
      top_supporting: normalize(await topFor(store, "supports_factor", rid)) as FactorMapEntry["top_supporting"],
      top_contradicting: normalize(await topFor(store, "contradicts_factor", rid)) as FactorMapEntry["top_contradicting"],
    });
  }
  return entries;
}

// ---------------------------------------------------------------------------
// case_timeline
// ---------------------------------------------------------------------------

export interface CaseTimelineInput {
  /** "court" = the real court-event timeline (court_event ∪ hearing ∪ deadline ∪
   * order); "master" = the extracted-from-the-corpora timeline (event ∪ message ∪
   * exhibit); "merged" (default) = both, each entry tagged with its `lane`. Owner
   * order 2026-09-07 13:09-13:16: these two timelines are kept deliberately
   * separate — never collapse the master timeline into the court-event one. */
  mode?: "court" | "master" | "merged";
  /** Court lane only: true = only entries whose date is >= now; false = only
   * entries whose date is < now; omitted = no filter. */
  upcoming?: boolean;
  from?: string;
  to?: string;
  known_by?: string;
  tables?: Array<"event" | "message" | "exhibit">;
}

export interface CaseTimelineEntry {
  lane: "court" | "master";
  table: string;
  id: string;
  /** The entry's primary sort date (occurred_at for the master lane; the
   * table's own date field — court_event.date/hearing.date/deadline.due/
   * order.entered — for the court lane), always an ISO string. */
  date: string;
  occurred_at: string | null;
  known_at: string | null;
  summary: string;
  source: Record<string, unknown> | null;
}

const MASTER_LANE_TABLES: Array<"event" | "message" | "exhibit"> = ["event", "message", "exhibit"];

const COURT_LANE_TABLES: ReadonlyArray<{ table: DataTable; dateField: string; summaryField: string }> = [
  { table: "court_event", dateField: "date", summaryField: "title" },
  { table: "hearing", dateField: "date", summaryField: "title" },
  { table: "deadline", dateField: "due", summaryField: "label" },
  { table: "order", dateField: "entered", summaryField: "title" },
];

async function masterLaneEntries(store: StoreOk, input: CaseTimelineInput): Promise<CaseTimelineEntry[]> {
  const tables = (input.tables?.filter((t) => TWO_CLOCK_TABLES.has(t)) as Array<"event" | "message" | "exhibit"> | undefined) ?? MASTER_LANE_TABLES;
  const entries: CaseTimelineEntry[] = [];
  for (const table of tables) {
    const field = SEARCH_FIELD[table as SearchableTable];
    const conditions: string[] = [];
    const params: Record<string, unknown> = {};
    if (input.from) {
      conditions.push("occurred_at >= <datetime>$from");
      params.from = input.from;
    }
    if (input.to) {
      conditions.push("occurred_at <= <datetime>$to");
      params.to = input.to;
    }
    if (input.known_by) {
      conditions.push("known_at <= <datetime>$known_by");
      params.known_by = input.known_by;
    }
    const where = conditions.length ? ` WHERE ${conditions.join(" AND ")}` : "";
    const rows = await store.db.query<Array<Record<string, unknown>>>(
      `SELECT id, occurred_at, known_at, source, ${field} AS summary FROM ${ident(table)}${where} ORDER BY occurred_at ASC LIMIT ${ROW_CAP};`,
      params,
    );
    for (const row of rows.at(-1) ?? []) {
      const occurredAt = normalize(row.occurred_at) as string;
      entries.push({
        lane: "master",
        table,
        id: refToString(row.id as RecordIdLike),
        date: occurredAt,
        occurred_at: occurredAt,
        known_at: row.known_at ? (normalize(row.known_at) as string) : null,
        summary: snippetOf(row.summary),
        source: row.source ? (normalize(row.source) as Record<string, unknown>) : null,
      });
    }
  }
  return entries;
}

async function courtLaneEntries(store: StoreOk, input: CaseTimelineInput): Promise<CaseTimelineEntry[]> {
  const nowIso = new Date().toISOString();
  const entries: CaseTimelineEntry[] = [];
  for (const spec of COURT_LANE_TABLES) {
    const rows = await store.db.query<Array<Record<string, unknown>>>(
      `SELECT id, ${spec.dateField} AS d, ${spec.summaryField} AS summary, source FROM ${ident(spec.table)} LIMIT ${ROW_CAP};`,
    );
    for (const row of rows.at(-1) ?? []) {
      if (!row.d) continue;
      const dateVal = normalize(row.d) as string;
      if (input.from && dateVal < input.from) continue;
      if (input.to && dateVal > input.to) continue;
      if (input.upcoming === true && dateVal < nowIso) continue;
      if (input.upcoming === false && dateVal >= nowIso) continue;
      entries.push({
        lane: "court",
        table: spec.table,
        id: refToString(row.id as RecordIdLike),
        date: dateVal,
        occurred_at: null,
        known_at: null,
        summary: snippetOf(row.summary),
        source: row.source ? (normalize(row.source) as Record<string, unknown>) : null,
      });
    }
  }
  return entries;
}

export async function caseTimeline(store: StoreOk, input: CaseTimelineInput): Promise<CaseTimelineEntry[]> {
  const mode = input.mode ?? "merged";
  const entries: CaseTimelineEntry[] = [];
  if (mode === "master" || mode === "merged") entries.push(...(await masterLaneEntries(store, input)));
  if (mode === "court" || mode === "merged") entries.push(...(await courtLaneEntries(store, input)));
  entries.sort((a, b) => a.date.localeCompare(b.date));
  return entries.slice(0, ROW_CAP);
}

// ---------------------------------------------------------------------------
// New registers — owner orders 2026-09-07 13:09-13:16: case_status, case_docket,
// case_memo, case_evidence_log, case_eval, case_reference, case_source.
// ---------------------------------------------------------------------------

export async function caseStatusGet(store: StoreOk): Promise<Record<string, unknown>> {
  const rows = await store.db.query<Array<Record<string, unknown>>>("SELECT * FROM case_status:current;");
  return normalize(rows.at(-1)?.[0] ?? {}) as Record<string, unknown>;
}

export async function caseStatusSet(store: StoreOk, data: Record<string, unknown>): Promise<Record<string, unknown>> {
  const merged = { ...data, last_updated: new Date().toISOString() };
  const rows = await store.db.query<Array<Record<string, unknown>>>("UPSERT case_status:current MERGE $data RETURN AFTER;", { data: merged });
  return normalize(rows.at(-1)?.[0] ?? {}) as Record<string, unknown>;
}

export interface CaseDocketFilter {
  status?: string;
  doc_type?: string;
  in_force?: boolean;
}

export interface CaseDocketEntry {
  table: "filing" | "draft" | "order" | "court_event";
  id: string;
  title: string;
  date: string | null;
  status: string | null;
  record: Record<string, unknown>;
}

/** filings + drafts + orders + upcoming court_events, filterable by
 * status/doc_type (filing, draft)/in_force (order). court_events here are
 * always "upcoming" (date >= now) — see case_timeline({mode:"court"}) for
 * the full past+upcoming court-event history. */
export async function caseDocket(store: StoreOk, filter: CaseDocketFilter = {}): Promise<CaseDocketEntry[]> {
  const out: CaseDocketEntry[] = [];

  // doc_type is a filing/draft-only concept and in_force is order-only — a
  // caller who filters by one implicitly asks only for the table(s) where
  // that filter is meaningful, not "everything else, unfiltered".
  const wantFilingsAndDrafts = filter.in_force === undefined;
  const wantOrders = filter.doc_type === undefined;
  const wantCourtEvents = filter.doc_type === undefined && filter.in_force === undefined;

  if (wantFilingsAndDrafts) {
    const filingRows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("filing")} LIMIT ${ROW_CAP};`);
    for (const row of filingRows.at(-1) ?? []) {
      if (filter.status && row.status !== filter.status) continue;
      if (filter.doc_type && row.doc_type !== filter.doc_type) continue;
      const normalized = normalize(row) as Record<string, unknown>;
      out.push({ table: "filing", id: refToString(row.id as RecordIdLike), title: String(row.title ?? ""), date: row.date ? (normalize(row.date) as string) : null, status: (row.status as string) ?? null, record: normalized });
    }

    const draftRows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("draft")} LIMIT ${ROW_CAP};`);
    for (const row of draftRows.at(-1) ?? []) {
      if (filter.status && row.status !== filter.status) continue;
      if (filter.doc_type && row.doc_type !== filter.doc_type) continue;
      const normalized = normalize(row) as Record<string, unknown>;
      out.push({ table: "draft", id: refToString(row.id as RecordIdLike), title: String(row.title ?? ""), date: row.date ? (normalize(row.date) as string) : null, status: (row.status as string) ?? null, record: normalized });
    }
  }

  if (wantOrders) {
    const orderRows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("order")} LIMIT ${ROW_CAP};`);
    for (const row of orderRows.at(-1) ?? []) {
      if (filter.in_force !== undefined && Boolean(row.in_force) !== filter.in_force) continue;
      if (filter.status && row.status !== filter.status) continue;
      const normalized = normalize(row) as Record<string, unknown>;
      out.push({ table: "order", id: refToString(row.id as RecordIdLike), title: String(row.title ?? "Untitled order"), date: row.entered ? (normalize(row.entered) as string) : null, status: (row.status as string) ?? null, record: normalized });
    }
  }

  if (wantCourtEvents) {
    const nowIso = new Date().toISOString();
    const courtEventRows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("court_event")} LIMIT ${ROW_CAP};`);
    for (const row of courtEventRows.at(-1) ?? []) {
      const date = row.date ? (normalize(row.date) as string) : null;
      if (!date || date < nowIso) continue;
      if (filter.status && row.status !== filter.status) continue;
      const normalized = normalize(row) as Record<string, unknown>;
      out.push({ table: "court_event", id: refToString(row.id as RecordIdLike), title: String(row.title ?? ""), date, status: (row.status as string) ?? null, record: normalized });
    }
  }

  out.sort((a, b) => (a.date ?? "").localeCompare(b.date ?? ""));
  return out;
}

export interface CaseMemoPutInput {
  id?: string;
  kind: string;
  title: string;
  text: string;
  status?: string;
  factors?: string[];
  author?: string;
  supersedes?: string;
  source?: Record<string, unknown>;
}

export async function caseMemoPut(store: StoreOk, input: CaseMemoPutInput): Promise<CasePutResult> {
  const data: Record<string, unknown> = {
    kind: input.kind,
    title: input.title,
    text: input.text,
    status: input.status ?? "open",
    factors: input.factors ?? null,
    author: input.author ?? null,
    supersedes: input.supersedes ?? null,
    source: input.source ?? null,
    created_at: new Date().toISOString(),
  };
  return casePut(store, { table: "memo", id: input.id, data });
}

export async function caseMemoList(store: StoreOk, filter: { kind?: string; status?: string } = {}): Promise<Record<string, unknown>[]> {
  const conditions: string[] = [];
  const params: Record<string, unknown> = {};
  if (filter.kind) { conditions.push("kind = $kind"); params.kind = filter.kind; }
  if (filter.status) { conditions.push("status = $status"); params.status = filter.status; }
  const where = conditions.length ? ` WHERE ${conditions.join(" AND ")}` : "";
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("memo")}${where} ORDER BY created_at DESC LIMIT ${ROW_CAP};`, params);
  return normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
}

/** One row per kind — the newest (by created_at) memo of each kind seen. */
export async function caseMemoLatestByKind(store: StoreOk): Promise<Record<string, Record<string, unknown>>> {
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("memo")} ORDER BY created_at DESC LIMIT ${ROW_CAP};`);
  const all = normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
  const out: Record<string, Record<string, unknown>> = {};
  for (const row of all) {
    const kind = String(row.kind ?? "unknown");
    if (!(kind in out)) out[kind] = row;
  }
  return out;
}

export interface CaseEvidenceLogAppendInput {
  id?: string;
  action: string;
  exhibit?: RecordRef;
  by?: string;
  hash?: string;
  path?: string;
  notes?: string;
  source?: Record<string, unknown>;
}

/** "for when we get to that point" — an append-only chain-of-custody log,
 * independent of the H1/H2/H3 custody hashing machinery elsewhere in the
 * platform (this is the plugin's own lightweight record). Owner order
 * 2026-09-07 13:09-13:16. */
export async function caseEvidenceLogAppend(store: StoreOk, input: CaseEvidenceLogAppendInput): Promise<CasePutResult> {
  const id = input.id ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
  const data: Record<string, unknown> = {
    action: input.action,
    by: input.by ?? null,
    hash: input.hash ?? null,
    path: input.path ?? null,
    notes: input.notes ?? null,
    logged_at: new Date().toISOString(),
    source: input.source ?? null,
  };
  const relations: CasePutRelation[] = input.exhibit ? [{ edge: "logs", from: { table: "evidence_log", id }, to: input.exhibit }] : [];
  return casePut(store, { table: "evidence_log", id, data, relations });
}

export async function caseEvidenceLogList(store: StoreOk, filter: { exhibit?: RecordRef; action?: string } = {}): Promise<Record<string, unknown>[]> {
  if (filter.exhibit) {
    const rid = parseRef(filter.exhibit);
    const rows = await store.db.query<Array<{ in: RecordIdLike }>>(`SELECT in FROM ${ident("logs")} WHERE out = $rid;`, { rid });
    const logIds = (rows.at(-1) ?? []).map((r) => r.in);
    const summaries = await resolveSummaries(store, logIds);
    let list = [...summaries.values()];
    if (filter.action) list = list.filter((r) => r.action === filter.action);
    list.sort((a, b) => String(b.logged_at ?? "").localeCompare(String(a.logged_at ?? "")));
    return normalize(list) as Record<string, unknown>[];
  }
  const conditions: string[] = [];
  const params: Record<string, unknown> = {};
  if (filter.action) { conditions.push("action = $action"); params.action = filter.action; }
  const where = conditions.length ? ` WHERE ${conditions.join(" AND ")}` : "";
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("evidence_log")}${where} ORDER BY logged_at DESC LIMIT ${ROW_CAP};`, params);
  return normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
}

export interface CaseEvalPutInput {
  id?: string;
  kind: string;
  title: string;
  subject?: string;
  score?: number;
  verdict?: string;
  text?: string;
  path?: string;
  tool_or_model?: string;
  source?: Record<string, unknown>;
}

export async function caseEvalPut(store: StoreOk, input: CaseEvalPutInput): Promise<CasePutResult> {
  const data: Record<string, unknown> = {
    kind: input.kind,
    title: input.title,
    subject: input.subject ?? null,
    score: input.score ?? null,
    verdict: input.verdict ?? null,
    text: input.text ?? null,
    path: input.path ?? null,
    tool_or_model: input.tool_or_model ?? null,
    created_at: new Date().toISOString(),
    source: input.source ?? null,
  };
  return casePut(store, { table: "eval", id: input.id, data });
}

export async function caseEvalList(store: StoreOk, filter: { kind?: string; subject?: string } = {}): Promise<Record<string, unknown>[]> {
  const conditions: string[] = [];
  const params: Record<string, unknown> = {};
  if (filter.kind) { conditions.push("kind = $kind"); params.kind = filter.kind; }
  if (filter.subject) { conditions.push("subject = $subject"); params.subject = filter.subject; }
  const where = conditions.length ? ` WHERE ${conditions.join(" AND ")}` : "";
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("eval")}${where} ORDER BY created_at DESC LIMIT ${ROW_CAP};`, params);
  return normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
}

/** Parses a reference-data file: a JSON array, `{entries:[...]}`/`{rows:[...]}`,
 * a single object, or JSONL (one JSON object per line). Reference data is
 * "the ruler, not the subject" (platform convention carried into this plugin) —
 * loaded whole, never analyzed, never migrated. */
function loadReferenceFile(path: string): Record<string, unknown>[] {
  const raw = readFileSync(path, "utf8").trim();
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed)) return parsed as Record<string, unknown>[];
    if (parsed && typeof parsed === "object") {
      const obj = parsed as Record<string, unknown>;
      if (Array.isArray(obj.entries)) return obj.entries as Record<string, unknown>[];
      if (Array.isArray(obj.rows)) return obj.rows as Record<string, unknown>[];
      return [obj];
    }
    return [];
  } catch {
    // Not a single JSON document — try JSONL (one object per line).
    return raw.split(/\r?\n/).filter((line) => line.trim().length > 0).map((line) => JSON.parse(line) as Record<string, unknown>);
  }
}

export interface CaseReferenceLoadInput {
  path?: string;
  fromPlugin?: boolean;
  pluginRoot?: string;
}

/** Retain complete legacy reference imports as citation-required shared drafts.
 * Inputs: explicit file or packaged reference roots. Outputs: counts, origin paths and retained proposal envelopes.
 * Effects: draft writes only, never unvalidated reference publication. Choose for reference migration, not interactive case facts.
 */
export async function caseReferenceLoad(store: StoreOk, input: CaseReferenceLoadInput): Promise<{ counts: number; loaded_from: string[]; library_proposals: Array<Record<string, unknown>> }> {
  const sources: Array<{ label: string; rows: Record<string, unknown>[] }> = [];
  if (input.path) sources.push({ label: input.path, rows: loadReferenceFile(input.path) });

  if (input.fromPlugin) {
    const root = input.pluginRoot ?? process.env.CLAUDE_PLUGIN_ROOT;
    if (root) {
      const dir = join(root, "content", "reference");
      if (existsSync(dir)) {
        for (const entry of readdirSync(dir)) {
          if (entry.endsWith(".json") || entry.endsWith(".jsonl")) {
            sources.push({ label: join(dir, entry), rows: loadReferenceFile(join(dir, entry)) });
          }
        }
      }
      // content/tools/court-language/lexicon.json is an EXISTING pattern file
      // (the deterministic detection lexicon behind court_language_review) —
      // it fits the reference shape once its fields are mapped (category,
      // pattern, severity all match directly; its "why" explanation becomes
      // `definition`). Ingested here rather than copied under
      // content/reference/ so there is exactly one copy of it (owner rule:
      // no dual sources of truth) — case_reference_match then also sees it.
      const lexiconPath = join(root, "content", "tools", "court-language", "lexicon.json");
      if (existsSync(lexiconPath)) {
        const rawRows = loadReferenceFile(lexiconPath);
        const mapped = rawRows.map((row) => ({
          kind: "lexicon",
          key: typeof row.category === "string" ? `lexicon-${row.category}` : undefined,
          category: row.category ?? null,
          pattern: row.pattern ?? null,
          definition: row.why ?? row.definition ?? null,
          severity: row.severity ?? null,
          source: { path: lexiconPath },
        }));
        sources.push({ label: lexiconPath, rows: mapped });
      }
    }
  }

  const libraryProposals: Array<Record<string, unknown>> = [];
  let count = 0;
  const loadedFrom: string[] = [];
  for (const { label, rows } of sources) {
    for (const [i, row] of rows.entries()) {
      const key = typeof row.key === "string" ? row.key : typeof row.id === "string" ? row.id : `${row.kind ?? "ref"}-${i}`;
      libraryProposals.push(await retainLibraryImport(store, `reference:${String(key)}`, {
        ...row, kind: row.kind ?? null, key: row.key ?? key,
        source: row.source ?? { path: label },
      }));
      count += 1;
    }
    loadedFrom.push(label);
  }
  return { counts: count, loaded_from: loadedFrom, library_proposals: libraryProposals };
}

export async function caseReferenceList(store: StoreOk, filter: { kind?: string; category?: string } = {}): Promise<Record<string, unknown>[]> {
  const conditions: string[] = [];
  const params: Record<string, unknown> = {};
  if (filter.kind) { conditions.push("kind = $kind"); params.kind = filter.kind; }
  if (filter.category) { conditions.push("category = $category"); params.category = filter.category; }
  const where = conditions.length ? ` WHERE ${conditions.join(" AND ")}` : "";
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("reference")}${where} LIMIT ${ROW_CAP};`, params);
  return normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
}

export interface ReferenceMatchHit {
  id: string;
  kind: string | null;
  category: string | null;
  matched: string;
  span: [number, number];
  via: "pattern" | "alias";
}

/** Matches every loaded reference row's `pattern` (as a case-insensitive regex)
 * or `aliases[]` (literal, case-insensitive) against `text`, returning every
 * hit with its character span — used to "customize outputs" against the
 * behavior-detection/ontology reference data (owner order 2026-09-07). */
export async function caseReferenceMatch(store: StoreOk, text: string): Promise<ReferenceMatchHit[]> {
  const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident("reference")} LIMIT ${ROW_CAP};`);
  const refs = normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
  const hits: ReferenceMatchHit[] = [];
  for (const ref of refs) {
    const id = String(ref.id ?? "");
    const kind = typeof ref.kind === "string" ? ref.kind : null;
    const category = typeof ref.category === "string" ? ref.category : null;
    if (typeof ref.pattern === "string" && ref.pattern.length > 0) {
      try {
        const re = new RegExp(ref.pattern, "gi");
        let match: RegExpExecArray | null;
        while ((match = re.exec(text)) !== null) {
          hits.push({ id, kind, category, matched: match[0], span: [match.index, match.index + match[0].length], via: "pattern" });
          if (match.index === re.lastIndex) re.lastIndex += 1; // guard zero-width patterns
        }
      } catch {
        // An invalid regex in a reference row must never crash matching for
        // every other row — skip it silently.
      }
    }
    if (Array.isArray(ref.aliases)) {
      for (const alias of ref.aliases) {
        if (typeof alias !== "string" || alias.length === 0) continue;
        const idx = text.toLowerCase().indexOf(alias.toLowerCase());
        if (idx >= 0) hits.push({ id, kind, category, matched: text.slice(idx, idx + alias.length), span: [idx, idx + alias.length], via: "alias" });
      }
    }
  }
  return hits;
}

export interface CaseSourceResult {
  id: string;
  source: Record<string, unknown> | null;
  path_exists: boolean | null;
  r2_path: string | null;
}

/** Given a record ref, returns its `source` block, whether `source.path`
 * exists LOCALLY (no network calls — a Case Bible / R2 path never being
 * fetched over the wire here), and the r2 pointer if present. */
export async function caseSourceOf(store: StoreOk, ref: RecordRef): Promise<CaseSourceResult> {
  const rid = parseRef(ref);
  const rows = await store.db.query<Array<Record<string, unknown>>>("SELECT source FROM $rid;", { rid });
  const row = rows.at(-1)?.[0] as Record<string, unknown> | undefined;
  const source = row?.source ? (normalize(row.source) as Record<string, unknown>) : null;
  const path = source && typeof source.path === "string" ? source.path : null;
  const r2Path = source && typeof source.r2_path === "string" ? source.r2_path : null;
  return { id: refToString(rid), source, path_exists: path ? existsSync(path) : null, r2_path: r2Path };
}

// ---------------------------------------------------------------------------
// case_record — the shared legal-record contract (propria.legal-record.v1).
// Claude Code · Opus 5.5 · 2026-09-27.
//
// The Family Law Toolkit and Advocatio open the same record with the same id and
// version (SINGLE-WORKDESK-CONVERGENCE.md, "Family Law Toolkit companion surface").
// The store itself computes the version: sha256 over SurrealDB's own string form of
// the record, whose object keys are always sorted, so it is deterministic and any
// client language gets the same answer by running RECORD_VERSION_SURQL unchanged.
// Advocatio carries a byte-identical copy of this query in
// modules/Legal-desktop/api/legal_workspace/services/family_court_toolkit.py.
// ---------------------------------------------------------------------------

export const RECORD_CONTRACT = "propria.legal-record.v1";

export const RECORD_VERSION_SURQL =
  "LET $r = (SELECT * OMIT embedding FROM ONLY type::record($tb, $id)); " +
  "RETURN IF $r = NONE { NONE } ELSE { { tb: record::tb($r.id), id: <string> record::id($r.id), " +
  "version: 'sha256:' + crypto::sha256(<string> $r), record: $r } };";

export interface CaseRecordResult {
  contract: typeof RECORD_CONTRACT;
  id: string;
  table: string;
  version: string;
  record: Record<string, unknown>;
  original_links?: import("./library-file-links.js").LibraryOriginalLink[];
}

/** Read one full shared record with its database version and separate exact original-file links.
 * Inputs: connected store and exact record reference. Outputs: versioned envelope or null.
 * Effects: bounded record and file-binding reads; links never enter the record body or change its hash.
 * Choose for all shared human/MCP detail surfaces rather than client-specific source copies.
 * Byline: Codex · GPT-6 · 2026-10-05.
 */
export async function caseRecord(store: StoreOk, ref: RecordRef): Promise<CaseRecordResult | null> {
  const rid = parseRef(ref);
  const full = refToString(rid);
  const table = full.slice(0, full.indexOf(":"));
  const id = full.slice(full.indexOf(":") + 1);
  const rows = await store.db.query<unknown[]>(RECORD_VERSION_SURQL, { tb: table, id });
  const row = rows.at(-1) as { tb?: string; id?: string; version?: string; record?: unknown } | null | undefined;
  if (!row || !row.version) return null;
  const record = normalize(row.record) as Record<string, unknown>;
  delete record.id;
  const { libraryOriginalLinks } = await import("./library-file-links.js");
  const original_links = await libraryOriginalLinks(store, `${table}:${id}`);
  return { contract: RECORD_CONTRACT, id: `${table}:${id}`, table, version: row.version, record, original_links };
}

// ---------------------------------------------------------------------------
// Sidecar compatibility aliases — `app/` (a separate, concurrently-developed
// Tauri work surface, see app/README.md "Where the store lives") dynamically
// dispatches store.ts exports BY NAME via app/sidecar/lib/store-client.mjs's
// `callStoreFn`, which calls `caseMemo`, `caseStatus`, `caseSource`,
// `caseReference`, `caseEvidenceLog`, `caseEvals` (each single-arity, no
// separate get/put/list split) — verified live by reading
// app/sidecar/server.mjs's route handlers 2026-09-07. This file's own
// established convention is one verb-function per concern (casePut vs
// caseQuery vs caseSearch, ... predates this addition), so the primary API
// above keeps that shape (caseMemoPut/List/LatestByKind, caseStatusGet/Set,
// caseSourceOf, caseReferenceLoad/List/Match, caseEvidenceLogAppend/List,
// caseEvalPut/List) — these are thin READ-ONLY aliases so the app's
// `callStoreFn` name-dispatch stops returning `{ queued: true }` for these
// surfaces without this file adopting a second naming convention. Do not
// remove without checking app/sidecar/lib/store-client.mjs's QUEUED_TOOLS
// list first.
// ---------------------------------------------------------------------------

export async function caseMemo(store: StoreOk): Promise<{ memos: Record<string, unknown>[] }> {
  return { memos: await caseMemoList(store, {}) };
}

export async function caseStatus(store: StoreOk): Promise<Record<string, unknown>> {
  return caseStatusGet(store);
}

export async function caseSource(store: StoreOk, args: { id?: RecordRef } = {}): Promise<CaseSourceResult | { available: false; reason: string }> {
  if (!args.id) return { available: false, reason: 'case_source requires an { id: "table:id" } record ref' };
  return caseSourceOf(store, args.id);
}

export async function caseReference(store: StoreOk, args: { match?: string } = {}): Promise<{ entries: unknown[] }> {
  if (args.match) return { entries: await caseReferenceMatch(store, args.match) };
  return { entries: await caseReferenceList(store, {}) };
}

export async function caseEvidenceLog(store: StoreOk): Promise<{ entries: Record<string, unknown>[] }> {
  return { entries: await caseEvidenceLogList(store, {}) };
}

export async function caseEvals(store: StoreOk): Promise<{ evals: Record<string, unknown>[] }> {
  return { evals: await caseEvalList(store, {}) };
}

// ---------------------------------------------------------------------------
// case_export / case_import
// ---------------------------------------------------------------------------

export function defaultExportDir(): string {
  return join(homedir(), ".config", "family-court-toolkit", "exports");
}

export interface CaseSnapshot {
  exported_at: string;
  embed_dim: number;
  tables: Record<DataTable, Record<string, unknown>[]>;
  edges: Record<EdgeTable, Array<{ id: string; in: string; out: string; [key: string]: unknown }>>;
}

export async function caseExport(store: StoreOk, path?: string): Promise<{ path: string; counts: Record<string, number> }> {
  const tables = {} as CaseSnapshot["tables"];
  const counts: Record<string, number> = {};
  for (const table of DATA_TABLES) {
    const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident(table)};`);
    const normalized = normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
    tables[table] = normalized;
    counts[table] = normalized.length;
  }
  const edges = {} as CaseSnapshot["edges"];
  for (const edge of EDGE_TABLES) {
    const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident(edge)};`);
    const normalized = normalize(rows.at(-1) ?? []) as CaseSnapshot["edges"][EdgeTable];
    edges[edge] = normalized;
    counts[edge] = normalized.length;
  }

  const snapshot: CaseSnapshot = { exported_at: new Date().toISOString(), embed_dim: store.dim, tables, edges };
  const outPath = path ?? join(defaultExportDir(), `case-export-${snapshot.exported_at.replace(/[:.]/g, "-")}.json`);
  mkdirSync(dirname(outPath), { recursive: true });
  writeFileSync(outPath, JSON.stringify(snapshot, null, 2), "utf8");
  return { path: outPath, counts };
}

/**
 * Platform export mechanism (owner order 2026-09-07 13:09-13:16): writes one
 * NDJSON file per table plus `edges.ndjson` and a `manifest.json`, shaped for
 * consumption by the probata evidence platform — each row is a case-extract/v1-
 * shaped object (`id`, `type`, and the record's own fields, provenance `source`
 * included when present). Full personal context is retained (the store
 * preserves full names and narratives).
 */
export async function caseExportPlatform(store: StoreOk, dirOverride?: string): Promise<{ dir: string; counts: Record<string, number> }> {
  const ts = new Date().toISOString().replace(/[:.]/g, "-");
  const dir = dirOverride ?? join(defaultExportDir(), `platform-${ts}`);
  mkdirSync(dir, { recursive: true });

  const counts: Record<string, number> = {};
  for (const table of DATA_TABLES) {
    const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident(table)};`);
    const normalized = normalize(rows.at(-1) ?? []) as Record<string, unknown>[];
    const lines = normalized.map((row) => JSON.stringify({ id: row.id, type: table, ...row }));
    writeFileSync(join(dir, `${table}.ndjson`), lines.length ? `${lines.join("\n")}\n` : "", "utf8");
    counts[table] = normalized.length;
  }

  const edgeLines: string[] = [];
  for (const edge of EDGE_TABLES) {
    const rows = await store.db.query<Array<Record<string, unknown>>>(`SELECT * FROM ${ident(edge)};`);
    const normalized = normalize(rows.at(-1) ?? []) as Array<Record<string, unknown>>;
    for (const row of normalized) {
      edgeLines.push(JSON.stringify({ from: row.in, to: row.out, edge, ...row }));
    }
    counts[edge] = normalized.length;
  }
  writeFileSync(join(dir, "edges.ndjson"), edgeLines.length ? `${edgeLines.join("\n")}\n` : "", "utf8");

  const manifest = {
    schema: "fct-platform-bundle/v1",
    exported_at: new Date().toISOString(),
    counts,
    personal_context: "preserved",
    store_url_redacted: "***",
    plugin_version: PLUGIN_VERSION,
  };
  writeFileSync(join(dir, "manifest.json"), JSON.stringify(manifest, null, 2), "utf8");

  return { dir, counts };
}

function isSnapshotShape(json: unknown): json is CaseSnapshot {
  return Boolean(json) && typeof json === "object" && "tables" in (json as object) && "edges" in (json as object);
}

async function importSnapshot(store: StoreOk, snapshot: CaseSnapshot): Promise<{ counts: Record<string, number>; library_proposals: Array<Record<string, unknown>> }> {
  const libraryProposals: Array<Record<string, unknown>> = [];
  const RecordId = recordIdCtor();
  const counts: Record<string, number> = {};
  for (const table of DATA_TABLES) {
    const rows = snapshot.tables[table] ?? [];
    for (const row of rows) {
      const idStr = String(row.id ?? "");
      const idx = idStr.indexOf(":");
      if (idx < 1) continue;
      const { id: _drop, ...rest } = row;
      const rid = new RecordId(table, idStr.slice(idx + 1));
      const casted = castDateFields(table, rest as Record<string, unknown>);
      if (table === "reference" || (table === "source" && rest.kind !== "case_document"))
        libraryProposals.push(await retainLibraryImport(store, `${table}:${idStr.slice(idx + 1)}`, casted));
      else if (table === "source") await putPersonalCaseSource(store, idStr.slice(idx + 1), casted);
      else await store.db.query("UPSERT $rid MERGE $data;", { rid, data: casted });
    }
    counts[table] = rows.length;
  }
  for (const edge of EDGE_TABLES) {
    const rows = snapshot.edges[edge] ?? [];
    for (const row of rows) {
      const { in: inRef, out: outRef, id: _drop, ...rest } = row as { in: string; out: string; id?: string; [k: string]: unknown };
      const from = parseRef(inRef);
      const to = parseRef(outRef);
      const query = Object.keys(rest).length > 0 ? `RELATE $from->${ident(edge)}->$to CONTENT $data;` : `RELATE $from->${ident(edge)}->$to;`;
      await store.db.query(query, { from, to, data: rest });
    }
    counts[edge] = rows.length;
  }
  return { counts, library_proposals: libraryProposals };
}

export async function caseImport(store: StoreOk, path: string): Promise<{ counts: Record<string, number>; kind: "snapshot" | "vincent" | "case-extract"; library_proposals?: Array<Record<string, unknown>> }> {
  const stat = statSync(path);
  if (stat.isDirectory()) {
    const { counts, library_proposals } = await caseImportExtract(store, path);
    return { counts, kind: "case-extract", library_proposals };
  }
  const raw = JSON.parse(readFileSync(path, "utf8"));
  if (raw && typeof raw === "object" && (raw as Record<string, unknown>).schema === "case-extract/v1") {
    const { counts, library_proposals } = await caseImportExtract(store, path);
    return { counts, kind: "case-extract", library_proposals };
  }
  if (isSnapshotShape(raw)) {
    const { counts, library_proposals } = await importSnapshot(store, raw);
    return { counts, kind: "snapshot", library_proposals };
  }
  if (
    raw &&
    typeof raw === "object" &&
    ("parties" in raw || "timeline" in raw || "timeline_events" in raw || "evidence" in raw || "evidence_matrix" in raw)
  ) {
    const counts = await importVincentSchema(store, raw as Record<string, unknown>);
    return { counts, kind: "vincent" };
  }
  throw new Error(
    `Unrecognized case-import JSON shape at ${path}. Expected a case-extract/v1 envelope (file or directory), a ` +
      `case_export snapshot ({tables, edges}), or a Vincent-style case schema ({parties, timeline, evidence, legal_issues, ...}).`,
  );
}

// ---------------------------------------------------------------------------
// Vincent-style case schema import (shared with scripts/import-vincent-schema.mjs)
// ---------------------------------------------------------------------------

/**
 * Extracts the case-schema JSON from a Vincent-style intake file. Handles
 * three shapes, in order: (1) the file is already plain JSON; (2) a proper
 * fenced ```json code block; (3) a JSON object embedded mid-sentence with NO
 * fence at all — verified live against the owner's actual intake file
 * (~/.config/family-court-toolkit/intake/vincent_case_schema_2025-12-15.md),
 * which reads "...legal argumentation.json{"parties": [...` with no code
 * fence, so shape (3) is brace-balance scanned rather than regex-fenced.
 *
 * That same real file also contains genuinely invalid JSON: every
 * `<ungrounded-authority probable-jurisdiction="US-MI">` tag has UNESCAPED
 * inner double quotes around `US-MI`, breaking JSON string syntax. This is
 * repaired narrowly (that literal attribute pattern only) before parsing —
 * see `repairKnownVincentQuoteBug` — rather than pulling in a general JSON
 * repair library for one known, describable corruption.
 */
export function extractVincentJson(fileContents: string): Record<string, unknown> {
  const trimmed = fileContents.trim();
  if (trimmed.startsWith("{")) {
    return JSON.parse(repairKnownVincentQuoteBug(trimmed));
  }
  const fenced = /```json\s*([\s\S]*?)```/.exec(fileContents);
  if (fenced) return JSON.parse(repairKnownVincentQuoteBug(fenced[1]));

  const balanced = extractBalancedJsonObject(fileContents);
  return JSON.parse(repairKnownVincentQuoteBug(balanced));
}

/** Scans from the first `{` in `text` and returns the substring up to its
 * balanced matching `}`, respecting (possibly malformed) string literals. */
function extractBalancedJsonObject(text: string): string {
  const start = text.indexOf("{");
  if (start === -1) throw new Error("No JSON object found in the Vincent-style case schema file (no '{' at all).");
  let depth = 0;
  let inString = false;
  let escape = false;
  for (let i = start; i < text.length; i++) {
    const ch = text[i];
    if (inString) {
      if (escape) escape = false;
      else if (ch === "\\") escape = true;
      else if (ch === '"') inString = false;
      continue;
    }
    if (ch === '"') {
      inString = true;
      continue;
    }
    if (ch === "{") depth++;
    else if (ch === "}") {
      depth--;
      if (depth === 0) return text.slice(start, i + 1);
    }
  }
  throw new Error("Unbalanced '{'/'}' while scanning for the embedded JSON object in the Vincent-style case schema file.");
}

/** Fixes the one known corruption in the owner's real intake file: unescaped
 * inner quotes around the jurisdiction value in `<ungrounded-authority
 * probable-jurisdiction="US-MI">` tags, which otherwise breaks JSON string
 * syntax. Swaps those two inner double quotes for single quotes — legal,
 * unambiguous JSON characters — rather than escaping them, since escaping
 * would require knowing the exact enclosing-string boundaries. A no-op on
 * any file that doesn't contain this exact pattern. */
function repairKnownVincentQuoteBug(jsonText: string): string {
  return jsonText.replace(/(probable-jurisdiction=)"([A-Z-]+)"/g, "$1'$2'");
}

function toInitialsLocal(value: unknown): string | null {
  if (typeof value !== "string") return null;
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return null;
  return `${parts.map((part) => part[0]?.toUpperCase() ?? "").join(".")}.`;
}

/** The real intake file's timeline dates are sometimes month-only
 * ("2024-05") rather than full dates — `new Date()` accepts that ISO partial
 * form (UTC midnight on the 1st), so this just validates + normalizes to a
 * full ISO string casePut can cast to a real datetime. */
function normalizeDateOnly(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error(`Invalid date "${value}" in Vincent-style case schema.`);
  return date.toISOString();
}

/** Extracts every `MCL 722.23(<letter>)`-style reference from a list of
 * statute/authority strings and returns the unique factor letters found. */
function factorLettersFrom(values: unknown): string[] {
  if (!Array.isArray(values)) return [];
  const letters = new Set<string>();
  for (const value of values) {
    if (typeof value !== "string") continue;
    for (const match of value.matchAll(/722\.23\(([a-lA-L])\)/g)) {
      letters.add(match[1].toLowerCase());
    }
  }
  return [...letters];
}

async function putNoteList(store: StoreOk, id: string, title: string, items: unknown, redact: (text: string) => string): Promise<boolean> {
  if (!Array.isArray(items) || items.length === 0) return false;
  const body = items.map((item) => `- ${redact(typeof item === "string" ? item : JSON.stringify(item))}`).join("\n");
  await casePut(store, { table: "note", id, data: { text: redact(`${title}\n\n${body}`) } });
  return true;
}

/**
 * Maps the owner's Vincent-style case schema onto the store model. Verified
 * live against the real intake file's actual top-level keys: `parties`
 * ({name, role}), `witnesses` ({name, role, relevance, evidence_refs}),
 * `timeline` ({date, event, evidence_refs, legal_issues_refs} — NOT
 * `timeline_events`), `evidence` ({id, type, description, source,
 * risk_level, relevance, timeline_refs, legal_issues_refs} — NOT
 * `evidence_matrix`), `legal_issues` ({id, issue, statutes, facts_refs,
 * arguments}), `user_goals` (string[]), `risk_matrix` ({evidence_id, risk}),
 * `documents` (string[] of filenames — NOT objects), `legal_authorities`
 * (string[]), `questions_for_further_development` (string[]).
 *
 * Full personal party fields and narrative names are retained in this private case store.
 * Inputs: parsed intake schema. Outputs: per-table counts. Effects: shared case writes.
 * Choose for personal intake, not published legal-source library updates.
 */
export async function importVincentSchema(store: StoreOk, schema: Record<string, unknown>): Promise<Record<string, number>> {
  const counts: Record<string, number> = { person: 0, child: 0, event: 0, exhibit: 0, source: 0, note: 0, evidences: 0, supports_factor: 0 };
  const nowIso = new Date().toISOString();

  // --- parties -> person / child -------------------------------------
  const parties = Array.isArray(schema.parties) ? schema.parties : [];
  for (const [i, party] of parties.entries()) {
    if (!party || typeof party !== "object") continue;
    const { id: importedId, ...p } = party as Record<string, unknown>;
    if (importedId !== undefined) p.source_record_id = importedId;
    const role = typeof p.role === "string" ? p.role : undefined;
    if (role && /child/i.test(role)) {
      const initials = typeof p.initials === "string" ? p.initials : (toInitialsLocal(p.name) ?? `CHILD-${i + 1}`);
      const age = typeof p.age === "number" ? p.age : undefined;
      await casePut(store, { table: "child", id: `vincent-party-${i + 1}`, data: { ...p, initials, ...(age === undefined ? {} : { age }) } });
      counts.child += 1;
      continue;
    }
    const initials = typeof p.initials === "string" ? p.initials : toInitialsLocal(p.name);
    await casePut(store, { table: "person", id: `vincent-party-${i + 1}`, data: { ...p, role: role ?? null, initials: initials ?? null } });
    counts.person += 1;
  }

  // Private case text is deliberately preserved verbatim, including names.
  const redactChildNames = (text: string): string => text;

  // --- timeline -> event, indexed as "timeline_N" (1-based) to match ---
  // --- the source file's own facts_refs convention, and by date (its  ---
  // --- own timeline_refs convention, e.g. evidence[].timeline_refs:    ---
  // --- ["2024-05"]) -----------------------------------------------------
  const timeline = Array.isArray(schema.timeline) ? schema.timeline : Array.isArray(schema.timeline_events) ? schema.timeline_events : [];
  const eventIdByRef = new Map<string, string>(); // "timeline_N" AND raw date string -> event id
  for (const [i, item] of timeline.entries()) {
    if (!item || typeof item !== "object") continue;
    const e = item as Record<string, unknown>;
    const rawDate = typeof e.date === "string" ? e.date : typeof e.occurred_at === "string" ? e.occurred_at : null;
    if (!rawDate) continue;
    const occurredAt = normalizeDateOnly(rawDate);
    const description = redactChildNames(typeof e.event === "string" ? e.event : typeof e.description === "string" ? e.description : "Untitled event");
    const knownAt = typeof e.known_at === "string" ? normalizeDateOnly(e.known_at) : occurredAt;
    const id = `vincent-timeline-${i + 1}`;
    await casePut(store, { table: "event", id, data: { occurred_at: occurredAt, known_at: knownAt, description } });
    counts.event += 1;
    eventIdByRef.set(`timeline_${i + 1}`, id);
    eventIdByRef.set(rawDate, id);

    for (const letter of factorLettersFrom(e.legal_issues_refs)) {
      await casePut(store, { table: "event", id, data: {}, relations: [{ edge: "supports_factor", from: { table: "event", id }, to: { table: "factor", id: letter } }] });
      counts.supports_factor += 1;
    }
  }

  // --- evidence -> exhibit, RELATEd to the timeline event(s) it backs --
  const evidence = Array.isArray(schema.evidence) ? schema.evidence : Array.isArray(schema.evidence_matrix) ? schema.evidence_matrix : [];
  const exhibitIdByEvidenceRef = new Map<string, string>();
  for (const [i, item] of evidence.entries()) {
    if (!item || typeof item !== "object") continue;
    const ex = item as Record<string, unknown>;
    const sourceRef = typeof ex.id === "string" ? ex.id : `evidence_${i + 1}`;
    const id = `vincent-${sourceRef}`;
    const timelineRefs = Array.isArray(ex.timeline_refs) ? ex.timeline_refs.filter((r): r is string => typeof r === "string") : [];
    const firstDate = timelineRefs.find((r) => /^\d{4}/.test(r));
    const occurredAt = firstDate ? normalizeDateOnly(firstDate) : nowIso;
    const label = redactChildNames(typeof ex.description === "string" ? ex.description : typeof ex.label === "string" ? ex.label : `Exhibit ${i + 1}`);
    await casePut(store, {
      table: "exhibit",
      id,
      data: {
        label,
        occurred_at: occurredAt,
        known_at: nowIso,
        exhibit_type: typeof ex.type === "string" ? ex.type : null,
        source_ref: typeof ex.source === "string" ? ex.source : null,
        risk_level: typeof ex.risk_level === "string" ? ex.risk_level : null,
      },
    });
    counts.exhibit += 1;
    exhibitIdByEvidenceRef.set(sourceRef, id);

    for (const ref of timelineRefs) {
      const eventId = eventIdByRef.get(ref);
      if (!eventId) continue;
      await casePut(store, { table: "exhibit", id, data: {}, relations: [{ edge: "evidences", from: { table: "exhibit", id }, to: { table: "event", id: eventId } }] });
      counts.evidences += 1;
    }

    for (const letter of factorLettersFrom(ex.legal_issues_refs)) {
      await casePut(store, { table: "exhibit", id, data: {}, relations: [{ edge: "supports_factor", from: { table: "exhibit", id }, to: { table: "factor", id: letter } }] });
      counts.supports_factor += 1;
    }
  }

  // --- legal_issues -> supports_factor edges from their facts_refs -----
  // --- (facts_refs mixes evidence_N and timeline_N ids in the real file) -
  const legalIssues = Array.isArray(schema.legal_issues) ? schema.legal_issues : [];
  for (const issue of legalIssues) {
    if (!issue || typeof issue !== "object") continue;
    const li = issue as Record<string, unknown>;
    const letters = factorLettersFrom(li.statutes);
    const factsRefs = Array.isArray(li.facts_refs) ? li.facts_refs.filter((r): r is string => typeof r === "string") : [];
    for (const ref of factsRefs) {
      const targetId = exhibitIdByEvidenceRef.get(ref) ?? eventIdByRef.get(ref);
      if (!targetId) continue;
      const targetTable: DataTable = exhibitIdByEvidenceRef.has(ref) ? "exhibit" : "event";
      for (const letter of letters) {
        await casePut(store, {
          table: targetTable,
          id: targetId,
          data: {},
          relations: [{ edge: "supports_factor", from: { table: targetTable, id: targetId }, to: { table: "factor", id: letter } }],
        });
        counts.supports_factor += 1;
      }
    }
  }

  // --- documents (plain filename strings) -> source --------------------
  const documents = Array.isArray(schema.documents) ? schema.documents : Array.isArray(schema.document_refs) ? schema.document_refs : [];
  for (const [i, doc] of documents.entries()) {
    const rawTitle = typeof doc === "string" ? doc : typeof doc === "object" && doc ? ((doc as Record<string, unknown>).title ?? (doc as Record<string, unknown>).name) : `Document ${i + 1}`;
    const title = redactChildNames(String(rawTitle ?? `Document ${i + 1}`));
    await casePut(store, { table: "source", id: `vincent-doc-${i + 1}`, data: { kind: "case_document", title } });
    counts.source += 1;
  }

  // --- everything else (witnesses, goals, risk matrix, authorities, ----
  // --- open questions) -> note records, so nothing is silently dropped -
  if (await putNoteList(store, "vincent-witnesses", "Witnesses", schema.witnesses, redactChildNames)) counts.note += 1;
  if (await putNoteList(store, "vincent-user-goals", "User goals", schema.user_goals, redactChildNames)) counts.note += 1;
  if (await putNoteList(store, "vincent-risk-matrix", "Risk matrix", schema.risk_matrix, redactChildNames)) counts.note += 1;
  if (await putNoteList(store, "vincent-legal-authorities", "Legal authorities", schema.legal_authorities, redactChildNames)) counts.note += 1;
  if (await putNoteList(store, "vincent-open-questions", "Questions for further development", schema.questions_for_further_development, redactChildNames)) counts.note += 1;

  return counts;
}

// ---------------------------------------------------------------------------
// case-extract/v1 importer (Case Bible intake -> case store), owner order
// 2026-09-07 13:09-13:16. Full record shape at
// E:\AI_Workspace\casebible\_intake\extracted-json-20260907\_SCHEMA-case-extract-v1.md
// (read-only reference; this store never writes there).
// ---------------------------------------------------------------------------

interface CaseExtractEnvelope {
  schema?: string;
  source?: { row?: string; path?: string; sha256?: string; kind?: string; authored_by?: string; source_date?: string };
  extracted_at?: string;
  extractor?: string;
  confidence?: string;
  records?: Array<Record<string, unknown>>;
}

/** Collects every `.json` file to import: `fileOrDir` itself if it's a file,
 * or every `.json` file under a `<row>/*.json` directory tree if it's a
 * directory — skipping `_INDEX.json` and any `_SCHEMA*` file (sidecar /
 * documentation files, never record files). */
function collectExtractFiles(fileOrDir: string): string[] {
  const stat = statSync(fileOrDir);
  if (stat.isFile()) return [fileOrDir];
  const out: string[] = [];
  const walk = (dir: string): void => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      if (!entry.name.endsWith(".json")) continue;
      if (entry.name === "_INDEX.json" || entry.name.startsWith("_SCHEMA")) continue;
      out.push(full);
    }
  };
  walk(fileOrDir);
  return out;
}

function buildChildRedactor(childNames: Array<{ full: string; initials: string }>): (text: string) => string {
  const escape = (s: string): string => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (text: string): string => {
    let out = text;
    for (const { full, initials } of childNames) {
      out = out.replace(new RegExp(`\\b${escape(full)}\\b`, "gi"), initials);
      const firstName = full.split(/\s+/)[0];
      if (firstName && firstName.length > 1) out = out.replace(new RegExp(`\\b${escape(firstName)}\\b`, "g"), initials);
    }
    return out;
  };
}

/** `event.occurred_at` may legitimately be the literal string `"unknown"`
 * (see the schema doc); this normalizes everything else via
 * `normalizeDateOnly` and passes `"unknown"`/missing straight through as
 * `null` so a real event without a firm date still imports. */
function normalizeExtractDate(value: unknown): string | null {
  if (typeof value !== "string" || value.length === 0 || value === "unknown") return null;
  return normalizeDateOnly(value);
}

export interface CaseImportExtractResult {
  counts: Record<string, number>;
  files_processed: number;
  files_skipped: string[];
  library_proposals: Array<Record<string, unknown>>;
}

/**
 * Imports one case-extract/v1 file, or every case-extract/v1 file under a
 * directory tree (one JSON file per source, `<row>/<slug>.json`). Idempotent
 * by `extract_id` (`<row>-<seq>`, used directly as the record id — a re-run
 * MERGEs onto the same records rather than duplicating them). Full names, aliases and
 * personal narratives remain unchanged in this private shared case store.
 * Inputs: local case-extract envelopes. Outputs: counts and skipped files.
 * Effects: case records are imported with provenance; legal-library publication uses the tracked validation path.
 * Choose for personal case intake, not an unvalidated legal-library overwrite.
 */
export async function caseImportExtract(store: StoreOk, fileOrDir: string): Promise<CaseImportExtractResult> {
  const files = collectExtractFiles(fileOrDir);
  const filesSkipped: string[] = [];
  const envelopes: Array<{ path: string; envelope: CaseExtractEnvelope }> = [];
  for (const file of files) {
    let parsed: CaseExtractEnvelope;
    try {
      parsed = JSON.parse(readFileSync(file, "utf8")) as CaseExtractEnvelope;
    } catch {
      filesSkipped.push(file);
      continue;
    }
    if (parsed.schema !== "case-extract/v1") {
      filesSkipped.push(file);
      continue;
    }
    envelopes.push({ path: file, envelope: parsed });
  }

  // Owner decision: preserve full personal context across private shared imports.
  const redact = (text: string): string => text;

  const libraryProposals: Array<Record<string, unknown>> = [];
  const counts: Record<string, number> = {};
  const bump = (key: string): void => {
    counts[key] = (counts[key] ?? 0) + 1;
  };

  for (const { envelope } of envelopes) {
    const sourceMeta = envelope.source ?? {};
    let seq = 0;
    for (const record of envelope.records ?? []) {
      seq += 1;
      const extractId = typeof record.id === "string" && record.id ? record.id : `${sourceMeta.row ?? "extract"}-${seq}`;
      const { id: _originalIdentity, ...fullRecord } = record;
      const provenance = {
        source: {
          path: sourceMeta.path ?? null,
          sha256: sourceMeta.sha256 ?? null,
          row: sourceMeta.row ?? null,
          locator: record.source_locator ?? null,
        },
        extract_id: extractId,
        imported_at: new Date().toISOString(),
      };

      switch (record.type) {
        case "person": {
          if (/child/i.test(String(record.role ?? ""))) {
            const initials = typeof record.initials === "string" && record.initials ? record.initials : toInitialsLocal(typeof record.name === "string" ? record.name : "") ?? "?.";
            // child.age is DDL `option<number>` — SurrealDB's option<T> accepts
            // NONE (the field simply absent) but rejects an explicit NULL, so
            // the key must be omitted entirely rather than set to null when
            // no age is known (verified live: "Expected none | number but
            // found NULL" otherwise).
            const data: Record<string, unknown> = { ...fullRecord, initials, ...provenance };
            if (typeof record.age === "number") data.age = record.age;
            await casePut(store, { table: "child", id: extractId, data });
            bump("child");
          } else {
            const name = typeof record.name === "string" ? record.name : "";
            const initials = typeof record.initials === "string" ? record.initials : toInitialsLocal(name);
            await casePut(store, {
              table: "person",
              id: extractId,
              data: {
                ...fullRecord,
                name: redact(name),
                role: record.role ?? null,
                initials: initials ?? null,
                aliases: record.aliases ?? null,
                relationship: record.relationship ?? null,
                notes: typeof record.notes === "string" ? redact(record.notes) : null,
                ...provenance,
              },
            });
            bump("person");
          }
          break;
        }
        case "event": {
          const occurredAt = normalizeExtractDate(record.occurred_at);
          const data: Record<string, unknown> = {
            ...fullRecord,
            description: redact(String(record.description ?? "")),
            location: record.location ?? null,
            participants: record.participants ?? null,
            evidence_refs: record.evidence_refs ?? null,
            factors: record.factors ?? null,
            tags: record.tags ?? null,
            occurred_text: typeof record.occurred_at === "string" ? record.occurred_at : null,
            ...provenance,
          };
          if (occurredAt) {
            data.occurred_at = occurredAt;
            data.known_at = normalizeExtractDate(record.known_at) ?? occurredAt;
          }
          await casePut(store, { table: "event", id: extractId, data });
          bump("event");
          break;
        }
        case "court_event": {
          const date = normalizeExtractDate(record.date);
          await casePut(store, {
            table: "court_event",
            id: extractId,
            data: {
                ...fullRecord,
              kind: record.kind ?? null,
              ...(date ? { date } : {}),
              title: redact(String(record.title ?? "")),
              court: record.court ?? null,
              judge_or_referee: record.judge_or_referee ?? null,
              case_no: record.case_no ?? null,
              outcome: record.outcome ?? null,
              document_refs: record.document_refs ?? null,
              ...(typeof record.status === "string" ? { status: record.status } : {}),
              ...provenance,
            },
          });
          bump("court_event");
          break;
        }
        case "filing": {
          const date = normalizeExtractDate(record.date);
          await casePut(store, {
            table: "filing",
            id: extractId,
            data: {
                ...fullRecord,
              title: redact(String(record.title ?? "")),
              doc_type: record.doc_type ?? null,
              filed_or_planned: record.filed_or_planned ?? null,
              ...(date ? { date } : {}),
              court: record.court ?? null,
              served_on: record.served_on ?? null,
              service_method: record.service_method ?? null,
              status: record.status ?? null,
              document_refs: record.document_refs ?? null,
              ...provenance,
            },
          });
          bump("filing");
          break;
        }
        case "draft": {
          const date = normalizeExtractDate(record.date);
          await casePut(store, {
            table: "draft",
            id: extractId,
            data: {
                ...fullRecord,
              title: redact(String(record.title ?? "")),
              doc_type: record.doc_type ?? null,
              version: record.version ?? null,
              status: record.status ?? null,
              path: record.path ?? null,
              notes: typeof record.notes === "string" ? redact(record.notes) : null,
              ...(date ? { date } : {}),
              ...provenance,
            },
          });
          bump("draft");
          break;
        }
        case "message": {
          const sentAt = normalizeExtractDate(record.sent_at);
          await casePut(store, {
            table: "message",
            id: extractId,
            data: {
                ...fullRecord,
              from: record.from ?? null,
              to: record.to ?? null,
              body: redact(String(record.body ?? "")),
              platform: record.platform ?? null,
              thread_id: record.thread_id ?? null,
              attachments: record.attachments ?? null,
              tags: record.tags ?? null,
              factors: record.factors ?? null,
              ...(sentAt ? { occurred_at: sentAt, known_at: sentAt } : {}),
              ...provenance,
            },
          });
          bump("message");
          break;
        }
        case "exhibit": {
          await casePut(store, {
            table: "exhibit",
            id: extractId,
            data: {
                ...fullRecord,
              label: redact(String(record.label_or_name ?? "")),
              path: record.path ?? null,
              sha256: record.sha256 ?? null,
              md5: record.md5 ?? null,
              r2_path: record.r2_path ?? null,
              mime: record.mime ?? null,
              size: record.size ?? null,
              description: typeof record.description === "string" ? redact(record.description) : null,
              tier: record.tier ?? null,
              related_events: record.related_events ?? null,
              known_at: new Date().toISOString(),
              ...provenance,
            },
          });
          bump("exhibit");
          break;
        }
        case "source_authority": {
          libraryProposals.push(await retainLibraryImport(store, `source:${extractId}`, { ...record, ...provenance }));
          bump("source");
          break;
        }
        case "issue":
        case "risk":
        case "goal": {
          const text = redact(
            String(
              record.summary ??
                record.mitigation ??
                record.notes ??
                record.title ??
                "",
            ),
          );
          await casePut(store, {
            table: "memo",
            id: extractId,
            data: {
                ...fullRecord,
              kind: record.type,
              title: redact(String(record.title ?? record.type)),
              text,
              status: record.status ?? "open",
              factors: record.factors ?? null,
              created_at: new Date().toISOString(),
              ...provenance,
            },
          });
          bump("memo");
          break;
        }
        case "note": {
          const kind = typeof record.kind === "string" ? record.kind : null;
          if (kind && ["strategy", "advice", "finding", "pattern"].includes(kind)) {
            await casePut(store, {
              table: "memo",
              id: extractId,
              data: {
                ...fullRecord,
                kind,
                title: `Note ${extractId}`,
                text: redact(String(record.text ?? "")),
                status: "open",
                created_at: new Date().toISOString(),
                ...provenance,
              },
            });
            bump("memo");
          } else {
            await casePut(store, { table: "note", id: extractId, data: { text: redact(String(record.text ?? "")), ...provenance } });
            bump("note");
          }
          break;
        }
        case "entity_rule": {
          libraryProposals.push(await retainLibraryImport(store, `reference:${extractId}`, { ...record, kind: "behavior_pattern", key: extractId, ...provenance }));
          bump("reference");
          break;
        }
        default:
          // Unknown/future record type — skip rather than throw, so one
          // unrecognized record never aborts an entire batch import.
          break;
      }
    }
  }

  return { counts, files_processed: envelopes.length, files_skipped: filesSkipped, library_proposals: libraryProposals };
}

// ---------------------------------------------------------------------------
// case_summary — same shape as core.ts's getCaseFacts(), computed from the store
// ---------------------------------------------------------------------------

export interface CaseSummary {
  configured: true;
  county: string | null;
  court: string | null;
  judge: string | null;
  referee: string | null;
  controlling_orders: Array<{ title: string; entered: string; served: string | null }>;
  next_hearing: string | null;
  deadlines: Array<{ label: string; due: string; rule: string | null }>;
  parties: string[];
  children: { count: number; entries: Array<Record<string, unknown> & { initials: string | null; age: number | string | null }> };
  case_context: { court: Record<string, unknown>; people: Array<Record<string, unknown>>; children: Array<Record<string, unknown>> };
  flags: string[];
  source: string;
  /** Row count per table (owner order 2026-09-07 13:09-13:16: a place to see
   * filed works, drafts, memos, reference rows, evidence-log entries, and
   * evals at a glance without a separate query per register). */
  counts: Record<string, number>;
}

/** Read the private shared case summary and retain complete personal context.
 * Inputs: admitted shared store. Outputs: planning fields and full court/person/child records.
 * Effects: read-only shared queries. Choose over public projections when preparing this owner's case.
 * Byline: Codex · GPT-6 · 2026-10-04.
 */
export async function caseSummary(store: StoreOk): Promise<CaseSummary> {
  const courtRows = await store.db.query<Array<Record<string, unknown>>>("SELECT * FROM court:main;");
  const court = (courtRows.at(-1)?.[0] ?? {}) as Record<string, unknown>;

  const orderRows = await store.db.query<Array<Record<string, unknown>>>(`SELECT title, entered, served FROM ${ident("order")} ORDER BY entered ASC;`);
  const controlling_orders = (orderRows.at(-1) ?? []).map((row) => ({
    title: String(row.title ?? "Untitled order"),
    entered: String(row.entered ?? "unknown"),
    served: row.served ? String(row.served) : null,
  }));

  const now = new Date().toISOString();
  const hearingRows = await store.db.query<Array<{ date?: string }>>(
    "SELECT date FROM hearing WHERE date >= <datetime>$now ORDER BY date ASC LIMIT 1;",
    { now },
  );
  let next_hearing = hearingRows.at(-1)?.[0]?.date ? (normalize(hearingRows.at(-1)![0].date) as string) : null;
  if (!next_hearing) {
    const anyHearing = await store.db.query<Array<{ date?: string }>>("SELECT date FROM hearing ORDER BY date DESC LIMIT 1;");
    next_hearing = anyHearing.at(-1)?.[0]?.date ? (normalize(anyHearing.at(-1)![0].date) as string) : null;
  }

  const deadlineRows = await store.db.query<Array<Record<string, unknown>>>("SELECT label, due, rule FROM deadline ORDER BY due ASC;");
  const deadlines = (deadlineRows.at(-1) ?? []).map((row) => ({
    label: String(row.label ?? "Unlabeled deadline"),
    due: String(row.due ?? "unknown"),
    rule: row.rule ? String(row.rule) : null,
  }));

  const personRows = await store.db.query<Array<Record<string, unknown>>>("SELECT * OMIT embedding FROM person;");
  const parties = (personRows.at(-1) ?? []).map((row) => {
    if (typeof row.name === "string" && row.name) return row.name;
    if (typeof row.initials === "string" && row.initials) return row.initials;
    return "?.";
  });

  const childRows = await store.db.query<Array<Record<string, unknown>>>("SELECT * OMIT embedding FROM child;");
  const childEntries = (childRows.at(-1) ?? []).map((row) => ({
    ...(normalize(row) as Record<string, unknown>),
    initials: typeof row.initials === "string" ? row.initials : null,
    age: typeof row.age === "number" || typeof row.age === "string" ? (row.age as number | string) : null,
  }));

  const counts: Record<string, number> = {};
  for (const table of DATA_TABLES) {
    const countRows = await store.db.query<Array<{ count: number }>>(`SELECT count() AS count FROM ${ident(table)} GROUP ALL;`);
    counts[table] = countRows.at(-1)?.[0]?.count ?? 0;
  }

  return {
    configured: true,
    county: typeof court.county === "string" ? court.county : null,
    court: typeof court.name === "string" ? court.name : typeof court.title === "string" ? court.title : null,
    judge: typeof court.judge === "string" ? court.judge : null,
    referee: typeof court.referee === "string" ? court.referee : null,
    controlling_orders,
    next_hearing,
    deadlines,
    parties,
    children: { count: childEntries.length, entries: childEntries },
    case_context: { court: normalize(court) as Record<string, unknown>, people: normalize(personRows.at(-1) ?? []) as Array<Record<string, unknown>>, children: normalize(childRows.at(-1) ?? []) as Array<Record<string, unknown>> },
    flags: Array.isArray(court.flags) ? (court.flags as string[]) : [],
    source: "surrealdb-case-store",
    counts,
  };
}
