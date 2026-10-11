import type { MatterMode, ProfferOperationSummary } from "@/lib/shared/types";

// Byline: Codex · GPT-6 · 2026-10-06
/** Merge a fresh first page over loaded rows while retaining older cursor pages.
 * Inputs: already ordered current rows and an ordered server page.
 * Output: unique rows ordered with the incoming page first and older loaded rows after it.
 * Side effects: none. Use for Activity refresh and pagination reconciliation.
 */
export function mergeOperationRows(
  current: readonly ProfferOperationSummary[],
  incoming: readonly ProfferOperationSummary[],
): ProfferOperationSummary[] {
  const rows = new Map<string, ProfferOperationSummary>();
  for (const operation of incoming) rows.set(operation.preview_handle, operation);
  for (const operation of current) {
    if (!rows.has(operation.preview_handle)) rows.set(operation.preview_handle, operation);
  }
  return Array.from(rows.values());
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Return the human-readable filename represented by a source locator.
 * Inputs: source reference string.
 * Output: decoded final path segment, or the original reference when no segment exists.
 * Side effects: none. Use in Activity and batch rows when no registered filename is available.
 */
export function sourceFilename(sourceRef: string): string {
  const withoutAuthority = sourceRef.replace(/^[a-z0-9]+:\/\/[^/]+\//i, "");
  const segment = withoutAuthority.split("/").filter(Boolean).at(-1);
  if (!segment) return sourceRef;
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Build the canonical Read URL for one mode-scoped Proffer attempt.
 * Inputs: preview handle and Dev/Live mode.
 * Output: /read URL carrying resource and mode query parameters.
 * Side effects: none. Use for opening an exact result from Activity or batch history.
 */
export function attemptReviewHref(previewHandle: string, mode: MatterMode): string {
  const query = new URLSearchParams({ resource: previewHandle, mode });
  return `/read?${query.toString()}`;
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Translate a Proffer lifecycle into the next plain-language operator step.
 * Inputs: one operation lifecycle value.
 * Output: concise action label for Activity.
 * Side effects: none. Use instead of exposing engine lifecycle names in row actions.
 */
export function nextOperationAction(lifecycle: ProfferOperationSummary["lifecycle"]): string {
  switch (lifecycle) {
    case "running": return "View progress";
    case "awaiting_repair_decision":
    case "awaiting_preview_decision": return "Review this import";
    case "failed": return "Start a new import";
    case "completed": return "View completed import";
    case "cancelled": return "View cancellation details";
    case "unavailable": return "Check this import";
  }
}
