// Byline: Claude Code · Opus 5.5 · 2026-09-25 (one proposed event: time, sources, entities, owner corrections)
"use client";

import { Flag, Pencil, RotateCcw, X } from "lucide-react";
import { useState } from "react";

import { RecordPeek, formatWhen } from "@/components/entities/record-peek";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { useEntityActions } from "@/hooks/use-entity-extraction";
import type { EntityProposal, EventProposal } from "@/lib/entity-extraction-client";
import type { MatterMode } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

type Actions = ReturnType<typeof useEntityActions>;

/** ISO (UTC) -> the browser's local "YYYY-MM-DDTHH:mm" for <input type=datetime-local>. */
function toLocalInput(iso: string | undefined) {
  if (!iso) return "";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** The key an event uses to name a proposal: its name key, else its first key. */
function referenceKey(proposal: EntityProposal) {
  return proposal.keys.find((key) => key.startsWith("name:")) ?? proposal.keys[0];
}

export function EventCard({
  event, previewHandle, mode, eventTypes, entities, selected, onSelect, actions,
}: {
  event: EventProposal;
  previewHandle: string;
  mode: MatterMode;
  eventTypes: string[];
  entities: EntityProposal[];
  selected: boolean;
  onSelect: (selected: boolean) => void;
  actions: Actions;
}) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(event.title);
  const [description, setDescription] = useState(event.description ?? "");
  const [when, setWhen] = useState(toLocalInput(event.occurred_at));
  const [chosen, setChosen] = useState<string[]>(event.resolved_entity_candidate_ids);
  const [peek, setPeek] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const editable = event.review_state === "pending";
  const id = event.candidate_id;
  const byId = new Map(entities.map((entity) => [entity.candidate_id, entity]));
  const involved = event.resolved_entity_candidate_ids.map((candidateId) => byId.get(candidateId)?.name ?? candidateId);

  const run = async (action: () => Promise<unknown>) => {
    setError(null);
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    }
  };

  const save = () => run(async () => {
    const keys = chosen.map((candidateId) => byId.get(candidateId)).filter((entity): entity is EntityProposal => Boolean(entity)).map(referenceKey);
    await actions.correctEvent({
      op: "edit", candidate_ids: [id], title: title.trim(), description,
      occurred_at: when ? new Date(when).toISOString() : undefined, set_entities: true, entity_keys: keys,
    });
    setEditing(false);
  });

  return (
    <li className={cn("border bg-card p-2 text-xs", event.review_state === "rejected" && "opacity-60")} data-testid="event-card">
      <div className="flex flex-wrap items-center gap-2">
        {editable && <input type="checkbox" aria-label={`Select ${event.title} to merge`} checked={selected} onChange={(change) => onSelect(change.target.checked)} />}
        <strong className="text-sm">{event.title}</strong>
        {editable ? (
          <select aria-label="Event type" className="h-7 border bg-background px-1 text-xs" value={event.event_type} onChange={(change) => void run(() => actions.correctEvent({ op: "edit", candidate_ids: [id], event_type: change.target.value }))}>
            {eventTypes.map((type) => <option key={type} value={type}>{type.replaceAll("_", " ")}</option>)}
          </select>
        ) : <Badge variant="outline">{event.event_type.replaceAll("_", " ")}</Badge>}
        <Badge variant={event.detected_by === "owner" ? "default" : "outline"}>{event.detected_by === "owner" ? "worth recalling" : "auto"}</Badge>
        {event.review_state === "approved" && <Badge>on the timeline</Badge>}
        {event.review_state === "rejected" && <Badge variant="outline">rejected</Badge>}
        {(event.flags ?? []).map((flag) => (
          <span key={flag.code} title={flag.detail} className="text-[#8a5a00] dark:text-[#ffd48a]"><Flag className="size-3.5" aria-label={flag.detail} /></span>
        ))}
        {(event.unresolved_entity_keys ?? []).length > 0 && (
          <span title={`Names no current entity covers: ${(event.unresolved_entity_keys ?? []).join(", ")}`} className="text-[#8a5a00] dark:text-[#ffd48a]"><Flag className="size-3.5" aria-label="unresolved entity" /></span>
        )}
      </div>
      <p className="mt-1">
        <span className="font-medium">{formatWhen(event.occurred_at)}</span>{" "}
        <span className="text-muted-foreground">
          {event.temporal_precision === "point" ? "exact" : "approximate — dated by the message"}
          {event.when_stated ? ` · stated “${event.when_stated}”` : ""}
          {event.source_available_from ? ` · visible from ${formatWhen(event.source_available_from)}` : ""}
        </span>
      </p>
      {event.description && !editing && <p className="mt-1 text-muted-foreground">{event.description}</p>}
      {involved.length > 0 && !editing && <p className="mt-1">Involves: {involved.join(", ")}</p>}

      {editing && (
        <form className="mt-2 space-y-1" onSubmit={(submit) => { submit.preventDefault(); void save(); }}>
          <Input aria-label="Event title" className="h-7 text-xs" value={title} maxLength={200} onChange={(change) => setTitle(change.target.value)} />
          <textarea aria-label="Event description" className="min-h-14 w-full border bg-background p-1 text-xs" value={description} maxLength={2000} onChange={(change) => setDescription(change.target.value)} />
          <label className="flex items-center gap-2">When <Input type="datetime-local" aria-label="Event time" className="h-7 w-52 text-xs" value={when} onChange={(change) => setWhen(change.target.value)} /></label>
          <fieldset className="flex flex-wrap gap-2">
            <legend className="text-muted-foreground">Who or what it involves</legend>
            {entities.filter((entity) => entity.review_state !== "rejected").map((entity) => (
              <label key={entity.candidate_id} className="inline-flex items-center gap-1">
                <input type="checkbox" checked={chosen.includes(entity.candidate_id)} onChange={(change) => setChosen((current) => change.target.checked ? [...current, entity.candidate_id] : current.filter((value) => value !== entity.candidate_id))} />
                {entity.name}
              </label>
            ))}
          </fieldset>
          <div className="flex gap-1">
            <Button type="submit" size="xs" disabled={!title.trim()}>Save</Button>
            <Button type="button" size="xs" variant="ghost" onClick={() => setEditing(false)}>Cancel</Button>
          </div>
        </form>
      )}

      <ol className="mt-2 space-y-1" aria-label="Source messages">
        {event.source_records.map((record) => (
          <li key={record.record_id}>
            <button type="button" className="w-full text-left hover:bg-accent/40" onClick={() => setPeek(peek === record.record_id ? null : record.record_id)}>
              <span className="font-mono text-muted-foreground">#{record.ordinal}</span> {formatWhen(record.occurred_at)}
              {record.snippet && <span className="block text-muted-foreground">{record.snippet}</span>}
            </button>
            {peek === record.record_id && <RecordPeek previewHandle={previewHandle} mode={mode} recordId={record.record_id} onClose={() => setPeek(null)} />}
          </li>
        ))}
      </ol>

      <div className="mt-2 flex flex-wrap items-center gap-1">
        {editable && !editing && <Button type="button" size="xs" variant="ghost" onClick={() => setEditing(true)}><Pencil /> Edit</Button>}
        {editable && <Button type="button" size="xs" variant="ghost" className="text-destructive" onClick={() => void run(() => actions.correctEvent({ op: "reject", candidate_ids: [id] }))}><X /> Reject</Button>}
        {event.review_state === "rejected" && <Button type="button" size="xs" variant="ghost" onClick={() => void run(() => actions.correctEvent({ op: "restore", candidate_ids: [id] }))}><RotateCcw /> Restore</Button>}
        {event.correction && <span className="ml-auto text-[10px] text-muted-foreground">{event.correction.op.replaceAll("_", " ")} by {event.correction.actor.username} · {formatWhen(event.correction.at)}</span>}
      </div>
      {error && <p className="mt-1 text-destructive" role="alert">{error}</p>}
    </li>
  );
}
