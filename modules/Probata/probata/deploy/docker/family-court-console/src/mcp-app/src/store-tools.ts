// Byline: Claude Code Â· Fable 5.1 Â· 2026-09-07
// Byline: Claude Code Â· Sonnet 5 Â· 2026-09-07 â€” 7 new tools (case_status,
// case_docket, case_memo, case_evidence_log, case_eval, case_reference,
// case_source) for the owner's 13:09-13:16 orders; case_export/case_import/
// case_timeline extended in place (additive â€” all nine original tools keep
// their original request shapes working).
//
// MCP tool registrations for the hosted shared SurrealDB case store (src/store.ts).
// Deliberately kept out of src/server.ts (another agent is editing that file
// concurrently) â€” call `registerStoreTools(server)` from server.ts to wire
// these seventeen tools in. See README-store.md for the exact snippet.
// Byline: Claude Code Â· Opus 5.5 Â· 2026-09-27 â€” case_record (shared legal-record contract).
//
// Every handler resolves the store first and, if the native module failed to
// load or the database failed to open, returns `{ available: false, reason }`
// instead of throwing â€” per the owner's graceful-degradation requirement.
// Handler validation errors (bad table name, child name rejection, refused
// write, etc.) are allowed to throw; the McpServer SDK converts a thrown
// error into a normal tool error result.

import { z } from "zod";
import { libraryPropose, libraryPublish } from "./case-library.js";
import { queueLibraryValidation } from "./library-validation-dispatch.js";
import {
  DATA_TABLES,
  EDGE_TABLES,
  SEARCHABLE_TABLES,
  type StoreOk,
  caseDocket,
  caseEvalList,
  caseEvalPut,
  caseEvidenceLogAppend,
  caseEvidenceLogList,
  caseExport,
  caseExportPlatform,
  caseFactorMap,
  caseGraph,
  caseImport,
  caseMemoLatestByKind,
  caseMemoList,
  caseMemoPut,
  casePut,
  caseQuery,
  caseRecord,
  caseReferenceList,
  caseReferenceMatch,
  caseSearch,
  caseSourceOf,
  caseStatusGet,
  caseStatusSet,
  caseSummary,
  caseTimeline,
  getStore,
} from "./store.js";

// Minimal structural type for the pieces of McpServer this file needs â€” kept
// local (rather than importing McpServer's type) so this file has no
// compile-time coupling to server.ts's exact SDK import path.
interface RegisterToolServer {
  registerTool(
    name: string,
    config: {
      title?: string;
      description: string;
      inputSchema?: Record<string, z.ZodTypeAny>;
      annotations?: Record<string, unknown>;
      _meta?: Record<string, unknown>;
    },
    handler: (args: Record<string, unknown>) => Promise<{ structuredContent: Record<string, unknown>; content: Array<{ type: "text"; text: string }> }>,
  ): unknown;
}

const readOnly = { readOnlyHint: true, destructiveHint: false, idempotentHint: true, openWorldHint: false } as const;
const readWrite = { readOnlyHint: false, destructiveHint: false, idempotentHint: true, openWorldHint: false } as const;

function toolResult(data: Record<string, unknown>) {
  return { structuredContent: data, content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

function unavailable(reason: string) {
  return toolResult({ available: false, reason });
}

const refSchema = z.union([z.string().min(3), z.object({ table: z.string().min(1), id: z.string().min(1) })]);

const dataTableSchema = z.enum(DATA_TABLES as unknown as [string, ...string[]]);
const edgeTableSchema = z.enum(EDGE_TABLES as unknown as [string, ...string[]]);
const searchableTableSchema = z.enum(SEARCHABLE_TABLES as unknown as [string, ...string[]]);

export function registerStoreTools(server: RegisterToolServer): void {
  // Byline: Codex, GPT-6, 2026-10-04. One shared mutation surface for all library clients.
  server.registerTool("library_propose", {
    title: "Save a shared library edit for citation validation",
    description: "Retain the full personalized proposed record in the shared case store. Inputs: target id, expected record version, partial patch, claim citations and rationale. Outputs: proposal id/hash/status. Published content is unchanged until validation and publication. Use for library edits from every surface.",
    inputSchema: {
      id: z.string().min(3).max(220), expected_version: z.string().max(80),
      patch: z.record(z.string(), z.unknown()), rationale: z.string().min(1).max(4000),
      citations: z.array(z.object({ source_id: z.string(), source_version: z.string(), pinpoint: z.string().min(1).max(512), claim: z.string().min(1).max(8000) })).min(1).max(64),
    }, annotations: readWrite,
  }, async args => {
    const store = await getStore();
    if (!store.available) return unavailable(store.reason);
    const saved = await libraryPropose(store, args as never);
    const dispatch = await queueLibraryValidation(store, String(saved.proposal_id));
    return toolResult({ ...saved, dispatch });
  });
  server.registerTool("library_validate", {
    title: "Retry validation of a saved shared library edit",
    description: "Start or join tracked validation for an already saved proposal. Input: proposal_id. Output: queued workflow or explicit retryable failure. Effects: starter call and dispatch metadata only. Use after a queue failure; this tool cannot publish or set validation flags.",
    inputSchema: { proposal_id: z.string().min(1).max(100) }, annotations: readWrite,
  }, async args => {
    const store = await getStore();
    if (!store.available) return unavailable(store.reason);
    return toolResult({ proposal_id: args.proposal_id, dispatch: await queueLibraryValidation(store, String(args.proposal_id)) });
  });
  server.registerTool("library_publish", {
    title: "Publish a validated shared library revision",
    description: "Apply an exact retained proposal only after its server-side validation receipt and source versions pass. Input: proposal_id. Outputs: shared record version and revision identity. Effects: atomic version check, preserved previous personal content and publication. Use after validation; conflicts remain visible.",
    inputSchema: { proposal_id: z.string().min(1).max(100) }, annotations: readWrite,
  }, async args => {
    const store = await getStore();
    if (!store.available) return unavailable(store.reason);
    return toolResult(await libraryPublish(store, String(args.proposal_id)));
  });
  server.registerTool(
    "case_put",
    {
      title: "Put a case-store record (+ optional relations)",
      description:
        "Save a version-checked record in the hosted shared SurrealDB case store (person, child, order, hearing, deadline, event, message, " +
        "exhibit, factor, note, court) and optionally RELATE it to other records in the same call. Full personal " +
        "case fields are preserved in the private shared store. Pass expected_version from case_record (or absent for a new explicit id) to refuse stale edits and retain complete revisions. Library source/reference edits use library_propose. " +
        "Returns unavailable with a reason if the shared store cannot connect.",
      inputSchema: {
        table: dataTableSchema,
        id: z.string().min(1).max(200).optional(),
        expected_version: z.string().regex(/^(?:absent|sha256:[a-f0-9]{64})$/).optional(),
        data: z.record(z.string(), z.unknown()),
        relations: z
          .array(z.object({ edge: edgeTableSchema, from: refSchema, to: refSchema, data: z.record(z.string(), z.unknown()).optional() }))
          .max(50)
          .optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await casePut(store as StoreOk, args as never);
      return toolResult({ available: true, ...result });
    },
  );

  server.registerTool(
    "case_query",
    {
      title: "Run a parameterised SurrealQL query against the case store",
      description:
        "Executes SELECT-only inspection against the shared case store. Mutation and external/custom function calls are refused; use governed tools for writes. " +
        "Result rows are capped at 200 per statement (a `truncated` flag notes when that cap hit). " +
        "Returns { available: false, reason } if the native surrealdb module failed to load.",
      inputSchema: {
        surql: z.string().min(1).max(10000),
        params: z.record(z.string(), z.unknown()).optional(),
        write: z.boolean().optional(),
      },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await caseQuery(store as StoreOk, args as never);
      return toolResult({ available: true, ...result });
    },
  );

  server.registerTool(
    "case_search",
    {
      title: "Search events, messages, notes, and exhibits",
      description:
        "Full-text (BM25) and, when NVIDIA_API_KEY/NIM_API_KEY is configured, vector (HNSW/cosine) search over " +
        "event.description, message.body, note.text, and exhibit.label, merged by reciprocal-rank fusion in hybrid mode. " +
        "Without embeddings configured, hybrid/vector modes degrade to text-only search and set degraded: true. Returns " +
        "id, table, snippet, score, occurred_at, known_at per hit.",
      inputSchema: {
        query: z.string().min(1).max(2000),
        tables: z.array(searchableTableSchema).max(4).optional(),
        k: z.number().int().min(1).max(100).optional(),
        mode: z.enum(["text", "vector", "hybrid"]).optional(),
      },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await caseSearch(store as StoreOk, args as never);
      return toolResult({ available: true, ...result });
    },
  );

  server.registerTool(
    "case_graph",
    {
      title: "Show a record's graph neighbourhood",
      description:
        "Returns the 1- or 2-hop neighbourhood of a record (id as \"table:id\", e.g. \"event:e2\") across the evidences, " +
        "supports_factor, contradicts_factor, sent_by, sent_to, and filed_in edges (or a subset via `edges`), with each " +
        "neighbor's edge, direction, id, and a short text summary.",
      inputSchema: {
        id: refSchema,
        depth: z.union([z.literal(1), z.literal(2)]).optional(),
        edges: z.array(edgeTableSchema).optional(),
      },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await caseGraph(store as StoreOk, args as never);
      return toolResult({ available: true, ...result });
    },
  );

  server.registerTool(
    "case_factor_map",
    {
      title: "Map events and exhibits onto the 12 MCL 722.23 factors",
      description:
        "For each of the 12 MCL 722.23 best-interest factors (a-l), returns the supporting/contradicting record counts " +
        "and the top 5 supporting and top 5 contradicting events/exhibits (by RELATE weight), each with a short summary.",
      inputSchema: {},
      annotations: readOnly,
      _meta: {},
    },
    async () => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const factors = await caseFactorMap(store as StoreOk);
      return toolResult({ available: true, factors });
    },
  );

  server.registerTool(
    "case_timeline",
    {
      title: "Build a case timeline â€” court lane, master lane, or both",
      description:
        "Returns timeline entries tagged by `lane`: \"court\" = the real court-event timeline (court_event âˆª hearing âˆª " +
        "deadline âˆª order); \"master\" = the extracted-from-the-corpora timeline (event âˆª message âˆª exhibit), each " +
        "carrying its own known_at. `mode` selects which lane(s) â€” default \"merged\" (both). `known_by` filters the " +
        "master lane to only what was known by that date (the two-clock discipline: occurred_at = when it happened, " +
        "known_at = when the owner learned it). `upcoming` filters the court lane to future (true) or past (false) " +
        "entries. These two timelines are kept deliberately separate; a merged view tags which lane each entry came from.",
      inputSchema: {
        mode: z.enum(["court", "master", "merged"]).optional(),
        upcoming: z.boolean().optional(),
        from: z.string().optional(),
        to: z.string().optional(),
        known_by: z.string().optional(),
        tables: z.array(z.enum(["event", "message", "exhibit"])).optional(),
      },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const entries = await caseTimeline(store as StoreOk, args as never);
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_export",
    {
      title: "Export the case store â€” a JSON snapshot, or a platform (probata) NDJSON bundle",
      description:
        "format: \"snapshot\" (default) writes every table and edge to a single JSON file (default " +
        "~/.config/family-court-toolkit/exports/<timestamp>.json) for backup/transfer. format: \"platform\" writes one " +
        "NDJSON file per table plus edges.ndjson and manifest.json (schema fct-platform-bundle/v1) to a new " +
        "~/.config/family-court-toolkit/exports/platform-<timestamp>/ directory, shaped for the probata evidence " +
        "platform's import; every row keeps its provenance and full private case context. " +
        "Returns the written path/dir and per-table/edge record counts.",
      inputSchema: { path: z.string().min(1).optional(), format: z.enum(["snapshot", "platform"]).optional() },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const { path, format } = args as { path?: string; format?: "snapshot" | "platform" };
      if (format === "platform") {
        const result = await caseExportPlatform(store as StoreOk, path);
        return toolResult({ available: true, format: "platform", ...result });
      }
      const result = await caseExport(store as StoreOk, path);
      return toolResult({ available: true, format: "snapshot", ...result });
    },
  );

  server.registerTool(
    "case_import",
    {
      title: "Import a case-extract/v1 batch, a case-store snapshot, or a Vincent-style case schema",
      description:
        "Loads `path` into the case store, auto-detecting the shape: a directory (or single file) of case-extract/v1 " +
        "envelopes (Case Bible intake â€” schema \"case-extract/v1\"), a case_export snapshot ({tables, edges}), or a " +
        "Vincent-style case schema (parties/timeline_events/evidence_matrix/...). Full personal names, aliases, " +
        "free-text context and custom fields are retained. Authority sources and references are retained as " +
        "citation-required drafts pending validation. Idempotent by extract_id/record id; returns per-table/edge counts and detected shape.",
      inputSchema: { path: z.string().min(1) },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await caseImport(store as StoreOk, (args as { path: string }).path);
      return toolResult({ available: true, ...result });
    },
  );

  server.registerTool(
    "case_summary",
    {
      title: "Summarize the case store (case_facts-compatible shape)",
      description:
        "Returns the same shape as core.ts's case_facts tool (county, court, judge, referee, controlling_orders, " +
        "next_hearing, deadlines, full party names and child records) from the shared case store. " +
        "Complete private court/person/child context accompanies planning fields; no initials-only redaction is applied.",
      inputSchema: {},
      annotations: readOnly,
      _meta: {},
    },
    async () => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const summary = await caseSummary(store as StoreOk);
      return toolResult({ available: true, ...summary });
    },
  );

  // -------------------------------------------------------------------------
  // New registers â€” owner orders 2026-09-07 13:09-13:16.
  // -------------------------------------------------------------------------

  server.registerTool(
    "case_status",
    {
      title: "Get or set the case-status singleton",
      description:
        "action: \"get\" returns the case_status:current record (phase, posture, next_court_event ref, " +
        "open_deadlines[], notes, last_updated). action: \"set\" MERGEs the given fields in and stamps last_updated. " +
        "One record for the whole case â€” \"keep a record of ... case status\" (owner order).",
      inputSchema: {
        action: z.enum(["get", "set"]),
        data: z.record(z.string(), z.unknown()).optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const { action, data } = args as { action: "get" | "set"; data?: Record<string, unknown> };
      const result = action === "set" ? await caseStatusSet(store as StoreOk, data ?? {}) : await caseStatusGet(store as StoreOk);
      return toolResult({ available: true, status: result });
    },
  );

  server.registerTool(
    "case_docket",
    {
      title: "List filings, drafts, orders, and upcoming court events",
      description:
        "Returns filings + drafts + orders + upcoming court_events (date >= now) in one list, sorted by date, each " +
        "tagged with its table. Filter by `status` (filing/draft/court_event) `doc_type` (filing/draft) or `in_force` " +
        "(order, boolean). Use case_timeline({mode:\"court\"}) for the full past+upcoming court-event history instead.",
      inputSchema: {
        status: z.string().max(80).optional(),
        doc_type: z.string().max(80).optional(),
        in_force: z.boolean().optional(),
      },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const entries = await caseDocket(store as StoreOk, args as never);
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_memo",
    {
      title: "Put or list analysis/strategy/weakness/direction memos",
      description:
        "action: \"put\" upserts a memo (kind: analysis|strategy|weakness|direction|finding|risk|goal|issue|advice; " +
        "title, text, status, factors[], author, supersedes). action: \"list\" filters by kind/status. " +
        "action: \"latest\" returns the newest memo per kind seen. \"keep a record of analysis and strategy, " +
        "weaknesses and case direction ... keep memos\" (owner order).",
      inputSchema: {
        action: z.enum(["put", "list", "latest"]),
        id: z.string().min(1).max(200).optional(),
        kind: z.string().max(80).optional(),
        title: z.string().max(1000).optional(),
        text: z.string().max(20000).optional(),
        status: z.string().max(80).optional(),
        factors: z.array(z.string()).max(12).optional(),
        author: z.string().max(200).optional(),
        supersedes: z.string().max(200).optional(),
        source: z.record(z.string(), z.unknown()).optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const input = args as Record<string, unknown> & { action: "put" | "list" | "latest" };
      if (input.action === "put") {
        const result = await caseMemoPut(store as StoreOk, input as never);
        return toolResult({ available: true, ...result });
      }
      if (input.action === "latest") {
        const latest = await caseMemoLatestByKind(store as StoreOk);
        return toolResult({ available: true, latest });
      }
      const entries = await caseMemoList(store as StoreOk, input as never);
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_evidence_log",
    {
      title: "Append to or list the evidence log",
      description:
        "action: \"append\" adds one entry (action: received|collected|hashed|reviewed|produced|disclosed|admitted| " +
        "excluded|returned; optional exhibit ref RELATEd via the `logs` edge, by, hash, path, notes) and stamps " +
        "logged_at. action: \"list\" filters by exhibit ref or action. A place to log evidence handling \"for when we " +
        "get to that point\" (owner order) â€” independent of the platform's own H1/H2/H3 custody hashing.",
      inputSchema: {
        action: z.enum(["append", "list"]),
        id: z.string().min(1).max(200).optional(),
        log_action: z.string().max(80).optional(),
        exhibit: refSchema.optional(),
        by: z.string().max(200).optional(),
        hash: z.string().max(200).optional(),
        path: z.string().max(2000).optional(),
        notes: z.string().max(4000).optional(),
        source: z.record(z.string(), z.unknown()).optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const input = args as Record<string, unknown> & { action: "append" | "list"; log_action?: string };
      if (input.action === "append") {
        const result = await caseEvidenceLogAppend(store as StoreOk, { ...input, action: input.log_action ?? "reviewed" } as never);
        return toolResult({ available: true, ...result });
      }
      const entries = await caseEvidenceLogList(store as StoreOk, { exhibit: input.exhibit as never, action: input.log_action } as never);
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_eval",
    {
      title: "Put or list evals/reports",
      description:
        "action: \"put\" upserts an eval/report row (kind: eval|report|review|audit; title, subject ref or free text, " +
        "score, verdict, text, path, tool_or_model). action: \"list\" filters by kind/subject. \"a place to store " +
        "evals and reports\" (owner order).",
      inputSchema: {
        action: z.enum(["put", "list"]),
        id: z.string().min(1).max(200).optional(),
        kind: z.string().max(80).optional(),
        title: z.string().max(1000).optional(),
        subject: z.string().max(500).optional(),
        score: z.number().optional(),
        verdict: z.string().max(200).optional(),
        text: z.string().max(20000).optional(),
        path: z.string().max(2000).optional(),
        tool_or_model: z.string().max(200).optional(),
        source: z.record(z.string(), z.unknown()).optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const input = args as Record<string, unknown> & { action: "put" | "list" };
      if (input.action === "put") {
        const result = await caseEvalPut(store as StoreOk, input as never);
        return toolResult({ available: true, ...result });
      }
      const entries = await caseEvalList(store as StoreOk, input as never);
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_reference",
    {
      title: "List or match shared reference data",
      description: "Read full personalized shared reference records. Inputs: list filters or match text. Outputs: reference rows or pattern hits. Effects: none. Use library_propose for interactive edits; legacy load requests are refused because published library changes require tracked citation validation.",
      inputSchema: {
        action: z.enum(["load", "list", "match"]),
        path: z.string().min(1).optional(),
        from_plugin: z.boolean().optional(),
        kind: z.string().max(80).optional(),
        category: z.string().max(80).optional(),
        text: z.string().max(20000).optional(),
      },
      annotations: readWrite,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const input = args as { action: "load" | "list" | "match"; path?: string; from_plugin?: boolean; kind?: string; category?: string; text?: string };
      if (input.action === "load") {
        throw new Error("Published library loading requires the tracked import and citation-validation path; use library_propose for interactive changes");
      }
      if (input.action === "match") {
        const hits = await caseReferenceMatch(store as StoreOk, input.text ?? "");
        return toolResult({ available: true, hits });
      }
      const entries = await caseReferenceList(store as StoreOk, { kind: input.kind, category: input.category });
      return toolResult({ available: true, entries });
    },
  );

  server.registerTool(
    "case_record",
    {
      title: "Open one record in the shared legal-record contract",
      description:
        "Given a record ref (\"table:id\"), returns { contract: \"propria.legal-record.v1\", id, table, version, record }. " +
        "version is computed by the case store (sha256 of the record's canonical form), so the Family Law Toolkit and " +
        "Advocatio show identical ids and versions for the same record; any correction yields a new version. " +
        "Returns { found: false } when no such record exists.",
      inputSchema: { id: refSchema },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const ref = (args as { id: never }).id;
      const result = await caseRecord(store as StoreOk, ref);
      if (!result) return toolResult({ available: true, found: false, id: typeof ref === "string" ? ref : null });
      return toolResult({ available: true, found: true, ...result });
    },
  );

  server.registerTool(
    "case_source",
    {
      title: "Show a record's original-source provenance",
      description:
        "Given a record ref (\"table:id\"), returns its `source` block ({path, sha256, r2_path, locator, url, row, " +
        "extract_id}) if it carries one, whether `source.path` exists LOCALLY (no network calls), and the r2 pointer. " +
        "\"every record links/points to its original source document\" (owner order).",
      inputSchema: { id: refSchema },
      annotations: readOnly,
      _meta: {},
    },
    async (args) => {
      const store = await getStore();
      if (!store.available) return unavailable(store.reason);
      const result = await caseSourceOf(store as StoreOk, (args as { id: never }).id);
      return toolResult({ available: true, ...result });
    },
  );
}
