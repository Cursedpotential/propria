// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Client-side mirror of mcp-app/src/store.ts's exported shapes. store.ts's
// case_docket/case_memo/case_status/case_source/case_reference/
// case_evidence_log/case_eval/case_export("platform") tools and mode-aware
// case_timeline are all real, landed features — store.ts even exposes
// compatibility alias functions (`caseMemo`, `caseStatus`, `caseSource`,
// `caseReference`, `caseEvidenceLog`, `caseEvals`) matching this app's
// `callStoreFn` dispatch names exactly (see store.ts's own "Sidecar
// compatibility aliases" comment block) — so the shapes below are
// transcribed directly from the landed store.ts, not guessed.
//
// Owner order 2026-09-07 15:29: the `Queued<T>` / `isQueued` speculative
// "not implemented yet" plumbing that existed for the concurrent-work
// window has been dropped — every shape here is unconditionally real.

export interface StoreUnavailable {
  available: false;
  reason: string;
}

export type StoreResponse<T> = T | StoreUnavailable;

export function isUnavailable<T>(value: StoreResponse<T>): value is StoreUnavailable {
  return typeof value === "object" && value !== null && (value as StoreUnavailable).available === false;
}

// ---------------------------------------------------------------------------
// Tables / edges
// ---------------------------------------------------------------------------

export const DATA_TABLES = [
  "person", "child", "order", "hearing", "deadline", "event", "message", "exhibit", "factor",
  "source", "note", "court", "court_event", "filing", "draft", "memo", "reference",
  "evidence_log", "eval", "case_status",
] as const;
export type DataTable = (typeof DATA_TABLES)[number];

export const SEARCHABLE_TABLES = ["event", "message", "note", "exhibit"] as const;
export type SearchableTable = (typeof SEARCHABLE_TABLES)[number];

// ---------------------------------------------------------------------------
// case_summary — the "/" route's headline data (still the primary source
// there; case_status is a separate, owner-populated singleton of unknown
// shape — see CaseStatusRecord below — used opportunistically, not as the
// sole source of truth for the headline card).
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
  children: { count: number; entries: Array<{ initials: string | null; age: number | string | null }> };
  flags: string[];
  source: string;
}

// ---------------------------------------------------------------------------
// case_search
// ---------------------------------------------------------------------------

export interface CaseSearchHit {
  id: string;
  table: SearchableTable;
  snippet: string;
  score: number;
  occurred_at: string | null;
  known_at: string | null;
}

export interface CaseSearchResult {
  hits: CaseSearchHit[];
  mode: "text" | "vector" | "hybrid";
  degraded: boolean;
  degraded_reason: string | null;
}

// ---------------------------------------------------------------------------
// case_graph
// ---------------------------------------------------------------------------

export interface CaseGraphNeighbor {
  id: string;
  table: string;
  edge: string;
  direction: "in" | "out";
  record: Record<string, unknown>;
}

export interface CaseGraphResult {
  id: string;
  neighbors: CaseGraphNeighbor[];
}

// ---------------------------------------------------------------------------
// case_timeline — mode-aware (court | master | merged).
// "court" = court_event ∪ hearing ∪ deadline ∪ order (deliberately kept
// separate from the master lane, owner order 2026-09-07 13:09-13:16).
// "master" = event ∪ message ∪ exhibit (the extracted-from-corpora
// timeline). "merged" (default) = both, each entry carrying its `lane`.
// ---------------------------------------------------------------------------

export const TIMELINE_MODES = ["court", "master", "merged"] as const;
export type TimelineMode = (typeof TIMELINE_MODES)[number];

export interface CaseTimelineEntry {
  lane: "court" | "master";
  table: string;
  id: string;
  /** Primary sort date (ISO): occurred_at for master lane; the table's own
   * date field (court_event.date / hearing.date / deadline.due / order.entered)
   * for the court lane. */
  date: string;
  occurred_at: string | null;
  known_at: string | null;
  summary: string;
  source: Record<string, unknown> | null;
}

/** The sidecar passes `mode` straight through to the real case_timeline tool. */
export interface CaseTimelineResponse {
  entries: CaseTimelineEntry[];
  mode: TimelineMode;
}

// ---------------------------------------------------------------------------
// case_factor_map
// ---------------------------------------------------------------------------

export interface FactorMapEntry {
  letter: string;
  title: string;
  support_count: number;
  contradict_count: number;
  top_supporting: Array<{ id: string; summary: string; weight: number | null }>;
  top_contradicting: Array<{ id: string; summary: string; weight: number | null }>;
}

// ---------------------------------------------------------------------------
// case_export — "snapshot" (default) and "platform" (caseExportPlatform) formats.
// ---------------------------------------------------------------------------

export interface CaseExportSnapshotResult {
  path: string;
  counts: Record<string, number>;
}

export interface CaseExportPlatformResult {
  dir: string;
  counts: Record<string, number>;
}

export type CaseExportResult = CaseExportSnapshotResult | CaseExportPlatformResult;

// ---------------------------------------------------------------------------
// case_docket — filings + drafts + orders + upcoming court_events.
// ---------------------------------------------------------------------------

export interface DocketEntry {
  table: "filing" | "draft" | "order" | "court_event";
  id: string;
  title: string;
  date: string | null;
  status: string | null;
  /** The full underlying row — e.g. `record.in_force` for an order,
   * `record.doc_type` for a filing/draft, `record.source` for provenance. */
  record: Record<string, unknown>;
}

export interface CaseDocketFilter {
  status?: string;
  doc_type?: string;
  in_force?: boolean;
}

export interface CaseDocketResponse {
  entries: DocketEntry[];
}

// ---------------------------------------------------------------------------
// case_memo — analysis/strategy/weakness/direction memos.
// ---------------------------------------------------------------------------

export interface CaseMemoRecord {
  id: string;
  kind: string;
  title: string;
  text: string;
  status: string;
  factors: string[] | null;
  author: string | null;
  supersedes: string | null;
  source: Record<string, unknown> | null;
  created_at: string;
}

export interface CaseMemoResponse {
  memos: CaseMemoRecord[];
}

// ---------------------------------------------------------------------------
// case_status — an owner-populated singleton (`case_status:current`).
// The exact field set is whatever `case_status_set` has been called with;
// treat it as an open record and render only the fields that show up.
// ---------------------------------------------------------------------------

export type CaseStatusRecord = Record<string, unknown>;
export type CaseStatusResponse = CaseStatusRecord;

// ---------------------------------------------------------------------------
// case_source — provenance for one record ref (docket/evidence row's source link).
// ---------------------------------------------------------------------------

export interface CaseSourceResult {
  id: string;
  source: Record<string, unknown> | null;
  path_exists: boolean | null;
  r2_path: string | null;
}

export type CaseSourceResponse = CaseSourceResult | { available: false; reason: string };

// ---------------------------------------------------------------------------
// case_reference — behavior-pattern / ontology grid + text-match tool.
// Two distinct shapes depending on whether a `match` string was passed.
// ---------------------------------------------------------------------------

export interface ReferenceRow {
  id: string;
  kind?: string;
  category?: string;
  pattern?: string;
  aliases?: string[];
  [key: string]: unknown;
}

export interface ReferenceMatchHit {
  id: string;
  kind: string | null;
  category: string | null;
  matched: string;
  span: [number, number];
  via: "pattern" | "alias";
}

export interface CaseReferenceResponse {
  entries: ReferenceRow[] | ReferenceMatchHit[];
}

export function isMatchHits(entries: ReferenceRow[] | ReferenceMatchHit[]): entries is ReferenceMatchHit[] {
  return entries.length === 0 || "matched" in entries[0];
}

// ---------------------------------------------------------------------------
// case_evidence_log — distinct from the live `exhibit` table (exhibits come
// from case_search over the "exhibit" table instead).
// ---------------------------------------------------------------------------

export interface EvidenceLogRecord {
  id: string;
  action: string;
  by: string | null;
  hash: string | null;
  path: string | null;
  notes: string | null;
  logged_at: string;
  source: Record<string, unknown> | null;
}

export interface CaseEvidenceLogResponse {
  entries: EvidenceLogRecord[];
}

export interface EvidenceResponse {
  exhibits: CaseSearchResult | { error: string };
  evidence_log: CaseEvidenceLogResponse;
}

// ---------------------------------------------------------------------------
// case_eval — evals/reports list + detail.
// ---------------------------------------------------------------------------

export interface EvalRecord {
  id: string;
  kind: string;
  title: string;
  subject: string | null;
  score: number | null;
  verdict: string | null;
  text: string | null;
  path: string | null;
  tool_or_model: string | null;
  created_at: string;
  source: Record<string, unknown> | null;
}

export interface CaseEvalsResponse {
  evals: EvalRecord[];
}
