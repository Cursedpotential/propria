// Byline: Claude Code · Opus 5 · 2026-09-22
// CENTRE region of Sources: the file rows, on Glide Data Grid.
//
// This file is the only place in the Sources screen that imports Glide, the
// same containment rule the Review message browser follows (pin 6.0.4-alpha24,
// one adapter file). Everything above it deals in plain row objects.
"use client";

import {
  CompactSelection,
  DataEditor,
  GridCellKind,
  type GridCell,
  type GridColumn,
  type GridSelection,
  type Item,
  type Theme,
} from "@glideapps/glide-data-grid";
import { useCallback, useMemo, useState } from "react";

import { useTheme } from "@/components/layout/theme-provider";
import { SOURCE_STATE_LABEL, formatBytes, type SourceState } from "@/components/sources/source-state";

import "@glideapps/glide-data-grid/dist/index.css";

/** Rows remaining below the viewport before the next listing page is requested. */
const PREFETCH_ROW_MARGIN = 30;

// Widths are chosen so the State and Unit marks are on screen at the panel's
// default width — a mark the operator has to scroll sideways to find is not a
// mark. Name takes the slack instead of a fixed 380 (live screenshot fix,
// 2026-09-22): at ~700px the old set ran 914px wide and cut "State" in half.
const COLUMNS: GridColumn[] = [
  { title: "Name", id: "name", width: 230, grow: 1 },
  { title: "Size", id: "size", width: 78 },
  { title: "Modified", id: "modified", width: 132 },
  { title: "State", id: "state", width: 104 },
  { title: "Unit", id: "unit", width: 104 },
];

export interface SourceGridRow {
  sourceRef: string;
  name: string;
  byteLength: number;
  lastModified: string | null;
  state: SourceState;
  /** "" when the file belongs to no unit; otherwise the unit's kind. */
  unitMark: string;
}

const FALLBACK_PALETTE = {
  light: {
    primary: "#4051b9",
    accent: "#e9ecfb",
    foreground: "#1d2228",
    mutedForeground: "#687078",
    card: "#fffefb",
    muted: "#ebe8e0",
    border: "#d5d1c9",
  },
  dark: {
    primary: "#8591f0",
    accent: "#313a66",
    foreground: "#f0f1ef",
    mutedForeground: "#b1b8bd",
    card: "#242e36",
    muted: "#2c373f",
    border: "#43505a",
  },
} as const;

function cssVariable(element: HTMLElement, name: string, fallback: string) {
  return getComputedStyle(element).getPropertyValue(name).trim() || fallback;
}

/** Glide paints to a canvas, so the surface tokens are read back, not inherited. */
function useGridTheme(host: HTMLElement | null): Partial<Theme> | undefined {
  const { resolvedTheme } = useTheme();
  const fallback = FALLBACK_PALETTE[resolvedTheme === "dark" ? "dark" : "light"];
  return useMemo(() => {
    if (!host) return undefined;
    return {
      accentColor: cssVariable(host, "--primary", fallback.primary),
      accentLight: cssVariable(host, "--accent", fallback.accent),
      textDark: cssVariable(host, "--foreground", fallback.foreground),
      textMedium: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textLight: cssVariable(host, "--muted-foreground", fallback.mutedForeground),
      textHeader: cssVariable(host, "--foreground", fallback.foreground),
      bgCell: cssVariable(host, "--card", fallback.card),
      bgCellMedium: cssVariable(host, "--muted", fallback.muted),
      bgHeader: cssVariable(host, "--muted", fallback.muted),
      bgHeaderHasFocus: cssVariable(host, "--accent", fallback.accent),
      bgHeaderHovered: cssVariable(host, "--accent", fallback.accent),
      borderColor: cssVariable(host, "--border", fallback.border),
      horizontalBorderColor: cssVariable(host, "--border", fallback.border),
      fontFamily: cssVariable(host, "--font-sans", "system-ui, sans-serif"),
      baseFontStyle: "13px",
      headerFontStyle: "600 12px",
    };
  }, [fallback, host]);
}

export function SourceRowsGrid({
  rows,
  selectedIndex,
  checkedRefs,
  onSelectIndex,
  onToggleChecked,
  onReachEnd,
}: {
  rows: readonly SourceGridRow[];
  selectedIndex: number;
  checkedRefs: ReadonlySet<string>;
  onSelectIndex: (index: number) => void;
  onToggleChecked: (sourceRefs: string[]) => void;
  onReachEnd: () => void;
}) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);

  const getCellContent = useCallback(
    ([column, row]: Item): GridCell => {
      const entry = rows[row];
      const text = (() => {
        if (!entry) return "";
        if (column === 0) return entry.name;
        if (column === 1) return formatBytes(entry.byteLength);
        if (column === 2) return entry.lastModified ? new Date(entry.lastModified).toLocaleString() : "—";
        if (column === 3) return SOURCE_STATE_LABEL[entry.state];
        return entry.unitMark;
      })();
      return { kind: GridCellKind.Text, data: text, displayData: text, allowOverlay: false, readonly: true };
    },
    [rows],
  );

  const checkedRows = useMemo(() => {
    let selection = CompactSelection.empty();
    rows.forEach((row, index) => {
      if (checkedRefs.has(row.sourceRef)) selection = selection.add(index);
    });
    return selection;
  }, [checkedRefs, rows]);

  const selection = useMemo<GridSelection>(
    () => ({
      columns: CompactSelection.empty(),
      rows: checkedRows,
      current:
        selectedIndex >= 0 && selectedIndex < rows.length
          ? {
              cell: [0, selectedIndex] as Item,
              range: { x: 0, y: selectedIndex, width: COLUMNS.length, height: 1 },
              rangeStack: [],
            }
          : undefined,
    }),
    [checkedRows, rows.length, selectedIndex],
  );

  const handleSelectionChange = useCallback(
    (next: GridSelection) => {
      const cell = next.current?.cell;
      if (cell) onSelectIndex(cell[1]);
      const marked: string[] = [];
      for (const index of next.rows) {
        const row = rows[index];
        if (row) marked.push(row.sourceRef);
      }
      onToggleChecked(marked);
    },
    [onSelectIndex, onToggleChecked, rows],
  );

  return (
    <div ref={setHost} className="h-full min-h-0 w-full" data-testid="source-rows-grid">
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
          gridSelection={selection}
          onGridSelectionChange={handleSelectionChange}
          rangeSelect="cell"
          rowMarkers="checkbox"
          rowSelect="multi"
          smoothScrollY
          keybindings={{ search: false }}
          onVisibleRegionChanged={(range) => {
            if (range.y + range.height + PREFETCH_ROW_MARGIN >= rows.length) onReachEnd();
          }}
        />
      )}
    </div>
  );
}
