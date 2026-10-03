// Byline: Claude Code · Sonnet · 2026-10-02
// ONE list of who still needs naming, most frequent first: numbers no person carries yet, and people still
// unconfirmed (a placeholder for a number nobody named, or a person only a contact export named). On the
// shared ImportedGrid (Glide: sort, filter, column toggle, virtualized rows); tapping a row opens the
// "Who is this?" sheet. Used by the mobile /m/unknown page and the desktop /unknown-numbers page.
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

import { ImportedGrid, type ImportedColumn } from "@/components/imported/imported-grid";
import { WhoIsThisSheet, prettyNumber } from "@/components/identity/who-is-this";
import { Button } from "@/components/ui/button";
import { importedApi, type UnknownNumber } from "@/lib/imported-client";

const count = new Intl.NumberFormat("en-US");
const KINDS = [["all", "All"], ["no_person", "No person yet"], ["unconfirmed", "Unconfirmed"]] as const;

const COLUMNS: ImportedColumn<UnknownNumber>[] = [
  { id: "who", title: "Who", width: 210, grow: 1, phone: true, text: (r) => (r.named && r.name ? r.name : r.number ? prettyNumber(r.number) : r.label) },
  { id: "number", title: "Number", width: 140, text: (r) => (r.number ? prettyNumber(r.number) : "") },
  { id: "seen", title: "Seen", width: 110, phone: true, text: (r) => count.format(r.total), sort: (r) => r.total },
  { id: "messages", title: "Messages", width: 100, text: (r) => count.format(r.messages), sort: (r) => r.messages },
  { id: "calls", title: "Calls", width: 80, text: (r) => count.format(r.calls), sort: (r) => r.calls },
  { id: "state", title: "State", width: 170, text: (r) => (r.kind === "no_person" ? "No person yet" : r.named ? "From contacts, unconfirmed" : "Placeholder"), color: (r) => (r.kind === "no_person" ? "#d9534f" : r.named ? "#c98a1b" : undefined) },
  { id: "candidates", title: "Contacts say", width: 220, text: (r) => r.candidates.join(", ") },
];

export function UnknownNumbersList({ className }: { className?: string }) {
  const [kind, setKind] = useState<(typeof KINDS)[number][0]>("all");
  const [active, setActive] = useState<UnknownNumber | null>(null);
  const query = useInfiniteQuery({
    queryKey: ["unknown-numbers", kind],
    queryFn: ({ pageParam, signal }) => importedApi.unknownNumbers(pageParam, undefined, kind, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  const total = query.data?.pages[0]?.total;
  return (
    <div className={className}>
      <div className="flex gap-2 overflow-x-auto px-4 pt-3" role="tablist" aria-label="Show">
        {KINDS.map(([value, label]) => (
          <Button key={value} type="button" role="tab" aria-selected={kind === value} variant={kind === value ? "default" : "outline"} onClick={() => setKind(value)} className="h-11 shrink-0 rounded-full">{label}</Button>
        ))}
      </div>
      <p className="px-4 pt-2 text-xs text-muted-foreground">{total === undefined ? "Loading" : `${count.format(total)} to name, most frequent first. Tap one to name it.`}</p>
      {query.isError ? <p className="p-4 text-sm text-destructive" role="alert">Could not load this list.</p> : null}
      <ImportedGrid
        className="h-[calc(100dvh-17rem)] min-h-96"
        rows={rows}
        columns={COLUMNS}
        filterLabel="Find a number or name"
        empty="Everyone is named."
        onRowClick={setActive}
        onReachEnd={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
      />
      {active ? (
        <WhoIsThisSheet
          open
          onOpenChange={(open) => { if (!open) setActive(null); }}
          number={active.number}
          label={active.label}
          currentName={active.named ? active.name : null}
          entityId={active.entity_id}
          candidates={active.candidates}
          context="the unnamed numbers list"
        />
      ) : null}
    </div>
  );
}
