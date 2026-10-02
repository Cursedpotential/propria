// Byline: Claude Code · Opus 5.5 · 2026-10-01
// The Case Bible events behind one catalog count, in the Glide grid, newest
// first, paged by time. Read-only: the catalog is never written from here.
"use client";

import { DataEditor, GridCellKind, type GridCell, type GridColumn, type Item } from "@glideapps/glide-data-grid";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

import { useGridTheme } from "@/components/sbv/grid-theme";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { getCatalogEvents, type CatalogEvent } from "@/lib/case-identity-client";

import "@glideapps/glide-data-grid/dist/index.css";

export interface CatalogEventsTarget {
  identifier: string;
  matchOn: "counterparty_phone" | "sender";
  label: string;
}

const COLUMNS: GridColumn[] = [
  { title: "When (UTC)", id: "time", width: 150 },
  { title: "Kind", id: "kind", width: 72 },
  { title: "Source", id: "source", width: 130 },
  { title: "Direction", id: "direction", width: 84 },
  { title: "Sender", id: "sender", width: 150 },
  { title: "Conversation", id: "conversation", width: 170 },
  { title: "Text", id: "body", width: 460, grow: 1 },
];

function cellText(event: CatalogEvent | undefined, column: number): string {
  if (!event) return "";
  switch (column) {
    case 0:
      return event.event_ts_utc ? event.event_ts_utc.replace("T", " ").slice(0, 19) : "";
    case 1:
      return event.event_kind;
    case 2:
      return event.source_format;
    case 3:
      return event.direction ?? "";
    case 4:
      return event.sender ?? "";
    case 5:
      return event.conversation_title ?? event.contact_name ?? "";
    default:
      return (event.body ?? "").replace(/\s+/g, " ");
  }
}

export function CatalogEventsSheet({ target, onClose }: { target: CatalogEventsTarget | null; onClose: () => void }) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);
  const events = useInfiniteQuery({
    queryKey: ["case-identity", "catalog-events", target?.identifier, target?.matchOn],
    queryFn: ({ pageParam }) => getCatalogEvents(target!.identifier, target!.matchOn, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (page) => (page.has_more ? page.next_before : undefined),
    enabled: Boolean(target),
  });
  const rows = useMemo(() => events.data?.pages.flatMap((page) => page.items) ?? [], [events.data]);
  const getCellContent = useCallback(
    ([column, row]: Item): GridCell => {
      const text = cellText(rows[row], column);
      return { kind: GridCellKind.Text, data: text, displayData: text, allowOverlay: true, readonly: true };
    },
    [rows],
  );

  return (
    <Sheet open={Boolean(target)} onOpenChange={(open) => (open ? undefined : onClose())}>
      <SheetContent side="right" className="w-[min(1100px,96vw)] sm:max-w-none">
        <SheetHeader>
          <SheetTitle>{target?.label}</SheetTitle>
          <SheetDescription>
            Case Bible catalog, <code>raw_duck.comm_events_20260918</code>, matched on{" "}
            {target?.matchOn === "counterparty_phone" ? "the other party's number" : "the sender"} ={" "}
            <code>{target?.identifier}</code>. Newest first; select a cell to read it whole.
          </SheetDescription>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 flex-col gap-2 px-4 pb-4">
          {events.isError && <p className="text-sm text-destructive">{(events.error as Error).message}</p>}
          <div ref={setHost} className="min-h-0 flex-1 rounded-md border" data-testid="case-catalog-events-grid">
            {theme && (
              <DataEditor
                width="100%"
                height="100%"
                columns={COLUMNS}
                rows={rows.length}
                rowHeight={30}
                headerHeight={32}
                theme={theme}
                getCellContent={getCellContent}
                rowMarkers="number"
                smoothScrollY
                keybindings={{ search: false }}
                onVisibleRegionChanged={(range) => {
                  if (range.y + range.height + 30 >= rows.length && events.hasNextPage && !events.isFetchingNextPage) {
                    void events.fetchNextPage();
                  }
                }}
              />
            )}
          </div>
          <p className="flex items-center gap-2 text-xs text-muted-foreground">
            {events.isFetching && <Loader2 className="h-3 w-3 animate-spin" />}
            {rows.length.toLocaleString()} shown{events.hasNextPage ? " · scroll for more" : ""}
          </p>
        </div>
      </SheetContent>
    </Sheet>
  );
}
