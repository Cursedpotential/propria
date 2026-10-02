// Byline: Claude Code · Sonnet · 2026-10-02
// ONE list of who still needs naming, most frequent first: numbers no person carries yet, and people
// still unconfirmed (a placeholder for a number nobody named, or a person only a contact export named).
// Shared by the mobile /m/unknown page and the desktop /unknown-numbers page.
import { useInfiniteQuery } from "@tanstack/react-query";
import { useState } from "react";

import { WhoIsThis, prettyNumber } from "@/components/identity/who-is-this";
import { importedApi } from "@/lib/imported-client";
import { cn } from "@/lib/utils";

const count = new Intl.NumberFormat("en-US");
const KINDS = [["all", "All"], ["no_person", "No person yet"], ["unconfirmed", "Unconfirmed"]] as const;

export function UnknownNumbersList({ className }: { className?: string }) {
  const [search, setSearch] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [kind, setKind] = useState<(typeof KINDS)[number][0]>("all");
  const query = useInfiniteQuery({
    queryKey: ["unknown-numbers", submitted, kind],
    queryFn: ({ pageParam, signal }) => importedApi.unknownNumbers(pageParam, submitted || undefined, kind, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const total = query.data?.pages[0]?.total;
  return (
    <div className={className}>
      <form className="flex gap-2 p-4 pb-2" role="search" onSubmit={(event) => { event.preventDefault(); setSubmitted(search.trim()); }}>
        <input type="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Find a number or name"
          aria-label="Find a number or name" className="h-12 min-w-0 flex-1 rounded-lg border border-input bg-card px-4 text-base" />
        <button type="submit" className="h-12 shrink-0 rounded-lg bg-primary px-5 font-semibold text-primary-foreground active:opacity-80">Find</button>
      </form>
      <div className="flex gap-2 overflow-x-auto px-4 pb-2" role="tablist" aria-label="Show">
        {KINDS.map(([value, label]) => (
          <button key={value} type="button" role="tab" aria-selected={kind === value} onClick={() => setKind(value)}
            className={cn("h-11 shrink-0 rounded-full border px-4 text-sm font-semibold", kind === value ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card")}>
            {label}
          </button>
        ))}
      </div>
      <p className="px-4 pb-2 text-xs text-muted-foreground">
        {total === undefined ? "Loading" : `${count.format(total)} to name, most frequent first.`}
      </p>
      {query.isError ? <p className="p-4 text-sm text-destructive" role="alert">Could not load this list.</p> : null}
      <ul className="divide-y divide-border">
        {items.map((item) => (
          <li key={`${item.kind}-${item.entity_id ?? item.number}`} className="flex items-center justify-between gap-3 px-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-[15px] font-semibold tabular-nums">
                {item.named && item.name ? item.name : item.number ? prettyNumber(item.number) : item.label}
              </p>
              <p className="text-xs text-muted-foreground">
                {item.named && item.number ? `${prettyNumber(item.number)} · ` : ""}
                {item.kind === "no_person" ? `${count.format(item.total)} rows, no person yet` : `${count.format(item.messages)} messages · ${count.format(item.calls)} calls`}
                {item.named ? " · from contacts, unconfirmed" : ""}
              </p>
              {item.candidates.length > 1 ? <p className="text-xs text-muted-foreground">Contacts say: {item.candidates.join(", ")}</p> : null}
            </div>
            <WhoIsThis number={item.number} label={item.label} currentName={item.named ? item.name : null} entityId={item.entity_id}
              candidates={item.candidates} context="the unnamed numbers list" />
          </li>
        ))}
      </ul>
      {query.hasNextPage ? (
        <div className="p-4">
          <button type="button" onClick={() => void query.fetchNextPage()} disabled={query.isFetchingNextPage}
            className="h-12 w-full rounded-lg border border-border bg-card text-sm font-semibold active:bg-muted disabled:opacity-60">
            {query.isFetchingNextPage ? "Loading" : "Load more"}
          </button>
        </div>
      ) : null}
    </div>
  );
}
