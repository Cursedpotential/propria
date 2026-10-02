// Byline: Claude Code · Sonnet · 2026-10-02
// Placeholders still unnamed, most frequent first, so the owner can work through the biggest ones.
// Shared by the mobile /m/unknown page and the desktop /unknown-numbers page.
import { useInfiniteQuery } from "@tanstack/react-query";
import { useState } from "react";

import { WhoIsThis, prettyNumber } from "@/components/identity/who-is-this";
import { importedApi } from "@/lib/imported-client";
import { cn } from "@/lib/utils";

const count = new Intl.NumberFormat("en-US");

export function UnknownNumbersList({ className }: { className?: string }) {
  const [search, setSearch] = useState("");
  const [submitted, setSubmitted] = useState("");
  const query = useInfiniteQuery({
    queryKey: ["unknown-numbers", submitted],
    queryFn: ({ pageParam, signal }) => importedApi.unknownNumbers(pageParam, submitted || undefined, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const total = query.data?.pages[0]?.total;
  return (
    <div className={className}>
      <form className="flex gap-2 p-4" role="search" onSubmit={(event) => { event.preventDefault(); setSubmitted(search.replace(/\D/g, "")); }}>
        <input type="search" inputMode="numeric" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Find a number"
          aria-label="Find an unnamed number" className="h-12 min-w-0 flex-1 rounded-lg border border-input bg-card px-4 text-base" />
        <button type="submit" className="h-12 shrink-0 rounded-lg bg-primary px-5 font-semibold text-primary-foreground active:opacity-80">Find</button>
      </form>
      <p className="px-4 pb-2 text-xs text-muted-foreground">
        {total === undefined ? "Loading" : `${count.format(total)} numbers still unnamed, most frequent first.`}
      </p>
      {query.isError ? <p className="p-4 text-sm text-destructive" role="alert">Could not load the unnamed numbers.</p> : null}
      <ul className="divide-y divide-border">
        {items.map((item) => (
          <li key={item.entity_id} className="flex items-center justify-between gap-3 px-4 py-3">
            <div className="min-w-0">
              <p className="text-[15px] font-semibold tabular-nums">{prettyNumber(item.number)}</p>
              <p className="text-xs text-muted-foreground">
                {count.format(item.messages)} messages · {count.format(item.calls)} calls
              </p>
              {item.candidates.length > 0 ? <p className="text-xs text-muted-foreground">Contacts say: {item.candidates.join(", ")}</p> : null}
            </div>
            <WhoIsThis number={item.number} entityId={item.entity_id} candidates={item.candidates} context="the unnamed numbers list" />
          </li>
        ))}
      </ul>
      {query.hasNextPage ? (
        <div className="p-4">
          <button type="button" onClick={() => void query.fetchNextPage()} disabled={query.isFetchingNextPage}
            className={cn("h-12 w-full rounded-lg border border-border bg-card text-sm font-semibold active:bg-muted disabled:opacity-60")}>
            {query.isFetchingNextPage ? "Loading" : "Load more"}
          </button>
        </div>
      ) : null}
    </div>
  );
}
