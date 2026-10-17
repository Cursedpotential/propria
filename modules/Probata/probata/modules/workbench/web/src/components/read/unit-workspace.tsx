// Unit-centered reading replaces the isolated-file layout after a source is selected.
// Reuses the existing SBV reader and extraction controls; no new storage authority.
// Byline: ChatGPT, 2026-10-17.
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useState } from "react";
import { ArrowLeft, FileJson, Layers, Search } from "lucide-react";
import { ConversationActionSheet } from "@/components/conversations/conversation-actions";
import { SelectableConversations, type SelectableItem } from "@/components/conversations/selectable-conversations";
import { toMessageRow } from "@/components/imported/record-rows";
import { Empty, ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { formatDate } from "@/components/mobile/mobile-format";
import { ReadContext } from "@/components/read/read-context";
import { UnitInspector } from "@/components/read/unit-inspector";
import { readHref } from "@/components/read/read-location";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { importedApi } from "@/lib/imported-client";
import { useOperatingMode } from "@/lib/fixed-case-context";
import { AppLink } from "@/lib/router-compat";
import type { UnitMessage } from "@/lib/unit-workspace-types";

type Representation = "reader" | "records" | "extractions" | "history";
const VIEWS: Array<{ id: Representation; label: string }> = [
  { id: "reader", label: "Conversation" }, { id: "records", label: "PostgreSQL records" },
  { id: "extractions", label: "Extraction results" }, { id: "history", label: "Source versions" },
];

/** Read a complete registered unit with one scope and a selection-linked inspector.
 * Inputs: source/thread IDs and shareable route state. Output: SBV messages, PG rows,
 * member navigation and real existing extraction/review tools. Effects: paged GETs
 * and explicit user actions; opening a view never imports, projects or promotes data.
 */
export function UnitWorkspace({ sourceId, threadId, params }: {
  sourceId: string; threadId: string | null; params: URLSearchParams;
}) {
  const router = useRouter();
  const mode = useOperatingMode();
  const memberId = params.get("unit_member");
  const appliedQuery = params.get("unit_q") ?? "";
  const selectedId = params.get("around");
  const view = VIEWS.find((item) => item.id === params.get("unit_view"))?.id ?? "reader";
  const [extractThread, setExtractThread] = useState<string | null>(null);
  const unit = useQuery({
    queryKey: ["imported-unit", sourceId, threadId],
    queryFn: ({ signal }) => importedApi.unit(sourceId, threadId, signal), retry: false,
  });
  const messages = useInfiniteQuery({
    queryKey: ["unit-messages", sourceId, threadId, memberId, appliedQuery, unit.data?.generation_signature],
    queryFn: async ({ pageParam, signal }) => {
      const result = await importedApi.unitMessages(sourceId, {
        threadId, memberId, q: appliedQuery, cursor: pageParam,
      }, signal);
      if (result.generation_signature !== unit.data?.generation_signature) {
        throw new Error("The unit generation changed. Refresh the unit before reading more records.");
      }
      return result;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    enabled: Boolean(unit.data), retry: false,
  });
  const rows = messages.data?.pages.flatMap((page) => page.items) ?? [];
  const selected = rows.find((item) => item.id === selectedId) ?? null;
  const effectiveParams = new URLSearchParams(params);
  effectiveParams.set("source", sourceId);
  const show = (changes: Record<string, string | null>) => {
    void router.navigate({ href: readHref(effectiveParams, changes), replace: true });
  };
  const openMessage = (message: UnitMessage) => show({ around: message.id });
  if (unit.isPending) return <Loading />;
  if (!unit.data) return <div className="p-6"><ErrorBox error={unit.error} onRetry={() => void unit.refetch()} /></div>;
  const detail = unit.data;
  const member = detail.members.find((item) => item.source.id === memberId);
  const currentThreads = detail.extraction_targets.filter((target) => !memberId || target.source_id === memberId);
  const generationCounts = new Map(detail.members.flatMap((item) => item.versions.map((version) => [version.generation_id, version.messages] as const)));
  const items: SelectableItem[] = currentThreads.map((target) => ({
    id: target.thread_id, contactName: target.label, type: "message", address: null,
    lastMessage: "Recorded conversation in this unit", lastDate: null,
    count: target.generation_ids.reduce((total, id) => total + (generationCounts.get(id) ?? 0), 0),
    href: readHref(effectiveParams, { thread: target.thread_id, unit_member: target.source_id, around: null }),
  }));

  return <div className="space-y-4 p-4 lg:p-5" data-testid="unit-workspace">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0">
        <AppLink className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground" href={readHref(effectiveParams, {
          source: null, thread: null, around: null, unit_member: null, unit_q: null, unit_view: null,
        })}><ArrowLeft className="size-3.5" /> Back to library</AppLink>
        <h1 className="mt-2 flex items-center gap-2 text-lg font-semibold"><Layers className="size-5 text-primary" />
          {detail.members[0]?.source.format === "Facebook" ? "Messaging workspace" : "Source workspace"}
        </h1>
        <p className="mt-1 break-all text-xs text-muted-foreground">{detail.root}</p>
      </div>
      <p className="text-xs text-muted-foreground">{detail.members.length} registered parts · {detail.message_count.toLocaleString()} messages in selected generations</p>
    </header>

    <div className="grid items-start gap-4 lg:grid-cols-[12.5rem_minmax(0,1fr)] 2xl:grid-cols-[13rem_minmax(0,1fr)_21rem]">
      <nav className="min-w-0 rounded-lg border bg-card" aria-label="Unit members">
        <div className="border-b p-3"><h2 className="text-xs font-semibold">Source unit</h2>
          <p className="mt-1 text-[11px] text-muted-foreground">{detail.membership_basis === "registered_facebook_folder" ? "Facebook conversation folder" : "Recorded export"}</p>
        </div>
        <div className="flex gap-1 overflow-x-auto p-2 lg:max-h-[65dvh] lg:flex-col lg:overflow-y-auto">
          <button type="button" aria-pressed={!memberId} className={`shrink-0 rounded px-3 py-2 text-left text-xs ${!memberId ? "bg-primary/15 font-semibold" : "hover:bg-accent"}`}
            onClick={() => show({ unit_member: null })}>All registered parts</button>
          {detail.members.map((item) => <button key={item.source.id} type="button" aria-pressed={memberId === item.source.id}
            className={`shrink-0 rounded p-2 text-left text-xs ${memberId === item.source.id ? "bg-primary/15 font-semibold" : "hover:bg-accent"}`}
            onClick={() => show({ unit_member: item.source.id })}>
            <span className="flex items-center gap-2"><FileJson className="size-4 shrink-0 text-muted-foreground" /><span className="break-all">{item.source.file_name}</span></span>
            <span className="mt-1 block pl-6 text-[10px] text-muted-foreground">{item.messages.toLocaleString()} messages</span>
          </button>)}
        </div>
        {detail.membership_basis !== "registered_facebook_folder" && detail.extraction_targets.length > 1 ? <label className="block border-t p-3 text-xs">
          Conversation
          <select aria-label="Conversation within this unit" className="mt-2 w-full rounded border bg-background p-2" value={threadId ?? ""}
            onChange={(event) => show({ thread: event.target.value || null })}>
            <option value="">All conversations in unit</option>
            {detail.extraction_targets.map((target) => <option key={target.thread_id} value={target.thread_id}>{target.label}</option>)}
          </select>
        </label> : null}
        <details className="border-t p-3 text-[11px] text-muted-foreground"><summary className="cursor-pointer">Membership coverage</summary>
          <p className="mt-2">{detail.coverage}</p><p className="mt-2">Folder association does not establish duplicate identity or an owner-designated atomic preservation unit.</p>
        </details>
      </nav>

      <section className="min-w-0 overflow-hidden rounded-lg border bg-card" aria-label="Selected unit content">
        <form className="flex items-center gap-2 border-b p-3" role="search" onSubmit={(event) => {
          event.preventDefault();
          const query = String(new FormData(event.currentTarget).get("unit_search") ?? "").trim();
          show({ unit_q: query || null });
        }}>
          <Search className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <Input key={appliedQuery} name="unit_search" type="search" defaultValue={appliedQuery} maxLength={200}
            aria-label={member ? "Search selected member" : "Search entire unit"} placeholder={member ? `Search ${member.source.file_name}` : "Search all messages in this unit"} className="min-w-0 border-0 shadow-none" />
          <Button type="submit" size="sm">Find</Button>
          {appliedQuery ? <Button variant="ghost" type="button" size="sm" onClick={() => show({ unit_q: null })}>Clear</Button> : null}
        </form>
        <div className="flex gap-1 overflow-x-auto border-b px-2 py-1" aria-label="Source representations">
          {VIEWS.map((item) => <button key={item.id} type="button" aria-pressed={view === item.id}
            className={`shrink-0 border-b-2 px-3 py-2 text-xs ${view === item.id ? "border-primary font-semibold" : "border-transparent text-muted-foreground hover:text-foreground"}`}
            onClick={() => show({ unit_view: item.id === "reader" ? null : item.id })}>{item.label}</button>)}
        </div>
        <div className="max-h-[68dvh] min-h-80 overflow-auto" aria-live="polite">
          {messages.isPending ? <Loading /> : null}
          {messages.isError ? <ErrorBox error={messages.error} onRetry={() => { void unit.refetch(); void messages.refetch(); }} /> : null}
          {view === "reader" ? <div className="space-y-3 p-4" aria-label="Messages across the unit">
            {rows.map((message, index) => {
              const row = toMessageRow(message, message.attribution?.record_ordinal ?? index);
              row.direction = message.attribution?.direction === "sent" ? "out" : message.attribution?.direction === "received" ? "in" : "unknown";
              return <article key={message.id} className="group" id={`unit-message-${message.id}`}>
                {index === 0 || formatDate(message.at) !== formatDate(rows[index - 1].at) ? <p className="py-2 text-center text-[11px] text-muted-foreground">{formatDate(message.at)}</p> : null}
                <MessageBubble row={row} previewHandle={message.attribution?.preview_handle ?? ""} mode={mode} showSenderLabel highlighted={message.id === selectedId} />
                <div className={`mt-1 flex items-center gap-2 text-[10px] text-muted-foreground ${message.outgoing ? "justify-end" : ""}`}>
                  <span>{message.source.file_name}</span><button className="rounded px-1 py-1 underline underline-offset-2 focus-visible:ring-2" onClick={() => openMessage(message)}>Inspect record</button>
                  {message.attachments > 0 ? <span>{message.attachments} attachment references · open record for retained fields</span> : null}
                </div>
              </article>;
            })}
            {!rows.length && !messages.isPending && !messages.isError ? <Empty>{appliedQuery ? "No matching normalized messages in this scope." : "No normalized messages were returned for these registered generations."}</Empty> : null}
          </div> : null}
          {view === "records" ? <div className="space-y-2 p-3" aria-label="PostgreSQL normalized records">
            <p className="mb-3 text-xs text-muted-foreground">Actual stored rows. Open a record to see every normalized field and its source coordinates.</p>
            {rows.map((message) => <button type="button" key={message.id} onClick={() => openMessage(message)} className={`block w-full rounded border p-3 text-left ${selectedId === message.id ? "border-primary bg-primary/5" : "hover:bg-accent"}`}>
              <span className="block text-xs font-medium">{message.sender.label} · {message.at ?? "Timestamp unknown"}</span>
              <span className="mt-1 block break-words text-xs">{message.body.slice(0, 220) || "Record without text"}</span>
              <span className="mt-2 block break-all font-mono text-[10px] text-muted-foreground">{message.source.file_name} · #{message.attribution?.record_ordinal} · {message.id}</span>
            </button>)}
          </div> : null}
          {view === "extractions" ? <div className="space-y-4 p-3">
            <p className="text-xs text-muted-foreground">Existing extraction controls apply to each recorded conversation below. They do not yet pin a whole-unit extraction to this viewing generation set.</p>
            {mode === "LIVE" ? <SelectableConversations items={items} placement="desktop" /> : null}
            {currentThreads.map((target) => <UnitExtraction key={target.thread_id} threadId={target.thread_id} label={target.label}
              params={effectiveParams} onRun={mode === "LIVE" ? () => setExtractThread(target.thread_id) : undefined} />)}
            {!currentThreads.length ? <p className="text-xs text-muted-foreground">No normalized message conversations are registered in this selection.</p> : null}
            <p className="text-[11px] text-muted-foreground">Targets cover the resolved unit membership, independently of message pagination. Choosing an extractor applies to that whole conversation, not just text-search matches.</p>
          </div> : null}
          {view === "history" ? <div className="space-y-3 p-3">
            <p className="text-xs text-muted-foreground">{detail.generation_policy}</p>
            {detail.members.map((item) => <details key={item.source.id} className="rounded border p-3"><summary className="cursor-pointer text-sm font-medium">{item.source.file_name}</summary>
              {item.versions.map((version) => <dl key={version.source_version_id} className="mt-3 space-y-1 break-all text-xs">
                <dt className="font-semibold">Selected source version</dt><dd>{version.source_version_id}</dd>
                <dt className="font-semibold">Normalization generation</dt><dd>{version.generation_id ?? "Not created"}</dd>
                <dt className="font-semibold">Retained versions / selected counts</dt><dd>{version.retained_versions} versions; {version.raw_records} parsed raw records; {version.normalized_records} normalized records</dd>
              </dl>)}
              <MemberHistory sourceId={item.source.id} />
            </details>)}
          </div> : null}
          {messages.hasNextPage && view !== "history" ? <div className="border-t p-3"><Button variant="outline" className="w-full" disabled={messages.isFetchingNextPage} onClick={() => void messages.fetchNextPage()}>{messages.isFetchingNextPage ? "Loading..." : appliedQuery ? "Load more matches" : "Load later messages"}</Button></div> : null}
        </div>
      </section>
      <div className="lg:col-span-2 2xl:col-span-1"><UnitInspector unit={detail} message={selected} recordId={selectedId} /></div>
    </div>
    {extractThread ? <ConversationActionSheet key={extractThread} kind="extract" threadIds={[extractThread]} open onOpenChange={(open) => !open && setExtractThread(null)} side="right" /> : null}
  </div>;
}

/** Read retained history only on request; expansion never reprocesses the source.
 * Input: exact import ID. Output: paged version rows. Effects: GETs only.
 */
function MemberHistory({ sourceId }: { sourceId: string }) {
  const [opened, setOpened] = useState(false);
  const history = useInfiniteQuery({ queryKey: ["unit-member-history", sourceId],
    queryFn: ({ pageParam, signal }) => importedApi.sourceHistory(sourceId, pageParam, signal),
    initialPageParam: 0, getNextPageParam: (last) => last.next_offset ?? undefined, enabled: opened,
  });
  return <div className="mt-3"><Button variant="outline" size="sm" onClick={() => setOpened((value) => !value)}>{opened ? "Hide retained history" : "Show all retained versions"}</Button>
    {opened ? <>{history.isError ? <ErrorBox error={history.error} onRetry={() => void history.refetch()} /> : null}
      {history.data?.pages.flatMap((page) => page.items).map((version) => <p key={version.source_version_id} className="mt-2 break-all text-[11px] text-muted-foreground">v{version.version_ordinal} · {version.source_version_id} · {version.normalized} extracted · {version.published ? "published" : "not published"}</p>)}
      {history.hasNextPage ? <Button className="mt-2" size="sm" variant="ghost" onClick={() => void history.fetchNextPage()}>More history</Button> : null}</> : null}
  </div>;
}


/** Fetch existing extracted outputs only when their target is expanded.
 * Inputs: exact native thread, label and route. Output: existing source-linked
 * findings and optional explicit run picker. Effects: on-demand GETs only.
 */
function UnitExtraction({ threadId, label, params, onRun }: {
  threadId: string; label: string; params: URLSearchParams; onRun?: () => void;
}) {
  const [open, setOpen] = useState(false);
  return <details className="rounded border p-3" onToggle={(event) => setOpen(event.currentTarget.open)}>
    <summary className="cursor-pointer text-sm font-medium">{label}</summary>
    {open ? <>{onRun ? <Button className="my-3" size="sm" variant="outline" onClick={onRun}>Available extractors</Button> : null}
      <ReadContext threadId={threadId} params={params} /></> : null}
  </details>;
}
