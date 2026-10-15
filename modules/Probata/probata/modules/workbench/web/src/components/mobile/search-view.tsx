// Byline: Claude Code · Sonnet · 2026-10-02
// One search box across the imported conversations (Weaviate ProfferChunks20261002 through /api/imported/search).
// A hit is a chunk; native chunks open the registered source in SBV, legacy chunks retain thread navigation.
import { useInfiniteQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useState } from "react";

import { Empty, ErrorBox, LoadMore, Loading, PageBar } from "@/components/mobile/mobile-ui";
import { formatDateTime } from "@/components/mobile/mobile-format";
import { Badge } from "@/components/ui/badge";
import { importedApi } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";
import { nativeSearchTargets } from "@/lib/search-read-target";
import { attemptReadHref } from "@/lib/workflow-links";

const LIVE_MODE = "LIVE" as const;

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

/** Search imported chunks and open the reader matching their actual source coordinates.
 * Inputs: query and display mode. Outputs: cited chunk cards and native source or legacy thread links.
 * Effects: paged search GETs and navigation. Choose for mobile imported search without per-message index assumptions.
 */
export function SearchView() {
  // The mobile shell is always the live case and has no FixedCaseProvider;
  // reading the fixed-case context here crashed /m/search (owner UX test 2026-10-15).
  const mode = LIVE_MODE;
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
        <Empty>Type a few words and search. Open a result’s source to read its messages.</Empty>
      ) : query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          <p className="px-4 pb-2 text-xs text-muted-foreground">{query.data?.pages[0]?.note}</p>
          <ul className="space-y-3 px-4 pb-4">
            {hits.length === 0 ? <Empty>No imported conversation matches “{submitted}”.</Empty> : hits.map((hit) => {
              const target = nativeSearchTargets(hit);
              const body = hit.body.length > 320 ? `${hit.body.slice(0, 320)}...` : hit.body;
              const card = (
                <>
                  <p className="whitespace-pre-line text-[15px] leading-snug"><Highlighted text={body} query={submitted} /></p>
                  <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <span className="font-semibold text-foreground/80">{hit.sender}</span>
                    <span>{formatDateTime(hit.at)}</span>
                    {hit.format ? <Badge variant="secondary">{hit.format}</Badge> : null}
                    {hit.kind === "conversation" && (hit.message_count ?? 0) > 1 ? <span>{hit.message_count} messages</span> : null}
                    {hit.device ? <span>{hit.device}</span> : null}
                  </div>
                </>
              );
              return (
                <li key={`${hit.collection ?? "imported"}:${hit.object_id ?? `${hit.id}-${hit.thread_id}`}`}>
                  {target.native ? <div className="rounded-xl border border-border bg-card p-4">{card}
                    <div className="mt-2 flex flex-wrap gap-3">{target.handles.map((handle) => <AppLink key={handle} href={attemptReadHref(handle, mode)} className="text-sm underline">Read source messages{target.handles.length > 1 ? ` · ${handle.slice(6)}` : ""}</AppLink>)}</div>
                    {!target.handles.length && <p className="mt-2 text-xs text-muted-foreground">This indexed passage has no verified human-source reader reference. Its excerpt and source coordinates remain available.</p>}
                  </div> : hit.thread_id ? (
                    <AppLink href={`/m/thread/${encodeURIComponent(hit.thread_id)}?${new URLSearchParams({ focus: hit.id })}`} className="block rounded-xl border border-border bg-card p-4 active:bg-muted">{card}</AppLink>
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
