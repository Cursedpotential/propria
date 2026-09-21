import { AlertTriangle, Check, Loader2 } from "lucide-react";

import {
  PROFFER_CHECKPOINT_FAILED_COPY,
  PROFFER_CHECKPOINT_WAITING_COPY,
  PROFFER_CONTEXT_CHECKPOINTS,
  profferCheckpointStatuses,
} from "@/lib/proffer-context-checkpoints";
import type { ProfferPreviewCheckpoint, ProfferPreviewEvent, ProfferPreviewReceipt } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

export function ContextFlowRail({
  started,
  phase,
  receipts,
  checkpoints,
  events,
}: {
  started: boolean;
  phase?: string;
  receipts?: ProfferPreviewReceipt[] | null;
  checkpoints?: ProfferPreviewCheckpoint[] | null;
  events?: ProfferPreviewEvent[];
}) {
  const statuses = profferCheckpointStatuses({ started, phase, receipts, checkpoints, events });

  return (
    // One line (owner ruling 2026-09-20 23:48: one-viewport Review). The status copy stays for
    // assistive tech and as a tooltip; it no longer takes a row of its own.
    <section className="flex items-center gap-x-4 overflow-x-auto whitespace-nowrap border-b bg-card px-3 py-1.5" aria-labelledby="context-flow-heading">
      <h2 id="context-flow-heading" className="text-xs font-semibold" title="All Review views unlock after the six processing checkpoints complete.">Context processing</h2>
      <p className="sr-only">All Review views unlock after the six processing checkpoints complete.</p>
      <ol className="flex items-center gap-x-3" aria-label="Context processing checkpoints">
        {PROFFER_CONTEXT_CHECKPOINTS.map((checkpoint, index) => {
          const status = statuses[checkpoint.type];
          const statusCopy = status === "completed"
            ? "Completed."
            : status === "running"
              ? "In progress."
              : status === "failed"
                ? PROFFER_CHECKPOINT_FAILED_COPY
                : PROFFER_CHECKPOINT_WAITING_COPY;
          return (
            <li
              key={checkpoint.type}
              data-checkpoint={checkpoint.type}
              data-status={status}
              className="flex items-center gap-1.5"
              title={statusCopy}
            >
              <span
                className={cn(
                  "grid h-5 w-5 place-items-center rounded-full border bg-card text-[10px] font-semibold",
                  status === "completed" && "border-[#2f9d67] bg-[#2f9d67] text-white",
                  status === "running" && "border-primary text-primary",
                  status === "failed" && "border-destructive bg-destructive text-destructive-foreground",
                )}
                aria-hidden="true"
              >
                {status === "completed" ? <Check className="h-3 w-3" /> : status === "running" ? <Loader2 className="h-3 w-3 animate-spin motion-reduce:animate-none" /> : status === "failed" ? <AlertTriangle className="h-3 w-3" /> : index + 1}
              </span>
              <strong className={cn("text-xs font-medium", status === "failed" && "text-destructive")}>{checkpoint.label}</strong>
              <span className="sr-only" role={status === "failed" ? "alert" : "status"}>
                {statusCopy}
              </span>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
