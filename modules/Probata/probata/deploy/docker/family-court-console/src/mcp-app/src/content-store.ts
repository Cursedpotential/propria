// Byline: Claude Code · Sonnet 5 · 2026-09-08
//
// "Content lives in the SurrealDB store" — thin, read-only cache layer over
// the `reference` and `source` tables that scripts/load-content-to-store.mjs
// populates from the plugin's on-disk content/ and skills/ files. This module
// never writes; loading is entirely the loader script's job.
//
// core.ts, court-language.ts, and survival-guide.ts each call getReference()/
// getSources() FIRST and fall back to their existing direct-file read
// whenever the store is unavailable OR has no matching row for that key — so
// an EMPTY mem:// store (the mem:// test situation, and any fresh plugin
// install before the loader has ever run) behaves exactly as those readers
// did before this file existed. Every failure mode here (store unreachable,
// row absent, malformed row) degrades to `null`/`[]` — this module never
// throws, because a throw here would take down a content reader that used to
// be a plain, reliable file read.
//
// In-process cache, 5 minute TTL, independent per key (`reference:<key>`) and
// for the whole `source` table listing. Reset via
// resetContentStoreCacheForTests() between mem:// store swaps in tests —
// otherwise a stale cached `null` (recorded before the store was populated)
// would outlive the store reset and silently fail a later assertion.

import { getStore, normalize, parseRef, type StoreOk } from "./store.js";

const CACHE_TTL_MS = 5 * 60 * 1000;

/** A `reference:<key>` row as read back from the store: `kind`/`key` plus
 * either a text `body` (markdown/whole-file content) or structured `data`
 * (parsed JSON), and the loader's own provenance fields. Shape matches
 * scripts/load-content-to-store.mjs's writes — see README-store.md. */
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

interface CacheEntry<T> {
  value: T;
  at: number;
}

let referenceCache = new Map<string, CacheEntry<ReferenceRow | null>>();
let sourcesCache: CacheEntry<Array<Record<string, unknown>>> | null = null;

function isFresh<T>(entry: CacheEntry<T> | null | undefined): entry is CacheEntry<T> {
  return !!entry && Date.now() - entry.at < CACHE_TTL_MS;
}

async function openStoreQuiet(): Promise<StoreOk | null> {
  try {
    const result = await getStore();
    return result.available ? result : null;
  } catch {
    return null;
  }
}

/**
 * Reads one `reference:<key>` row. Returns `null` when the store is
 * unavailable (no driver, connection failure) OR when no row exists for that
 * key (empty store, or that content was never loaded) — both cases mean
 * "the caller must fall back to its file read," so they are deliberately not
 * distinguished here.
 */
export async function getReference(key: string): Promise<ReferenceRow | null> {
  const cached = referenceCache.get(key);
  if (isFresh(cached)) return cached.value;

  const store = await openStoreQuiet();
  if (!store) return null;

  try {
    const rid = parseRef({ table: "reference", id: key });
    const rows = await store.db.query<Array<Record<string, unknown>>>("SELECT * FROM $rid;", { rid });
    const raw = rows.at(-1)?.[0];
    const row = raw ? ((normalize(raw) as ReferenceRow) ?? null) : null;
    referenceCache.set(key, { value: row, at: Date.now() });
    return row;
  } catch {
    // A malformed key or a transient query failure must never crash a
    // content reader — treat it exactly like "no such row."
    return null;
  }
}

/**
 * Reads every row of the `source` table (the ledger.json entries the loader
 * writes as `source:<id>`). Returns `null` when the store is unavailable;
 * returns `[]` (not `null`) when the store is reachable but the table is
 * empty — callers treat both as "fall back to the file read," but only
 * `null` means "the store itself could not be asked."
 */
export async function getSources(): Promise<Array<Record<string, unknown>> | null> {
  if (isFresh(sourcesCache)) return sourcesCache.value;

  const store = await openStoreQuiet();
  if (!store) return null;

  try {
    const rows = await store.db.query<Array<Record<string, unknown>>>("SELECT * FROM source LIMIT 1000;");
    const list = (normalize(rows.at(-1) ?? []) as Array<Record<string, unknown>>) ?? [];
    sourcesCache = { value: list, at: Date.now() };
    return list;
  } catch {
    return null;
  }
}

/** Test-only: clears both in-process caches so a test that swaps the
 * underlying mem:// store (store.resetStoreForTests()) doesn't read a stale
 * cached answer recorded against the previous store instance. */
export function resetContentStoreCacheForTests(): void {
  referenceCache = new Map();
  sourcesCache = null;
}
