// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Read views for the Review Actions repair builder: the validator's checklist, one repair
// run's per-step progress and receipts, where the repaired result re-entered (one run, or
// one batch of runs), and the "Repaired → <new run>" line a repaired run's history carries.
"use client";

import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, Circle, Loader2, MinusCircle, XCircle } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import { getProfferBatch, getProfferRepairRun } from "@/lib/api-client";
import { refName, repairReentryFromStages, ruleLabel, summaryLine, toolLabel } from "@/lib/repair-plan";
import type {
  MatterMode,
  ProfferBatchStatus,
  ProfferOperationStage,
  ProfferRepairCheck,
  ProfferRepairRunStatus,
  ProfferRepairStepState,
} from "@/lib/shared/types";
import { cn } from "@/lib/utils";

type OpenRun = (previewHandle: string) => void;

/** Every named check inline: pass or fail, with the engine's own reason. */
export function RepairCheckList({ checks }: { checks: readonly ProfferRepairCheck[] }) {
  return (
    <ul className="space-y-1" aria-label="Plan checks" data-testid="repair-checks">
      {checks.map((check) => (
        <li key={check.rule} className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-1.5 text-xs" data-status={check.status}>
          {check.status === "pass" ? (
            <CheckCircle2 className="mt-0.5 size-3.5 text-emerald-600" aria-hidden="true" />
          ) : (
            <XCircle className="mt-0.5 size-3.5 text-destructive" aria-hidden="true" />
          )}
          <span className="min-w-0">
            <span className="font-medium">{ruleLabel(check.rule)}</span>
            <span className="sr-only"> {check.status === "pass" ? "passed" : "failed"}</span>
            {check.reason && (
              <span className={cn("block break-words text-[11px]", check.status === "pass" ? "text-muted-foreground" : "text-destructive")}>
                {check.reason}
              </span>
            )}
          </span>
        </li>
      ))}
    </ul>
  );
}

const STEP_ICON: Record<ProfferRepairStepState, typeof Circle> = {
  pending: Circle,
  running: Loader2,
  succeeded: CheckCircle2,
  failed: XCircle,
  skipped: MinusCircle,
};

const STEP_TONE: Record<ProfferRepairStepState, string> = {
  pending: "text-muted-foreground",
  running: "animate-spin text-primary motion-reduce:animate-none",
  succeeded: "text-emerald-600",
  failed: "text-destructive",
  skipped: "text-muted-foreground",
};

const RUN_LABEL: Record<ProfferRepairRunStatus["status"], string> = {
  running: "running",
  completed: "finished",
  failed: "stopped",
};

/** One repair run: each step's state, output, counts and receipt, then where the result went. */
export function RepairRunView({ status, onOpenRun }: { status: ProfferRepairRunStatus; onOpenRun: OpenRun }) {
  return (
    <div className="space-y-2" data-testid="repair-run" data-status={status.status}>
      <p className="text-xs">
        <span className="font-medium">Repair run {RUN_LABEL[status.status]}</span>
        {status.reason && <span className="block break-words text-[11px] text-destructive">{status.reason}</span>}
      </p>
      <ol className="space-y-1.5" aria-label="Repair steps">
        {status.steps.map((step) => {
          const Icon = STEP_ICON[step.status];
          const counts = summaryLine(step.summary);
          return (
            <li key={step.step_id} className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-1.5 text-xs" data-status={step.status}>
              <Icon className={cn("mt-0.5 size-3.5", STEP_TONE[step.status])} aria-hidden="true" />
              <div className="min-w-0">
                <p className="truncate" title={step.activity}>
                  <span className="font-mono text-[10px] text-muted-foreground">{step.step_id}</span> {toolLabel(step.activity)}
                  <span className="text-muted-foreground"> · {step.status}</span>
                </p>
                {step.reason && <p className="break-words text-[11px] text-destructive">{step.reason}</p>}
                {step.output_ref && (
                  <p className="truncate font-mono text-[10px]" title={step.output_ref}>
                    → {refName(step.output_ref)}
                    {step.output_sha256 && <span className="text-muted-foreground"> · sha256 {step.output_sha256.slice(0, 12)}</span>}
                  </p>
                )}
                {counts && <p className="text-[11px] text-muted-foreground">{counts}</p>}
                {step.receipt_ref && (
                  <p className="truncate font-mono text-[10px] text-muted-foreground" title={step.receipt_ref}>receipt {step.receipt_ref}</p>
                )}
              </div>
            </li>
          );
        })}
      </ol>
      {status.checks.length > 0 && <RepairCheckList checks={status.checks} />}
      {status.reentry_preview_handle && (
        <RepairedRunLine lead="Re-entered →" handle={status.reentry_preview_handle} name={lastOutputName(status)} onOpenRun={onOpenRun} />
      )}
      {status.reentry_batch_id && (
        <RepairBatchRuns mode={status.matter_mode} batchId={status.reentry_batch_id} lead="Re-entered →" onOpenRun={onOpenRun} />
      )}
      {status.reentry_receipt_ref && (
        <p className="truncate font-mono text-[10px] text-muted-foreground" title={status.reentry_receipt_ref}>
          re-entry receipt {status.reentry_receipt_ref}
        </p>
      )}
    </div>
  );
}

function lastOutputName(status: ProfferRepairRunStatus) {
  const produced = [...status.steps].reverse().find((step) => step.status === "succeeded" && step.output_ref);
  return refName(produced?.output_ref);
}

/** "Repaired → <new run>": a link that opens the re-entered run in Review. */
export function RepairedRunLine({
  lead,
  handle,
  name,
  fallback,
  error,
  onOpenRun,
}: {
  lead: string;
  handle: string | null | undefined;
  name?: string;
  fallback?: string;
  error?: string;
  onOpenRun: OpenRun;
}) {
  return (
    <p className="flex min-w-0 items-center gap-1 text-xs" data-testid="repaired-link" title={error || undefined}>
      <span className="shrink-0 font-medium">{lead}</span>
      {handle ? (
        <Button
          type="button"
          variant="link"
          size="xs"
          className="h-auto min-w-0 justify-start truncate p-0 text-xs"
          title={`Open run ${handle} in Review`}
          onClick={() => onOpenRun(handle)}
        >
          {name ? `${name} · run ${handle.slice(0, 8)}` : `run ${handle.slice(0, 8)}`}
        </Button>
      ) : (
        <span className="min-w-0 truncate font-mono text-[10px] text-muted-foreground">{fallback ?? "new run"}</span>
      )}
    </p>
  );
}

/** One batch's runs, each opening in Review once it is bound. */
export function BatchRunList({ batch, lead, onOpenRun }: { batch: ProfferBatchStatus; lead: string; onOpenRun: OpenRun }) {
  const shown = batch.items.slice(0, 20);
  const { counts } = batch;
  return (
    <div className="space-y-1 text-xs" data-testid="repair-batch-runs">
      <p>
        <span className="font-medium">{lead}</span> {counts.total} run{counts.total === 1 ? "" : "s"}
        <span className="text-muted-foreground">
          {" "}· {counts.done} done · {counts.queued + counts.running} in progress · {counts.waiting_on_gate} waiting · {counts.failed} failed
        </span>
      </p>
      <ul className="space-y-0.5">
        {shown.map((item) => (
          <li key={item.key} className="flex min-w-0 items-center gap-1">
            {item.preview_handle ? (
              <Button
                type="button"
                variant="link"
                size="xs"
                className="h-auto min-w-0 justify-start truncate p-0 text-xs"
                title={`Open run ${item.preview_handle} in Review`}
                onClick={() => onOpenRun(item.preview_handle)}
              >
                {refName(item.source_ref || item.key)}
              </Button>
            ) : (
              <span className="min-w-0 truncate">{refName(item.source_ref || item.key)}</span>
            )}
            <span className="shrink-0 text-muted-foreground">· {item.status.replaceAll("_", " ")}</span>
          </li>
        ))}
      </ul>
      {batch.items.length > shown.length && <p className="text-[11px] text-muted-foreground">+{batch.items.length - shown.length} more</p>}
    </div>
  );
}

export function RepairBatchRuns({ mode, batchId, lead, onOpenRun }: { mode: MatterMode; batchId: string; lead: string; onOpenRun: OpenRun }) {
  const query = useQuery({
    queryKey: ["proffer-batch", mode, batchId],
    queryFn: ({ signal }) => getProfferBatch(batchId, mode, signal),
    retry: false,
    refetchInterval: (current) => (current.state.data && !current.state.data.terminal ? 5_000 : false),
  });
  if (!query.data) {
    return (
      <p className="flex min-w-0 items-center gap-1 text-xs" title={query.error?.message}>
        <span className="shrink-0 font-medium">{lead}</span>
        <span className="min-w-0 truncate font-mono text-[10px] text-muted-foreground">batch {batchId}</span>
      </p>
    );
  }
  return <BatchRunList batch={query.data} lead={lead} onOpenRun={onOpenRun} />;
}

function RepairedRun({ mode, repairWorkflowId, onOpenRun }: { mode: MatterMode; repairWorkflowId: string; onOpenRun: OpenRun }) {
  const query = useQuery({
    queryKey: ["proffer-repair-run", mode, repairWorkflowId],
    queryFn: ({ signal }) => getProfferRepairRun(repairWorkflowId, mode, signal),
    retry: false,
    staleTime: 60_000,
  });
  return (
    <RepairedRunLine
      lead="Repaired →"
      handle={query.data?.reentry_preview_handle}
      name={query.data ? lastOutputName(query.data) : undefined}
      fallback={query.isPending ? "…" : "new run"}
      error={query.error?.message}
      onOpenRun={onOpenRun}
    />
  );
}

/**
 * A run a repair plan re-entered shows "Repaired → <new run>". The link is read from the run's
 * own operation history (the engine's `repair.reentry` receipt); nothing renders without one.
 * The run's repair gate is not closed by this: the owner still answers it (decision 4A).
 */
export function RepairedLink({ stages, mode, onOpenRun }: { stages: readonly ProfferOperationStage[]; mode: MatterMode; onOpenRun: OpenRun }) {
  const link = useMemo(() => repairReentryFromStages(stages), [stages]);
  if (!link) return null;
  return link.kind === "run" ? (
    <RepairedRun mode={mode} repairWorkflowId={link.repairWorkflowId} onOpenRun={onOpenRun} />
  ) : (
    <RepairBatchRuns mode={mode} batchId={link.batchId} lead="Repaired →" onOpenRun={onOpenRun} />
  );
}
