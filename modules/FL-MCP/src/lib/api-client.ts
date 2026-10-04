// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — bounded sidecar requests.
//
// Low-level fetch wrappers over the sidecar's /api/store/* endpoints. Kept
// deliberately thin and un-opinionated; src/lib/queries.ts wraps these in
// TanStack Query `queryOptions()` factories for route loaders/components.

import { resolveSidecarBaseUrl } from "./sidecar";
import type {
  CaseDocketResponse,
  CaseEvalsResponse,
  CaseExportResult,
  CaseGraphResult,
  CaseMemoResponse,
  CaseReferenceResponse,
  ReferenceLibraryPage,
  CaseSearchResult,
  CaseSourceResponse,
  CaseStatusResponse,
  CaseSummary,
  CaseTimelineResponse,
  EvidenceResponse,
  FactorMapEntry,
  StoreResponse,
  TimelineMode,
} from "@/types/store";

const SIDECAR_REQUEST_TIMEOUT_MS = 6_000;

async function sidecarFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const base = await resolveSidecarBaseUrl();
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), SIDECAR_REQUEST_TIMEOUT_MS);
  const abortFromCaller = () => controller.abort();
  init?.signal?.addEventListener("abort", abortFromCaller, { once: true });

  let res: Response;
  try {
    res = await fetch(`${base}${path}`, {
      ...init,
      headers: { "content-type": "application/json", ...init?.headers },
      signal: controller.signal,
    });
  } catch (error) {
    if (controller.signal.aborted && !init?.signal?.aborted) {
      throw new Error("The local case store did not respond within 6 seconds.", { cause: error });
    }
    throw error;
  } finally {
    window.clearTimeout(timer);
    init?.signal?.removeEventListener("abort", abortFromCaller);
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(typeof body?.error === "string" ? body.error : `Sidecar request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export const storeApi = {
  summary: () => sidecarFetch<StoreResponse<CaseSummary>>("/api/store/summary"),

  search: (params: { q: string; mode?: "text" | "vector" | "hybrid"; k?: number; tables?: string[] }) => {
    const usp = new URLSearchParams({ q: params.q });
    if (params.mode) usp.set("mode", params.mode);
    if (params.k) usp.set("k", String(params.k));
    if (params.tables?.length) usp.set("tables", params.tables.join(","));
    return sidecarFetch<StoreResponse<CaseSearchResult>>(`/api/store/search?${usp}`);
  },

  graph: (params: { id: string; depth?: 1 | 2; edges?: string[] }) => {
    const usp = new URLSearchParams({ id: params.id });
    if (params.depth) usp.set("depth", String(params.depth));
    if (params.edges?.length) usp.set("edges", params.edges.join(","));
    return sidecarFetch<StoreResponse<CaseGraphResult>>(`/api/store/graph?${usp}`);
  },

  timeline: (params: { mode?: TimelineMode; from?: string; to?: string; knownBy?: string; tables?: string[] } = {}) => {
    const usp = new URLSearchParams();
    if (params.mode) usp.set("mode", params.mode);
    if (params.from) usp.set("from", params.from);
    if (params.to) usp.set("to", params.to);
    if (params.knownBy) usp.set("known_by", params.knownBy);
    if (params.tables?.length) usp.set("tables", params.tables.join(","));
    return sidecarFetch<StoreResponse<CaseTimelineResponse>>(`/api/store/timeline?${usp}`);
  },

  factorMap: () => sidecarFetch<StoreResponse<{ entries: FactorMapEntry[] }>>("/api/store/factor-map"),

  docket: () => sidecarFetch<StoreResponse<CaseDocketResponse>>("/api/store/docket"),
  memos: () => sidecarFetch<StoreResponse<CaseMemoResponse>>("/api/store/memos"),
  status: () => sidecarFetch<StoreResponse<CaseStatusResponse>>("/api/store/status"),
  source: (id: string) => sidecarFetch<StoreResponse<CaseSourceResponse>>(`/api/store/source?id=${encodeURIComponent(id)}`),
  reference: (match?: string) =>
    sidecarFetch<StoreResponse<CaseReferenceResponse>>(`/api/store/reference${match ? `?match=${encodeURIComponent(match)}` : ""}`),
  /**
   * Reads one bounded page from the shared reference library endpoint.
   * Inputs are optional search/page parameters; output is a page response or
   * visible shared-store unavailability. It performs an HTTP GET only; use
   * reference() for ontology matches and the legacy compact reference list.
   */
  referenceLibrary: (params: { q?: string; limit?: number; offset?: number } = {}) => {
    const usp = new URLSearchParams();
    if (params.q) usp.set("q", params.q);
    if (params.limit !== undefined) usp.set("limit", String(params.limit));
    if (params.offset !== undefined) usp.set("offset", String(params.offset));
    return sidecarFetch<StoreResponse<ReferenceLibraryPage>>(`/api/store/reference/library?${usp}`);
  },
  evidence: () => sidecarFetch<StoreResponse<EvidenceResponse>>("/api/store/evidence"),
  evals: () => sidecarFetch<StoreResponse<CaseEvalsResponse>>("/api/store/evals"),

  exportSnapshot: (path?: string) =>
    sidecarFetch<StoreResponse<CaseExportResult>>("/api/store/export", {
      method: "POST",
      body: JSON.stringify({ format: "snapshot", path }),
    }),
};

export const authApi = {
  status: () =>
    sidecarFetch<{ configured: boolean; source: string; tokenLength: number | null; chat: { available: boolean; source: string; fix?: string } }>(
      "/api/auth/status",
    ),
};
