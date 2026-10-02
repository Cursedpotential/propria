// Byline: Claude Code · Sonnet · 2026-10-02
// One search box across the imported conversations (Weaviate ProfferChunks20261002 through /api/imported/search).
// A hit is a run of messages, one line per message; it opens its thread at the first of them.
import { useInfiniteQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useState } from "react";

import { Chip, Empty, ErrorBox, LoadMore, Loading, PageBar } from "@/components/mobile/mobile-ui";
import { formatDateTime } from "@/components/mobile/mobile-format";
import { importedApi } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";

function Highlighted({ text, query }: { text: string; query: string }) {
  const raw = query.split(/\s+/).filter((word) => word.length > 1).map((word) => word.toLowerCase());
  if (!raw.length) return <>{text}</>;
  const escaped = raw.map((word) => word.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const parts = text.split(new RegExp(`(${escaped.join("|")})`, "gi"));
  return (
    <>
      {parts.map((part, index) =>
        raw.includes(part.toLowerCase())
          ? <mark key={index} className="rounded bg-amber-200 px-0.5 text-foreground dark:bg-amber-500/40">{part}</mark>
          : <span key={index}>{part}</span>,
      )}
    </>
  );
}

export function SearchView() {
  const [draft, setDraft] = useState("");
  const [submitted, setSubmitted] = useState("");
  const query = useInfiniteQuery({
    queryKey: ["m-search", submitted],
    queryFn: ({ pageParam, signal }) => importedApi.search(submitted, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
    enabled: submitted.length > 0,
  });
  const hits = query.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <div>
      <PageBar title="Search" subtitle="Across every imported conversation" />
      <form
        className="flex gap-2 p-4"
        onSubmit={(event) => {
          event.preventDefault();
          setSubmitted(draft.trim());
        }}
        role="search"
      >
        <input
          type="search"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder="Words from a message"
          enterKeyHint="search"
          autoComplete="off"
          aria-label="Search imported messages"
          className="h-12 min-w-0 flex-1 rounded-lg border border-input bg-card px-4 text-base"
        />
        <button type="submit" className="flex h-12 w-14 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground active:opacity-80" aria-label="Search">
          <Search className="size-5" />
        </button>
      </form>
      {!submitted ? (
        <Empty>Type a few words and search. Each result opens its conversation at that message.</Empty>
      ) : query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          <p className="px-4 pb-2 text-xs text-muted-foreground">{query.data?.pages[0]?.note}</p>
          <ul className="space-y-3 px-4 pb-4">
            {hits.length === 0 ? <Empty>No imported conversation matches “{submitted}”.</Empty> : hits.map((hit) => {
              const body = hit.body.length > 320 ? `${hit.body.slice(0, 320)}...` : hit.body;
              const card = (
                <>
                  <p className="whitespace-pre-line text-[15px] leading-snug"><Highlighted text={body} query={submitted} /></p>
                  <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <span className="font-semibold text-foreground/80">{hit.sender}</span>
                    <span>{formatDateTime(hit.at)}</span>
                    {hit.format ? <Chip>{hit.format}</Chip> : null}
                    {hit.kind === "conversation" && (hit.message_count ?? 0) > 1 ? <span>{hit.message_count} messages</span> : null}
                    {hit.device ? <span>{hit.device}</span> : null}
                  </div>
                </>
              );
              return (
                <li key={`${hit.id}-${hit.thread_id}`}>
                  {hit.thread_id ? (
                    <AppLink href={`/m/thread/${hit.thread_id}?focus=${hit.id}`} className="block rounded-xl border border-border bg-card p-4 active:bg-muted">{card}</AppLink>
                  ) : <div className="rounded-xl border border-border bg-card p-4">{card}</div>}
                </li>
              );
            })}
          </ul>
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
        </>
      )}
    </div>
  );
}
