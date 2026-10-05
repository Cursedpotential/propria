// Byline: Codex / GPT-6 / 2026-10-04 - bounded shared excerpts; caseRecord versions separate from source SHA.
// Byline: Claude Code · Fable 5.1 · 2026-09-07
//
// survival_guide loader. Owner correction (2026-09-07 09:51): this tool does NOT
// author complete static guides. It resolves shared event-pack records, the shared
// writing template, optional current shared case facts, and bounded pinpoint excerpts
// for sources the pack cites — the calling MODEL writes the guide from those inputs.
//
// Path resolution mirrors core.ts's pluginRootPath(): this module is bundled to
// dist/survival-guide.js, a sibling of dist/core.js and dist/server.js, both two
// directories below the plugin root that holds content/.
//
// Shared records are authoritative. The synchronous list below remains only for
// the existing server schema's startup enum until its owner changes that adapter;
// every actual invocation discovers and validates event packs from shared records.

import { readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { getSharedCaseFacts, getSharedRuleCatalogContext, ruleCorrectionProposalsForCitations, type RuleCorrectionProposal } from "./core.js";
import { getReference, getReferenceExcerpt, type ReferenceExcerpt } from "./content-store.js";
import { getStore, normalize } from "./store.js";

function pluginRootPath(...segments: string[]): string {
  const here = dirname(fileURLToPath(import.meta.url));
  return join(here, "..", "..", ...segments);
}

const EVENTS_DIR = pluginRootPath("content", "tools", "survival-guide", "events");

export interface SurvivalGuideContextPack {
  id: string;
  title: string;
  what_it_is: string;
  who_is_in_the_room?: string[];
  who_reads_it?: string[];
  sequence: string[];
  prepare: string[];
  applicable_rules: Array<{ cite: string; summary: string; source: string; section: string; disposition?: string }>;
  deadlines: Array<{ label: string; rule_preset: string | null; cite: string; status: string }>;
  traps: Array<{ trap: string; safe_move: string }>;
  do_not: string[];
  phrases: { openers: string[]; closers: string[] };
  safety_gates: string[];
  exit_checklist: string[];
  sources: Array<{ file: string; section: string }>;
}

/** Return packaged event IDs for the existing synchronous server schema compatibility only.
 * Inputs: none. Outputs: sorted local names; these are not the current event catalog.
 * Effects: one bounded directory listing. Pick discoverSharedSurvivalGuideEvents() for runtime shared records.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export function listSurvivalGuideEvents(): string[] {
  return readdirSync(EVENTS_DIR)
    .filter((name) => name.endsWith(".json"))
    .map((name) => name.slice(0, -5))
    .sort();
}

const EVENT_PAGE_SIZE = 100;
const EVENT_ROW_BUDGET = 5000;

/** Discover all current shared event-pack identities with ordered bounded keyset pages.
 * Inputs: none; each page carries at most 100 reference identities and the full scan caps at 5,000.
 * Outputs: sorted unique event IDs; malformed, duplicate, nonadvancing, empty, or over-budget catalogs throw.
 * Effects: read-only shared queries. Pick over local filenames so newly shared packs are discovered without a rebuild.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function discoverSharedSurvivalGuideEvents(): Promise<string[]> {
  const store = await getStore();
  if (!store.available) throw new Error("Shared survival-guide catalog unavailable");
  const ids: string[] = [];
  const seen = new Set<string>();
  let after: string | null = null;
  for (let pageNumber = 0; pageNumber <= EVENT_ROW_BUDGET / EVENT_PAGE_SIZE; pageNumber++) {
    const query = after === null
      ? `SELECT record::id(id) AS content_cursor, key, kind FROM reference WHERE kind = 'event_pack' ORDER BY content_cursor ASC LIMIT ${EVENT_PAGE_SIZE};`
      : `SELECT record::id(id) AS content_cursor, key, kind FROM reference WHERE kind = 'event_pack' AND record::id(id) > $after ORDER BY content_cursor ASC LIMIT ${EVENT_PAGE_SIZE};`;
    const results = await store.db.query(query, after === null ? {} : { after });
    if (!Array.isArray(results) || results.length !== 1 || !Array.isArray(results[0])) throw new Error("Malformed shared survival-guide catalog page");
    const page = normalize(results[0]) as unknown;
    if (!Array.isArray(page) || page.length > EVENT_PAGE_SIZE) throw new Error("Malformed or oversized shared survival-guide catalog page");
    let cursor: string | null = after;
    for (const value of page) {
      if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Malformed shared event-pack identity");
      const row = value as Record<string, unknown>;
      const key = row.key;
      const next = row.content_cursor;
      if (row.kind !== "event_pack" || typeof key !== "string" || !/^[a-z0-9][a-z0-9-]{0,99}$/.test(key)
        || typeof next !== "string" || next !== key || seen.has(key) || (cursor !== null && next <= cursor)) {
        throw new Error("Malformed, duplicate, or nonadvancing shared event-pack catalog");
      }
      seen.add(key);
      ids.push(key);
      cursor = next;
    }
    if (ids.length > EVENT_ROW_BUDGET) throw new Error("Shared event-pack catalog exceeds 5,000-row budget");
    if (page.length < EVENT_PAGE_SIZE) {
      if (ids.length === 0) throw new Error("Shared survival-guide event catalog is empty");
      return ids;
    }
    if (cursor === null) throw new Error("Shared event-pack catalog page did not advance");
    after = cursor;
  }
  throw new Error("Shared event-pack catalog page budget exhausted");
}

/** Resolve one event pack by exact identity from the current shared event catalog.
 * Inputs: shared event ID. Outputs: validated context pack.
 * Effects: bounded shared listing plus one exact reference read; missing/malformed rows throw with no packaged fallback.
 * Pick this over local event JSON so additions/updates in shared records take effect immediately.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function loadContextPack(event: string): Promise<SurvivalGuideContextPack> {
  const ids = await discoverSharedSurvivalGuideEvents();
  if (!ids.includes(event)) {
    throw new Error(`Unknown survival_guide event "${event}". Known events: ${ids.join(", ")}`);
  }
  const ref = await getReference(event);
  if (ref) {
    if (ref.kind !== "event_pack" || (ref.key !== undefined && ref.key !== event)) throw new Error(`Shared survival-guide identity mismatch: ${event}`);
    const data = ref.data as SurvivalGuideContextPack | undefined;
    if (!data || data.id !== event || typeof data.title !== "string" || !Array.isArray(data.sequence)
      || !Array.isArray(data.prepare) || !Array.isArray(data.applicable_rules) || !Array.isArray(data.deadlines)
      || !Array.isArray(data.traps) || !Array.isArray(data.do_not) || !data.phrases
      || !Array.isArray(data.safety_gates) || !Array.isArray(data.exit_checklist) || !Array.isArray(data.sources)) {
      throw new Error(`Malformed shared survival-guide context: ${event}`);
    }
    return data;
  }
  throw new Error(`Shared survival-guide event pack missing: ${event}`);
}

export interface SurvivalGuideResult {
  event: string;
  format: "full" | "card" | "json";
  release_status?: Record<string, unknown>;
  template?: string;
  context_pack: SurvivalGuideContextPack;
  rule_correction_proposals: RuleCorrectionProposal[];
  rule_source_traceability: Array<Record<string, unknown>>;
  case_facts?: unknown;
  source_excerpts?: Record<string, string>;
  source_excerpt_resolutions?: ReferenceExcerpt[];
}

/** Read the current exact shared full or card survival-guide template.
 * Inputs: full/card format. Outputs: nonempty template text.
 * Effects: one shared read; missing/malformed records throw, with no packaged fallback. Pick this over local templates.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
async function loadTemplate(format: "full" | "card"): Promise<string> {
  const key = format === "card" ? "survival-guide-card-template" : "survival-guide-template";
  const ref = await getReference(key);
  if (ref) {
    if (typeof ref.body !== "string" || !ref.body.length) throw new Error(`Malformed shared survival-guide template: ${key}`);
    return ref.body;
  }
  throw new Error(`Shared survival-guide template missing: ${key}`);
}

/** Assemble a survival-guide context with current bounded shared source excerpts.
 * Inputs: current shared event ID, output format and optional shared case-facts inclusion.
 * Outputs: context/template, release status, exact source links/gaps, correction proposals, and bounded excerpt/provenance entries (at most 32 sources).
 * Effects: shared reads; failures throw, source/pinpoint gaps stay visible with empty excerpts, no excerpt file fallback.
 * Pick over writing a finished guide: this supplies model inputs; all event, template, authority, and case facts come from shared records.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export async function buildSurvivalGuide(input: {
  event: string;
  format?: "full" | "card" | "json";
  include_case_facts?: boolean;
}): Promise<SurvivalGuideResult> {
  const format = input.format ?? "full";
  const contextPack = await loadContextPack(input.event);
  if (contextPack.sources.length > 32 || contextPack.sources.some(source => !source || typeof source.file !== "string"
    || source.file.length > 512 || typeof source.section !== "string" || !source.section.trim() || source.section.length > 256)) {
    throw new Error("Invalid survival-guide source selections or 32-source budget exceeded");
  }

  const citations = [
    ...contextPack.applicable_rules.map((rule) => rule.cite),
    ...contextPack.deadlines.map((deadline) => deadline.cite),
  ].filter((citation, index, all) => typeof citation === "string" && citation.trim().length > 0 && all.indexOf(citation) === index);
  const catalogContext = await getSharedRuleCatalogContext(citations);
  const result: SurvivalGuideResult = {
    event: input.event,
    format,
    release_status: catalogContext.release_status,
    context_pack: contextPack,
    rule_correction_proposals: ruleCorrectionProposalsForCitations(contextPack.applicable_rules.map((rule) => rule.cite)),
    rule_source_traceability: catalogContext.rule_source_traceability,
  };

  if (format !== "json") {
    result.template = await loadTemplate(format);
  }

  if (input.include_case_facts) {
    result.case_facts = await getSharedCaseFacts();
  }

  const excerpts: Record<string, string> = {};
  const resolutions: ReferenceExcerpt[] = [];
  for (const source of contextPack.sources) {
    const resolution = await getReferenceExcerpt(source.file, source.section);
    resolutions.push(resolution);
    excerpts[source.file] = resolution.excerpt;
  }
  result.source_excerpts = excerpts;
  result.source_excerpt_resolutions = resolutions;

  return result;
}

/** Render model context and explicitly located shared excerpt evidence as Markdown.
 * Inputs: assembled survival-guide result. Outputs: Markdown preserving proposal status, excerpt text and named unresolved items.
 * Effects: none. Pick over raw JSON for full/card display; no claim that an unresolved pinpoint was read.
 * Byline: Codex · GPT-6-Luna · 2026-10-04.
 */
export function renderSurvivalGuideMarkdown(result: SurvivalGuideResult): string {
  const lines: string[] = [];
  lines.push(`# survival_guide — ${result.context_pack.title} (${result.format})`);
  lines.push("");
  if (result.release_status) {
    const release = result.release_status;
    lines.push("## Shared release status");
    lines.push(String(release.label ?? release.status ?? "Shared release status unavailable"));
    if (typeof release.record_ref === "string") {
      lines.push(`Record: ${release.record_ref}; version: ${String(release.record_version ?? "unavailable")}`);
    }
    if (typeof release.reason === "string") lines.push(release.reason);
    lines.push("");
  }
  const traceabilityGaps = result.rule_source_traceability.flatMap((trace) => {
    const citation = typeof trace.claimed_citation === "string" ? trace.claimed_citation : "Unlabeled rule citation";
    const linked = Array.isArray(trace.shared_sources) ? trace.shared_sources.filter((source): source is Record<string, unknown> => Boolean(source && typeof source === "object")) : [];
    if (trace.traceability_status !== "EXACT_SHARED_SOURCE_RECORD") {
      return [`${citation}: ${String(trace.traceability_status ?? "shared_source_gap")}`];
    }
    return linked.flatMap((source) => {
      const ref = typeof source.record_ref === "string" ? source.record_ref : "shared source record";
      const gaps = Array.isArray(source.traceability_gaps) ? source.traceability_gaps.filter((gap): gap is string => typeof gap === "string") : [];
      return [...new Set(gaps)].map((gap) => `${citation} (${ref}): ${gap}`);
    });
  });
  if (traceabilityGaps.length) {
    lines.push("## Rule-source traceability gaps");
    for (const gap of traceabilityGaps) lines.push(`- ${gap}`);
    lines.push("");
  }
  if (result.template) {
    lines.push("## Writing template");
    lines.push(result.template);
    lines.push("");
  }
  lines.push("## Context pack");
  lines.push("```json");
  lines.push(JSON.stringify(result.context_pack, null, 2));
  lines.push("```");
  if (result.rule_correction_proposals.length) {
    lines.push("");
    lines.push("## Dated rule-correction proposals — source corpus unchanged");
    for (const proposal of result.rule_correction_proposals) {
      lines.push(`### ${proposal.rule_identity} — ${proposal.proposal_status}`);
      if (proposal.rule_heading) lines.push(proposal.rule_heading);
      lines.push(`Recorded: ${proposal.recorded_on}; audit date: ${proposal.audit_date}; source: ${proposal.audit_source}; authority status: ${proposal.authority_status}`);
      lines.push(`Source locations: ${proposal.source_locations.join("; ")}`);
      lines.push(`Gap: ${proposal.gap}`);
      lines.push(`Proposed correction: ${proposal.proposed_correction}`);
      lines.push("");
    }
  }
  if (result.case_facts !== undefined) {
    lines.push("");
    lines.push("## Case facts");
    lines.push("```json");
    lines.push(JSON.stringify(result.case_facts, null, 2));
    lines.push("```");
  }
  if (result.source_excerpt_resolutions) {
    lines.push("");
    lines.push("## Shared source excerpts and resolution gaps");
    for (const item of result.source_excerpt_resolutions) {
      lines.push(`### ${item.source_path} — ${item.requested_pinpoint}`);
      lines.push(`Resolution: ${item.resolution_status}${item.gap_reason ? ` (${item.gap_reason})` : ""}`);
      lines.push(`Reference: ${item.reference_id ?? "unmapped"}; source SHA: ${item.source_sha256 ?? "unavailable"}; record version: ${item.record_version ?? "unavailable"}`);
      if (item.excerpt_truncated) lines.push("Excerpt limited to 4096 Unicode characters; this is not the complete requested text.");
      lines.push("```");
      lines.push(item.excerpt);
      lines.push("```");
    }
  }
  return lines.join("\n");
}
