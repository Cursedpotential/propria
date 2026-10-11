// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// /conversations — the imported conversations on the desktop: pick a source file, check whole conversations, then Extract or
// Send to Surreal; read one conversation's messages and what each extractor found in it. The same actions as /m, built from
// the same pieces: ImportedGrid for the sources, ConversationList (with its row checkboxes) for the conversations,
// MessageBubble for the messages, and components/conversations/* for the actions and the read-only Extractions view.
"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { ConversationToolbar } from "@/components/conversations/conversation-actions";
import { ExtractionsView } from "@/components/conversations/extractions-view";
import { SelectableConversations, type SelectableItem } from "@/components/conversations/selectable-conversations";
import { ImportedGrid, type ImportedColumn } from "@/components/imported/imported-grid";
import { toMessageRow } from "@/components/imported/record-rows";
import { formatCount, formatDate, statusLabel } from "@/components/mobile/mobile-format";
import { Empty, ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Button } from "@/components/ui/button";
import { importedApi, type ImportedSource } from "@/lib/imported-client";
import { useBrowserSearchParams } from "@/lib/router-compat";

const SOURCE_COLUMNS: ImportedColumn<ImportedSource>[] = [
  { id: "file", title: "File", width: 220, grow: 1, phone: true, text: (s) => s.file_name },
  { id: "status", title: "Status", width: 140, phone: true, text: (s) => statusLabel(s.status), sort: (s) => s.status },
  { id: "format", title: "Format", width: 80, text: (s) => s.format },
  { id: "device", title: "Phone", width: 130, text: (s) => s.device ?? "" },
  { id: "normalized", title: "Records", width: 90, text: (s) => formatCount(s.normalized), sort: (s) => s.normalized },
  { id: "imported", title: "Imported", width: 110, text: (s) => formatDate(s.imported_at), sort: (s) => s.imported_at ?? "" },
];

function SourcePicker({ onPick }: { onPick: (source: ImportedSource) => void }) {
  const query = useInfiniteQuery({
    queryKey: ["m-sources"],
    queryFn: ({ pageParam, signal }) => importedApi.sources(pageParam, undefined, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <ImportedGrid
      className="h-72"
      rows={rows}
      columns={SOURCE_COLUMNS}
      filterLabel="Filter files, phones, status"
      empty="Nothing imported yet."
      onRowClick={onPick}
      onReachEnd={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
    />
  );
}

function SourceConversations({ sourceId }: { sourceId: string }) {
  const query = useInfiniteQuery({
    queryKey: ["m-threads", sourceId],
    queryFn: ({ pageParam, signal }) => importedApi.threads(sourceId, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const items = useMemo<SelectableItem[]>(
    () => (query.data?.pages.flatMap((page) => page.items) ?? []).map((thread) => ({
      id: thread.id,
      href: `/conversations?source=${encodeURIComponent(sourceId)}&thread=${encodeURIComponent(thread.id)}`,
      contactName: thread.title,
      address: thread.participants.find((p) => !p.mine)?.id ?? null,
      type: thread.calls && !thread.messages ? "call" : "message",
      lastMessage: thread.last_message,
      lastDate: thread.last_at,
      count: thread.messages || thread.records,
      tag: thread.party === "first_party" ? "First-party" : thread.party === "third_party" ? "Third-party" : thread.party === "mixed" ? "First + third" : null,
    })),
    [query.data, sourceId],
  );
  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <>
      <SelectableConversations items={items} placement="desktop" />
      {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
    </>
  );
}

function Messages({ threadId }: { threadId: string }) {
  const query = useInfiniteQuery({
    queryKey: ["m-messages", threadId, null],
    queryFn: ({ pageParam, signal }) => importedApi.messages(threadId, { cursor: pageParam, around: null }, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.older_cursor ?? undefined,
  });
  const messages = useMemo(() => [...(query.data?.pages ?? [])].reverse().flatMap((page) => page.items), [query.data]);
  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <div className="space-y-1.5 px-3 py-3" aria-label="Conversation">
      {query.hasNextPage
        ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load earlier messages" />
        : <p className="py-2 text-center text-xs text-muted-foreground">Start of this conversation</p>}
      {messages.map((message, index) => (
        <MessageBubble key={message.id} row={toMessageRow(message, index)} previewHandle="" mode="LIVE" showSenderLabel={!message.outgoing} highlighted={false} />
      ))}
    </div>
  );
}

export default function ConversationsPage() {
  const router = useRouter();
  const params = useBrowserSearchParams();
  const sourceId = params.get("source");
  const threadId = params.get("thread");
  const [view, setView] = useState<"messages" | "extractions">("messages");
  return (
    <div className="mx-auto grid max-w-7xl gap-4 p-4 lg:grid-cols-[minmax(0,26rem)_minmax(0,1fr)]">
      <div className="min-w-0 space-y-4">
        <header>
          <h1 className="text-xl font-semibold">Imported conversations</h1>
          <p className="pt-1 text-sm text-muted-foreground">
            Pick a file, check whole conversations, then extract entities and events or send them to Surreal.
          </p>
        </header>
        <section aria-label="Source files"><SourcePicker onPick={(source) => void router.navigate({ href: `/conversations?source=${encodeURIComponent(source.id)}` })} /></section>
        <section aria-label="Conversations" className="rounded-lg border border-border">
          {sourceId ? <SourceConversations sourceId={sourceId} /> : <Empty>Choose a file above to list its conversations.</Empty>}
        </section>
      </div>
      <section aria-label="Conversation" className="min-w-0 rounded-lg border border-border">
        {threadId ? (
          <>
            <ConversationToolbar threadId={threadId} placement="desktop" />
            <div className="flex gap-2 px-3 pt-3" role="group" aria-label="View">
              <Button type="button" variant={view === "messages" ? "default" : "outline"} className="h-10" onClick={() => setView("messages")}>Messages</Button>
              <Button type="button" variant={view === "extractions" ? "default" : "outline"} className="h-10" onClick={() => setView("extractions")}>Extractions</Button>
            </div>
            {view === "messages" ? <Messages threadId={threadId} /> : <ExtractionsView threadId={threadId} />}
          </>
        ) : (
          <Empty>Open a conversation to read it and see what the extractors found.</Empty>
        )}
      </section>
    </div>
  );
}
