import type { MatterMode, ProfferOperationSummary } from "@/lib/shared/types";

/** Merge a fresh first page over loaded rows while retaining older cursor pages. */
export function mergeOperationRows(
  current: readonly ProfferOperationSummary[],
  incoming: readonly ProfferOperationSummary[],
): ProfferOperationSummary[] {
  const rows = new Map(current.map((operation) => [operation.preview_handle, operation]));
  for (const operation of incoming) rows.set(operation.preview_handle, operation);
  return Array.from(rows.values());
}

/** Return the source filename represented by a locator when its final segment is readable. */
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

/** Link to the existing mode-scoped review and safe re-entry controls for one attempt. */
export function attemptReviewHref(previewHandle: string, mode: MatterMode): string {
  const query = new URLSearchParams({ mode, resource: previewHandle });
  return `/review?${query.toString()}`;
}

/** Translate a Proffer lifecycle into the next plain-language operator step. */
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
