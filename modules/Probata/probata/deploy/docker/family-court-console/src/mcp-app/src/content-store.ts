// Byline: Codex · GPT-6 · 2026-10-04
// Fresh, read-only shared content. Only an explicitly configured mem:// fixture
// may signal an absent row with null for its caller's packaged-file fallback.
import { getStore, normalize, parseRef, type StoreOk } from "./store.js";

export interface ReferenceRow {
  kind?: string;
  key?: string;
  body?: string;
  data?: unknown;
  source_path?: string;
  sha256?: string;
  loaded_at?: string;
  [extra: string]: unknown;
}

const SOURCE_PAGE_SIZE = 200;
const SOURCE_ROW_BUDGET = 10000;

// Byline: Codex · GPT-6 · 2026-10-04 — bounded shared pinpoint excerpts.
const EXCERPT_CHAR_LIMIT = 4096;
const HEADING_CHAR_LIMIT = 512;
const EXCERPT_BODY_CHAR_BUDGET = 16 * 1024 * 1024;

export interface ReferenceExcerpt {
  source_path: string;
  reference_id: string | null;
  source_sha256: string | null;
  /** Content-addressed version: identical to source_sha256, not a revision counter. */
  record_version: string | null;
  requested_pinpoint: string;
  resolution_status: "resolved" | "source_gap" | "pinpoint_gap" | "source_changed";
  gap_reason: string | null;
  resolved_heading: string | null;
  excerpt: string;
  excerpt_truncated: boolean;
}

/** Map an exact content source path using the existing reference-loader key contract.
 * Inputs: relative plugin path. Outputs: loader key, or null for paths outside supported content.
 * Effects: none; rejects traversal, separators and overlong input. Pick over filename guessing or skill auto-import.
 */
function excerptReferenceKey(path: string): string | null {
  if (path.length > 512 || !path.startsWith("content/") || path.includes("\\") || /[\x00-\x1f]/.test(path)
    || path.split("/").some(part => !part || part === "." || part === "..")) return null;
  return path.slice(8).toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 140);
}

/** Resolve one current shared source to a bounded, explicitly located text excerpt.
 * Inputs: exact content path and requested pinpoint (at most 256 characters).
 * Outputs: provenance plus up to 4096 Unicode characters; named empty gaps for missing sources/pinpoints or changed records.
 * Effects: at most two exact-record SDK SELECTs, no cache/content writes/file fallback. Read errors and malformed metadata throw.
 * Pick over getReference for large text: the server projects bounded metadata/text, never the full body.
 * A version is the retained source SHA, not proof of revalidation; only whole-file requests or unique Markdown headings resolve.
 */
export async function getReferenceExcerpt(sourcePath: string, pinpoint: string): Promise<ReferenceExcerpt> {
  if (typeof sourcePath !== "string" || sourcePath.length > 512 || typeof pinpoint !== "string" || !pinpoint.trim() || pinpoint.length > 256) {
    throw new Error("Invalid shared excerpt request");
  }
  const key = excerptReferenceKey(sourcePath);
  const result: ReferenceExcerpt = {
    source_path: sourcePath, reference_id: key ? `reference:${key}` : null,
    source_sha256: null, record_version: null, requested_pinpoint: pinpoint,
    resolution_status: "source_gap", gap_reason: "unsupported_source_path", resolved_heading: null,
    excerpt: "", excerpt_truncated: false,
  };
  if (!key) return result;
  const store = await openContentStore();
  const rid = parseRef({ table: "reference", id: key });
  const whole = pinpoint.trim() === "whole file" || pinpoint.trim() === "whole member";
  const escaped = pinpoint.trim().replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  // Numeric/rule identifiers may prefix a heading title; prose pinpoints must match its entire title.
  const identifier = /^(?:\d+(?:\.\d+)+|[A-Z]\.\d+|Rule \d+(?:\.\d+)+)$/.test(pinpoint.trim());
  const pattern = `(?i)^#{1,6}[ \\t]+(?:\\*\\*)?${escaped}${identifier ? "(?:[ \\t:—–]|\\*\\*|\\r?$)" : "(?:\\*\\*)?[ \\t]*\\r?$"}`;
  const metadata = contentPage(await store.db.query(
    `SELECT string::slice(source_path ?? '', 0, 513) AS source_path,
      string::slice(sha256 ?? '', 0, 65) AS source_sha256, string::len(body ?? '') AS body_chars,
      ${whole ? "[]" : "array::slice(array::filter(string::split(body ?? '', '\\n'), |$line| string::matches($line, $pattern)), 0, 2).map(|$line| string::slice($line, 0, 512))"} AS headings
      FROM $rid LIMIT 2;`, { rid, pattern },
  ), 2);
  if (metadata.length === 0) { result.gap_reason = "reference_missing"; return result; }
  if (metadata.length !== 1) throw new Error(`Ambiguous shared excerpt reference: ${key}`);
  const row = metadata[0];
  if (typeof row.source_path !== "string" || typeof row.source_sha256 !== "string" || !/^[a-f0-9]{64}$/.test(row.source_sha256)
    || !Number.isSafeInteger(row.body_chars) || (row.body_chars as number) <= 0 || !Array.isArray(row.headings)
    || row.headings.length > 2 || row.headings.some(h => typeof h !== "string" || Array.from(h).length > HEADING_CHAR_LIMIT)) {
    throw new Error(`Malformed shared excerpt metadata: ${key}`);
  }
  result.source_sha256 = row.source_sha256;
  result.record_version = row.source_sha256;
  if (row.source_path.replace(/\\/g, "/") !== sourcePath) { result.gap_reason = "reference_path_mismatch"; return result; }
  if ((row.body_chars as number) > EXCERPT_BODY_CHAR_BUDGET) throw new Error("Shared excerpt source exceeds 16Mi-character read budget");
  const headings = row.headings as string[];
  if (!whole && (headings.length !== 1 || Array.from(headings[0]).length >= HEADING_CHAR_LIMIT)) {
    result.resolution_status = "pinpoint_gap";
    result.gap_reason = headings.length > 1 ? "ambiguous_heading" : headings.length === 1 ? "heading_budget_exceeded" : "pinpoint_not_resolved";
    return result;
  }
  const marker = whole ? "" : headings[0];
  const page = contentPage(await store.db.query(
    `SELECT ${whole ? "1" : "array::len(string::split(body, $marker))"} AS parts,
      string::slice(${whole ? "body" : "string::split(body, $marker)[1] ?? ''"}, 0, 4097) AS excerpt
      FROM $rid WHERE sha256 = $sha AND source_path = $path LIMIT 2;`,
    { rid, sha: row.source_sha256, path: row.source_path, marker },
  ), 2);
  if (page.length === 0) { result.resolution_status = "source_changed"; result.gap_reason = "reference_changed_or_disappeared"; return result; }
  if (page.length !== 1 || !Number.isSafeInteger(page[0].parts) || typeof page[0].excerpt !== "string"
    || Array.from(page[0].excerpt).length > EXCERPT_CHAR_LIMIT + 1) throw new Error(`Malformed bounded shared excerpt: ${key}`);
  if (!whole && page[0].parts !== 2) { result.resolution_status = "pinpoint_gap"; result.gap_reason = "ambiguous_heading_occurrence"; return result; }
  const text = marker + page[0].excerpt;
  // End at the next Markdown heading: do not silently append an unrelated section.
  const nextHeading = whole ? null : /\n#{1,6}[ \t]+/.exec(text);
  const located = nextHeading ? text.slice(0, nextHeading.index + 1) : text;
  const characters = Array.from(located);
  result.excerpt = characters.slice(0, EXCERPT_CHAR_LIMIT).join("");
  result.excerpt_truncated = characters.length > EXCERPT_CHAR_LIMIT;
  result.resolved_heading = whole ? null : marker;
  result.resolution_status = "resolved";
  result.gap_reason = null;
  return result;
}

/** Identify the explicit embedded fixture configuration used by these readers.
 * Inputs: none; reads the environment setting that getStore() prioritizes over secrets/defaults.
 * Outputs: true only for an explicit mem:// URL; unknown/local/remote settings are strict.
 * Effects: no writes. Pick this over duplicating private URL/secrets resolution; these callers pass no override.
 */
function fixtureFallback(): boolean {
  return process.env.CUSTODY_CASE_DB?.trim() === "mem://";
}

/** Open the configured content store without disguising availability errors.
 * Inputs: none; getStore() owns endpoint resolution and connection reuse.
 * Outputs: a connected store or an error; fixture mode does not suppress connection failures.
 * Effects: delegates connection acquisition; no content writes. Use for reads instead of a quiet fallback opener.
 */
async function openContentStore(): Promise<StoreOk> {
  const store = await getStore();
  if (!store.available) throw new Error("Shared content store unavailable");
  return store;
}

/** Validate one normalized database page before a caller can consume its rows.
 * Inputs: SDK query results and the requested maximum page size.
 * Outputs: object records; malformed result envelopes/rows or oversized pages throw.
 * Effects: none. Use for content queries instead of coercing missing results to an empty table.
 */
function contentPage(results: unknown, limit: number): Record<string, unknown>[] {
  if (!Array.isArray(results) || results.length !== 1 || !Array.isArray(results[0])) {
    throw new Error("Malformed shared content query result");
  }
  const page = normalize(results[0]);
  if (!Array.isArray(page) || page.length > limit || page.some(row => !row || typeof row !== "object" || Array.isArray(row))) {
    throw new Error("Malformed shared content page");
  }
  return page as Record<string, unknown>[];
}

/** Read the current exact reference without caching or silent packaged fallback.
 * Inputs: reference key.
 * Outputs: a body/data record; null only for an absent explicit mem:// fixture row.
 * Effects: one bounded database read; missing production rows, malformed rows and failures throw.
 * Pick this for content consumers; caseRecord is the sibling for a versioned legal-record envelope.
 */
export async function getReference(key: string): Promise<ReferenceRow | null> {
  const fixture = fixtureFallback();
  const store = await openContentStore();
  const rid = parseRef({ table: "reference", id: key });
  const page = contentPage(await store.db.query("SELECT * FROM $rid LIMIT 2;", { rid }), 2);
  if (page.length === 0 && fixture) return null;
  if (page.length !== 1) throw new Error(`Shared reference missing or ambiguous: ${key}`);
  const row = page[0];
  const hasBody = typeof row.body === "string" && row.body.length > 0;
  const hasData = row.data !== null && typeof row.data === "object";
  if (!hasBody && !hasData) throw new Error(`Malformed shared reference content: ${key}`);
  if (row.body !== undefined && typeof row.body !== "string") throw new Error(`Malformed shared reference body: ${key}`);
  return row as ReferenceRow;
}

/** Read current source records using bounded ordered keyset pages.
 * Inputs: none; hard limits are 200 rows/page and 10,000 rows per complete listing.
 * Outputs: the complete bounded array; an empty explicit mem:// fixture array permits file fallback.
 * Effects: read-only queries; empty production tables, excess rows, malformed/nonadvancing pages throw.
 * Pick this for existing ledger consumers; this is a fresh traversal, not a transactional snapshot across pages.
 */
export async function getSources(): Promise<Array<Record<string, unknown>> | null> {
  const fixture = fixtureFallback();
  const store = await openContentStore();
  const all: Record<string, unknown>[] = [];
  let after: string | null = null;
  const seen = new Set<string>();
  for (let pageNumber = 0; pageNumber <= SOURCE_ROW_BUDGET / SOURCE_PAGE_SIZE; pageNumber++) {
    const results = await store.db.query(
      after === null
        ? `SELECT *, record::id(id) AS content_cursor FROM source ORDER BY content_cursor ASC LIMIT ${SOURCE_PAGE_SIZE};`
        : `SELECT *, record::id(id) AS content_cursor FROM source WHERE record::id(id) > $after ORDER BY content_cursor ASC LIMIT ${SOURCE_PAGE_SIZE};`,
      after === null ? {} : { after },
    );
    const page = contentPage(results, SOURCE_PAGE_SIZE);
    let cursor = after;
    for (const row of page) {
      const id = row.id;
      if (typeof id !== "string" || !id.startsWith("source:") || id.length <= 7 || seen.has(id) || typeof row.content_cursor !== "string" || row.content_cursor !== id.slice(7) || (cursor !== null && row.content_cursor <= cursor)) {
        throw new Error("Malformed or nonadvancing shared source page");
      }
      seen.add(id);
      cursor = row.content_cursor;
      delete row.content_cursor;
    }
    if (all.length + page.length > SOURCE_ROW_BUDGET) throw new Error("Shared source listing exceeds 10000-row budget");
    all.push(...page);
    if (page.length < SOURCE_PAGE_SIZE) {
      if (all.length === 0 && !fixture) throw new Error("Shared source table is empty");
      return all;
    }
    after = cursor;
    if (after === undefined || after === null) throw new Error("Malformed shared source cursor");
  }
  throw new Error("Shared source listing page budget exhausted");
}

/** Preserve the existing test reset API after removal of all content caches.
 * Inputs/outputs: none. Effects: none; every call now performs fresh reads.
 * Pick this compatibility hook for existing fixtures; no production invalidation is required.
 */
export function resetContentStoreCacheForTests(): void {}
