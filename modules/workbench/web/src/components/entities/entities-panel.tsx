// Byline: Claude Code · Opus 5.5 · 2026-09-25 (Entities & events: propose -> correct -> commit, one entry point)
"use client";

import { Flag, GitMerge, Play, ScanSearch, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";

import { EntityCard } from "@/components/entities/entity-card";
import { EventCard } from "@/components/entities/event-card";
import { formatWhen } from "@/components/entities/record-peek";
import { ValidationChecklist, WorkflowSteps } from "@/components/entities/workflow-steps";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useEntityActions, useEntityProposals, useEntityWritesPending, useWorkflowProgress } from "@/hooks/use-entity-extraction";
import {
  newIdempotencyKey,
  type ExtractionRunSummary,
  type ProposalsResponse,
  type ValidationReport,
  type WorkflowProgress,
} from "@/lib/entity-extraction-client";
import type { MatterMode } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

const STATE_ORDER: Record<string, number> = { pending: 0, approved: 1, rejected: 2 };
const OWNER_RUNS = new Set(["owner.review", "probata.extraction.commit"]);

/** One line per extractor: its newest run, with any model flags. */
function latestRuns(runs: ExtractionRunSummary[]) {
  const newest = new Map<string, ExtractionRunSummary>();
  for (const run of runs) {
    if (!OWNER_RUNS.has(run.extractor) && !newest.has(run.extractor)) newest.set(run.extractor, run);
  }
  return [...newest.values()];
}

function invalidBatches(run: ExtractionRunSummary) {
  const value = run.stats.invalid_batches;
  return Array.isArray(value) ? (value as Array<{ first_ordinal: number; last_ordinal: number; reason: string }>) : [];
}

/** Running from the click until the workflow reports an outcome. */
function inFlight(workflowId: string | null, progress: { data?: WorkflowProgress; error: unknown }) {
  if (!workflowId || progress.error) return false;
  return !progress.data || progress.data.outcome === "running";
}

/**
 * Review's Entities tab. Extraction only proposes; nothing reaches the
 * registry or the timeline until the owner validates and runs the workflow.
 * A validation belongs to the proposals it read: any change hides it and the
 * workflow needs a fresh one (the engine also refuses a stale digest).
 */
export function EntitiesPanel({ previewHandle, mode }: { previewHandle: string; mode: MatterMode }) {
  const proposals = useEntityProposals(previewHandle, mode);
  const actions = useEntityActions(previewHandle, mode);
  const writing = useEntityWritesPending(previewHandle, mode);
  const [extractionId, setExtractionId] = useState<string | null>(null);
  const [commitId, setCommitId] = useState<string | null>(null);
  const extraction = useWorkflowProgress("extraction", extractionId, previewHandle, mode);
  const commit = useWorkflowProgress("commit", commitId, previewHandle, mode);
  const [view, setView] = useState<"entities" | "events">("entities");
  const [useModel, setUseModel] = useState(true);
  const [validation, setValidation] = useState<{ report: ValidationReport; basis: ProposalsResponse | undefined } | null>(null);
  const [selectedEntities, setSelectedEntities] = useState<string[]>([]);
  const [selectedEvents, setSelectedEvents] = useState<string[]>([]);
  const [mergeName, setMergeName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const data = proposals.data;
  const report = validation && validation.basis === data && !writing ? validation.report : undefined;

  const entities = useMemo(() => [...(data?.entities ?? [])].sort((left, right) =>
    (STATE_ORDER[left.review_state] ?? 3) - (STATE_ORDER[right.review_state] ?? 3) || right.mention_count - left.mention_count), [data]);
  const events = useMemo(() => [...(data?.events ?? [])].sort((left, right) =>
    (STATE_ORDER[left.review_state] ?? 3) - (STATE_ORDER[right.review_state] ?? 3) || (left.occurred_at ?? "").localeCompare(right.occurred_at ?? "")), [data]);
  const runs = useMemo(() => latestRuns(data?.extractions ?? []), [data]);
  const pendingEntities = entities.filter((entity) => entity.review_state === "pending").length;
  const pendingEvents = events.filter((event) => event.review_state === "pending").length;
  const extracting = actions.extract.isPending || inFlight(extractionId, extraction);
  const committing = actions.commit.isPending || inFlight(commitId, commit);
  const selected = view === "entities" ? selectedEntities : selectedEvents;

  const run = async (action: () => Promise<unknown>) => {
    setError(null);
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    }
  };

  const extract = () => run(async () => {
    const started = await actions.extract.mutateAsync({ useModel, key: newIdempotencyKey("extract") });
    setExtractionId(started.workflow_id);
  });
  const validate = () => run(async () => {
    const basis = data;
    setValidation({ report: await actions.validate.mutateAsync(), basis });
  });
  const commitNow = () => run(async () => {
    if (!report?.ok) return;
    const started = await actions.commit.mutateAsync({ digest: report.digest, key: newIdempotencyKey("commit") });
    setCommitId(started.workflow_id);
    setValidation(null);
  });
  const merge = () => run(async () => {
    const name = mergeName.trim() || undefined;
    if (view === "entities") {
      await actions.correctEntity({ op: "merge", candidate_ids: selectedEntities, name });
      setSelectedEntities([]);
    } else {
      await actions.correctEvent({ op: "merge", candidate_ids: selectedEvents, title: name });
      setSelectedEvents([]);
    }
    setMergeName("");
  });
  const toggle = (setter: typeof setSelectedEntities, id: string) => (on: boolean) =>
    setter((current) => (on ? [...current, id] : current.filter((value) => value !== id)));

  return (
    <div className="space-y-2" data-testid="entities-panel">
      <header className="flex flex-wrap items-center gap-2 border bg-card px-3 py-2">
        <h2 className="text-sm font-semibold">Entities and events</h2>
        <label className="ml-auto inline-flex items-center gap-1 text-xs" title="Also read message text with the extraction model: names, places, organizations and events">
          <input type="checkbox" checked={useModel} onChange={(change) => setUseModel(change.target.checked)} /> Read message text
        </label>
        <Button type="button" size="sm" onClick={() => void extract()} disabled={extracting || committing} data-testid="extract-entities-button">
          <ScanSearch /> {extracting ? "Extracting…" : "Extract entities"}
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={() => void validate()} disabled={!data || writing || actions.validate.isPending || committing}>
          <ShieldCheck /> Validate
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => void commitNow()}
          disabled={!report?.ok || writing || committing}
          title={report?.ok ? "Commit these proposals to the registry and the timeline" : "Validate first; every check must pass"}
          data-testid="run-entity-workflow-button"
        >
          <Play /> Run workflow
        </Button>
      </header>

      {runs.length > 0 && (
        <p className="flex flex-wrap gap-x-4 gap-y-1 px-1 text-[11px] text-muted-foreground">
          {runs.map((extractionRun) => (
            <span key={extractionRun.id} className="inline-flex items-center gap-1">
              {extractionRun.extractor.replace("probata.", "")} · {extractionRun.status} · {formatWhen(extractionRun.finished_at ?? extractionRun.started_at)}
              {invalidBatches(extractionRun).length > 0 && (
                <span
                  className="text-[#8a5a00] dark:text-[#ffd48a]"
                  title={invalidBatches(extractionRun).map((batch) => `messages #${batch.first_ordinal}–#${batch.last_ordinal}: ${batch.reason}`).join("\n")}
                >
                  <Flag className="inline size-3" aria-label="some model output failed validation and was not used" />
                </span>
              )}
            </span>
          ))}
        </p>
      )}

      {(extraction.data || commit.data) && (
        <div className="grid gap-2 lg:grid-cols-2">
          <WorkflowSteps progress={extraction.data} title="Extraction" />
          <WorkflowSteps progress={commit.data} title="Commit" />
        </div>
      )}
      <ValidationChecklist report={report} />
      {(error || proposals.error || extraction.error || commit.error) && (
        <p className="flex items-start gap-1 px-1 text-xs text-destructive" role="alert">
          <Flag className="mt-0.5 size-3 shrink-0" aria-hidden="true" />
          {error ?? ((proposals.error ?? extraction.error ?? commit.error) as Error).message}
        </p>
      )}

      <nav className="flex gap-1 border-b" role="tablist" aria-label="Proposal views">
        {([["entities", `People, places and organizations (${pendingEntities})`], ["events", `Events (${pendingEvents})`]] as const).map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={view === id}
            onClick={() => setView(id)}
            className={cn("border-b-2 px-3 py-1.5 text-xs font-semibold", view === id ? "border-primary text-primary" : "border-transparent text-muted-foreground")}
          >
            {label}
          </button>
        ))}
      </nav>

      {selected.length > 1 && (
        <div className="flex flex-wrap items-center gap-2 border bg-accent/30 px-2 py-1 text-xs">
          <GitMerge className="size-3.5" aria-hidden="true" />
          Merge {selected.length} into one
          <Input
            aria-label={view === "entities" ? "Name after merging" : "Title after merging"}
            className="h-7 w-48 text-xs"
            placeholder={view === "entities" ? "name (optional)" : "title (optional)"}
            maxLength={200}
            value={mergeName}
            onChange={(change) => setMergeName(change.target.value)}
          />
          <Button type="button" size="xs" onClick={() => void merge()} disabled={writing}>Merge</Button>
          <Button type="button" size="xs" variant="ghost" onClick={() => (view === "entities" ? setSelectedEntities([]) : setSelectedEvents([]))}>Clear</Button>
        </div>
      )}

      {proposals.isLoading && <p className="text-xs text-muted-foreground">Loading proposals…</p>}
      {data && view === "entities" && (entities.length === 0 ? (
        <p className="border p-4 text-center text-xs text-muted-foreground">No proposals yet. Extract entities to propose the people, places and organizations in this run.</p>
      ) : (
        <ul className="space-y-1">
          {entities.map((proposal) => (
            <EntityCard
              key={proposal.candidate_id}
              proposal={proposal}
              previewHandle={previewHandle}
              mode={mode}
              entityTypes={data.entity_types}
              aliasKinds={data.alias_kinds}
              selected={selectedEntities.includes(proposal.candidate_id)}
              onSelect={toggle(setSelectedEntities, proposal.candidate_id)}
              actions={actions}
            />
          ))}
        </ul>
      ))}
      {data && view === "events" && (events.length === 0 ? (
        <p className="border p-4 text-center text-xs text-muted-foreground">No events proposed. Extraction proposes them from message text; open any message to mark one worth recalling.</p>
      ) : (
        <ul className="space-y-1">
          {events.map((event) => (
            <EventCard
              key={event.candidate_id}
              event={event}
              previewHandle={previewHandle}
              mode={mode}
              eventTypes={data.event_types}
              entities={entities}
              selected={selectedEvents.includes(event.candidate_id)}
              onSelect={toggle(setSelectedEvents, event.candidate_id)}
              actions={actions}
            />
          ))}
        </ul>
      ))}
    </div>
  );
}
