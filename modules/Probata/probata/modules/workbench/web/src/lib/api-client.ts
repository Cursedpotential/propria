// Byline: Claude Code · Sonnet (agent) · 2026-07-22 (C3: records, schemas, verify, parse-dryrun, flags; C4: knowledge search/browse + Graphiti pane added 2026-07-23)
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-24: Graphiti client calls removed; Graphiti is retired, D-070)
// Byline: Codex · GPT-5 · 2026-08-15 (run reports, review actions, and court readiness)
// Byline: Codex · GPT-5 · 2026-08-18 (conversation intake and governed entities)
// Byline amendment: Codex · GPT-5 · 2026-08-18 (third-party review client)
// Byline: Codex · GPT-5 · 2026-08-18 (native evidence horizon search parameter)
// Byline: Codex · GPT-5 · 2026-08-28 (proffer workflow (formerly Universal Import Workflow) client)
// Byline: Claude Code · Opus 5.5 · 2026-09-25 (run source-context read-back for Review Actions)
// Byline: Claude Code · Opus 5.5 · 2026-09-26 (repair workflow builder client; ApiError keeps its body)
/**
 * API client for the Knowledge Workbench.
 *
 * The static export (`output: "export"`) is served same-origin by the
 * platform's FastAPI backend, so the base URL is empty by default — every
 * call resolves relative to the page's own origin. `VITE_API_URL` remains a
 * supported build-time override for browser development against a backend on
 * a different port.
 */
import type {
  CustodyTier,
  CourtCase,
  CourtCaseStatus,
  EvidenceDetail,
  EvidenceSourceContent,
  EvidenceConversationContext,
  CourtReadiness,
  EvidenceItemListResponse,
  EvidencePromotionResult,
  EvidenceReviewDecision,
  EvidenceReviewListResponse,
  EvidenceReviewResult,
  FileAnalysis,
  FileTextResponse,
  Flag,
  FlagCreateRequest,
  FlagStatus,
  FlagTargetKind,
  FlagUpdateRequest,
  HealthDepsResponse,
  KnowledgeContentsResponse,
  KnowledgeItemDetail,
  KnowledgeSourceRef,
  KnowledgeSourceResolution,
  KnowledgeSearchResponse,
  Matter,
  MatterDetail,
  MatterListResponse,
  ParseDryrunResponse,
  PgSchemaName,
  RecordMetaPatch,
  RecordRow,
  RecordsListResponse,
  RetryFromStage,
  RunAbortResponse,
  RunContinueResponse,
  RunCreateResponse,
  RunDetail,
  RunMode,
  RunReport,
  RunReviewAction,
  RunReviewActionRequest,
  RunRetryResponse,
  RunSummary,
  SchemasResponse,
  TableDetail,
  ThirdPartyApprovalRequest,
  ThirdPartyApprovalResponse,
  AcquisitionAssertionInput,
  GovernedEntitiesResponse,
  GovernedEntity,
  MessageCorpus,
  StagedFile,
  StagedFileMeta,
  ToolServerGroup,
  MonitoredActionCapability,
  MonitoredActionRun,
  StartAtomicToolActionRequest,
  UploadResponse,
  VerifyResponse,
  WeaviateDetail,
  Workflow,
  ProfferOperationDetail,
  ProfferOperationLifecycle,
  ProfferOperationListResponse,
  ProfferProposalResourceCatalog,
  ProfferOperatorSnapshot,
  ProfferDecisionResponse,
  ProfferPreviewResponse,
  ProfferRepairDecisionRequest,
  ProfferRepairDecisionResponse,
  ProfferPreviewMessagesResponse,
  ProfferContentResponse,
  ProfferPotentialPromotionFlag,
  ProfferPotentialPromotionFlagList,
  ProfferPotentialPromotionFlagRequest,
  ProfferStartRequest,
  ProfferStartResponse,
  ProfferUploadResponse,
  ProfferSourceBrowserResponse,
  ProfferSourceInspection,
  ProfferSourceObject,
  ProfferHumanSourceAssertions,
  ProfferObservedSource,
  ProfferRunSourceContext,
  ProfferSourceContextReceipt,
  ProfferHandlerSelectionDecisionRequest,
  ProfferHandlerSelectionDecisionResponse,
  ProfferBatchStartRequest,
  ProfferBatchStartResponse,
  ProfferBatchStatus,
  ProfferRepairCheck,
  ProfferRepairPlan,
  ProfferRepairProposeResponse,
  ProfferRepairRunResponse,
  ProfferRepairRunStatus,
  ProfferRepairToolsResponse,
  ProfferRepairValidateResponse,
  DecodedExistsResponse,
  CatalogUnitLookup,
  CatalogUnitsUnderPrefix,
  CatalogProvenance,
  SourceUnitKind,
  SourceUnitMark,
  SourceUnitMarkList,
  SourceUnitProposal,
  MatterMode,
} from "./shared/types";

const API_BASE = import.meta.env.VITE_API_URL || "";

/** Typed API error with HTTP status code for caller-side branching. */
export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    /** The parsed error body, when the server sent JSON (e.g. a refused repair run's checks). */
    public readonly body: unknown = undefined,
  ) {
    super(message);
    this.name = "ApiError";
  }

  /** True for 408, 429, 500, 502, 503, 504 — worth retrying. */
  get isRetryable(): boolean {
    return [408, 429, 500, 502, 503, 504].includes(this.status);
  }

  get isNotFound(): boolean {
    return this.status === 404;
  }

  get isConflict(): boolean {
    return this.status === 409;
  }

  /** 410 — the spine has retired the resource (e.g. a stale gate action
   * racing a run that moved on). Distinct from isConflict(409): a 409
   * means "not in the right state right now", a 410 means "gone for good". */
  get isGone(): boolean {
    return this.status === 410;
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, init);
  } catch {
    // Network failure (offline, DNS, CORS, etc.)
    throw new ApiError("Network error — check your connection", 0);
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      body.detail || `API error: ${res.status}`,
      res.status,
      body,
    );
  }
  return res.json();
}

export async function getHealth() {
  return apiFetch<{ status: string }>("/health");
}

/** Console header's dependency status chips (C2.6 requirement 4) —
 * `{lancedb, object_store, pg, milvus, checked_at}`. */
export async function getHealthDeps() {
  return apiFetch<HealthDepsResponse>("/api/health/deps");
}

export interface ListFilesParams {
  status?: string;
  detected_type?: string;
}

export async function listFiles(params: ListFilesParams = {}) {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.detected_type) qs.set("detected_type", params.detected_type);
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<StagedFile[]>(`/api/files${suffix}`);
}

export async function getFile(id: string) {
  return apiFetch<StagedFile>(`/api/files/${encodeURIComponent(id)}`);
}

export async function updateFileMeta(id: string, patch: Partial<StagedFileMeta>) {
  return apiFetch<StagedFile>(`/api/files/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

/** Full, untruncated extracted text for the Preview modal (C2.7). */
export async function getFileText(id: string) {
  return apiFetch<FileTextResponse>(`/api/files/${encodeURIComponent(id)}/text`);
}

/** Re-run the server's detect.py sniffing + basic shape stats (C2.7). */
export async function analyzeFile(id: string) {
  return apiFetch<FileAnalysis>(`/api/files/${encodeURIComponent(id)}/analyze`, {
    method: "POST",
  });
}

// ---------------------------------------------------------------------------
// Repair control surface
// ---------------------------------------------------------------------------

export interface RepairToolCard {
  id: string;
  category: string;
  description: string;
  execution_policy: "manual_or_auto" | "manual_approval_required" | string;
  side_effect: string;
}

export async function listRepairTools() {
  return apiFetch<RepairToolCard[]>("/api/repairs/tools");
}

export async function runAutomaticRepairAssessment(path: string, format?: string) {
  return apiFetch<Record<string, unknown>>("/api/repairs/automatic-assessment", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, format: format || null, sample_limit: 25 }),
  });
}

export async function executeRepairTool(
  toolId: string,
  payload: Record<string, unknown>,
  approved = false,
) {
  return apiFetch<Record<string, unknown>>("/api/repairs/execute", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tool_id: toolId, payload, approved }),
  });
}

// ---------------------------------------------------------------------------
// Runs (C1 Operator Console)
// ---------------------------------------------------------------------------

export interface ListRunsParams {
  status?: string;
  limit?: number;
}

export async function listRuns(params: ListRunsParams = {}) {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.limit) qs.set("limit", String(params.limit));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<RunSummary[]>(`/api/runs${suffix}`);
}

export async function getRun(runId: string) {
  return apiFetch<RunDetail>(`/api/runs/${encodeURIComponent(runId)}`);
}

/** Start a run from an already-staged file (JSON body — no re-upload). */
export async function createRunFromStaged(params: {
  stagedId: string;
  workflow: Workflow | string;
  domain: string;
  mode: RunMode;
  custodyTier?: CustodyTier;
  sourceMeta?: Record<string, unknown>;
  messageCorpus: MessageCorpus;
  sourcePrincipal: string;
  callerOwnsConversation: boolean;
  acquisition?: AcquisitionAssertionInput;
}) {
  return apiFetch<RunCreateResponse>("/api/runs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      staged_id: params.stagedId,
      workflow: params.workflow,
      domain: params.domain,
      mode: params.mode,
      custody_tier: params.custodyTier ?? null,
      source_meta: params.sourceMeta ?? null,
      message_corpus: params.messageCorpus,
      source_principal: params.sourcePrincipal,
      caller_owns_conversation: params.callerOwnsConversation,
      acquisition: params.acquisition ?? null,
    }),
  });
}

/** Start a run from a freshly dropped file (multipart — never lands in staging). */
export async function createRunFromFile(params: {
  file: File;
  workflow: Workflow | string;
  domain: string;
  mode: RunMode;
  custodyTier?: CustodyTier;
  sourceMeta?: Record<string, unknown>;
  messageCorpus: MessageCorpus;
  sourcePrincipal: string;
  callerOwnsConversation: boolean;
  acquisition?: AcquisitionAssertionInput;
}) {
  const formData = new FormData();
  formData.append("file", params.file);
  formData.append("workflow", params.workflow);
  formData.append("domain", params.domain);
  formData.append("mode", params.mode);
  if (params.custodyTier) formData.append("custody_tier", params.custodyTier);
  if (params.sourceMeta) formData.append("source_meta", JSON.stringify(params.sourceMeta));
  formData.append("message_corpus", params.messageCorpus);
  formData.append("source_principal", params.sourcePrincipal);
  formData.append("caller_owns_conversation", String(params.callerOwnsConversation));
  if (params.acquisition) formData.append("acquisition", JSON.stringify(params.acquisition));
  return apiFetch<RunCreateResponse>("/api/runs", { method: "POST", body: formData });
}

// ---------------------------------------------------------------------------
// Run gate controls (C2)
// ---------------------------------------------------------------------------

/** Release a gated (paused) run past its current stage boundary. Throws
 * ApiError(409) if the run isn't paused. */
export async function continueRun(runId: string) {
  return apiFetch<RunContinueResponse>(`/api/runs/${encodeURIComponent(runId)}/continue`, {
    method: "POST",
  });
}

/** Abort a running or gated run. While `running`, this takes effect at the
 * next stage boundary rather than instantly. Throws ApiError(409) if the
 * run is already terminal. */
export async function abortRun(runId: string) {
  return apiFetch<RunAbortResponse>(`/api/runs/${encodeURIComponent(runId)}/abort`, {
    method: "POST",
  });
}

/** Start a fresh run from a terminal-failed one. The returned `run_id` is
 * the NEW run (not the one passed in) — open that run to watch it.
 * Throws ApiError(409) if the source run isn't terminal-failed.
 *
 * `fromStage` (C2.6, optional): pass `"knowledge"` to skip straight to
 * re-running the knowledge stage over the parent's already-stored records
 * instead of a full custody->parse->store->knowledge rerun — see
 * `RetryFromStage`'s doc comment. Omit for the pre-C2.6 full-rerun. */
export async function retryRun(runId: string, fromStage?: RetryFromStage) {
  return apiFetch<RunRetryResponse>(`/api/runs/${encodeURIComponent(runId)}/retry`, {
    method: "POST",
    ...(fromStage
      ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify({ from_stage: fromStage }) }
      : {}),
  });
}

export async function getRunReport(runId: string) {
  return apiFetch<RunReport>(`/api/runs/${encodeURIComponent(runId)}/report`);
}

export async function createRunReviewAction(runId: string, payload: RunReviewActionRequest) {
  return apiFetch<RunReviewAction>(`/api/runs/${encodeURIComponent(runId)}/review-actions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

// ---------------------------------------------------------------------------
// Records (C3 — parse-quality review + curation)
// ---------------------------------------------------------------------------

export interface ListRecordsParams {
  artifactId?: string;
  runId?: string;
  q?: string;
  limit?: number;
  offset?: number;
}

export async function listRecords(params: ListRecordsParams = {}) {
  const qs = new URLSearchParams();
  if (params.artifactId) qs.set("artifact_id", params.artifactId);
  if (params.runId) qs.set("run_id", params.runId);
  if (params.q) qs.set("q", params.q);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<RecordsListResponse>(`/api/records${suffix}`);
}

/** Curation-only edit (title/labels/attrs_patch) — never touches evidence
 * blobs/hashes. Returns the updated record row. */
export async function patchRecordMeta(recordId: string, patch: RecordMetaPatch) {
  return apiFetch<RecordRow>(`/api/records/${encodeURIComponent(recordId)}/meta`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

// ---------------------------------------------------------------------------
// Data Explorer (read-only PostgreSQL + Weaviate inspection views)
// ---------------------------------------------------------------------------

export async function getSchemas() {
  return apiFetch<SchemasResponse>("/api/schemas");
}

export async function approveThirdPartyConversation(
  conversationId: string,
  review: ThirdPartyApprovalRequest,
) {
  return apiFetch<ThirdPartyApprovalResponse>(
    `/api/third-party-conversations/${encodeURIComponent(conversationId)}/approve`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(review),
    },
  );
}

export async function listGovernedEntities(query = "", limit = 20) {
  const qs = new URLSearchParams({ limit: String(limit) });
  if (query.trim()) qs.set("q", query.trim());
  return apiFetch<GovernedEntitiesResponse>(`/api/entities?${qs.toString()}`);
}

export async function createGovernedEntity(payload: { display_name: string; entity_type?: string }) {
  return apiFetch<GovernedEntity>("/api/entities", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function getTableDetail(schema: PgSchemaName, table: string, limit = 5) {
  const qs = new URLSearchParams({ limit: String(limit) });
  return apiFetch<TableDetail>(
    `/api/schemas/postgresql/${encodeURIComponent(schema)}/${encodeURIComponent(table)}?${qs.toString()}`,
  );
}

export async function getWeaviateDetail(collection: string, limit = 5) {
  const qs = new URLSearchParams({ limit: String(limit) });
  return apiFetch<WeaviateDetail>(
    `/api/schemas/weaviate/${encodeURIComponent(collection)}?${qs.toString()}`,
  );
}

// ---------------------------------------------------------------------------
// Verify (C3 — active hash verification)
// ---------------------------------------------------------------------------

export async function verifySha256(sha256: string) {
  return apiFetch<VerifyResponse>(`/api/verify/${encodeURIComponent(sha256)}`, {
    method: "POST",
  });
}

// ---------------------------------------------------------------------------
// Parse dry-run (C3 — "the real parser candidates")
// ---------------------------------------------------------------------------

/** Dry-run parse an already-staged file by sha256 — no run is created. */
export async function parseDryrunSha(sha256: string) {
  return apiFetch<ParseDryrunResponse>("/api/runs/parse-dryrun", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sha256 }),
  });
}

/** Dry-run parse a fresh, not-yet-staged file — no run is created. */
export async function parseDryrunFile(file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return apiFetch<ParseDryrunResponse>("/api/runs/parse-dryrun", { method: "POST", body: formData });
}

// ---------------------------------------------------------------------------
// Corroboration flags (C3 — requirements addendum 6)
// ---------------------------------------------------------------------------

export interface ListFlagsParams {
  status?: FlagStatus;
  targetKind?: FlagTargetKind;
}

export async function listFlags(params: ListFlagsParams = {}) {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.targetKind) qs.set("target_kind", params.targetKind);
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<Flag[]>(`/api/flags${suffix}`);
}

export async function createFlag(payload: FlagCreateRequest) {
  return apiFetch<Flag>("/api/flags", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateFlag(flagId: string, patch: FlagUpdateRequest) {
  return apiFetch<Flag>(`/api/flags/${encodeURIComponent(flagId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

// ---------------------------------------------------------------------------
// Knowledge (Weaviate projection search + canonical PostgreSQL browse/detail)
// ---------------------------------------------------------------------------

export interface SearchKnowledgeParams {
  caseId?: string;
  lane?: string;
  limit?: number;
  /** Optional as-lived ceiling. The evidence service clamps omitted searches
   * to the current instant and never accepts disclosure tiers from callers. */
  horizon?: string;
}

export async function searchKnowledge(query: string, params: SearchKnowledgeParams = {}) {
  const qs = new URLSearchParams({ q: query });
  qs.set("case_id", params.caseId || "primary");
  if (params.lane) qs.set("lane", params.lane);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.horizon) qs.set("horizon", params.horizon);
  return apiFetch<KnowledgeSearchResponse>(`/api/knowledge/search?${qs.toString()}`);
}

export interface ListKnowledgeContentsParams {
  caseId: string;
  lane: string;
  limit?: number;
  offset?: number;
}

export async function listKnowledgeContents(params: ListKnowledgeContentsParams) {
  const qs = new URLSearchParams();
  qs.set("case_id", params.caseId);
  qs.set("lane", params.lane);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<KnowledgeContentsResponse>(`/api/knowledge/contents${suffix}`);
}

export async function getKnowledgeContent(artifactId: string, caseId: string) {
  const qs = new URLSearchParams({ case_id: caseId });
  return apiFetch<KnowledgeItemDetail>(
    `/api/knowledge/contents/${encodeURIComponent(artifactId)}?${qs.toString()}`,
  );
}

// ---------------------------------------------------------------------------
// Matter workspace (framework-neutral spine API, via Workbench proxy)
// ---------------------------------------------------------------------------

export async function listMatters(limit = 50, offset = 0, mode?: MatterMode) {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (mode) qs.set("mode", mode);
  return apiFetch<MatterListResponse>(`/api/matters?${qs.toString()}`);
}

export async function createMatter(payload: {
  title: string;
  description?: string;
  partition_key?: string;
  created_by?: "owner";
}, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<Matter>(`/api/matters?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function getMatter(matterId: string, mode?: MatterMode) {
  const query = new URLSearchParams();
  if (mode) query.set("mode", mode);
  const suffix = query.size ? `?${query.toString()}` : "";
  return apiFetch<MatterDetail>(`/api/matters/${encodeURIComponent(matterId)}${suffix}`);
}

export interface CaseManagementCapabilities {
  registry_available: boolean;
  advanced_evidence_available: boolean;
  advanced_evidence_reason: string;
}

export async function getCaseManagementCapabilities() {
  return apiFetch<CaseManagementCapabilities>("/api/case-management/capabilities");
}

// ---------------------------------------------------------------------------
// proffer workflow (formerly Universal Import Workflow) — production acquisition and decision boundary
// ---------------------------------------------------------------------------

export async function uploadProfferSource(file: File, mode: MatterMode) {
  // Existing authenticated server ingress writes Nexus; only the server authors its acquisition reference.
  const staged = await uploadFile(file);
  return acquireStagedProfferSource(staged.id, mode);
}

export function acquireStagedProfferSource(stagedId: string, mode: MatterMode) {
  return apiFetch<ProfferUploadResponse>(`/api/proffer/staged/${encodeURIComponent(stagedId)}/acquisition?mode=${mode}`, {
    method: "POST",
  }).then((result) => {
    if (result.matter_mode !== mode) throw new ApiError("Staged acquisition did not confirm TEST/REAL mode", 502);
    return result;
  });
}

export function listProfferSources(params: {
  mode: MatterMode;
  rootId?: string;
  fileTypes?: string[];
  prefix?: string;
  continuationToken?: string;
  filter?: string;
  pageSize?: number;
  signal?: AbortSignal;
}) {
  const query = new URLSearchParams();
  query.set("mode", params.mode);
  query.set("filter_scope", "root");
  if (params.rootId) query.set("root_id", params.rootId);
  for (const fileType of params.fileTypes ?? []) query.append("file_type", fileType);
  if (params.prefix) query.set("prefix", params.prefix);
  if (params.continuationToken) query.set("continuation_token", params.continuationToken);
  if (params.filter) query.set("filter", params.filter);
  if (params.pageSize) query.set("page_size", String(params.pageSize));
  const suffix = query.size ? `?${query.toString()}` : "";
  return apiFetch<ProfferSourceBrowserResponse>(`/api/proffer/sources${suffix}`, { signal: params.signal }).then((response) => {
    if (response.matter_mode !== params.mode) throw new ApiError("The source browser did not confirm the active TEST/REAL mode", 502);
    if (params.rootId && response.active_root_id !== params.rootId) throw new ApiError("The source browser returned a different source location", 502);
    if ((params.filter?.trim() ?? "") && (response.filter !== params.filter?.trim() || !response.filter_applied || response.filter_scope !== "root")) {
      throw new ApiError("The source browser did not confirm a complete backing-source search", 502);
    }
    return response;
  });
}

export function inspectProfferSource(source: ProfferSourceObject, mode: MatterMode, rootId: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode, root_id: rootId });
  return apiFetch<ProfferSourceInspection>(`/api/proffer/source-inspection?${query.toString()}`, {
    method: "POST",
    signal,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      key: source.key,
      source_ref: source.source_ref,
      root_id: rootId,
      expected_byte_length: source.byte_length,
      expected_etag: source.etag ?? null,
    }),
  }).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The source inspection did not confirm the active TEST/REAL mode", 502);
    if (response.active_root_id !== rootId || response.source_ref !== source.source_ref) throw new ApiError("The source inspection did not confirm the selected R2 source location", 502);
    return response;
  });
}

export function createProfferSourceContext(payload: {
  request_id: string;
  matter_id: string;
  court_case_id: string;
  source_ref: string;
  observed_source: ProfferObservedSource;
  supersedes_ref?: string | null;
  assertions: ProfferHumanSourceAssertions;
  change_reason: string;
  matter_mode: MatterMode;
}) {
  const query = new URLSearchParams({ mode: payload.matter_mode });
  return apiFetch<ProfferSourceContextReceipt>(`/api/proffer/source-contexts?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  }).then((response) => {
    if (response.matter_mode !== payload.matter_mode) throw new ApiError("The source-context receipt did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

/**
 * One run's registration facts and newest operator context revision (Review Actions panel).
 * Byline: Claude Code · Opus 5.5 · 2026-09-25.
 */
export function getProfferRunSourceContext(previewHandle: string, mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferRunSourceContext>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/source-context?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.preview_handle !== previewHandle || response.matter_mode !== mode) {
      throw new ApiError("The run source context crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function startProffer(payload: ProfferStartRequest) {
  const query = new URLSearchParams({ mode: payload.matter_mode });
  return apiFetch<ProfferStartResponse>(`/api/proffer/start?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  }).then((response) => {
    if (response.matter_mode !== payload.matter_mode) throw new ApiError("The started preview did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

export function decideProfferHandler(
  previewHandle: string,
  mode: MatterMode,
  payload: ProfferHandlerSelectionDecisionRequest,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferHandlerSelectionDecisionResponse>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/handler-selection?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  ).then((response) => {
    if (response.matter_mode !== mode || response.preview_handle !== previewHandle) {
      throw new ApiError("The handler-selection decision did not confirm this mode and preview", 502);
    }
    return response;
  });
}

export function listProfferOperations(params: {
  status?: ProfferOperationLifecycle;
  cursor?: string;
  limit?: number;
} = {}, signal?: AbortSignal) {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit) query.set("limit", String(params.limit));
  const suffix = query.size ? `?${query.toString()}` : "";
  return apiFetch<ProfferOperationListResponse>(`/api/proffer/operations${suffix}`, { signal });
}

export function getProfferOperation(previewHandle: string, signal?: AbortSignal) {
  return apiFetch<ProfferOperationDetail>(
    `/api/proffer/operations/${encodeURIComponent(previewHandle)}`,
    { signal },
  );
}

export function getProfferPreview(previewHandle: string, mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferPreviewResponse>(`/api/proffer/previews/${encodeURIComponent(previewHandle)}?${query.toString()}`, { signal }).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The preview did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

export function getProfferOperatorSnapshot(previewHandle: string, mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferOperatorSnapshot>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/operator?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.preview_handle !== previewHandle || response.matter_mode !== mode) {
      throw new ApiError("The operator snapshot crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function decideProfferRepair(previewHandle: string, mode: MatterMode, payload: ProfferRepairDecisionRequest) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferRepairDecisionResponse>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/repair-decision?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  ).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The repair decision did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

export function decideProffer(
  previewHandle: string,
  mode: MatterMode,
  payload: { approved: boolean; reason: string },
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferDecisionResponse>(`/api/proffer/previews/${encodeURIComponent(previewHandle)}/decision?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  }).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The decision response did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

/**
 * Cancel one run through the engine's Temporal cancellation (D05-C06). The actor comes from the
 * request identity, never the browser; the engine keeps who and why in the run's own history.
 * Byline: Claude Code · Opus 5.5 · 2026-09-28.
 */
export function cancelProfferRun(previewHandle: string, mode: MatterMode, reason: string) {
  const query = new URLSearchParams({ mode });
  return apiFetch<{ preview_handle: string; status: "cancel_requested"; matter_mode: MatterMode }>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/cancel?${query.toString()}`,
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ reason }) },
  ).then((response) => {
    if (response.preview_handle !== previewHandle || response.matter_mode !== mode) {
      throw new ApiError("The cancel response crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

/** Server-side filters for `getProfferPreviewMessages`. A cursor is bound to the exact
 * filter that minted it, so any change here must restart paging from the first page. */
export interface ProfferPreviewMessageFilters {
  /** Case-insensitive body substring match, max 200 chars. */
  q?: string;
  hasAttachments?: boolean;
  /** Participant id to restrict results to a single sender. */
  sender?: string;
  /** RFC3339 lower bound (inclusive), by `sent_at`. */
  from?: string;
  /** RFC3339 upper bound (inclusive), by `sent_at`. */
  to?: string;
}

export function getProfferPreviewMessages(
  previewHandle: string,
  mode: MatterMode,
  cursor?: string,
  limit = 100,
  signal?: AbortSignal,
  filters?: ProfferPreviewMessageFilters,
) {
  const query = new URLSearchParams({ limit: String(limit), mode });
  if (cursor) query.set("cursor", cursor);
  if (filters?.q) query.set("q", filters.q.slice(0, 200));
  if (filters?.hasAttachments) query.set("has_attachments", "true");
  if (filters?.sender) query.set("sender", filters.sender);
  if (filters?.from) query.set("from", filters.from);
  if (filters?.to) query.set("to", filters.to);
  return apiFetch<ProfferPreviewMessagesResponse>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/messages?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The preview messages did not confirm the active TEST/REAL mode", 502);
    return response;
  });
}

const SHA256_HEX = /^[0-9a-f]{64}$/i;

/**
 * Builds the same-origin URL for one attachment's post-ingest derived media,
 * streamed from `<key>.derived/media/<sha256><ext>` in B2 by the engine.
 * Contract relayed 2026-09-20 (Control surfaces decisions): `GET
 * /api/proffer/previews/{handle}/media/{sha256}`, content-type resolved
 * server-side from the manifest/extension. Returns `null` for a malformed
 * sha256 so callers never point a media element at an unvalidated path —
 * this endpoint is for POST-ingest derived media only; it has no bearing on
 * unprocessed source preview (see attachment-preview.tsx for that gap).
 */
export function getProfferPreviewMediaUrl(previewHandle: string, mode: MatterMode, sha256: string): string | null {
  if (!SHA256_HEX.test(sha256)) return null;
  const query = new URLSearchParams({ mode });
  return `${API_BASE}/api/proffer/previews/${encodeURIComponent(previewHandle)}/media/${sha256.toLowerCase()}?${query.toString()}`;
}

export function getProfferPreviewContent(
  previewHandle: string,
  mode: MatterMode,
  recordCursor?: string,
  chunkCursor?: string,
  limit = 100,
  signal?: AbortSignal,
) {
  const query = new URLSearchParams({ limit: String(limit), mode });
  if (recordCursor) query.set("record_cursor", recordCursor);
  if (chunkCursor) query.set("chunk_cursor", chunkCursor);
  return apiFetch<ProfferContentResponse>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/content?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.preview_handle !== previewHandle || response.matter_mode !== mode) {
      throw new ApiError("The preview content crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function listProfferProposalResources(
  mode: MatterMode,
  params: { status?: ProfferOperationLifecycle; cursor?: string; limit?: number } = {},
  signal?: AbortSignal,
) {
  const query = new URLSearchParams({ mode });
  if (params.status) query.set("status", params.status);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit) query.set("limit", String(params.limit));
  return apiFetch<ProfferProposalResourceCatalog>(`/api/proffer/proposal-resources?${query.toString()}`, { signal }).then((response) => {
    if (response.scope !== "context_review_resources" || response.matter_mode !== mode) {
      throw new ApiError("The Review catalog crossed its Context scope or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function listProfferPotentialPromotionFlags(
  previewHandle: string,
  mode: MatterMode,
  signal?: AbortSignal,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferPotentialPromotionFlagList>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/potential-promotion-flags?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.flags.some((flag) => flag.preview_handle !== previewHandle || flag.matter_mode !== mode)) {
      throw new ApiError("A potential-promotion flag crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function createProfferPotentialPromotionFlag(
  previewHandle: string,
  mode: MatterMode,
  payload: ProfferPotentialPromotionFlagRequest,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferPotentialPromotionFlag>(
    `/api/proffer/previews/${encodeURIComponent(previewHandle)}/potential-promotion-flags?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  ).then((response) => {
    if (response.preview_handle !== previewHandle || response.matter_mode !== mode) {
      throw new ApiError("The potential-promotion flag crossed its preview or TEST/REAL boundary", 502);
    }
    return response;
  });
}

export function createProfferPreviewEventSource(previewHandle: string, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return new EventSource(
    `${API_BASE}/api/proffer/previews/${encodeURIComponent(previewHandle)}/events?${query.toString()}`,
    { withCredentials: true },
  );
}

export async function createCourtCase(
  matterId: string,
  payload: {
    caption: string;
    court_name?: string;
    docket_number?: string;
    jurisdiction?: string;
    case_type?: string;
    status?: CourtCaseStatus;
    filed_on?: string;
    closed_on?: string;
    is_primary?: boolean;
    created_by?: "owner";
  },
  mode: MatterMode,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<CourtCase>(`/api/matters/${encodeURIComponent(matterId)}/court-cases?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function resolveKnowledgeSource(matterId: string, source: KnowledgeSourceRef, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<KnowledgeSourceResolution>(
    `/api/matters/${encodeURIComponent(matterId)}/knowledge/resolve?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(source),
    },
  );
}

export async function createEvidenceItem(
  matterId: string,
  payload: {
    court_case_id: string;
    source: KnowledgeSourceRef & { normalized_record_id: string };
    title: string;
    description?: string;
    quote?: string;
    evidence_type?: string;
    created_by?: "owner";
  },
  mode: MatterMode,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<EvidencePromotionResult>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export async function listEvidenceItems(matterId: string, mode: MatterMode, limit = 50, offset = 0) {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset), mode });
  return apiFetch<EvidenceItemListResponse>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items?${qs.toString()}`,
  );
}

export async function getEvidenceDetail(matterId: string, evidenceItemId: string, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<EvidenceDetail>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}?${query.toString()}`,
  );
}

export async function getEvidenceSourceContent(matterId: string, evidenceItemId: string, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<EvidenceSourceContent>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}/source-content?${query.toString()}`,
  );
}

export async function getEvidenceConversationContext(matterId: string, evidenceItemId: string, mode: MatterMode) {
  const query = new URLSearchParams({ before: "25", after: "25", mode });
  return apiFetch<EvidenceConversationContext>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}/conversation-context?${query.toString()}`,
  );
}

export async function getCourtReadiness(matterId: string, evidenceItemId: string, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<CourtReadiness>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}/court-readiness?${query.toString()}`,
  );
}

export async function reviewEvidenceItem(
  matterId: string,
  evidenceItemId: string,
  payload: {
    decision: EvidenceReviewDecision;
    rationale: string;
    reviewer?: "owner";
  },
  mode: MatterMode,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<EvidenceReviewResult>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}/reviews?${query.toString()}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export async function listEvidenceReviews(matterId: string, evidenceItemId: string, mode: MatterMode) {
  const query = new URLSearchParams({ mode });
  return apiFetch<EvidenceReviewListResponse>(
    `/api/matters/${encodeURIComponent(matterId)}/evidence-items/${encodeURIComponent(evidenceItemId)}/reviews?${query.toString()}`,
  );
}

// ---------------------------------------------------------------------------
// Tool Explorer (MCP servers)
// ---------------------------------------------------------------------------

export async function listTools() {
  return apiFetch<ToolServerGroup[]>("/api/tools");
}

/**
 * Capability handshake for the only browser-supported atomic-tool path.
 * A missing route is intentionally reported by callers as unavailable; the
 * browser must never fall back to a legacy direct-call route for execution.
 */
export async function getMonitoredActionCapability() {
  return apiFetch<MonitoredActionCapability>("/api/monitored-actions/capabilities");
}

export async function startAtomicToolAction(request: StartAtomicToolActionRequest) {
  return apiFetch<MonitoredActionRun>("/api/monitored-actions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}

export async function getMonitoredAction(actionId: string) {
  return apiFetch<MonitoredActionRun>(`/api/monitored-actions/${encodeURIComponent(actionId)}`);
}

export async function cancelMonitoredAction(actionId: string) {
  return apiFetch<MonitoredActionRun>(`/api/monitored-actions/${encodeURIComponent(actionId)}/cancel`, {
    method: "POST",
  });
}

/** Upload a file with progress reporting (non-streaming — one JSON response). */
export function uploadFile(
  file: File,
  onProgress?: (percent: number) => void,
): Promise<UploadResponse> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const formData = new FormData();
    formData.append("file", file);

    xhr.upload.addEventListener("progress", (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100));
      }
    });

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText));
        } catch {
          reject(new ApiError("Malformed response from server", xhr.status));
        }
      } else {
        try {
          const body = JSON.parse(xhr.responseText);
          reject(new ApiError(body.detail || `Upload failed: ${xhr.status}`, xhr.status));
        } catch {
          reject(new ApiError(`Upload failed: ${xhr.status}`, xhr.status));
        }
      }
    });

    xhr.addEventListener("error", () =>
      reject(new ApiError("Network error — check your connection", 0)),
    );
    xhr.addEventListener("abort", () =>
      reject(new ApiError("Upload aborted", 0)),
    );

    xhr.open("POST", `${API_BASE}/api/upload`);
    xhr.send(formData);
  });
}

// Byline: Codex · 2026-09-20. Read-only pre-ingest catalog discovery.
export function getDiscoveryCapabilities(signal?: AbortSignal) {
  return apiFetch<import("./discovery-types").DiscoveryCapabilities>("/api/intake/discovery/capabilities", { signal });
}
export function listDiscoveryTree(parent = "", cursor?: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ parent, limit: "100" });
  if (cursor) query.set("cursor", cursor);
  return apiFetch<import("./discovery-types").DiscoveryPage>(`/api/intake/discovery/tree?${query}`, { signal });
}
export function searchDiscovery(q: string, parent = "", mode = "filename_substring", cursor?: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ q, parent, mode, limit: "100" });
  if (cursor) query.set("cursor", cursor);
  return apiFetch<import("./discovery-types").DiscoveryPage>(`/api/intake/discovery/search?${query}`, { signal });
}

export function listDiscoveryUnits(unitType: string, afterId = -1, signal?: AbortSignal) {
  const query = new URLSearchParams({ unit_type: unitType, after_id: String(afterId), limit: "100" });
  return apiFetch<import("./discovery-types").DiscoveryUnits>(`/api/intake/discovery/units?${query}`, { signal });
}
export function getDiscoveryUnitMembers(unitId: number, signal?: AbortSignal, cursor?: string) {
  const query = new URLSearchParams({ limit: "100" });
  if (cursor) query.set("cursor", cursor);
  return apiFetch<import("./discovery-types").DiscoveryUnitMembers>(`/api/intake/discovery/units/${unitId}/members?${query}`, { signal });
}

// Byline: Claude Code · Opus 5 · 2026-09-22. Sources screen: folder batches,
// decode-state marks, catalog units and provenance, hand-marked units.

/** Start one durable batch over one folder (engine POST /reference-import/start-batch). */
export function startProfferBatch(payload: ProfferBatchStartRequest) {
  const query = new URLSearchParams({ mode: payload.matter_mode });
  return apiFetch<ProfferBatchStartResponse>(`/api/proffer/start-batch?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  }).then((response) => {
    if (response.matter_mode !== payload.matter_mode || response.batch_id !== payload.batch_id) {
      throw new ApiError("The started batch did not confirm this batch and Test/Live mode", 502);
    }
    return response;
  });
}

/** Per-item status and counts for one running batch. */
export function getProfferBatch(batchId: string, mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferBatchStatus>(`/api/proffer/batches/${encodeURIComponent(batchId)}?${query.toString()}`, { signal }).then((response) => {
    if (response.matter_mode !== mode || response.batch_id !== batchId) {
      throw new ApiError("The batch status did not confirm this batch and Test/Live mode", 502);
    }
    return response;
  });
}

// ---------------------------------------------------------------------------
// Repair workflow builder (Review Actions). BFF /api/proffer/repair/* passes the engine's
// /reference-import/repair/* routes through; every call names the TEST/REAL mode.
// Byline: Claude Code · Opus 5.5 · 2026-09-26.
// ---------------------------------------------------------------------------

/** The engine refused to start a plan (422); `checks` says which rules failed and why. */
export class RepairRunRefusedError extends ApiError {
  constructor(
    message: string,
    public readonly checks: ProfferRepairCheck[],
  ) {
    super(message, 422);
    this.name = "RepairRunRefusedError";
  }
}

/** Every repair-capable Activity, for the step picker. */
export function listProfferRepairTools(mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferRepairToolsResponse>(`/api/proffer/repair/tools?${query.toString()}`, { signal }).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The repair tool list did not confirm the active Test/Live mode", 502);
    return response;
  });
}

/** The engine's candidate plans for one Review run's source (its own repair report decides). */
export function proposeProfferRepairs(
  mode: MatterMode,
  body: { source_ref: string; preview_handle: string },
  signal?: AbortSignal,
) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferRepairProposeResponse>(`/api/proffer/repair/propose?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  }).then((response) => {
    if (response.matter_mode !== mode) throw new ApiError("The repair proposals did not confirm the active Test/Live mode", 502);
    return response;
  });
}

/** Every named check against the plan; the engine fails closed. */
export function validateProfferRepairPlan(plan: ProfferRepairPlan, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode: plan.matter_mode });
  return apiFetch<ProfferRepairValidateResponse>(`/api/proffer/repair/validate?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(plan),
    signal,
  }).then((response) => {
    if (response.matter_mode !== plan.matter_mode) throw new ApiError("The plan validation did not confirm the active Test/Live mode", 502);
    return response;
  });
}

/** Starts the plan on Temporal. A refusal throws RepairRunRefusedError with the engine's checks. */
export function runProfferRepairPlan(plan: ProfferRepairPlan) {
  const query = new URLSearchParams({ mode: plan.matter_mode });
  return apiFetch<ProfferRepairRunResponse>(`/api/proffer/repair/run?${query.toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(plan),
  }).then(
    (response) => {
      if (response.matter_mode !== plan.matter_mode) throw new ApiError("The repair run did not confirm the active Test/Live mode", 502);
      return response;
    },
    (error: unknown) => {
      const body = error instanceof ApiError ? error.body : undefined;
      const checks = body && typeof body === "object" && Array.isArray((body as { checks?: unknown }).checks)
        ? (body as { checks: ProfferRepairCheck[] }).checks
        : null;
      if (error instanceof ApiError && error.status === 422 && checks) throw new RepairRunRefusedError(error.message, checks);
      throw error;
    },
  );
}

/** One repair run's per-step progress, receipts and re-entry. */
export function getProfferRepairRun(workflowId: string, mode: MatterMode, signal?: AbortSignal) {
  const query = new URLSearchParams({ mode });
  return apiFetch<ProfferRepairRunStatus>(
    `/api/proffer/repair/runs/${encodeURIComponent(workflowId)}?${query.toString()}`,
    { signal },
  ).then((response) => {
    if (response.workflow_id !== workflowId || response.matter_mode !== mode) {
      throw new ApiError("The repair run status crossed its run or Test/Live boundary", 502);
    }
    return response;
  });
}

/** Which of these sources already have a decode manifest. Never inferred from a name. */
function chunks<T>(items: readonly T[], size: number): T[][] {
  const out: T[][] = [];
  for (let start = 0; start < items.length; start += size) out.push(items.slice(start, start + size));
  return out;
}

// Server caps per request (api/app/types/proffer_decoded_exists.py, runtime/intake_discovery.py).
// Sources loads 200 rows a page, so from the second page on the lists are split to stay under them.
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (past 200 files these calls answered 422).
const DECODED_EXISTS_MAX = 200;
const UNIT_LOOKUP_MAX_ROOTS = 200;
const UNIT_LOOKUP_MAX_KEYS = 400;

export async function getDecodedExists(sourceRefs: string[], signal?: AbortSignal): Promise<DecodedExistsResponse> {
  const pages = await Promise.all(
    chunks(sourceRefs, DECODED_EXISTS_MAX).map((batch) =>
      apiFetch<DecodedExistsResponse>("/api/proffer/decoded/exists", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source_refs: batch }),
        signal,
      }),
    ),
  );
  return { items: pages.flatMap((page) => page.items) };
}

/** Catalog units for the listed rows: folders that are units, keys that are members. */
export async function lookupCatalogUnits(roots: string[], keys: string[], signal?: AbortSignal): Promise<CatalogUnitLookup> {
  const rootBatches = chunks(roots, UNIT_LOOKUP_MAX_ROOTS);
  const keyBatches = chunks(keys, UNIT_LOOKUP_MAX_KEYS);
  const requests = Array.from({ length: Math.max(rootBatches.length, keyBatches.length, 1) }, (_, index) => ({
    roots: rootBatches[index] ?? [],
    keys: keyBatches[index] ?? [],
  }));
  const pages = await Promise.all(
    requests.map((body) =>
      apiFetch<CatalogUnitLookup>("/api/intake/discovery/unit-lookup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        signal,
      }),
    ),
  );
  const units = new Map(pages.flatMap((page) => page.units).map((unit) => [unit.unit_id, unit]));
  return {
    units: [...units.values()],
    members: pages.flatMap((page) => page.members),
    backend: pages[0].backend,
    source_links_verified: pages.every((page) => page.source_links_verified),
  };
}

/** Catalog provenance for one vault object. */
export function getCatalogProvenance(vaultKey: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ vault_key: vaultKey });
  return apiFetch<CatalogProvenance>(`/api/intake/discovery/catalog/by-vault-key?${query.toString()}`, { signal });
}

export function listSourceUnitMarks(signal?: AbortSignal) {
  return apiFetch<SourceUnitMarkList>("/api/sources/unit-marks", { signal });
}

export function recordSourceUnitMark(payload: {
  unit_root: string;
  unit_type: SourceUnitKind;
  label?: string;
  confirm?: boolean;
}) {
  return apiFetch<SourceUnitMark>("/api/sources/unit-marks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

/** What this folder looks like, from the names the browser just listed. */
export function proposeSourceUnit(unitRoot: string, names: string[], signal?: AbortSignal) {
  return apiFetch<SourceUnitProposal>("/api/sources/unit-proposal", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ unit_root: unitRoot, names }),
    signal,
  });
}

/** Relationship neighbors for one indexed object (Intake's Surreal file graph). */
export function getDiscoveryNeighbors(table: string, key: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ limit: "50" });
  return apiFetch<{ [key: string]: unknown }>(
    `/api/intake/discovery/neighbors/${encodeURIComponent(table)}/${encodeURIComponent(key)}?${query.toString()}`,
    { signal },
  );
}

/** Which recorded catalog units the files under ONE vault folder belong to. */
export function getUnitsUnderPrefix(prefix: string, signal?: AbortSignal) {
  const query = new URLSearchParams({ prefix });
  return apiFetch<CatalogUnitsUnderPrefix>(`/api/intake/discovery/units/under-prefix?${query.toString()}`, { signal });
}
