import type { ProfferPreviewCheckpoint, ProfferPreviewEvent, ProfferPreviewReceipt } from "@/lib/shared/types";

export const PROFFER_CONTEXT_CHECKPOINTS = [
  { type: "raw_source_verification", label: "Raw source verification" },
  { type: "parser_selection", label: "Parser selection" },
  { type: "parser_execution", label: "Parser execution" },
  { type: "normalization", label: "Normalization" },
  { type: "storage", label: "Storage" },
  { type: "completeness", label: "Completeness" },
] as const;

export type ProfferContextCheckpointType = (typeof PROFFER_CONTEXT_CHECKPOINTS)[number]["type"];
export type ProfferContextCheckpointStatus = "waiting" | "running" | "completed" | "failed";

export const PROFFER_CHECKPOINT_WAITING_COPY = "Waiting for this checkpoint.";
export const PROFFER_CHECKPOINT_FAILED_COPY = "Stopped at this checkpoint. Saved upstream artifacts may still be inspected.";

export function checkpointLabel(type: ProfferContextCheckpointType) {
  return PROFFER_CONTEXT_CHECKPOINTS.find((checkpoint) => checkpoint.type === type)?.label ?? type;
}

/** Map reported checkpoint evidence without guessing that unreported work is running.
 * Inputs: receipts, live checkpoints and authoritative lifecycle. Outputs: checkpoint statuses.
 * Effects: none; use for progress display independently of artifact availability.
 */
export function profferCheckpointStatuses({
  phase,
  lifecycle,
  terminal = false,
  receipts = [],
  checkpoints = [],
}: {
  started: boolean;
  phase?: string;
  lifecycle?: string | null;
  terminal?: boolean;
  receipts?: ProfferPreviewReceipt[] | null;
  checkpoints?: ProfferPreviewCheckpoint[] | null;
  events?: ProfferPreviewEvent[];
}): Record<ProfferContextCheckpointType, ProfferContextCheckpointStatus> {
  const statuses = Object.fromEntries(
    PROFFER_CONTEXT_CHECKPOINTS.map(({ type }) => [type, "waiting"]),
  ) as Record<ProfferContextCheckpointType, ProfferContextCheckpointStatus>;

  for (const receipt of receipts ?? []) {
    statuses[receipt.receipt_type] = receipt.status === "completed" || receipt.status === "skipped"
      ? "completed"
      : receipt.status === "failed"
        ? "failed"
        : receipt.status === "running" ? "running" : "waiting";
  }

  for (const checkpoint of checkpoints ?? []) {
    statuses[checkpoint.checkpoint] = checkpoint.status === "pending" ? "waiting" : checkpoint.status;
  }

  const stopped = terminal || ["failed", "cancelled", "completed", "rerun_required", "unavailable"].includes(lifecycle ?? phase ?? "");
  if (stopped) {
    for (const { type } of PROFFER_CONTEXT_CHECKPOINTS) {
      if (statuses[type] === "running") statuses[type] = "waiting";
    }
  }
  return statuses;
}

export function profferContextFlowComplete(
  receipts: ProfferPreviewReceipt[] | null | undefined,
  checkpoints?: ProfferPreviewCheckpoint[] | null,
) {
  return PROFFER_CONTEXT_CHECKPOINTS.every(({ type }) => {
    const durableReceiptComplete = receipts?.some(
      (receipt) => receipt.receipt_type === type && receipt.status === "completed",
    );
    const liveCheckpointComplete = !checkpoints?.length || checkpoints.some(
      (checkpoint) => checkpoint.checkpoint === type && checkpoint.status === "completed",
    );
    return Boolean(durableReceiptComplete && liveCheckpointComplete);
  });
}
