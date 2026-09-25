// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Review Actions: start a fresh run of the selected source that carries the operator's
// context, parser and repair choices, then answer that new run's own gates with those
// choices when it reaches them. A re-run always gets a new request identity (the
// Temporal workflow id); the finished or failed run is never presented as resumed.
//
// The start contract has no handler field. A chosen parser is honored by starting
// with OPERATOR_HANDLER_SELECTION, which makes the engine stop at its parser step
// (workflow.go: `in.ParserOptionsRef == OperatorHandlerSelectionOptions`), and then
// recording the chosen candidate against that run's own recommendation.
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { createProfferSourceContext, decideProfferHandler, decideProfferRepair, startProffer } from "@/lib/api-client";
import type {
  MatterMode,
  ProfferHumanSourceAssertions,
  ProfferObservedSource,
  ProfferOperatorSnapshot,
  ProfferParserCandidate,
  ProfferPreviewResponse,
} from "@/lib/shared/types";

/** Starts a run that stops at the parser step for an operator choice. */
export const OPERATOR_HANDLER_SELECTION = "operator-handler-selection/v1";
/** What Intake starts with: the engine selects the registered handler itself. */
export const ENGINE_HANDLER_SELECTION = "pending-handler-selection/v1";

export interface RerunRequest {
  declaredFormat: string;
  /** Context carried into the new run as its first revision; null starts it without context. */
  context: { assertions: ProfferHumanSourceAssertions; observedSource: ProfferObservedSource; changeReason: string } | null;
  /** A candidate to record at the new run's parser step, "choose" to stop there, null to let the engine choose. */
  parser: ProfferParserCandidate | "choose" | null;
  /** "original": if the new run's repair check stops it, continue with the retained original. */
  repair: "original" | null;
}

export interface PendingGateAnswers {
  handler?: ProfferParserCandidate;
  repair?: "original";
}

/** The new run's own offer of the chosen parser (same id and path; its version may have moved on). */
export function offeredCandidate(preview: ProfferPreviewResponse, choice: ProfferParserCandidate): ProfferParserCandidate | null {
  const offered = preview.recommended_handler ? [preview.recommended_handler, ...(preview.alternative_handlers ?? [])] : [];
  const sameRoute = offered.filter((candidate) => candidate.handler_id === choice.handler_id && candidate.execution_path === choice.execution_path);
  return sameRoute.find((candidate) => candidate.handler_version === choice.handler_version) ?? sameRoute[0] ?? null;
}

function message(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback;
}

export function useReviewRerun({
  mode,
  previewHandle,
  snapshot,
  preview,
  loadSnapshot,
  onStarted,
}: {
  mode: MatterMode;
  previewHandle: string;
  snapshot: ProfferOperatorSnapshot | null;
  preview: ProfferPreviewResponse | null;
  loadSnapshot: () => Promise<void>;
  onStarted: (previewHandle: string) => void;
}) {
  const [starting, setStarting] = useState(false);
  const [pending, setPending] = useState<Record<string, PendingGateAnswers>>({});
  const [startedHandles, setStartedHandles] = useState<string[]>([]);
  // Each gate of each run is answered automatically at most once; a later recovery
  // stop (after a failed parse) is always left to the operator.
  const answeredRef = useRef(new Set<string>());

  const rerun = useCallback(async (request: RerunRequest) => {
    if (!snapshot || starting) return;
    setStarting(true);
    try {
      const requestId = `proffer-${snapshot.matter_id}-${crypto.randomUUID()}`;
      let sourceContextRef: string | null = null;
      if (request.context) {
        const receipt = await createProfferSourceContext({
          request_id: requestId,
          matter_id: snapshot.matter_id,
          court_case_id: snapshot.court_case_id,
          source_ref: snapshot.source_ref,
          observed_source: request.context.observedSource,
          assertions: request.context.assertions,
          change_reason: request.context.changeReason,
          matter_mode: mode,
        });
        sourceContextRef = receipt.source_context_ref;
      }
      const started = await startProffer({
        request_id: requestId,
        source_ref: snapshot.source_ref,
        declared_format: request.declaredFormat,
        parser_options_ref: request.parser ? OPERATOR_HANDLER_SELECTION : ENGINE_HANDLER_SELECTION,
        matter_id: snapshot.matter_id,
        court_case_id: snapshot.court_case_id,
        source_context_ref: sourceContextRef,
        matter_mode: mode,
      });
      const answers: PendingGateAnswers = {};
      if (request.parser && request.parser !== "choose") answers.handler = request.parser;
      if (request.repair) answers.repair = request.repair;
      if (answers.handler || answers.repair) {
        setPending((current) => ({ ...current, [started.preview_handle]: answers }));
      }
      setStartedHandles((current) => [...current, started.preview_handle]);
      toast.success("Started a new run of this source");
      onStarted(started.preview_handle);
    } catch (error) {
      toast.error(message(error, "The new run could not be started"));
    } finally {
      setStarting(false);
    }
  }, [mode, onStarted, snapshot, starting]);

  const pendingForRun = previewHandle ? pending[previewHandle] : undefined;

  useEffect(() => {
    if (!previewHandle || !pendingForRun || !preview || preview.preview_handle !== previewHandle) return;
    if (preview.phase === "awaiting_repair_decision" && pendingForRun.repair === "original") {
      const key = `${previewHandle}:repair`;
      if (answeredRef.current.has(key)) return;
      answeredRef.current.add(key);
      decideProfferRepair(previewHandle, mode, { approved: true, apply_repair: false })
        .then(() => {
          toast.success("Repair step answered: continue with the retained original");
          return loadSnapshot();
        })
        .catch((error: unknown) => toast.error(message(error, "The repair step could not be answered")));
      return;
    }
    if (preview.phase === "awaiting_handler_selection" && pendingForRun.handler && preview.handler_recommendation_ref) {
      const key = `${previewHandle}:handler`;
      if (answeredRef.current.has(key)) return;
      const offered = offeredCandidate(preview, pendingForRun.handler);
      if (!offered) return; // the Parser section says so and the operator picks by hand
      answeredRef.current.add(key);
      decideProfferHandler(previewHandle, mode, {
        recommendation_ref: preview.handler_recommendation_ref,
        handler_id: offered.handler_id,
        handler_version: offered.handler_version,
        execution_path: offered.execution_path,
        compatibility_ref: offered.compatibility_ref,
      })
        .then(() => {
          toast.success(`Parser step answered: ${offered.handler_id} ${offered.handler_version}`);
          return loadSnapshot();
        })
        .catch((error: unknown) => toast.error(message(error, "The parser step could not be answered")));
    }
  }, [loadSnapshot, mode, pendingForRun, preview, previewHandle]);

  // A running run is re-read every few seconds so its progress, and any stop that needs an
  // answer, shows up without a manual refresh. A run started here is also re-read while it is
  // not readable yet, and while it still owes itself an answer. Reading stops at the human
  // preview decision and at the end of the run.
  const ownRun = Boolean(previewHandle) && startedHandles.includes(previewHandle);
  const current = snapshot?.preview_handle === previewHandle ? snapshot : null;
  const watching = Boolean(previewHandle) && (current
    ? !current.terminal && current.lifecycle !== "awaiting_preview_decision"
      && (current.lifecycle === "running" || (ownRun && Boolean(pendingForRun)))
    : ownRun);
  useEffect(() => {
    if (!watching) return;
    const timer = window.setInterval(() => void loadSnapshot(), 4000);
    return () => window.clearInterval(timer);
  }, [loadSnapshot, watching]);

  return { rerun, starting, pendingForRun };
}
