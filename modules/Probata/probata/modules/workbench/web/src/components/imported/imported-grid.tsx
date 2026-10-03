// Byline: Claude Code · Sonnet · 2026-10-02
// One Glide Data Grid adapter for every imported list (sources, files, unknown numbers): virtualized rows,
// header-click sorting, a filter box and a column toggle, on the Workbench's own grid theme
// (components/sbv/grid-theme.ts). Glide is the settled data-grid library of this app (see
// components/sbv/message-browser-grid.tsx and components/sources/source-rows-grid.tsx); this file is the one
// place the imported views import it. On a phone it starts with only the columns marked `phone` and the
// column toggle brings the rest back.
"use client";

import {
  CompactSelection,
  DataEditor,
  GridCellKind,
  type GridCell,
  type GridColumn,
  type GridSelection,
  type Item,
} from "@glideapps/glide-data-grid";
import { ArrowDown, ArrowUp, Check, Columns3 } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

import { useGridTheme } from "@/components/sbv/grid-theme";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { useIsMobile } from "@/hooks/use-mobile";
import { cn } from "@/lib/utils";

import "@glideapps/glide-data-grid/dist/index.css";

export interface ImportedColumn<Row> {
  id: string;
  title: string;
  width: number;
  /** Takes the leftover width when it is the only growing column. */
  grow?: number;
  text: (row: Row) => string;
  /** A colour for this cell's text (a status, for example). */
  color?: (row: Row) => string | undefined;
  /** What the column sorts by; defaults to its text. */
  sort?: (row: Row) => string | number;
  /** Shown on a phone before the owner toggles more on. */
  phone?: boolean;
}

interface ImportedGridProps<Row> {
  rows: readonly Row[];
  columns: readonly ImportedColumn<Row>[];
  onRowClick: (row: Row) => void;
  /** Called when the owner scrolls near the last loaded row (for paged lists). */
  onReachEnd?: () => void;
  filterLabel?: string;
  empty?: string;
  className?: string;
}

const PREFETCH_ROW_MARGIN = 20;

export function ImportedGrid<Row>({ rows, columns, onRowClick, onReachEnd, filterLabel = "Filter", empty = "Nothing to show.", className }: ImportedGridProps<Row>) {
  const phone = useIsMobile();
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);
  const [filter, setFilter] = useState("");
  const [sort, setSort] = useState<{ id: string; dir: 1 | -1 } | null>(null);
  const [hidden, setHidden] = useState<ReadonlySet<string> | null>(null);

  const visible = useMemo(() => {
    const defaults = new Set(columns.filter((column) => (phone ? column.phone : true)).map((column) => column.id));
    const chosen = hidden === null ? defaults : new Set(columns.filter((column) => !hidden.has(column.id)).map((column) => column.id));
    return columns.filter((column) => chosen.has(column.id));
  }, [columns, hidden, phone]);

  const shown = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    let out = needle
      ? rows.filter((row) => columns.some((column) => column.text(row).toLowerCase().includes(needle)))
      : [...rows];
    if (sort) {
      const column = columns.find((candidate) => candidate.id === sort.id);
      if (column) {
        const key = column.sort ?? column.text;
        out = out.sort((a, b) => {
          const left = key(a);
          const right = key(b);
          return (typeof left === "number" && typeof right === "number" ? left - right : String(left).localeCompare(String(right), undefined, { numeric: true })) * sort.dir;
        });
      }
    }
    return out;
  }, [columns, filter, rows, sort]);

  const gridColumns = useMemo<GridColumn[]>(
    () => visible.map((column) => ({ id: column.id, title: column.title + (sort?.id === column.id ? (sort.dir === 1 ? " ▲" : " ▼") : ""), width: column.width, grow: column.grow })),
    [sort, visible],
  );

  const getCellContent = useCallback(
    ([col, row]: Item): GridCell => {
      const column = visible[col];
      const entry = shown[row];
      const text = entry && column ? column.text(entry) : "";
      const color = entry && column?.color ? column.color(entry) : undefined;
      return {
        kind: GridCellKind.Text, data: text, displayData: text, allowOverlay: false, readonly: true,
        themeOverride: color ? { textDark: color } : undefined,
      };
    },
    [shown, visible],
  );

  const selection = useMemo<GridSelection>(() => ({ columns: CompactSelection.empty(), rows: CompactSelection.empty(), current: undefined }), []);
  const rowHeight = phone ? 48 : 34;

  function toggleColumn(id: string) {
    const current = new Set(visible.map((column) => column.id));
    if (current.has(id)) current.delete(id);
    else current.add(id);
    setHidden(new Set(columns.filter((column) => !current.has(column.id)).map((column) => column.id)));
  }

  return (
    <div className={cn("flex min-h-0 flex-col", className)}>
      <div className="flex items-center gap-2 px-4 py-2">
        <Input value={filter} onChange={(event) => setFilter(event.target.value)} placeholder={filterLabel} aria-label={filterLabel} className="h-11 flex-1 text-base" />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" className="h-11 shrink-0 gap-2" aria-label="Choose columns">
              <Columns3 className="size-4" /> <span className="hidden sm:inline">Columns</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {columns.map((column) => (
              <DropdownMenuItem key={column.id} onSelect={(event) => { event.preventDefault(); toggleColumn(column.id); }} className="min-h-10 gap-2">
                <Check className={cn("size-4", visible.some((v) => v.id === column.id) ? "opacity-100" : "opacity-0")} /> {column.title}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <p className="px-4 pb-1 text-xs text-muted-foreground" aria-live="polite">
        {shown.length} {shown.length === 1 ? "row" : "rows"}
        {sort ? <> · sorted by {columns.find((c) => c.id === sort.id)?.title} {sort.dir === 1 ? <ArrowUp className="inline size-3" /> : <ArrowDown className="inline size-3" />}</> : " · tap a column title to sort"}
      </p>
      <div ref={setHost} className="min-h-0 flex-1" data-testid="imported-grid">
        {shown.length === 0 ? (
          <p className="px-6 py-12 text-center text-sm text-muted-foreground">{empty}</p>
        ) : theme ? (
          <DataEditor
            width="100%"
            height="100%"
            columns={gridColumns}
            rows={shown.length}
            rowHeight={rowHeight}
            headerHeight={rowHeight}
            theme={theme}
            getCellContent={getCellContent}
            gridSelection={selection}
            onGridSelectionChange={() => undefined}
            onCellClicked={([, row]) => { const entry = shown[row]; if (entry) onRowClick(entry); }}
            onHeaderClicked={(col) => {
              const id = visible[col]?.id;
              if (id) setSort((prev) => (prev?.id === id ? (prev.dir === 1 ? { id, dir: -1 } : null) : { id, dir: 1 }));
            }}
            smoothScrollY
            smoothScrollX
            keybindings={{ search: false }}
            onVisibleRegionChanged={(range) => { if (onReachEnd && range.y + range.height + PREFETCH_ROW_MARGIN >= shown.length) onReachEnd(); }}
          />
        ) : null}
      </div>
    </div>
  );
}
