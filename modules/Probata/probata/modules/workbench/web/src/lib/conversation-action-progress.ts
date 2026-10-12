// Byline: Codex · GPT-6 · 2026-10-12.
import type { ActionStatus, ExtractorInfo } from "@/lib/conversation-actions-client";
import type { WorkflowProgress } from "@/lib/entity-extraction-client";

/** Identify failed extractor retries whose exact conversation is known from the accepted selection.
 * Inputs: actual progress, saved thread IDs and requested extractor IDs. Output: safe single-extractor choices.
 * Effects: none. Pick for the existing conversation start route; ambiguous multi-thread/run mixtures
 * require server retry coordinates and never derive identity from display labels.
 */
export function failedConversationExtractors(status: ActionStatus, threadIds: readonly string[], requested: readonly string[]): string[] {
  if (status.kind !== "extraction" || threadIds.length !== 1 || ["pending", "queued", "running", "retrying"].includes(status.outcome)) return [];
  return requested.filter((id) => {
    const stages = status.steps.filter((step) => step.extractor === id);
    return stages.length > 0 && stages.every((step) => step.status === "failed");
  });
}

/** Derive extraction presentation from business outcome and actual requested extractor steps.
 * Inputs: worker progress, registry labels and the exact extractor selection saved at start.
 * Output: progress and an explicit success/failure summary. Effects: none.
 * Pick for conversation extraction; resolution, HTTP acceptance and Temporal completion
 * never count as extractor success. Send receipts keep their existing business outcome.
 */
export function conversationActionProgress(status: ActionStatus, extractors: readonly ExtractorInfo[], requested: readonly string[] = []) {
  const label = (id?: string) => extractors.find((extractor) => extractor.id === id)?.label ?? (id === "surreal" ? "surreal-case" : id);
  const progress: WorkflowProgress = {
    workflow_id: status.workflow_id, outcome: status.outcome,
    steps: status.steps.map((step) => ({
      step: step.step === "resolve" && status.kind === "extraction" ? "Resolve source conversations (preparation)" : step.conversation ? `${step.conversation} · ${label(step.extractor) ?? ""}` : step.step,
      status: step.status, detail: step.detail ?? "", counts: step.counts ?? {}, flags: step.flags ?? [],
    })),
  };
  if (status.kind !== "extraction") return { progress, summary: "" };
  const stages = status.steps.filter((step) => Boolean(step.extractor) && (!requested.length || requested.includes(step.extractor!)));
  const succeeded = stages.filter((step) => step.status === "completed").length;
  const unsuccessful = stages.filter((step) => step.status !== "completed").length;
  const missing = requested.filter((id) => !stages.some((step) => step.extractor === id)).length;
  const resolution = status.steps.find((step) => step.step === "resolve");
  const unresolved = resolution?.status === "failed" || (resolution?.counts?.skipped ?? 0) > 0;
  const business = status.outcome.toLowerCase();
  const running = ["running", "pending", "queued", "retrying"].includes(business);
  const failed = ["failed", "canceled", "cancelled", "terminated", "timed_out"].includes(business);
  let summary: string;
  if (failed || resolution?.status === "failed") {
    progress.outcome = "failed";
    summary = `${succeeded} requested extractor stage${succeeded === 1 ? "" : "s"} succeeded. The extraction failed; inspect the errors below.`;
  } else if (running) {
    progress.outcome = business === "retrying" ? "retrying" : business === "pending" || business === "queued" ? "pending" : "running";
    summary = `${succeeded} requested extractor stage${succeeded === 1 ? "" : "s"} succeeded so far. Source resolution alone does not complete extraction.`;
  } else if (!succeeded) {
    progress.outcome = "failed";
    summary = "No requested extractor completed successfully. Source resolution and workflow completion do not establish extraction success.";
  } else if (unsuccessful || missing || unresolved) {
    progress.outcome = "partial";
    summary = `${succeeded} requested extractor stage${succeeded === 1 ? "" : "s"} succeeded; other requested stages failed, were skipped or have no successful result. Inspect their details before retrying.`;
  } else {
    progress.outcome = business === "completed_with_flags" ? "completed_with_flags" : "completed";
    summary = `All ${succeeded} reported requested extractor stage${succeeded === 1 ? "" : "s"} completed successfully.`;
  }
  return { progress, summary };
}
