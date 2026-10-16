// Byline: Codex · GPT-6 · 2026-10-06.
import { ApiError } from "@/lib/api-client";
import type { ProfferStartRequest, ProfferStartResponse } from "@/lib/shared/types";

/** An earlier import the starter refused to duplicate (409 already_imported).
 * Owner spec 2026-10-15: "this has already been imported -- view it?".
 * Byline: Claude Code · Opus 5.5 · 2026-10-16 */
export interface AlreadyImported {
  status: "imported" | "in_progress";
  sourceKey: string;
  importedAt: string | null;
  name: string;
}

/** Read the starter's duplicate refusal from an API error, or null. */
export function alreadyImportedFrom(error: unknown, name: string): AlreadyImported | null {
  if (!(error instanceof ApiError) || error.status !== 409) return null;
  const body = error.body as { already_imported?: { status?: unknown; source_key?: unknown; imported_at?: unknown } } | undefined;
  const found = body?.already_imported;
  if (!found || (found.status !== "imported" && found.status !== "in_progress") || typeof found.source_key !== "string") return null;
  return { status: found.status, sourceKey: found.source_key, importedAt: typeof found.imported_at === "string" ? found.imported_at : null, name };
}

/** The Read view of a source: its key as an unpadded URL-safe base64 ID (the Imported API's encode_id). */
export function readSourceHref(sourceKey: string): string {
  const bytes = new TextEncoder().encode(sourceKey);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  const id = btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return `/read?${new URLSearchParams({ source: id })}`;
}

export interface SourceSubmission {
  request: ProfferStartRequest;
  response?: ProfferStartResponse;
}

/** Submit a selection once and retain accepted receipts if a later file fails.
 * Inputs: stable per-file requests, start transport, receipt callback.
 * Output: accepted count and optional failure. Effects: calls the existing start API
 * sequentially and records its receipts on entries. Retry uses the same request IDs;
 * already acknowledged sources are skipped. Use for Sources multi-file submission.
 */
export async function processSelection(
  entries: SourceSubmission[],
  start: (request: ProfferStartRequest) => Promise<ProfferStartResponse>,
  onAccepted: (entry: SourceSubmission) => void,
  name = "",
): Promise<{ accepted: number; error: string | null; alreadyImported?: AlreadyImported }> {
  let accepted = 0;
  for (const entry of entries) {
    try {
      if (!entry.response) entry.response = await start(entry.request);
      accepted += 1;
      onAccepted(entry);
    } catch (error) {
      const alreadyImported = alreadyImportedFrom(error, name);
      if (alreadyImported) return { accepted, error: null, alreadyImported };
      return { accepted, error: error instanceof Error ? error.message : "Processing could not be started." };
    }
  }
  return { accepted, error: null };
}
