// Byline: Codex · GPT-6 · 2026-10-06
import { ExtractionsView } from "@/components/conversations/extractions-view";
import { readHref } from "@/components/read/read-location";
import { useThreadExtractions } from "@/hooks/use-conversation-actions";
import type { ExtractionGroup } from "@/lib/conversation-actions-client";
import { AppLink } from "@/lib/router-compat";

/** Show exact extractor/run/version references and the messages supporting findings.
 * Inputs: one extraction group and canonical thread/source URL context.
 * Output: expandable provenance and source-record links; effects: navigation only.
 * Pick alongside ExtractionsView to retain its findings UI and expose original locators.
 */
function ExtractionCitations({ group, params }: { group: ExtractionGroup; params: URLSearchParams }) {
  const citedEntities = group.entities.filter((entity) => entity.mentions.length > 0);
  return (
    <details className="rounded-md border border-border p-3 text-xs">
      <summary className="cursor-pointer font-semibold">{group.label}: citations and run details</summary>
      <div className="mt-3 space-y-3">
        <ul className="space-y-2 break-all">
          {group.runs.map((run) => <li key={run.id}><strong>{run.extractor}</strong> · version {run.version || "not supplied"}<br />Run {run.id} · {run.status}{run.model_id ? <><br />Model {run.model_id}</> : null}{run.error ? <p className="text-destructive">{run.error}</p> : null}</li>)}
        </ul>
        {citedEntities.map((entity) => (
          <div key={entity.id} className="space-y-1">
            <p className="font-medium">{entity.name} · {entity.review_state} · confidence {entity.confidence}</p>
            <p className="break-all text-muted-foreground">Entity {entity.id} · run {entity.run_id}</p>
            <ul className="space-y-2">
              {entity.mentions.map((mention, index) => <li key={`${mention.record_id}:${index}`}>
                {mention.snippet ? <p className="whitespace-pre-wrap break-words">“{mention.snippet}”</p> : null}
                {mention.record_id ? <AppLink href={readHref(params, { around: mention.record_id })} className="break-all underline underline-offset-4">Source message {mention.record_id}</AppLink> : <p className="text-muted-foreground">This mention has no source record ID.</p>}
              </li>)}
            </ul>
          </div>
        ))}
        {group.events.map((event) => (
          <div key={event.id} className="space-y-1">
            <p className="font-medium">{event.title} · {event.review_state} · confidence {event.confidence}</p>
            <p className="break-all text-muted-foreground">Event {event.id} · run {event.run_id}</p>
            {event.description ? <p className="whitespace-pre-wrap break-words">{event.description}</p> : null}
            {event.record_ids.length ? <ul className="space-y-1">{event.record_ids.map((id) => <li key={id}><AppLink href={readHref(params, { around: id })} className="break-all underline underline-offset-4">Source message {id}</AppLink></li>)}</ul> : <p className="text-muted-foreground">This event has no source record ID.</p>}
          </div>
        ))}
        {!citedEntities.length && !group.events.length ? <p className="text-muted-foreground">No source-record citations were supplied for this extraction.</p> : null}
      </div>
    </details>
  );
}

/** Present existing extracted findings and their supplied provenance beside a conversation.
 * Inputs: original thread ID and source-aware route query. Output: context with cited records.
 * Effects: the shared read-only extraction query; no new store or inference.
 * Pick for reading existing context; explicit extraction tools remain in ConversationToolbar.
 */
export function ReadContext({ threadId, params }: { threadId: string; params: URLSearchParams }) {
  const query = useThreadExtractions(threadId);
  return (
    <aside className="min-w-0 rounded-lg border border-border bg-card" aria-label="Extracted context and citations">
      <header className="space-y-1 p-4"><h2 className="text-sm font-semibold">Extracted context</h2><p className="text-xs text-muted-foreground">Findings by extractor. Open citations to read the source messages and inspect run details.</p></header>
      <ExtractionsView threadId={threadId} />
      <div className="space-y-3 p-4 pt-0">
        {query.data?.extractors.map((group) => <ExtractionCitations key={group.id} group={group} params={params} />)}
        {query.data?.extractors.length ? <p className="text-xs text-muted-foreground">This extraction response provides record IDs and run versions. Source byte hashes and validation receipts are not supplied here; inspect processing previews for their available custody details.</p> : null}
      </div>
    </aside>
  );
}
