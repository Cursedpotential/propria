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
