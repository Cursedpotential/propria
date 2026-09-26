// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Byline amendment: Claude Code · Opus 5.5 · 2026-09-26 — the repair builder replaces the disabled
// apply-repair choice; a repaired run shows "Repaired → <new run>" from its history.
// Review Actions: always present for the selected run, whatever state it is in.
// Owner 2026-09-25 00:16: "Can't modify any metadata. I can't add any context, I can't
// choose a parser, I can't choose a repair. I can't do anything."
//
// Every control calls an endpoint that already exists:
//   context            -> POST /api/proffer/source-contexts (append-only revision)
//   parser, at its stop -> POST /api/proffer/previews/{handle}/handler-selection
//   repair, at its stop -> POST /api/proffer/previews/{handle}/repair-decision
//   repair plan         -> /api/proffer/repair/{propose,tools,validate,run,runs/{id}}
//                          (repair-builder.tsx): a separate plan whose result re-enters as a
//                          new run; this run's repair gate stays open (owner decision 4A)
//   anything else       -> POST /api/proffer/start: a fresh run of the same source that
//                          answers its own stops with the choices made here
//                          (hooks/use-review-rerun.ts).
"use client";

import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Loader2, RotateCcw, ShieldCheck } from "lucide-react";
import { type ReactNode, useState } from "react";

import { ParserSelectionPanel } from "@/components/intake/parser-selection-panel";
import { RepairBuilder } from "@/components/sbv/repair-builder";
import { RepairedLink } from "@/components/sbv/repair-run-view";
import { ReviewContextSection } from "@/components/sbv/review-context-section";
import { Button } from "@/components/ui/button";
import { offeredCandidate, type PendingGateAnswers, type RerunRequest } from "@/hooks/use-review-rerun";
import { getProfferRunSourceContext } from "@/lib/api-client";
import { declaredFormat } from "@/lib/declared-format";
import type {
  ProfferContentResponse,
  ProfferOperatorAvailability,
  ProfferOperatorSnapshot,
  ProfferParserCandidate,
  ProfferPreviewResponse,
} from "@/lib/shared/types";
import { cn } from "@/lib/utils";

function candidateKey(candidate: ProfferParserCandidate) {
  return [candidate.handler_id, candidate.handler_version, candidate.execution_path, candidate.compatibility_ref].join("\u0000");
}

/** One line for a stopped run: where it stopped and the part of the reason a person needs. */
function failureSummary(snapshot: ProfferOperatorSnapshot) {
  const failedStage = [...snapshot.stages].reverse().find((stage) => stage.status === "failed");
  const reason = snapshot.reason || failedStage?.reason || "";
  if (!reason && snapshot.lifecycle !== "failed") return null;
  const gateway = /tool gateway "([^"]+)" returned (\d+)/.exec(reason);
  const text = gateway
    ? `${gateway[1]} returned ${gateway[2]}`
    : reason.replace(/^activity error \([^)]*\):\s*/, "").trim() || "the run stopped without a reported reason";
  const stageId = failedStage?.stage ?? snapshot.current_stage ?? "";
  return {
    stage: stageId.replace(/_activity$/, "").replaceAll("_", " "),
    text,
    repairCheck: stageId.startsWith("assess_source_repair") || Boolean(gateway?.[1].startsWith("repair.")),
  };
}

function StatusLine({ label, value }: { label: string; value: ProfferOperatorAvailability }) {
  const text = value.status === "available" ? (value.ref ?? "ready") : value.status === "pending" ? "not run yet" : "not reported";
  return (
    <p className="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-2 text-xs" title={value.reason || undefined}>
      <span className="text-muted-foreground">{label}</span>
      <span className="truncate font-mono text-[11px]">{text}</span>
    </p>
  );
}

function SectionTitle({ id, children }: { id: string; children: ReactNode }) {
  return <h3 id={id} className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{children}</h3>;
}

export function ReviewActionsPanel({
  className,
  snapshot,
  preview,
  content,
  actionNames,
  actionPending,
  decision,
  onSelectHandler,
  onRetainOriginal,
  onRerun,
  rerunPending,
  pendingAnswers,
  onOpenRun,
}: {
  className?: string;
  snapshot: ProfferOperatorSnapshot;
  preview: ProfferPreviewResponse;
  content: ProfferContentResponse | null;
  actionNames: ReadonlySet<string>;
  actionPending: boolean;
  /** The Approve / Reject block; present only at the preview decision stop. */
  decision: ReactNode;
  onSelectHandler: (candidate: ProfferParserCandidate) => void;
  onRetainOriginal: () => void;
  onRerun: (request: RerunRequest) => void;
  rerunPending: boolean;
  pendingAnswers?: PendingGateAnswers;
  /** Opens another run (a repaired run's re-entry) in Review. */
  onOpenRun: (previewHandle: string) => void;
}) {
  const [selectedKey, setSelectedKey] = useState("");
  const [repairChoice, setRepairChoice] = useState<"original" | null>(null);
  const runContextQuery = useQuery({
    queryKey: ["proffer-run-source-context", snapshot.matter_mode, snapshot.preview_handle],
    queryFn: ({ signal }) => getProfferRunSourceContext(snapshot.preview_handle, snapshot.matter_mode, signal),
    retry: false,
    staleTime: 15_000,
  });
  const runContext = runContextQuery.data ?? null;
  const current = runContext?.current ?? null;

  const candidates = preview.recommended_handler ? [preview.recommended_handler, ...(preview.alternative_handlers ?? [])] : [];
  const selectedCandidate = candidates.find((candidate) => candidateKey(candidate) === selectedKey) ?? null;
  const atHandlerStop = preview.phase === "awaiting_handler_selection";
  const atRepairStop = actionNames.has("retain_original");
  const failure = failureSummary(snapshot);
  const sourceName = snapshot.source_ref.split("/").filter(Boolean).at(-1) ?? snapshot.source_ref;
  const format = runContext?.registration?.declared_format || content?.package.declared_format || declaredFormat({ name: sourceName });
  const blockedReason = runContextQuery.isPending ? "Reading how this run was registered…" : null;
  const blocked = rerunPending || Boolean(blockedReason);
  const plannedParser = atHandlerStop ? null : selectedCandidate;
  const handlerNotOffered = Boolean(atHandlerStop && pendingAnswers?.handler && !offeredCandidate(preview, pendingAnswers.handler));

  function request(overrides: Partial<Pick<RerunRequest, "parser" | "repair">> = {}): RerunRequest {
    return {
      declaredFormat: format,
      context: current
        ? {
            assertions: current.assertions,
            observedSource: current.observed_source,
            changeReason: `Carried from run ${snapshot.preview_handle.slice(0, 8)} (context revision ${current.revision}) into a re-run`,
          }
        : null,
      parser: overrides.parser !== undefined ? overrides.parser : plannedParser,
      repair: overrides.repair !== undefined ? overrides.repair : repairChoice,
    };
  }

  const carries = [
    format,
    current ? `context rev ${current.revision}` : "no context",
    plannedParser ? `parser ${plannedParser.handler_id}` : "engine picks the parser",
    repairChoice === "original" ? "skip repair if asked" : "stop if repair is needed",
  ].join(" · ");

  return (
    <aside aria-label="Actions" className={cn("space-y-3 border bg-card p-3", className)} data-testid="review-actions-panel">
      <h2 className="text-sm font-semibold">Actions</h2>

      {decision && (
        <section aria-label="Decision" className="space-y-2 border-b pb-3">
          <SectionTitle id="review-decision-heading">Decision</SectionTitle>
          {decision}
        </section>
      )}

      <ReviewContextSection
        snapshot={snapshot}
        runContext={runContext}
        loading={runContextQuery.isPending}
        error={runContextQuery.error ? runContextQuery.error.message : null}
        refreshing={runContextQuery.isFetching}
        rerunPending={blocked}
        onSaved={() => void runContextQuery.refetch()}
        onRerunWithContext={() => onRerun(request())}
      />

      <section aria-labelledby="review-parser-heading" className="space-y-1.5 border-t pt-3" data-testid="review-actions-parser">
        <SectionTitle id="review-parser-heading">Parser</SectionTitle>
        {handlerNotOffered && pendingAnswers?.handler && (
          <p className="text-xs text-destructive" role="status">{pendingAnswers.handler.handler_id} is not offered for this run. Choose one below.</p>
        )}
        <ParserSelectionPanel
          inspection={null}
          preview={preview}
          selectedCandidateKey={selectedKey}
          handlerDecisionRef={null}
          submitting={actionPending}
          onSelect={(candidate) => setSelectedKey(candidateKey(candidate))}
          onRecordDecision={() => selectedCandidate && onSelectHandler(selectedCandidate)}
          rerun={atHandlerStop ? undefined : {
            pending: blocked,
            disabledReason: blockedReason,
            onRerun: (candidate) => onRerun(request({ parser: candidate ?? "choose" })),
          }}
          compact
        />
      </section>

      <section aria-labelledby="review-repair-heading" className="space-y-1.5 border-t pt-3" data-testid="review-actions-repair">
        <SectionTitle id="review-repair-heading">Repair</SectionTitle>
        {/* Read from this run's own history (the engine's repair.reentry receipt); the gate
            controls below stay, because the owner still answers this run's gate (decision 4A). */}
        <RepairedLink stages={snapshot.stages} mode={snapshot.matter_mode} onOpenRun={onOpenRun} />
        {failure?.repairCheck && (
          <div className="flex items-center gap-2 text-xs">
            <AlertTriangle className="size-3.5 shrink-0 text-destructive" aria-hidden="true" />
            <span className="min-w-0 flex-1 truncate" title={snapshot.reason || failure.text}>Repair check failed: {failure.text}</span>
            <Button type="button" size="sm" variant="outline" className="h-7 px-2 text-xs" disabled={blocked} onClick={() => onRerun(request())}>
              <RotateCcw className="size-3.5" /> Retry
            </Button>
          </div>
        )}
        <StatusLine label="Assessment" value={snapshot.repair_state.assessment_report} />
        <StatusLine label="Affected parts" value={snapshot.repair_state.affected_units} />
        <fieldset className="space-y-1 text-xs">
          <legend className="sr-only">Repair choice</legend>
          <label className="flex items-start gap-2">
            <input type="radio" name="review-repair-choice" className="mt-0.5" checked={repairChoice === "original"} onChange={() => setRepairChoice("original")} />
            <span>Continue without repair, using the kept original</span>
          </label>
        </fieldset>
        {atRepairStop ? (
          <Button type="button" size="sm" className="w-full" disabled={actionPending} onClick={onRetainOriginal}>
            <ShieldCheck className="size-3.5" /> Retain sealed original and continue
          </Button>
        ) : (
          <Button type="button" size="sm" variant="outline" className="w-full" disabled={blocked || repairChoice !== "original"} onClick={() => onRerun(request({ repair: "original" }))}>
            <RotateCcw className="size-3.5" /> Re-run with this choice
          </Button>
        )}
        <RepairBuilder snapshot={snapshot} onOpenRun={onOpenRun} />
      </section>

      <section aria-labelledby="review-process-heading" className="space-y-1.5 border-t pt-3" data-testid="review-actions-process">
        <SectionTitle id="review-process-heading">Process</SectionTitle>
        {failure && !failure.repairCheck && (
          <p className="flex items-center gap-2 text-xs" title={snapshot.reason || failure.text}>
            <AlertTriangle className="size-3.5 shrink-0 text-destructive" aria-hidden="true" />
            <span className="min-w-0 truncate">Stopped at {failure.stage || "an unreported stage"}: {failure.text}</span>
          </p>
        )}
        {(pendingAnswers?.handler || pendingAnswers?.repair) && (
          <p className="text-[11px] text-muted-foreground">
            This run will answer its own stops: {[pendingAnswers.handler ? `parser ${pendingAnswers.handler.handler_id}` : null, pendingAnswers.repair ? "continue without repair" : null].filter(Boolean).join(" · ")}
          </p>
        )}
        <p className="text-[11px] text-muted-foreground" title="What a re-run carries">Carries: {carries}</p>
        <Button type="button" className="w-full" disabled={blocked} onClick={() => onRerun(request())} data-testid="review-rerun">
          {rerunPending ? <Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> : <RotateCcw className="size-4" />} Re-run this source
        </Button>
        {blockedReason && <p className="text-[11px] text-muted-foreground">{blockedReason}</p>}
      </section>
    </aside>
  );
}
