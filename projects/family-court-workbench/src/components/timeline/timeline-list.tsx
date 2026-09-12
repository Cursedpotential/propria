// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import type { CaseTimelineEntry } from "@/types/store";

const columns: ColumnDef<CaseTimelineEntry>[] = [
  {
    accessorKey: "date",
    header: "Date",
    cell: (ctx) => <span className="font-mono text-xs tabular-nums text-text-secondary">{ctx.getValue<string>()}</span>,
    size: 180,
  },
  {
    accessorKey: "lane",
    header: "Lane",
    cell: (ctx) => <Badge tone={ctx.getValue<string>() === "court" ? "accent" : "neutral"}>{ctx.getValue<string>()}</Badge>,
    size: 90,
  },
  {
    accessorKey: "table",
    header: "Type",
    cell: (ctx) => <Badge tone="neutral">{ctx.getValue<string>()}</Badge>,
    size: 100,
  },
  {
    accessorKey: "summary",
    header: "Summary",
    cell: (ctx) => <span className="truncate">{ctx.getValue<string>()}</span>,
  },
];

/** Virtualised (TanStack Virtual) list over a TanStack Table row model — the row height is fixed (36px) which is what makes windowing safe here. */
export function TimelineList({ entries }: { entries: CaseTimelineEntry[] }) {
  const parentRef = React.useRef<HTMLDivElement>(null);

  const table = useReactTable({
    data: entries,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const rows = table.getRowModel().rows;
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 36,
    overscan: 12,
  });

  return (
    <div className="flex flex-col overflow-hidden rounded-[var(--radius-md)] border border-border">
      <div className="grid grid-cols-[180px_90px_100px_1fr] gap-2 border-b border-border bg-surface-raised px-3 py-1.5 text-xs font-medium uppercase tracking-wide text-text-tertiary">
        {table.getFlatHeaders().map((h) => (
          <div key={h.id}>{flexRender(h.column.columnDef.header, h.getContext())}</div>
        ))}
      </div>
      <div ref={parentRef} className="h-[520px] overflow-y-auto">
        {rows.length === 0 && <p className="p-4 text-sm text-text-tertiary">No entries in this lane/date range.</p>}
        <div style={{ height: virtualizer.getTotalSize(), position: "relative" }}>
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const row = rows[virtualRow.index];
            return (
              <div
                key={row.id}
                className="absolute left-0 top-0 grid w-full grid-cols-[180px_90px_100px_1fr] items-center gap-2 border-b border-border px-3 text-sm"
                style={{ height: virtualRow.size, transform: `translateY(${virtualRow.start}px)` }}
              >
                {row.getVisibleCells().map((cell) => (
                  <div key={cell.id} className="min-w-0">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
