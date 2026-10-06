// Byline: Codex · GPT-6 · 2026-10-06
import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { SelectableConversations, type SelectableItem } from "@/components/conversations/selectable-conversations";
import { ImportedGrid, type ImportedColumn } from "@/components/imported/imported-grid";
import { formatCount, statusLabel } from "@/components/mobile/mobile-format";
import { Empty, ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { ReadConversation } from "@/components/read/read-conversation";
import { readHref } from "@/components/read/read-location";
import { ReadSearch } from "@/components/read/read-search";
import { Button } from "@/components/ui/button";
import { importedApi, type ImportedSource } from "@/lib/imported-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";

const SOURCE_COLUMNS: ImportedColumn<ImportedSource>[] = [
  { id: "file", title: "Source file", width: 220, grow: 1, phone: true, text: (source) => source.file_name },
  { id: "format", title: "Format", width: 90, phone: true, text: (source) => source.format },
  { id: "status", title: "Status", width: 130, text: (source) => statusLabel(source.status) },
  { id: "records", title: "Records", width: 90, text: (source) => formatCount(source.normalized), sort: (source) => source.normalized },
];

/** List imported source files using the established paged grid and API.
 * Input: current route context. Output: source selection links; effects: GETs and navigation.
 * Pick for entering Read from a file rather than a content-search hit.
 */
function ReadSources({ params }: { params: URLSearchParams }) {
  const router = useRouter();
  const query = useInfiniteQuery({
    queryKey: ["m-sources"],
    queryFn: ({ pageParam, signal }) => importedApi.sources(pageParam, undefined, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  if (query.isPending) return <Loading />;
  return (
    <>
      {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
      <ImportedGrid className="h-64" rows={rows} columns={SOURCE_COLUMNS} filterLabel="Filter loaded source files" empty="No imported sources yet."
        onRowClick={(source) => void router.navigate({ href: readHref(params, { source: source.id, thread: null, around: null }) })} />
      {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load more sources" /> : null}
    </>
  );
}

/** List and locally filter paged conversations belonging to one original source.
 * Inputs: original source ID and query context. Output: complete-conversation selection.
 * Effects: GETs, URL navigation, and explicit existing bulk extraction/send actions.
 * Pick after choosing a source file; shared SelectableConversations owns selection/actions.
 */
function ReadThreads({ sourceId, params }: { sourceId: string; params: URLSearchParams }) {
  const [filter, setFilter] = useState("");
  const query = useInfiniteQuery({
    queryKey: ["m-threads", sourceId],
    queryFn: ({ pageParam, signal }) => importedApi.threads(sourceId, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const threads = query.data?.pages.flatMap((page) => page.items) ?? [];
  const visible = threads.filter((thread) => [thread.title, thread.last_message, ...thread.participants.map((person) => person.label)].join(" ").toLowerCase().includes(filter.toLowerCase()));
  const items: SelectableItem[] = visible.map((thread) => ({
    id: thread.id,
    href: readHref(params, { source: sourceId, thread: thread.id, around: null }),
    contactName: thread.title,
    address: thread.participants.find((person) => !person.mine)?.id ?? null,
    type: thread.calls && !thread.messages ? "call" : "message",
    lastMessage: thread.last_message,
    lastDate: thread.last_at,
    count: thread.messages || thread.records,
    tag: thread.party === "first_party" ? "First-party" : thread.party === "third_party" ? "Third-party" : thread.party === "mixed" ? "First + third" : null,
  }));
  if (query.isPending) return <Loading />;
  return (
    <div className="space-y-3">
      {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
      {query.data ? <p className="break-words text-sm font-medium">{query.data.pages[0].source.file_name}</p> : null}
      <input type="search" aria-label="Filter loaded conversations" value={filter} onChange={(event) => setFilter(event.target.value)} placeholder="Filter loaded conversations" className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm" />
      <div className="max-h-[32rem] overflow-y-auto"><SelectableConversations items={items} placement="desktop" /></div>
      {filter && !items.length ? <p className="text-xs text-muted-foreground">Clear the filter or load more conversations.</p> : null}
      {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load more conversations" /> : null}
    </div>
  );
}

/** Compose source selection, content search and a cited conversation reader.
 * Input: source/thread/around/q URL parameters. Output: an adaptive Read workspace.
 * Effects: child GETs and user navigation; no automatic corpus writes.
 * Pick for imported reading; processing previews use the existing preview client.
 */
export function ReadWorkspace() {
  const params = useBrowserSearchParams();
  const sourceId = params.get("source");
  const threadId = params.get("thread");
  const around = params.get("around");
  return (
    <div className="mx-auto max-w-[100rem] space-y-5 p-4 lg:p-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div><h1 className="text-xl font-semibold">Read</h1><p className="mt-1 text-sm text-muted-foreground">Open a conversation, read its messages, and follow extracted context back to the source.</p></div>
        <Button asChild variant="outline"><AppLink href={readHref(params, { view: "review" })}>Processing previews</AppLink></Button>
      </header>
      <div className="grid items-start gap-5 xl:grid-cols-[24rem_minmax(0,1fr)]">
        <aside className="min-w-0 space-y-4" aria-label="Reading selection">
          <details key={sourceId ?? "no-source"} open={!sourceId} className="rounded-lg border border-border bg-card p-3">
            <summary className="cursor-pointer text-sm font-semibold">{sourceId ? "Change source file" : "Source files"}</summary>
            <section className="mt-3" aria-label="Source files"><ReadSources params={params} /></section>
          </details>
          <section id="read-conversations" className="scroll-mt-4 rounded-lg border border-border bg-card p-3" aria-label="Conversations">
            <h2 className="mb-3 text-sm font-semibold">Conversations</h2>
            {sourceId ? <ReadThreads key={sourceId} sourceId={sourceId} params={params} /> : <Empty>Choose a source file, or search message content.</Empty>}
          </section>
        </aside>
        <div className="min-w-0 space-y-5">
          <ReadSearch key={params.get("q") ?? ""} params={params} />
          {threadId ? <ReadConversation key={`${threadId}:${around ?? ""}`} threadId={threadId} around={around} params={params} />
            : <section className="rounded-lg border border-border bg-card"><Empty>Choose a conversation or open a search result to start reading. Extracted context and source details appear alongside it.</Empty></section>}
        </div>
      </div>
    </div>
  );
}
