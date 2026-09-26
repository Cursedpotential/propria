/**
 * Byline: Claude Code · Sonnet 5 · 2026-09-14
 *
 * One adapter module for every call into the `casebible_index` `serve` HTTP API
 * (loopback only, e.g. http://127.0.0.1:8765). The renderer never holds a
 * database credential -- every privileged lookup (Parquet lake, Weaviate,
 * Surreal graph) happens inside that backend process; this module only ever
 * speaks plain HTTP to it, same as the existing `filesystemSearchIndex` Tauri
 * bridge speaks to Weaviate through the same backend.
 *
 * Base URL is configurable via `VITE_INTAKE_BACKEND_API_URL` (build-time),
 * defaulting to the `casebible-corpus serve` default port 8765. Deliberately
 * `localhost`, not `127.0.0.1`: the Tauri desktop CSP (`apps/src-tauri/
 * tauri.conf.json`, connect-src) allows `http://localhost:*` but NOT
 * `127.0.0.1` -- CSP source matching is a literal hostname string match, not a
 * DNS/hosts-file resolution, so the two are different origins to the browser
 * engine even though they resolve to the same loopback address.
 *
 * KNOWN LIMITATION (found live, not yet resolved): the portal's web deploy
 * (`progress-board/server.mjs`) sends `connect-src 'self'` for
 * `/intake/xplorer/*`, which blocks this fetch entirely in that browser
 * context -- only the native Tauri desktop app can reach this backend today.
 * See the receipt for the exact blocked-request evidence and options.
 */

const BASE_URL = (import.meta.env.VITE_INTAKE_BACKEND_API_URL ?? 'http://localhost:8765').replace(
  /\/$/,
  '',
);

class IntakeBackendError extends Error {
  constructor(
    message: string,
    readonly status?: number,
  ) {
    super(message);
    this.name = 'IntakeBackendError';
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    });
  } catch {
    throw new IntakeBackendError(
      'Backend unreachable (is `casebible-corpus serve` running?)',
      undefined,
    );
  }
  if (!response.ok) {
    let detail = '';
    try {
      const body = (await response.json()) as { detail?: string };
      detail = body.detail ?? '';
    } catch {
      // Body was not JSON; fall through with an empty detail.
    }
    throw new IntakeBackendError(detail || `Backend returned ${response.status}`, response.status);
  }
  return (await response.json()) as T;
}

// ── Filesystem lookup (native metadata panel) ───────────────────────────────

export interface LookupResult {
  found: boolean;
  reason?: string;
  relative_path?: string;
  filename?: string;
  extension?: string;
  media_type?: string;
  byte_size?: number;
  content_sha256?: string;
  source_created_at?: string;
  source_modified_at?: string;
  indexed_at?: string;
  title?: string;
  document_type?: string;
  document_date?: string | null;
  date_basis?: string;
  short_summary?: string;
  detailed_summary?: string;
  people?: string[];
  organizations?: string[];
  locations?: string[];
  dates_mentioned?: string[];
  topics?: string[];
  keywords?: string[];
  case_relevance?: string;
  language?: string;
  confidence?: number;
  review_notes?: string[];
  review_state?: string;
  record_role?: string;
  index_status?: string;
  extraction_method?: string;
  extraction_notes?: string[];
  page_count?: number | null;
  text_char_count?: number;
  chunk_count?: number;
  summary_coverage?: string;
  summary_coverage_ratio?: number;
  summary_model?: string;
  embedding_model?: string;
  embedding_dimensions?: number;
  schema_version?: string;
  snapshot?: string;
  duplicates: { relative_path: string; filename: string }[];
}

export const lookupPath = (absolutePath: string): Promise<LookupResult> =>
  request<LookupResult>(`/filesystem/lookup?path=${encodeURIComponent(absolutePath)}`);

// ── CocoIndex semantic search (Parquet lake, via NIM embeddings) ───────────

export interface CocoIndexHit {
  score: number;
  semantic_score: number;
  lexical_score: number;
  document_id: string;
  version_id: string;
  chunk_id: string;
  relative_path: string;
  filename: string;
  document_type: string;
  document_date: string | null;
  title: string;
  short_summary: string;
  chunk_ordinal: number;
  char_start: number;
  char_end: number;
  text: string;
}

export interface CocoIndexSearchResponse {
  query: string;
  model: string;
  snapshot: string | null;
  hits: CocoIndexHit[];
}

export const searchCocoIndex = (
  query: string,
  options?: { limit?: number; hybrid?: boolean; documentType?: string; pathPrefix?: string },
): Promise<CocoIndexSearchResponse> =>
  request<CocoIndexSearchResponse>('/search', {
    method: 'POST',
    body: JSON.stringify({
      query,
      limit: options?.limit ?? 10,
      hybrid: options?.hybrid ?? true,
      document_type: options?.documentType ?? null,
      path_prefix: options?.pathPrefix ?? null,
    }),
  });

// ── DuckDB SQL over the lake (`documents` / `chunks` views only) ───────────

export interface LakeQueryResult {
  columns: string[];
  rows: unknown[][];
  row_count: number;
  truncated: boolean;
  views_available: string[];
}

export const runLakeQuery = (sql: string, limit = 200): Promise<LakeQueryResult> =>
  request<LakeQueryResult>('/filesystem/duckdb', {
    method: 'POST',
    body: JSON.stringify({ sql, limit }),
  });

// ── Surreal graph (read-only; a separate agent may be writing concurrently) ─

export interface GraphNeighborhood {
  root: unknown;
  edges: unknown[];
}

export const graphNeighbors = (
  table: string,
  key: string,
  edgeLimit = 50,
): Promise<GraphNeighborhood> =>
  request<GraphNeighborhood>(
    `/filesystem/graph/neighbors/${encodeURIComponent(table)}/${encodeURIComponent(key)}?edge_limit=${edgeLimit}`,
  );

export const graphStatus = (): Promise<{ status: string; version?: string }> =>
  request('/filesystem/graph/status');

// ── Index run status ─────────────────────────────────────────────────────

export interface FilesystemRunStatus {
  state?: string;
  coverage?: string;
  failure_events?: number;
  files_observed?: number;
  files_transformed?: number;
  run_id?: string;
  source_id?: string;
  reported_at?: string;
}

export const filesystemStatus = (): Promise<FilesystemRunStatus> => request('/filesystem/status');

export { IntakeBackendError };
