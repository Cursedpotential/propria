// Byline: Codex · GPT-6 · 2026-10-06
import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { formatDateTime } from "@/components/mobile/mobile-format";
import { Empty, ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { readHref, searchHitHref } from "@/components/read/read-location";
import { Button } from "@/components/ui/button";
import { importedApi } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";

/** Search indexed conversation chunks and call-log entries through the existing search API.
 * Input: q and reading context from the URL. Output: paged hits with original message links.
 * Effects: GETs after submitting a query and URL navigation; no indexing or writes.
 * Pick for message content, rather than the source/conversation list filters.
 */
export function ReadSearch({ params }: { params: URLSearchParams }) {
  const router = useRouter();
  const submitted = params.get("q")?.trim() ?? "";
  const [draft, setDraft] = useState(submitted);
  const query = useInfiniteQuery({
    queryKey: ["m-search", submitted],
    queryFn: ({ pageParam, signal }) => importedApi.search(submitted, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
    enabled: submitted.length > 0,
  });
  const hits = query.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <section className="rounded-lg border border-border bg-card p-4" aria-label="Content search">
      <form role="search" className="flex flex-wrap gap-2" onSubmit={(event) => { event.preventDefault(); void router.navigate({ href: readHref(params, { q: draft.trim() || null }) }); }}>
        <label htmlFor="read-content-search" className="w-full text-sm font-semibold">Search message content</label>
        <input id="read-content-search" type="search" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Words from a message" className="h-10 min-w-0 flex-1 rounded-md border border-input bg-background px-3 text-sm" />
        <Button type="submit">Search</Button>
        {submitted ? <Button asChild variant="ghost"><AppLink href={readHref(params, { q: null })}>Clear search</AppLink></Button> : null}
        <p className="w-full text-xs text-muted-foreground">Searches indexed conversation chunks and call-log entries. Conversations without published chunks are not searched.</p>
      </form>
      {submitted ? (
        <div className="mt-4" aria-live="polite">
          {query.isPending ? <Loading /> : null}
          {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
          {query.data ? <p className="mb-3 text-xs text-muted-foreground">{hits.length} loaded results for “{submitted}” · {query.data.pages[0].note}</p> : null}
          {query.isSuccess && !hits.length ? <Empty>No indexed conversation chunks match “{submitted}”. Try different words.</Empty> : null}
          <ul className="max-h-96 space-y-3 overflow-y-auto">
            {hits.map((hit) => (
              <li key={`${hit.id}:${hit.thread_id}`} className="rounded-md border border-border p-3">
                <p className="whitespace-pre-wrap break-words text-sm leading-relaxed">{hit.body}</p>
                <p className="mt-2 text-xs text-muted-foreground">{[hit.sender, formatDateTime(hit.at), hit.source, hit.format, hit.device].filter(Boolean).join(" · ")}</p>
                {hit.thread_id ? <AppLink className="mt-2 inline-block text-sm underline underline-offset-4" href={searchHitHref(params, hit.thread_id, hit.id)}>Read in conversation</AppLink>
                  : <p className="mt-2 text-xs text-muted-foreground">{hit.kind === "call_log" ? "Call-log result; no conversation link is supplied by the search API." : "Original conversation link unavailable; no conversation mapping is supplied by the search API."}</p>}
              </li>
            ))}
          </ul>
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load more results" /> : null}
        </div>
      ) : null}
    </section>
  );
}
