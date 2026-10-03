// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// What each extractor found in one conversation, side by side, read-only (desktop and /m). One card per extractor, tagged
// with the extractor that made it; entities and events inside, with the message each came from. Editing stays in Review.
"use client";

import { formatDate, errorText } from "@/components/mobile/mobile-format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useThreadExtractions } from "@/hooks/use-conversation-actions";
import type { ExtractionGroup, ExtractedEntity, ExtractedEvent } from "@/lib/conversation-actions-client";

const STATUS_TEXT: Record<ExtractionGroup["status"], string> = {
  running: "running",
  completed: "done",
  failed: "failed",
  completed_with_failures: "done with failures",
};

function EntityRow({ entity }: { entity: ExtractedEntity }) {
  return (
    <li className="py-1.5" data-testid="extracted-entity">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-sm font-medium">{entity.name}</span>
        <Badge variant="outline" className="text-[11px]">{entity.type}</Badge>
        <span className="text-xs text-muted-foreground">{entity.mention_count} mention{entity.mention_count === 1 ? "" : "s"}</span>
      </div>
      {entity.aliases.length ? <p className="text-xs text-muted-foreground">also: {entity.aliases.join(", ")}</p> : null}
      {entity.mentions[0]?.snippet ? <p className="truncate text-xs italic text-muted-foreground">&ldquo;{entity.mentions[0].snippet}&rdquo;</p> : null}
    </li>
  );
}

function EventRow({ event }: { event: ExtractedEvent }) {
  const when = event.occurred_at ? formatDate(event.occurred_at) : event.when_stated ?? "undated";
  return (
    <li className="py-1.5" data-testid="extracted-event">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-xs tabular-nums text-muted-foreground">{when}</span>
        <Badge variant="outline" className="text-[11px]">{event.type.replaceAll("_", " ")}</Badge>
        {event.precision === "uncertain" ? <Badge variant="secondary" className="text-[11px]">date uncertain</Badge> : null}
      </div>
      <p className="text-sm">{event.title}</p>
    </li>
  );
}

function Group({ group }: { group: ExtractionGroup }) {
  const failures = group.runs.filter((run) => run.status === "failed" && run.error);
  return (
    <Card className="gap-2 py-3" data-testid={`extraction-group-${group.id}`}>
      <CardHeader className="px-4">
        <CardTitle className="flex flex-wrap items-center gap-1.5 text-sm">
          {group.label}
          {group.compare_only ? <Badge variant="outline" className="text-[11px]">compare only</Badge> : null}
          <Badge variant={group.status === "failed" ? "destructive" : "secondary"} className="text-[11px]">{STATUS_TEXT[group.status]}</Badge>
        </CardTitle>
        <p className="text-xs text-muted-foreground">{group.counts.entities} entities · {group.counts.events} events</p>
      </CardHeader>
      <CardContent className="space-y-2 px-4">
        {failures.map((run) => <p key={run.id} className="text-xs text-destructive">{run.error}</p>)}
        {group.entities.length ? (
          <section aria-label={`${group.label} entities`}>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Entities</h3>
            <ul className="divide-y divide-border">{group.entities.map((entity) => <EntityRow key={entity.id} entity={entity} />)}</ul>
          </section>
        ) : null}
        {group.events.length ? (
          <section aria-label={`${group.label} events`}>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Events</h3>
            <ul className="divide-y divide-border">{group.events.map((event) => <EventRow key={event.id} event={event} />)}</ul>
          </section>
        ) : null}
        {!group.entities.length && !group.events.length && group.status !== "running" && !failures.length
          ? <p className="text-sm text-muted-foreground">This extractor found nothing in this conversation.</p> : null}
      </CardContent>
    </Card>
  );
}

/** Every extractor's findings for the conversation, grouped by extractor. */
export function ExtractionsView({ threadId, onExtract }: { threadId: string; onExtract?: () => void }) {
  const query = useThreadExtractions(threadId);
  if (query.isPending) return <div className="space-y-3 p-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-24 w-full" /></div>;
  if (query.isError) {
    return (
      <div className="m-4 rounded-lg border border-destructive/40 bg-destructive/10 p-4 text-sm" role="alert">
        <p className="font-semibold text-destructive">Could not load the extractions</p>
        <p className="mt-1">{errorText(query.error)}</p>
        <Button type="button" variant="outline" className="mt-3 h-12" onClick={() => void query.refetch()}>Try again</Button>
      </div>
    );
  }
  const groups = query.data.extractors;
  if (groups.length === 0) {
    return (
      <div className="space-y-3 px-4 py-8 text-center">
        <p className="text-sm text-muted-foreground">Nothing has been extracted from this conversation yet.</p>
        {onExtract ? <Button type="button" className="h-12" onClick={onExtract}>Extract</Button> : null}
      </div>
    );
  }
  return (
    <div className="space-y-3 px-4 pb-6" aria-label="Extractions by extractor">
      {groups.map((group) => <Group key={group.id} group={group} />)}
      {query.data.truncated ? <p className="text-xs text-muted-foreground">Only the first 2,000 of each list are shown.</p> : null}
      {onExtract ? <Button type="button" variant="outline" className="h-12 w-full" onClick={onExtract}>Extract again</Button> : null}
    </div>
  );
}
