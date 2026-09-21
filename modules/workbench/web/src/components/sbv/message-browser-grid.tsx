// Byline: Claude Code · Opus 5 · 2026-09-20 (Glide Data Grid message list with cursor-driven paging)
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
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";

import "@glideapps/glide-data-grid/dist/index.css";

/** Rows remaining below the viewport before the next cursor page is requested. */
const PREFETCH_ROW_MARGIN = 40;

const COLUMNS: GridColumn[] = [
  { title: "Time", id: "time", width: 190 },
  { title: "Sender", id: "sender", width: 160 },
  { title: "Direction", id: "direction", width: 96 },
  { title: "Message", id: "body", width: 620, grow: 1 },
  { title: "Files", id: "attachments", width: 70 },
];

function cssVariable(element: HTMLElement, name: string, fallback: string) {
  const value = getComputedStyle(element).getPropertyValue(name).trim();
  return value || fallback;
}

/** Used only when a custom property is missing; mirrors `src/app/globals.css`. */
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

/**
 * Glide paints to a canvas, so it cannot inherit the surface tokens through CSS.
 * The palette is read back from the live custom properties whenever the resolved
 * light/dark theme changes, which keeps the grid inside the existing design system.
 */
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

interface MessageBrowserGridProps {
  rows: PreviewMessageRow[];
  selectedIndex: number;
  onSelectIndex: (index: number) => void;
  onReachEnd: () => void;
}

export function MessageBrowserGrid({
  rows,
  selectedIndex,
  onSelectIndex,
  onReachEnd,
}: MessageBrowserGridProps) {
  const [host, setHost] = useState<HTMLDivElement | null>(null);
  const theme = useGridTheme(host);

  const getCellContent = useCallback(
    ([column, row]: Item): GridCell => {
      const entry = rows[row];
      const text = (() => {
        if (!entry) return "";
        if (column === 0) return entry.timeLabel;
        if (column === 1) return entry.senderName;
        if (column === 2) return entry.direction === "out" ? "out" : "in";
        if (column === 3) return entry.bodyLine;
        return entry.attachmentCount ? String(entry.attachmentCount) : "";
      })();
      return {
        kind: GridCellKind.Text,
        data: text,
        displayData: text,
        allowOverlay: false,
        readonly: true,
      };
    },
    [rows],
  );

  const selection = useMemo<GridSelection>(
    () => ({
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
      current:
        selectedIndex >= 0 && selectedIndex < rows.length
          ? {
              cell: [0, selectedIndex] as Item,
              range: { x: 0, y: selectedIndex, width: COLUMNS.length, height: 1 },
              rangeStack: [],
            }
          : undefined,
    }),
    [rows.length, selectedIndex],
  );

  const handleSelectionChange = useCallback(
    (next: GridSelection) => {
      const cell = next.current?.cell;
      if (!cell) return;
      onSelectIndex(cell[1]);
    },
    [onSelectIndex],
  );

  return (
    <div ref={setHost} className="h-full min-h-0 w-full" data-testid="message-browser-grid">
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
          rowMarkers="number"
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
