// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Repair builder, in Review Actions → Repair (replaces the disabled "Apply the proposed repair").
// Owner 2026-09-25 18:25: "Apply proposed repair … should propose options based on file name and
// then allow me to build out the workflow; it should validate that the workflow will follow the
// designated rules … allow me to run that workflow, run against Temporal". Ratified option A.
//
//   Propose  -> POST /api/proffer/repair/propose   the engine's proposals for this run's source
//   Compose  -> an editable step list drawn from GET /api/proffer/repair/tools
//   Validate -> POST /api/proffer/repair/validate  every named check, pass or fail, with its reason
//   Run      -> POST /api/proffer/repair/run, then GET /api/proffer/repair/runs/{id} until it ends
//
// The engine decides every proposal, tool and check; nothing here is invented or pre-judged.
"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2, Play, RotateCcw, ShieldCheck, Wand2 } from "lucide-react";
import { useState } from "react";

import { RepairStepEditor } from "@/components/sbv/repair-plan-steps";
import { RepairCheckList, RepairRunView } from "@/components/sbv/repair-run-view";
import { Button } from "@/components/ui/button";
import {
  RepairRunRefusedError,
  getProfferRepairRun,
  listProfferRepairTools,
  proposeProfferRepairs,
  runProfferRepairPlan,
  validateProfferRepairPlan,
} from "@/lib/api-client";
import {
  draftFromProposal,
  newPlanId,
  noRepairLine,
  noStepsProposed,
  planFromDraft,
  planKey,
  toolLabel,
  type RepairDraftStep,
} from "@/lib/repair-plan";
import type {
  ProfferOperatorSnapshot,
  ProfferRepairCheck,
  ProfferRepairPlan,
  ProfferRepairProposal,
  ProfferRepairProposeResponse,
  ProfferRepairValidateResponse,
} from "@/lib/shared/types";

/** The engine's proposals: each with its rationale and steps; a stepless one is the "wait" option. */
export function RepairProposalList({
  response,
  onUse,
}: {
  response: ProfferRepairProposeResponse;
  onUse: (proposal: ProfferRepairProposal) => void;
}) {
  return (
    <div className="space-y-1.5" data-testid="repair-proposals">
      {noStepsProposed(response.proposals) && <p className="text-xs" role="status">{noRepairLine(response.signature)}</p>}
      {response.proposals.length > 0 && (
        <ol className="space-y-1" aria-label="Proposed repairs">
          {response.proposals.map((proposal, index) => (
            <li key={`${index}:${proposal.steps.map((step) => step.activity).join(",")}`} className="space-y-1 border p-1.5">
              <p className="text-[11px] leading-4">{proposal.rationale}</p>
              {proposal.steps.length > 0 && (
                <div className="flex flex-wrap items-center gap-1">
                  {proposal.steps.map((step) => (
                    <span key={step.step_id} className="rounded-sm border px-1 text-[10px]" title={step.activity}>
                      {toolLabel(step.activity)}
                    </span>
                  ))}
                  <Button type="button" variant="secondary" size="xs" className="ml-auto" onClick={() => onUse(proposal)}>
                    Use this plan
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}

export function RepairBuilder({
  snapshot,
  onOpenRun,
}: {
  snapshot: ProfferOperatorSnapshot;
  onOpenRun: (previewHandle: string) => void;
}) {
  const mode = snapshot.matter_mode;
  const [draft, setDraft] = useState<RepairDraftStep[] | null>(null);
  const [planId, setPlanId] = useState("");
  const [validation, setValidation] = useState<{ key: string; result: ProfferRepairValidateResponse } | null>(null);
  const [refusal, setRefusal] = useState<{ key: string; detail: string; checks: ProfferRepairCheck[] } | null>(null);
  const [workflowId, setWorkflowId] = useState<string | null>(null);

  const validate = useMutation({
    mutationFn: (candidate: ProfferRepairPlan) => validateProfferRepairPlan(candidate),
    onSuccess: (result, candidate) => setValidation({ key: planKey(candidate), result }),
  });
  const start = useMutation({
    mutationFn: (candidate: ProfferRepairPlan) => runProfferRepairPlan(candidate),
    onSuccess: (response) => setWorkflowId(response.workflow_id),
    onError: (error, candidate) => {
      if (error instanceof RepairRunRefusedError) setRefusal({ key: planKey(candidate), detail: error.message, checks: error.checks });
    },
  });
  const run = useQuery({
    queryKey: ["proffer-repair-run", mode, workflowId],
    queryFn: ({ signal }) => getProfferRepairRun(workflowId ?? "", mode, signal),
    enabled: Boolean(workflowId),
    retry: 2,
    // Poll while the run is in flight; stop once it ends or cannot be read.
    refetchInterval: (current) =>
      current.state.status === "error" || (current.state.data && current.state.data.status !== "running") ? false : 2_000,
  });

  function openDraft(steps: RepairDraftStep[]) {
    setDraft(steps);
    setPlanId(newPlanId(snapshot.preview_handle));
    setValidation(null);
    setRefusal(null);
    setWorkflowId(null);
    validate.reset();
    start.reset();
  }

  const propose = useMutation({
    mutationFn: () => proposeProfferRepairs(mode, { source_ref: snapshot.source_ref, preview_handle: snapshot.preview_handle }),
    // Nothing to run: open the step builder on the full tool list straight away.
    onSuccess: (response) => {
      if (noStepsProposed(response.proposals)) openDraft([]);
    },
  });
  const tools = useQuery({
    queryKey: ["proffer-repair-tools", mode],
    queryFn: ({ signal }) => listProfferRepairTools(mode, signal),
    enabled: draft !== null,
    retry: false,
    staleTime: 5 * 60_000,
  });

  const plan = draft
    ? planFromDraft({ planId, sourceRef: snapshot.source_ref, previewHandle: snapshot.preview_handle, mode, steps: draft })
    : null;
  const key = plan ? planKey(plan) : "";
  // A result belongs to the exact plan it judged; any edit makes it stale and hides it.
  const checked = validation && validation.key === key ? validation.result : null;
  const refused = refusal && refusal.key === key ? refusal : null;
  const runEnded = Boolean(run.data && run.data.status !== "running");

  return (
    <div className="space-y-2 border-t pt-2" data-testid="repair-builder">
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-xs font-semibold">Repair plan</h4>
        <Button type="button" variant="outline" size="xs" disabled={propose.isPending} onClick={() => propose.mutate()}>
          {propose.isPending ? <Loader2 className="animate-spin motion-reduce:animate-none" /> : <Wand2 />} Propose repairs
        </Button>
      </div>
      {propose.error && <p className="text-[11px] text-destructive" role="status">{propose.error.message}</p>}
      {propose.data && <RepairProposalList response={propose.data} onUse={(proposal) => openDraft(draftFromProposal(proposal))} />}

      {draft && !workflowId && (
        <div className="space-y-1.5">
          <RepairStepEditor
            steps={draft}
            tools={tools.data?.tools ?? []}
            toolsLoading={tools.isPending}
            toolsError={tools.error ? tools.error.message : null}
            disabled={start.isPending}
            onChange={setDraft}
          />
          {draft.length > 0 && (
            <div className="flex items-center gap-1.5">
              <Button
                type="button"
                variant="outline"
                size="xs"
                disabled={!plan || validate.isPending || start.isPending}
                onClick={() => plan && validate.mutate(plan)}
              >
                {validate.isPending ? <Loader2 className="animate-spin motion-reduce:animate-none" /> : <ShieldCheck />} Validate
              </Button>
              <Button
                type="button"
                size="xs"
                disabled={!plan || !checked?.ok || start.isPending}
                title={checked?.ok ? undefined : "Run starts once every check passes"}
                onClick={() => plan && start.mutate(plan)}
                data-testid="repair-run-button"
              >
                {start.isPending ? <Loader2 className="animate-spin motion-reduce:animate-none" /> : <Play />} Run
              </Button>
            </div>
          )}
          {validate.error && <p className="text-[11px] text-destructive" role="status">{validate.error.message}</p>}
          {checked && <RepairCheckList checks={checked.checks} />}
          {refused && (
            <>
              <p className="break-words text-[11px] text-destructive" role="status">{refused.detail}</p>
              <RepairCheckList checks={refused.checks} />
            </>
          )}
          {start.error && !(start.error instanceof RepairRunRefusedError) && (
            <p className="text-[11px] text-destructive" role="status">{start.error.message}</p>
          )}
        </div>
      )}

      {workflowId &&
        (run.data ? (
          <RepairRunView status={run.data} onOpenRun={onOpenRun} />
        ) : (
          <p className="flex items-center gap-1.5 text-xs" role="status">
            {run.error ? (
              <>
                <span className="min-w-0 flex-1 text-destructive">{run.error.message}</span>
                <Button type="button" variant="ghost" size="xs" onClick={() => void run.refetch()}>
                  <RotateCcw /> Check again
                </Button>
              </>
            ) : (
              <>
                <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /> Starting the repair run…
              </>
            )}
          </p>
        ))}
      {workflowId && runEnded && (
        <Button type="button" variant="ghost" size="xs" onClick={() => openDraft(draft ?? [])}>
          <RotateCcw /> Build another plan
        </Button>
      )}
    </div>
  );
}
