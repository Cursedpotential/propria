// Byline: Claude Code · Opus 5 · 2026-09-22
// The one state mark every Sources row carries, and where it comes from.
//
// Four states, in the owner's words (2026-09-21 rethink): not processed ·
// decoded · in context · failed. Each is read from a durable server fact —
// never guessed from a file name:
//
//   failed / in context  <- GET /api/proffer/proposal-resources (one row per
//                           run, matched to the inspected provider version)
//   decoded              <- POST /api/proffer/decoded/exists (the derivation
//                           manifest object exists for that source)
//   not processed        <- neither of the above said anything
//
// Legacy path-only receipts and decode existence cannot verify the selected
// immutable version. They remain available as explicitly unpinned history;
// version status stays unverified instead of inheriting another version's state.

import type { ProfferProposalResource } from "@/lib/shared/types";

export type SourceState = "not_processed" | "decoded" | "in_context" | "failed" | "version_unverified";

export const SOURCE_STATE_LABEL: Record<SourceState, string> = {
  not_processed: "not processed",
  decoded: "decoded",
  in_context: "in context",
  failed: "failed",
  version_unverified: "version status unverified",
};

/** Tailwind classes per state; one small mark on the item, never a banner. */
export const SOURCE_STATE_CLASS: Record<SourceState, string> = {
  not_processed: "border-border bg-muted text-muted-foreground",
  decoded: "border-[#4051b9] bg-[#e9ecfb] text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]",
  in_context: "border-[#2f9d67] bg-[#e2f3e9] text-[#17794b] dark:bg-[#203d31] dark:text-[#72d9a1]",
  failed: "border-[#b5433b] bg-[#fbe9e7] text-[#8f302a] dark:bg-[#3a2422] dark:text-[#ffb5ae]",
  version_unverified: "border-border bg-muted text-muted-foreground",
};

export interface SourceStateFacts {
  /** source_ref -> the newest run lifecycle that touched it. */
  runs: ReadonlyMap<string, ProfferProposalResource[]>;
  /** source_ref -> a decode manifest exists. */
  decoded: ReadonlySet<string>;
}

export const EMPTY_STATE_FACTS: SourceStateFacts = { runs: new Map(), decoded: new Set() };

/** Group run records by the source they ran against, newest first. */
export function runsBySource(items: readonly ProfferProposalResource[]) {
  const map = new Map<string, ProfferProposalResource[]>();
  for (const item of items) {
    const list = map.get(item.source_ref) ?? [];
    list.push(item);
    map.set(item.source_ref, list);
  }
  for (const list of map.values()) {
    list.sort((left, right) => right.created_at.localeCompare(left.created_at));
  }
  return map;
}

/** Separate history for an inspected immutable version from path-only legacy receipts.
 * Inputs: source locator, indexed runs and optional inspected provider version.
 * Output: exact-version and explicitly unpinned histories plus other-version count.
 * Effects: none; choose before assigning current status, never combine versions by path alone.
 */
export function sourceRunHistory(sourceRef: string, facts: SourceStateFacts, providerVersion?: string | null) {
  const selected = historyLocator(sourceRef);
  const version = providerVersion || selected.version;
  const exact: ProfferProposalResource[] = [];
  const unpinned: ProfferProposalResource[] = [];
  let otherVersionCount = 0;
  for (const [locator, runs] of facts.runs) {
    const candidate = historyLocator(locator);
    if (candidate.base !== selected.base) continue;
    if (!candidate.version) unpinned.push(...runs);
    else if (version && candidate.version === version) exact.push(...runs);
    else otherVersionCount += runs.length;
  }
  exact.sort((left, right) => right.created_at.localeCompare(left.created_at));
  unpinned.sort((left, right) => right.created_at.localeCompare(left.created_at));
  return { exact, unpinned, otherVersionCount };
}

/** Read only the engine's supported single-version locator syntax for history joins.
 * Input: source URI. Output: canonical base plus exact version, or literal unmatched key.
 * Effects: none; choose to equate valid URL encodings without admitting unrelated query fields.
 */
function historyLocator(sourceRef: string): { base: string; version: string | null } {
  try {
    const parsed = new URL(sourceRef);
    if (parsed.hash || parsed.username || parsed.password) return { base: sourceRef, version: null };
    const entries = [...parsed.searchParams.entries()];
    if (entries.length && (entries.length !== 1 || entries[0][0] !== "versionId" || !entries[0][1] || entries[0][1] === "null")) return { base: sourceRef, version: null };
    parsed.search = "";
    return { base: parsed.toString(), version: entries[0]?.[1] ?? null };
  } catch { return { base: sourceRef, version: null }; }
}

/** Assign current source state only from an exactly matched inspected provider version.
 * Inputs: source locator, durable facts and provider version. Output: current status
 * or explicitly unverified version status. Effects: none; path-only history/decodes
 * never establish that the selected immutable version was processed.
 */
export function sourceState(sourceRef: string, facts: SourceStateFacts, providerVersion?: string | null): SourceState {
  const history = sourceRunHistory(sourceRef, facts, providerVersion);
  const runs = history.exact;
  if (runs.some((run) => run.lifecycle === "completed")) return "in_context";
  if (runs.length && runs.every((run) => run.lifecycle === "failed")) return "failed";
  if (history.unpinned.length || history.otherVersionCount || facts.decoded.has(sourceRef)) return "version_unverified";
  if (runs.length) return "not_processed";
  return "not_processed";
}

/** The handler the detected format recommends, and the alternatives the owner may pick instead. */
export interface HandlerChoice {
  id: string;
  label: string;
  formats: string[];
}

export const HANDLER_CHOICES: HandlerChoice[] = [
  { id: "xml", label: "Message backup decoder (SMS/MMS/calls XML)", formats: ["xml"] },
  { id: "message_export_json", label: "Message export reader (JSON)", formats: ["message_export_json"] },
  { id: "delimited_text", label: "Delimited text reader (CSV/TSV/TXT)", formats: ["delimited_text"] },
  { id: "markdown", label: "Markdown reader", formats: ["markdown"] },
  { id: "html", label: "HTML reader", formats: ["html"] },
  { id: "pdf", label: "PDF extractor", formats: ["pdf"] },
  { id: "docx", label: "Word document extractor", formats: ["docx"] },
  { id: "image", label: "Image handler", formats: ["image"] },
  { id: "archive", label: "Archive expander", formats: ["archive"] },
  { id: "unknown_binary", label: "Unclassified binary (detect on the engine)", formats: ["unknown_binary"] },
];

/** Detected format from the object's own extension — the same map the engine's preflight uses. */
export function detectedFormat(name: string): string {
  const extension = name.split(".").pop()?.toLowerCase() ?? "";
  const formats: Record<string, string> = {
    xml: "xml",
    json: "message_export_json",
    ndjson: "message_export_json",
    md: "markdown",
    txt: "delimited_text",
    csv: "delimited_text",
    tsv: "delimited_text",
    pdf: "pdf",
    png: "image",
    jpg: "image",
    jpeg: "image",
    gif: "image",
    webp: "image",
    avif: "image",
    tif: "image",
    tiff: "image",
    bmp: "image",
    docx: "docx",
    html: "html",
    htm: "html",
    zip: "archive",
    tar: "archive",
    tgz: "archive",
    gz: "archive",
    "7z": "archive",
    rar: "archive",
  };
  return formats[extension] ?? "unknown_binary";
}

export function formatBytes(value: number | null | undefined): string {
  if (value === null || value === undefined) return "—";
  if (value < 1024) return `${value} bytes`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}
