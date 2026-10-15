// Byline: Codex · GPT-6 · 2026-10-12.
import { useEffect, useRef } from "react";
import { toast } from "sonner";

/** Present an observed business state without using Temporal execution completion as success.
 * Input: server business status. Output: common visible state, retaining unknown values.
 * Effects: none. Pick for standalone and composed operation surfaces, never execution routing.
 */
export function operationState(status: unknown): string {
  if (typeof status !== "string" || !status) return "pending";
  if (["accepted", "scheduled", "queued", "waiting"].includes(status)) return "pending";
  if (["enriching", "registering", "preparing", "processing"].includes(status)) return "running";
  if (["complete", "completed_with_flags"].includes(status)) return "completed";
  if (["partial_enrichment", "completed_with_failures"].includes(status)) return "partial";
  return status;
}

/** Notify observed business failure, partial output, completion and recovery through the existing in-app Toaster.
 * Inputs: durable workflow/run identity, business status, operation label and actual error detail.
 * Output: none. Effects: deduplicated Sonner notifications only; no external message or execution.
 * Pick alongside detailed status panels for independent and composed Temporal operations.
 */
export function useOperationNotifications(workflowId: string | undefined, runId: string | undefined, status: string | undefined, label: string, detail?: string) {
  const observed = useRef({ events: new Set<string>(), failed: false });
  useEffect(() => {
    if (!workflowId || !status) return;
    const state = operationState(status);
    if (!["failed", "partial", "completed"].includes(state)) return;
    const event = `${workflowId}:${runId ?? ""}:${state}`;
    if (observed.current.events.has(event)) return;
    if (observed.current.events.size >= 64) observed.current.events.delete(observed.current.events.values().next().value!);
    observed.current.events.add(event);
    const options = { id: event, description: [detail, `Workflow ${workflowId}`, runId ? `Run ${runId}` : undefined].filter(Boolean).join(" · ") };
    if (state === "failed") { observed.current.failed = true; toast.error(`${label} failed`, options); }
    else if (state === "partial") { observed.current.failed = true; toast.warning(`${label} partially completed`, options); }
    else { toast.success(`${label} ${observed.current.failed ? "recovered" : "completed"}`, options); observed.current.failed = false; }
  }, [workflowId, runId, status, label, detail]);
}
