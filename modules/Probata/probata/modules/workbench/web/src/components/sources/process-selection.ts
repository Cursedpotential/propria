// Byline: Codex · GPT-6 · 2026-10-06.
import type { ProfferStartRequest, ProfferStartResponse } from "@/lib/shared/types";

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
): Promise<{ accepted: number; error: string | null }> {
  let accepted = 0;
  for (const entry of entries) {
    try {
      if (!entry.response) entry.response = await start(entry.request);
      accepted += 1;
      onAccepted(entry);
    } catch (error) {
      return { accepted, error: error instanceof Error ? error.message : "Processing could not be started." };
    }
  }
  return { accepted, error: null };
}
