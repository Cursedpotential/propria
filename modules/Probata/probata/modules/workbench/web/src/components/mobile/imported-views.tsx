// Byline: Claude Code · Sonnet · 2026-10-02
// Imported (sources -> conversations -> messages) and Calls for the mobile shell, built on the Workbench's own
// components rather than new UI:
//   sources      ImportedGrid (Glide Data Grid: sort, filter, column toggle, virtualized rows)
//   conversations  ConversationList (ported from the SBV fork's ConversationList.jsx)
//   messages     MessageBubble (the SBV-derived component the Review thread view uses)
//   calls        CallsTable (the SBV-derived call-history list Review uses)
// Read-only. "Who is this?" comes from the identity components those rows already host.
// Byline amendment: Claude Code · Sonnet 5.5 · 2026-10-02 (conversations are checkable, and a conversation has Extractions,
// Extract and Send to Surreal: components/conversations/*; those are the only actions on this view that start work).
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useLayoutEffect, useMemo, useRef } from "react";

import { ConversationSelectionBar, ConversationToolbar } from "@/components/conversations/conversation-actions";
import { ImportedGrid, type ImportedColumn } from "@/components/imported/imported-grid";
import { toCallRow, toMessageRow } from "@/components/imported/record-rows";
import { formatCount, formatDate, formatRange, statusLabel } from "@/components/mobile/mobile-format";
import { Empty, ErrorBox, LoadMore, Loading, PageBar, StatusBadge } from "@/components/mobile/mobile-ui";
import { CallsTable } from "@/components/sbv/calls-table";
import { ConversationList, type ConversationListItem } from "@/components/sbv/conversation-list";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useSelection } from "@/hooks/use-conversation-actions";
import { importedApi, type ImportedSource, type SourceStatus } from "@/lib/imported-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

const STATUS_COLOR: Record<SourceStatus, string | undefined> = {
  committed: "#2f9e6e", awaiting_review: "#c98a1b", parked: "#d2691e", failed: "#d9534f", running: "#3f8fd2", not_finished: undefined, skipped: undefined,
};

/** Split parents read "Split into N conversations - N done / M failed", never failed or not finished. */
export function sourceStatusText(source: ImportedSource) {
  if (source.split) {
    const { total, done, failed } = source.split;
    return `Split into ${total} · ${done} done${failed ? ` / ${failed} failed` : ""}`;
  }
  return statusLabel(source.status);
}

const SOURCE_COLUMNS: ImportedColumn<ImportedSource>[] = [
  { id: "file", title: "File", width: 200, grow: 1, phone: true, text: (s) => s.file_name },
  { id: "status", title: "Status", width: 190, phone: true, text: sourceStatusText, color: (s) => STATUS_COLOR[s.status], sort: (s) => s.status },
  { id: "format", title: "Format", width: 80, text: (s) => s.format },
  { id: "device", title: "Phone", width: 130, text: (s) => s.device ?? "" },
  { id: "owner", title: "Owner", width: 80, text: (s) => s.owner ?? "" },
  { id: "raw", title: "Raw", width: 80, text: (s) => formatCount(s.raw), sort: (s) => s.raw },
  { id: "normalized", title: "Normalized", width: 100, text: (s) => formatCount(s.normalized), sort: (s) => s.normalized },
  { id: "committed", title: "Done", width: 80, text: (s) => formatCount(s.committed), sort: (s) => s.committed },
  { id: "files", title: "Files", width: 70, text: (s) => String(s.files), sort: (s) => s.files },
  { id: "imported", title: "Imported", width: 120, text: (s) => formatDate(s.imported_at), sort: (s) => s.imported_at ?? "" },
];

export function SourcesView() {
  const router = useRouter();
  const summary = useQuery({ queryKey: ["m-summary"], queryFn: ({ signal }) => importedApi.summary(signal), staleTime: 15_000 });
  const query = useInfiniteQuery({
    queryKey: ["m-sources"],
    queryFn: ({ pageParam, signal }) => importedApi.sources(pageParam, undefined, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  const totals = summary.data?.totals;
  const byStatus = (summary.data?.files_by_status ?? {}) as Partial<Record<SourceStatus, number>>;
  return (
    <div className="flex flex-col">
      <PageBar title="Imported" subtitle="Every source file, newest first" />
      {totals ? (
        <Card className="mx-4 mt-3 gap-0 py-3">
          <CardContent className="space-y-2 px-4">
            <p className="text-sm">
              <strong>{formatCount(totals.committed)}</strong> of {formatCount(totals.normalized)} records done
              <span className="text-muted-foreground"> · {formatCount(totals.raw)} raw · {formatCount(totals.sources)} sources</span>
            </p>
            <div className="flex flex-wrap gap-1.5">
              {(Object.entries(byStatus) as [SourceStatus, number][]).map(([status, count]) => (
                <Badge key={status} variant="secondary">{formatCount(count)} {statusLabel(status).toLowerCase()}</Badge>
              ))}
            </div>
            {summary.data && !summary.data.run_state_available ? <p className="text-xs text-muted-foreground">Run status is still loading; finished files already show as done.</p> : null}
          </CardContent>
        </Card>
      ) : null}
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <ImportedGrid
          className="h-[calc(100dvh-16rem)] min-h-96"
          rows={rows}
          columns={SOURCE_COLUMNS}
          filterLabel="Filter files, phones, owners, status"
          empty="Nothing imported yet."
          onRowClick={(source) => void router.navigate({ href: `/m/source/${source.id}` })}
          onReachEnd={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
        />
      )}
    </div>
  );
}

export function SourceThreadsView({ sourceId }: { sourceId: string }) {
  const { selected, toggle, clear } = useSelection();
  const query = useInfiniteQuery({
    queryKey: ["m-threads", sourceId],
    queryFn: ({ pageParam, signal }) => importedApi.threads(sourceId, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const source = query.data?.pages[0]?.source;
  const items = useMemo<(ConversationListItem & { id: string })[]>(
    () => (query.data?.pages.flatMap((page) => page.items) ?? []).map((thread) => ({
      id: thread.id,
      href: `/m/thread/${thread.id}`,
      contactName: thread.title,
      address: thread.participants.find((p) => !p.mine)?.id ?? null,
      type: thread.calls && !thread.messages ? "call" : "message",
      lastMessage: thread.last_message,
      lastDate: thread.last_at,
      count: thread.messages || thread.records,
      tag: thread.party === "first_party" ? "First-party" : thread.party === "third_party" ? "Third-party" : thread.party === "mixed" ? "First + third" : null,
    })),
    [query.data],
  );
  return (
    <div>
      <PageBar title={source?.file_name ?? "Source"} subtitle={source ? [source.format, source.device, source.owner].filter(Boolean).join(" · ") : undefined} back="/m" />
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          {source ? <SourceSummary source={source} /> : null}
          <ConversationList items={items} selection={{ selected, onToggle: toggle }} />
          <ConversationSelectionBar threadIds={items.map((item) => item.id).filter((id) => selected.has(id))} onClear={clear} placement="mobile" />
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
        </>
      )}
    </div>
  );
}

function SourceSummary({ source }: { source: ImportedSource }) {
  const counts = Object.entries(source.status_counts) as [SourceStatus, number][];
  return (
    <Card className="mx-4 my-3 gap-0 py-3">
      <CardContent className="space-y-2 px-4">
        <div className="flex items-center justify-between gap-3">
          {source.split ? <p className="text-sm font-semibold">{sourceStatusText(source)}</p> : <StatusBadge status={source.status} />}
          <Sheet>
            <SheetTrigger asChild><Button type="button" variant="outline" size="sm" className="h-10">Details</Button></SheetTrigger>
            <SheetContent side="bottom" className="max-h-[85dvh] overflow-y-auto">
              <SheetHeader>
                <SheetTitle>{source.file_name}</SheetTitle>
                <SheetDescription>{source.casevault_key}</SheetDescription>
              </SheetHeader>
              <dl className="grid grid-cols-3 gap-2 px-4 pb-4 text-center">
                {([["Raw", source.raw], ["Normalized", source.normalized], ["Done", source.committed]] as [string, number][]).map(([label, value]) => (
                  <div key={label} className="rounded-md bg-muted/70 px-1 py-2"><dd className="text-base font-semibold tabular-nums">{formatCount(value)}</dd><dt className="text-[11px] text-muted-foreground">{label}</dt></div>
                ))}
              </dl>
              <ul className="space-y-1 px-4 pb-6 text-sm">
                <li>{formatCount(source.files)} files · {formatRange(source.first_at, source.last_at)}</li>
                {counts.map(([status, count]) => <li key={status}>{formatCount(count)} {statusLabel(status).toLowerCase()}</li>)}
                {source.failed_attempts > 0 ? <li className="text-muted-foreground">{formatCount(source.failed_attempts)} earlier attempts failed; the file's current status is what counts.</li> : null}
              </ul>
            </SheetContent>
          </Sheet>
        </div>
        <p className="text-xs text-muted-foreground">{formatCount(source.normalized)} records · {formatRange(source.first_at, source.last_at)}</p>
      </CardContent>
    </Card>
  );
}

export function ThreadView({ threadId }: { threadId: string }) {
  const params = useBrowserSearchParams();
  const focus = params.get("focus");
  const query = useInfiniteQuery({
    queryKey: ["m-messages", threadId, focus],
    queryFn: ({ pageParam, signal }) => importedApi.messages(threadId, { cursor: pageParam, around: pageParam ? null : focus }, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.older_cursor ?? undefined,
  });
  // Pages arrive newest first; each page is oldest-first inside, like a chat.
  const messages = useMemo(() => [...(query.data?.pages ?? [])].reverse().flatMap((page) => page.items), [query.data]);
  const head = query.data?.pages[0];
  const title = useMemo(() => {
    const others = [...new Set(messages.filter((m) => !m.sender.mine).map((m) => m.sender.label))];
    return others.length ? others.slice(0, 3).join(", ") : head?.conversation ?? "Conversation";
  }, [messages, head]);
  const bottomRef = useRef<HTMLDivElement>(null);
  const pageCount = query.data?.pages.length ?? 0;
  const scrolledFor = useRef<string | null>(null);
  useLayoutEffect(() => {
    const key = `${threadId}:${focus ?? ""}`;
    if (pageCount !== 1 || scrolledFor.current === key) return;
    if (focus) document.getElementById(`msg-${focus}`)?.scrollIntoView({ block: "center" });
    else bottomRef.current?.scrollIntoView({ block: "end" });
    scrolledFor.current = key;
  }, [pageCount, focus, threadId]);
  return (
    <div>
      <PageBar title={title} subtitle={head ? [head.source.format, head.source.device, head.source.file_name].filter(Boolean).join(" · ") : undefined} back={head ? `/m/source/${head.source.id}` : "/m"} />
      <ConversationToolbar threadId={threadId} placement="mobile" />
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <div className="space-y-1.5 px-3 py-3" aria-label="Conversation">
          {query.hasNextPage
            ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load earlier messages" />
            : <p className="py-2 text-center text-xs text-muted-foreground">Start of this conversation</p>}
          {messages.map((message, index) => {
            const day = formatDate(message.at);
            const header = index === 0 || day !== formatDate(messages[index - 1].at);
            return (
              <div key={message.id} id={`msg-${message.id}`}>
                {header ? <p className="py-2 text-center text-xs font-semibold text-muted-foreground">{day}</p> : null}
                <MessageBubble row={toMessageRow(message, index)} previewHandle="" mode="REAL" showSenderLabel={!message.outgoing} highlighted={message.id === focus} />
                {message.party ? <p className={`px-2 text-[10px] text-muted-foreground ${message.outgoing ? "text-right" : ""}`}>{message.party === "first_party" ? "first-party" : "third-party"}</p> : null}
              </div>
            );
          })}
          {focus && head?.newer_cursor ? (
            <div className="p-2 text-center">
              <Button asChild variant="outline" className="h-12"><AppLink href={`/m/thread/${threadId}`}>Jump to the latest messages</AppLink></Button>
            </div>
          ) : null}
          <div ref={bottomRef} />
        </div>
      )}
    </div>
  );
}

export function CallsView() {
  const query = useInfiniteQuery({
    queryKey: ["m-calls"],
    queryFn: ({ pageParam, signal }) => importedApi.calls(pageParam, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });
  const summary = query.data?.pages[0]?.summary;
  const rows = useMemo(() => (query.data?.pages.flatMap((page) => page.items) ?? []).map(toCallRow), [query.data]);
  const cells: [string, number][] = summary ? [["Calls", summary.total], ["Missed", summary.missed], ["In", summary.incoming], ["Out", summary.outgoing]] : [];
  return (
    <div>
      <PageBar title="Calls" subtitle={query.data ? `Read from ${query.data.pages[0].read_from}` : undefined} />
      <Button asChild variant="outline" className="mx-4 mt-3 h-12 w-[calc(100%-2rem)] justify-between border-amber-500/60 bg-amber-100 text-amber-950 dark:bg-amber-950 dark:text-amber-100">
        <AppLink href="/m/unknown"><span>Unnamed numbers, most frequent first</span><span aria-hidden="true">&rsaquo;</span></AppLink>
      </Button>
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          {summary ? (
            <dl className="grid grid-cols-4 gap-2 p-4 pb-2 text-center">
              {cells.map(([label, value]) => (
                <Card key={label} className="gap-0 py-2"><CardContent className="px-1"><dd className="text-base font-semibold tabular-nums">{formatCount(value)}</dd><dt className="text-[11px] text-muted-foreground">{label}</dt></CardContent></Card>
              ))}
            </dl>
          ) : null}
          <div className="p-4 pt-2">
            {rows.length === 0 ? <Empty>No calls imported yet.</Empty> : <CallsTable rows={rows} />}
          </div>
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
        </>
      )}
    </div>
  );
}
