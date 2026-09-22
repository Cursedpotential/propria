// Byline: Claude Code · Opus 5 · 2026-09-22
// The one state mark every Sources row carries, and where it comes from.
//
// Four states, in the owner's words (2026-09-21 rethink): not processed ·
// decoded · in context · failed. Each is read from a durable server fact —
// never guessed from a file name:
//
//   failed / in context  <- GET /api/proffer/proposal-resources (one row per
//                           run, keyed by the run's own source_ref)
//   decoded              <- POST /api/proffer/decoded/exists (the derivation
//                           manifest object exists for that source)
//   not processed        <- neither of the above said anything
//
// Known limit, stated plainly rather than hidden: both facts resolve through
// the source's object path (`source_ref`), so a file that is later moved to a
// canonical home loses the link until the engine records a move. Nothing here
// keys on the *folder* a file happens to sit in, so an unsorted file in Triage
// carries exactly the same marks as a sorted one.

import type { ProfferProposalResource } from "@/lib/shared/types";

export type SourceState = "not_processed" | "decoded" | "in_context" | "failed";

export const SOURCE_STATE_LABEL: Record<SourceState, string> = {
  not_processed: "not processed",
  decoded: "decoded",
  in_context: "in context",
  failed: "failed",
};

/** Tailwind classes per state; one small mark on the item, never a banner. */
export const SOURCE_STATE_CLASS: Record<SourceState, string> = {
  not_processed: "border-border bg-muted text-muted-foreground",
  decoded: "border-[#4051b9] bg-[#e9ecfb] text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]",
  in_context: "border-[#2f9d67] bg-[#e2f3e9] text-[#17794b] dark:bg-[#203d31] dark:text-[#72d9a1]",
  failed: "border-[#b5433b] bg-[#fbe9e7] text-[#8f302a] dark:bg-[#3a2422] dark:text-[#ffb5ae]",
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

export function sourceState(sourceRef: string, facts: SourceStateFacts): SourceState {
  const runs = facts.runs.get(sourceRef) ?? [];
  if (runs.some((run) => run.lifecycle === "completed")) return "in_context";
  if (runs.length && runs.every((run) => run.lifecycle === "failed")) return "failed";
  if (facts.decoded.has(sourceRef)) return "decoded";
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
