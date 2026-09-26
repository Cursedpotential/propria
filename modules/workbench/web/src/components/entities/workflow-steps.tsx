// Byline: Claude Code · Opus 5.5 · 2026-09-25 (step list for the extraction and commit workflows)
"use client";

import { Check, CircleDashed, Flag, LoaderCircle, MinusCircle, X } from "lucide-react";

import type { ValidationReport, WorkflowProgress } from "@/lib/entity-extraction-client";
import { cn } from "@/lib/utils";

const STEP_LABEL: Record<string, string> = {
  rules: "Participants (rules)",
  model: "Names, places and events (model)",
  reconcile: "Group aliases",
  validate: "Validate",
  entities: "Write entities",
  aliases: "Write aliases",
  mentions: "Link mentions",
  events: "Write events",
  timeline_members: "Add to timeline",
  finalize: "Record receipt",
  projection: "Timesketch projection",
  workflow: "Workflow",
};

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case "completed":
      return <Check className="size-3.5 text-[#2f9d67]" aria-label="done" />;
    case "failed":
      return <X className="size-3.5 text-destructive" aria-label="failed" />;
    case "running":
      return <LoaderCircle className="size-3.5 animate-spin text-primary" aria-label="running" />;
    case "skipped":
      return <MinusCircle className="size-3.5 text-muted-foreground" aria-label="skipped" />;
    default:
      return <CircleDashed className="size-3.5 text-muted-foreground" aria-label="waiting" />;
  }
}

export function WorkflowSteps({ progress, title }: { progress: WorkflowProgress | undefined; title: string }) {
  if (!progress) return null;
  return (
    <section aria-label={title} className="border bg-card p-2 text-xs" data-testid={`workflow-steps-${title.toLowerCase().replaceAll(" ", "-")}`}>
      <p className="mb-1 font-semibold">
        {title} <span className="font-normal text-muted-foreground">· {progress.outcome.replaceAll("_", " ")}</span>
      </p>
      <ol className="space-y-1">
        {progress.steps.map((step) => (
          <li key={step.step} className="flex items-start gap-2">
            <span className="mt-0.5 shrink-0"><StatusIcon status={step.status} /></span>
            <span className="min-w-0">
              <span className={cn(step.status === "failed" && "text-destructive")}>{STEP_LABEL[step.step] ?? step.step}</span>
              {step.detail && <span className="text-muted-foreground"> — {step.detail}</span>}
              {step.flags.map((flag, index) => (
                <span key={`${flag.code}-${index}`} className="mt-0.5 flex items-start gap-1 text-[11px] text-[#8a5a00] dark:text-[#ffd48a]">
                  <Flag className="mt-0.5 size-3 shrink-0" aria-hidden="true" />
                  {flag.detail}
                </span>
              ))}
            </span>
          </li>
        ))}
      </ol>
    </section>
  );
}

const RULE_LABEL: Record<string, string> = {
  live_mode: "Live (REAL) run",
  has_proposals: "Something to commit",
  current_generation: "Proposals match the run's current data",
  names_and_types: "Every entity has a name and type",
  aliases_unique: "Every alias belongs to one entity",
  match_targets_live: "Merge targets still exist",
  no_duplicate_committed: "No duplicate of a committed entity",
  mentions_resolve: "Every mention points at a message of this run",
  events_have_time: "Every event has a time",
  events_have_sources: "Every event has a source message",
  events_source_clock: "Every event carries its sources' availability",
  event_entities_resolve: "Every entity an event names exists",
  events_unique: "No duplicate events",
  bounded: "Within size bounds",
};

export function ValidationChecklist({ report }: { report: ValidationReport | undefined }) {
  if (!report) return null;
  return (
    <section aria-label="Validation" className="border bg-card p-2 text-xs" data-testid="entity-validation-checklist">
      <p className="mb-1 font-semibold">
        Validation <span className={cn("font-normal", report.ok ? "text-[#2f9d67]" : "text-destructive")}>· {report.ok ? "all checks pass" : "fix the failing checks"}</span>
      </p>
      <ul className="space-y-1">
        {report.checks.map((check) => (
          <li key={check.rule} className="flex items-start gap-2">
            <span className="mt-0.5 shrink-0">
              {check.status === "pass" ? <Check className="size-3.5 text-[#2f9d67]" aria-label="pass" /> : <X className="size-3.5 text-destructive" aria-label="fail" />}
            </span>
            <span>
              {RULE_LABEL[check.rule] ?? check.rule}
              {check.status === "fail" && <span className="block text-muted-foreground">{check.reason}</span>}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
